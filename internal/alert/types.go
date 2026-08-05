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

	direction := "up"
	if r.PercentChange < 0 {
		direction = "down"
	}

	if r.Alert.CalculationMethod == "from_peak" {
		extremeName := "peak"
		if r.Alert.Threshold >= 0 {
			extremeName = "trough"
		}
		return fmt.Sprintf("%s (%s) is %s %.2f%% from %s ($%.2f) over the last %d days (threshold: %.2f%%)",
			r.Alert.Name, r.Alert.Symbol, direction, absFloat(r.PercentChange), extremeName, r.ExtremePrice, r.Alert.Period, r.Alert.Threshold)
	}

	return fmt.Sprintf("%s (%s) is %s %.2f%% over the last %d days (threshold: %.2f%%)",
		r.Alert.Name, r.Alert.Symbol, direction, absFloat(r.PercentChange), r.Alert.Period, r.Alert.Threshold)
}

// TelegramMessage generates a Markdown-formatted message for Telegram
func (r *AlertResult) TelegramMessage() string {
	if r.Error != nil {
		return fmt.Sprintf("❌ *Error* evaluating alert *%s*:\n%v", r.Alert.Name, r.Error)
	}

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
			"*%s* (%s) is %s *%.2f%%* from %s ($%.2f) over the last %d days\n\n"+
			"• Current: $%.2f\n"+
			"• %s Price: $%.2f\n"+
			"• Threshold: %.2f%%\n"+
			"• Time: %s",
			emoji, r.Alert.Name,
			r.Alert.Name, r.Alert.Symbol, direction, absFloat(r.PercentChange), strings.ToLower(extremeName), r.ExtremePrice, r.Alert.Period,
			r.CurrentPrice,
			extremeName, r.ExtremePrice,
			r.Alert.Threshold,
			r.EvaluatedAt.Format("2006-01-02 15:04:05 MST"))
	}

	return fmt.Sprintf("%s *%s* Alert\n\n"+
		"*%s* (%s) is %s *%.2f%%* over the last %d days\n\n"+
		"• Current: $%.2f\n"+
		"• %d days ago: $%.2f\n"+
		"• Threshold: %.2f%%\n"+
		"• Time: %s",
		emoji, r.Alert.Name,
		r.Alert.Name, r.Alert.Symbol, direction, absFloat(r.PercentChange), r.Alert.Period,
		r.CurrentPrice,
		r.Alert.Period, r.PastPrice,
		r.Alert.Threshold,
		r.EvaluatedAt.Format("2006-01-02 15:04:05 MST"))
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
