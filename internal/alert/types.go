package alert

import (
	"fmt"
	"strings"
	"time"
)

// Alert represents an alert configuration for evaluation
type Alert struct {
	Name              string
	Symbol            string
	Threshold         float64 // Percentage threshold (negative for drops, positive for gains)
	Period            int     // Period in days
	CalculationMethod string  // "from_peak" or "period_to_period"
	Enabled           bool
}

// AlertResult represents the result of evaluating an alert
type AlertResult struct {
	Alert         Alert
	Triggered     bool
	CurrentPrice  float64
	PastPrice     float64
	ExtremePrice  float64 // The peak or trough price if using "from_peak"
	PercentChange float64
	EvaluatedAt   time.Time
	Error         error
}

// IsError returns true if the alert evaluation resulted in an error
func (r *AlertResult) IsError() bool {
	return r.Error != nil
}

// Message generates a human-readable message for the alert result
func (r *AlertResult) Message() string {
	if r.Error != nil {
		return fmt.Sprintf("Error evaluating alert '%s': %v", r.Alert.Name, r.Error)
	}

	if !r.Triggered {
		return fmt.Sprintf("Alert '%s' not triggered: %s is %.2f%% (threshold: %.2f%%)",
			r.Alert.Name, r.Alert.Symbol, r.PercentChange, r.Alert.Threshold)
	}

	cur := CurrencySymbol(r.Alert.Symbol)
	direction := "up"
	if r.PercentChange < 0 {
		direction = "down"
	}

	if r.Alert.CalculationMethod == "from_peak" {
		extremeName := "peak"
		if r.Alert.Threshold >= 0 {
			extremeName = "trough"
		}
		return fmt.Sprintf("%s (%s) is %s %.2f%% from %s (%s%.2f) over the last %d days (threshold: %.2f%%)",
			r.Alert.Name, r.Alert.Symbol, direction, absFloat(r.PercentChange), extremeName, cur, r.ExtremePrice, r.Alert.Period, r.Alert.Threshold)
	}

	return fmt.Sprintf("%s (%s) is %s %.2f%% over the last %d days (threshold: %.2f%%)",
		r.Alert.Name, r.Alert.Symbol, direction, absFloat(r.PercentChange), r.Alert.Period, r.Alert.Threshold)
}

// TelegramMessage generates a Markdown-formatted message for Telegram
func (r *AlertResult) TelegramMessage() string {
	if r.Error != nil {
		return fmt.Sprintf("❌ *Error* evaluating alert *%s*:\n%v", r.Alert.Name, r.Error)
	}

	cur := CurrencySymbol(r.Alert.Symbol)
	direction := "📈 up"
	emoji := "🚀"
	if r.PercentChange < 0 {
		direction = "📉 down"
		emoji = "⚠️"
	}

	if r.Alert.CalculationMethod == "from_peak" {
		extremeName := "Peak"
		if r.Alert.Threshold >= 0 {
			extremeName = "Trough"
		}
		return fmt.Sprintf("%s *%s* Alert\n\n"+
			"*%s* (%s) is %s *%.2f%%* from %s (%s%.2f) over the last %d days\n\n"+
			"• Current: %s%.2f\n"+
			"• %s Price: %s%.2f\n"+
			"• Threshold: %.2f%%\n"+
			"• Time: %s",
			emoji, r.Alert.Name,
			r.Alert.Name, r.Alert.Symbol, direction, absFloat(r.PercentChange), strings.ToLower(extremeName), cur, r.ExtremePrice, r.Alert.Period,
			cur, r.CurrentPrice,
			extremeName, cur, r.ExtremePrice,
			r.Alert.Threshold,
			r.EvaluatedAt.Format("2006-01-02 15:04:05 MST"))
	}

	return fmt.Sprintf("%s *%s* Alert\n\n"+
		"*%s* (%s) is %s *%.2f%%* over the last %d days\n\n"+
		"• Current: %s%.2f\n"+
		"• %d days ago: %s%.2f\n"+
		"• Threshold: %.2f%%\n"+
		"• Time: %s",
		emoji, r.Alert.Name,
		r.Alert.Name, r.Alert.Symbol, direction, absFloat(r.PercentChange), r.Alert.Period,
		cur, r.CurrentPrice,
		r.Alert.Period, cur, r.PastPrice,
		r.Alert.Threshold,
		r.EvaluatedAt.Format("2006-01-02 15:04:05 MST"))
}

// CurrencySymbol returns the currency prefix for a ticker's prices. Alpha Vantage
// quotes Indian listings (".BSE" / ".NSE" suffix) in rupees; everything else this
// tool tracks is quoted in US dollars.
func CurrencySymbol(symbol string) string {
	upper := strings.ToUpper(symbol)
	if strings.HasSuffix(upper, ".BSE") || strings.HasSuffix(upper, ".NSE") {
		return "₹"
	}
	return "$"
}

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// EvaluationSummary represents a summary of all alert evaluations
type EvaluationSummary struct {
	TotalAlerts     int
	TriggeredAlerts int
	SkippedAlerts   int
	ErrorAlerts     int
	APIRequests     int // Network fetches made; lower than alerts evaluated when symbols repeat
	Results         []AlertResult
}

// GetTriggeredResults returns only the results where alerts were triggered
func (s *EvaluationSummary) GetTriggeredResults() []AlertResult {
	var triggered []AlertResult
	for _, r := range s.Results {
		if r.Triggered && !r.IsError() {
			triggered = append(triggered, r)
		}
	}
	return triggered
}

// GetNotifiableResults returns the triggered results that should be sent as
// notifications. When several triggered alerts are tiers of the same check
// (same symbol, period, calculation method and direction, e.g. SPY at -10%,
// -12% and -15%), only the deepest triggered tier is kept — its message already
// carries the actual percentage move, so the shallower tiers add nothing.
// Order follows the first appearance of each group.
func (s *EvaluationSummary) GetNotifiableResults() []AlertResult {
	type groupKey struct {
		symbol string
		period int
		method string
		drop   bool
	}
	var order []groupKey
	best := make(map[groupKey]AlertResult)
	for _, r := range s.GetTriggeredResults() {
		method := r.Alert.CalculationMethod
		if method == "" {
			method = "from_peak"
		}
		k := groupKey{r.Alert.Symbol, r.Alert.Period, method, r.Alert.Threshold < 0}
		cur, seen := best[k]
		if !seen {
			order = append(order, k)
			best[k] = r
		} else if absFloat(r.Alert.Threshold) > absFloat(cur.Alert.Threshold) {
			best[k] = r
		}
	}
	out := make([]AlertResult, 0, len(order))
	for _, k := range order {
		out = append(out, best[k])
	}
	return out
}

// GetErrorResults returns only the results that had errors
func (s *EvaluationSummary) GetErrorResults() []AlertResult {
	var errors []AlertResult
	for _, r := range s.Results {
		if r.IsError() {
			errors = append(errors, r)
		}
	}
	return errors
}
