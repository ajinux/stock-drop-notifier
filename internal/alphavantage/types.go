package alphavantage

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DailyPrice represents a single day's OHLCV data
type DailyPrice struct {
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

// TimeSeriesResponse represents the parsed response from Alpha Vantage
type TimeSeriesResponse struct {
	MetaData   MetaData
	TimeSeries []DailyPrice
}

// MetaData contains metadata about the time series
type MetaData struct {
	Information   string
	Symbol        string
	LastRefreshed string
	OutputSize    string
	TimeZone      string
}

// rawTimeSeriesResponse is the raw JSON structure from Alpha Vantage
type rawTimeSeriesResponse struct {
	MetaData     rawMetaData         `json:"Meta Data"`
	TimeSeries   map[string]rawOHLCV `json:"Time Series (Daily)"`
	ErrorMessage string              `json:"Error Message"`
	Note         string              `json:"Note"`
	Information  string              `json:"Information"`
}

type rawMetaData struct {
	Information   string `json:"1. Information"`
	Symbol        string `json:"2. Symbol"`
	LastRefreshed string `json:"3. Last Refreshed"`
	OutputSize    string `json:"4. Output Size"`
	TimeZone      string `json:"5. Time Zone"`
}

type rawOHLCV struct {
	Open   string `json:"1. open"`
	High   string `json:"2. high"`
	Low    string `json:"3. low"`
	Close  string `json:"4. close"`
	Volume string `json:"5. volume"`
}

// ParseTimeSeriesResponse parses the raw JSON response into a structured format
func ParseTimeSeriesResponse(data []byte) (*TimeSeriesResponse, error) {
	var raw rawTimeSeriesResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Check for API errors
	if raw.ErrorMessage != "" {
		return nil, fmt.Errorf("API error: %s", raw.ErrorMessage)
	}

	// Check for rate limit message
	if raw.Note != "" {
		return nil, &RateLimitError{Message: raw.Note}
	}

	// Check for information message (usually indicates an issue)
	if raw.Information != "" && raw.TimeSeries == nil {
		// Check for premium feature required
		if strings.Contains(raw.Information, "premium") {
			return nil, &PremiumRequiredError{
				Feature: "full historical data",
				Message: "periods longer than ~4 months require a premium Alpha Vantage subscription",
			}
		}
		return nil, fmt.Errorf("API information: %s", raw.Information)
	}

	// Parse time series data
	var timeSeries []DailyPrice
	for dateStr, ohlcv := range raw.TimeSeries {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse date %s: %w", dateStr, err)
		}

		open, err := strconv.ParseFloat(ohlcv.Open, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse open price: %w", err)
		}

		high, err := strconv.ParseFloat(ohlcv.High, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse high price: %w", err)
		}

		low, err := strconv.ParseFloat(ohlcv.Low, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse low price: %w", err)
		}

		closePrice, err := strconv.ParseFloat(ohlcv.Close, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse close price: %w", err)
		}

		volume, err := strconv.ParseInt(ohlcv.Volume, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse volume: %w", err)
		}

		timeSeries = append(timeSeries, DailyPrice{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closePrice,
			Volume: volume,
		})
	}

	// Sort by date descending (most recent first)
	sort.Slice(timeSeries, func(i, j int) bool {
		return timeSeries[i].Date.After(timeSeries[j].Date)
	})

	return &TimeSeriesResponse{
		MetaData: MetaData{
			Information:   raw.MetaData.Information,
			Symbol:        raw.MetaData.Symbol,
			LastRefreshed: raw.MetaData.LastRefreshed,
			OutputSize:    raw.MetaData.OutputSize,
			TimeZone:      raw.MetaData.TimeZone,
		},
		TimeSeries: timeSeries,
	}, nil
}

// RateLimitError represents an API rate limit error
type RateLimitError struct {
	Message string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded: %s", e.Message)
}

// SymbolNotFoundError represents an invalid symbol error
type SymbolNotFoundError struct {
	Symbol string
}

func (e *SymbolNotFoundError) Error() string {
	return fmt.Sprintf("symbol not found: %s", e.Symbol)
}

// PremiumRequiredError represents an error when a premium feature is needed
type PremiumRequiredError struct {
	Feature string
	Message string
}

func (e *PremiumRequiredError) Error() string {
	return fmt.Sprintf("premium subscription required for %s: %s", e.Feature, e.Message)
}
