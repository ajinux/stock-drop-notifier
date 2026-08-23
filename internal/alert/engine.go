package alert

import (
	"fmt"
	"time"

	"github.com/ajinux/stock-drop-notifier/internal/alphavantage"
	"github.com/ajinux/stock-drop-notifier/internal/config"
)

// DefaultRateLimitDelay is the delay between API calls to avoid rate limiting
const DefaultRateLimitDelay = 1200 * time.Millisecond // 1.2 seconds

// Engine evaluates alert conditions against market data
type Engine struct {
	client         *alphavantage.Client
	rateLimitDelay time.Duration
}

// NewEngine creates a new alert evaluation engine
func NewEngine(client *alphavantage.Client) *Engine {
	return &Engine{
		client:         client,
		rateLimitDelay: DefaultRateLimitDelay,
	}
}

// WithRateLimitDelay sets a custom rate limit delay (useful for testing)
func (e *Engine) WithRateLimitDelay(delay time.Duration) *Engine {
	e.rateLimitDelay = delay
	return e
}

// EvaluateAlert evaluates a single alert and returns the result
func (e *Engine) EvaluateAlert(alert Alert) AlertResult {
	result := AlertResult{
		Alert:       alert,
		EvaluatedAt: time.Now(),
	}

	// Fetch time series data with appropriate output size for the period
	// For periods > 120 days, this will use "full" output to get enough historical data
	response, err := e.client.GetDailyTimeSeriesForPeriod(alert.Symbol, alert.Period)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch data for %s: %w", alert.Symbol, err)
		return result
	}

	var percentChange float64
	var extremePrice float64

	if alert.CalculationMethod == config.CalculationMethodFromPeak || alert.CalculationMethod == "" {
		// Use CalculateExtremePriceChange
		// isNegative is true if threshold < 0
		var err error
		percentChange, extremePrice, err = e.client.CalculateExtremePriceChange(response.TimeSeries, alert.Period, alert.Threshold < 0)
		if err != nil {
			result.Error = fmt.Errorf("failed to calculate price change: %w", err)
			return result
		}
		result.ExtremePrice = extremePrice
	} else {
		// Use CalculatePriceChange
		var err error
		percentChange, err = e.client.CalculatePriceChange(response.TimeSeries, alert.Period)
		if err != nil {
			result.Error = fmt.Errorf("failed to calculate price change: %w", err)
			return result
		}
	}

	// Get current and past prices
	if len(response.TimeSeries) > 0 {
		result.CurrentPrice = response.TimeSeries[0].Close
		if alert.CalculationMethod == config.CalculationMethodFromPeak || alert.CalculationMethod == "" {
			// Find the actual price from 'days' ago for the pastPrice field
			targetDate := response.TimeSeries[0].Date.AddDate(0, 0, -alert.Period)
			for _, price := range response.TimeSeries {
				if price.Date.Before(targetDate) || price.Date.Equal(targetDate) {
					result.PastPrice = price.Close
					break
				}
				result.PastPrice = price.Close
			}
		} else {
			result.PastPrice = result.CurrentPrice / (1 + percentChange/100)
		}
	}

	result.PercentChange = percentChange

	// Check if threshold is breached
	// For negative thresholds (drops): trigger if actual change <= threshold
	// For positive thresholds (gains): trigger if actual change >= threshold
	if alert.Threshold < 0 {
		result.Triggered = percentChange <= alert.Threshold
	} else {
		result.Triggered = percentChange >= alert.Threshold
	}

	return result
}

// EvaluateAll evaluates all alerts and returns a summary
func (e *Engine) EvaluateAll(alerts []Alert) EvaluationSummary {
	summary := EvaluationSummary{
		TotalAlerts: len(alerts),
		Results:     make([]AlertResult, 0, len(alerts)),
	}

	isFirstRequest := true
	for _, alert := range alerts {
		// Skip disabled alerts
		if !alert.Enabled {
			summary.SkippedAlerts++
			continue
		}

		// Add delay between API requests to avoid rate limiting (1 req/sec limit)
		// Skip delay for the first request
		if !isFirstRequest && e.rateLimitDelay > 0 {
			time.Sleep(e.rateLimitDelay)
		}
		isFirstRequest = false

		result := e.EvaluateAlert(alert)
		summary.Results = append(summary.Results, result)

		if result.IsError() {
			summary.ErrorAlerts++
		} else if result.Triggered {
			summary.TriggeredAlerts++
		}
	}

	return summary
}

// EvaluateFromConfig evaluates alerts from configuration
func (e *Engine) EvaluateFromConfig(alertConfigs []config.AlertConfig) EvaluationSummary {
	alerts := make([]Alert, len(alertConfigs))
	for i, cfg := range alertConfigs {
		period, _ := config.ParsePeriod(cfg.Condition.Period) // Already validated
		method := cfg.Condition.CalculationMethod
		if method == "" {
			method = config.CalculationMethodFromPeak
		}
		alerts[i] = Alert{
			Name:              cfg.Name,
			Symbol:            cfg.Symbol,
			Threshold:         cfg.Condition.Threshold,
			Period:            period,
			CalculationMethod: method,
			Enabled:           cfg.Enabled,
		}
	}
	return e.EvaluateAll(alerts)
}

// AlertFromConfig converts a config.AlertConfig to an Alert
func AlertFromConfig(cfg config.AlertConfig) (Alert, error) {
	period, err := config.ParsePeriod(cfg.Condition.Period)
	if err != nil {
		return Alert{}, err
	}

	method := cfg.Condition.CalculationMethod
	if method == "" {
		method = config.CalculationMethodFromPeak
	}

	return Alert{
		Name:              cfg.Name,
		Symbol:            cfg.Symbol,
		Threshold:         cfg.Condition.Threshold,
		Period:            period,
		CalculationMethod: method,
		Enabled:           cfg.Enabled,
	}, nil
}
