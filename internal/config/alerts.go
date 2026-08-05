package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// CalculationMethodFromPeak measures change from the period's peak/trough to current price (default)
	CalculationMethodFromPeak = "from_peak"
	// CalculationMethodPeriodToPeriod measures change from start of period to current price (original behavior)
	CalculationMethodPeriodToPeriod = "period_to_period"
)

// AlertCondition defines the condition that triggers an alert
type AlertCondition struct {
	Type              string  `yaml:"type"`               // "percent_change"
	Threshold         float64 `yaml:"threshold"`          // e.g., -5.0 for 5% drop, 10.0 for 10% gain
	Period            string  `yaml:"period"`             // e.g., "7d" for 7 days, "30d" for 30 days
	CalculationMethod string  `yaml:"calculation_method"` // "from_peak" (default) or "period_to_period"
}

// AlertConfig represents a single alert configuration
type AlertConfig struct {
	Name      string         `yaml:"name"`
	Symbol    string         `yaml:"symbol"`
	Condition AlertCondition `yaml:"condition"`
	Enabled   bool           `yaml:"enabled"`
}

// AlertsFile represents the alerts.yaml file structure
type AlertsFile struct {
	Alerts []AlertConfig `yaml:"alerts"`
}

// DefaultAlertsPath is the default path to the alerts configuration file
const DefaultAlertsPath = "alerts.yaml"

// LoadAlerts loads alert configurations from a YAML file
func LoadAlerts(path string) (*AlertsFile, error) {
	if path == "" {
		path = DefaultAlertsPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty alerts if file doesn't exist
			return &AlertsFile{Alerts: []AlertConfig{}}, nil
		}
		return nil, fmt.Errorf("failed to read alerts file: %w", err)
	}

	var alertsFile AlertsFile
	if err := yaml.Unmarshal(data, &alertsFile); err != nil {
		return nil, fmt.Errorf("failed to parse alerts file: %w", err)
	}

	// Validate all alerts
	for i, alert := range alertsFile.Alerts {
		if err := ValidateAlert(&alert); err != nil {
			return nil, fmt.Errorf("invalid alert at index %d: %w", i, err)
		}
	}

	return &alertsFile, nil
}

// SaveAlerts saves alert configurations to a YAML file
func SaveAlerts(path string, alerts *AlertsFile) error {
	if path == "" {
		path = DefaultAlertsPath
	}

	data, err := yaml.Marshal(alerts)
	if err != nil {
		return fmt.Errorf("failed to marshal alerts: %w", err)
	}

	// Add header comment
	header := "# Stock Market Alert Configuration\n"
	content := header + string(data)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write alerts file: %w", err)
	}

	return nil
}

// ValidateAlert validates a single alert configuration
func ValidateAlert(alert *AlertConfig) error {
	var errs []string

	if alert.Name == "" {
		errs = append(errs, "name is required")
	}

	if alert.Symbol == "" {
		errs = append(errs, "symbol is required")
	}

	if alert.Condition.Type == "" {
		errs = append(errs, "condition.type is required")
	} else if alert.Condition.Type != "percent_change" {
		errs = append(errs, fmt.Sprintf("unsupported condition type: %s", alert.Condition.Type))
	}

	if alert.Condition.Period == "" {
		errs = append(errs, "condition.period is required")
	} else if _, err := ParsePeriod(alert.Condition.Period); err != nil {
		errs = append(errs, fmt.Sprintf("invalid period format: %s", alert.Condition.Period))
	}

	if alert.Condition.CalculationMethod != "" &&
		alert.Condition.CalculationMethod != CalculationMethodFromPeak &&
		alert.Condition.CalculationMethod != CalculationMethodPeriodToPeriod {
		errs = append(errs, fmt.Sprintf("invalid calculation method: %s", alert.Condition.CalculationMethod))
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// ParsePeriod parses a period string into days
// Supported formats:
//   - "7d", "30d", "90d", "365d" - days
//   - "1w", "2w", "4w" - weeks (converted to days)
//   - "1m", "3m", "6m", "12m" - months (approximated as 30 days each)
//   - "1y", "2y" - years (approximated as 365 days each)
func ParsePeriod(period string) (int, error) {
	period = strings.ToLower(strings.TrimSpace(period))

	// Match patterns like "7d", "30d", "1w", "1m", "1y"
	re := regexp.MustCompile(`^(\d+)([dwmy])$`)
	matches := re.FindStringSubmatch(period)
	if matches == nil {
		return 0, fmt.Errorf("invalid period format: %s (expected format like '7d', '1w', '1m', or '1y')", period)
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}

	if value <= 0 {
		return 0, fmt.Errorf("period must be positive: %d", value)
	}

	unit := matches[2]
	var days int

	switch unit {
	case "d":
		days = value
	case "w":
		days = value * 7
	case "m":
		days = value * 30 // Approximate month as 30 days
	case "y":
		days = value * 365 // Approximate year as 365 days
	default:
		return 0, fmt.Errorf("unknown period unit: %s", unit)
	}

	return days, nil
}

// FormatPeriod converts days back to a human-readable period string
func FormatPeriod(days int) string {
	switch {
	case days >= 365 && days%365 == 0:
		return fmt.Sprintf("%dy", days/365)
	case days >= 30 && days%30 == 0:
		return fmt.Sprintf("%dm", days/30)
	case days >= 7 && days%7 == 0:
		return fmt.Sprintf("%dw", days/7)
	default:
		return fmt.Sprintf("%dd", days)
	}
}

// AddAlert adds a new alert to the alerts file
func (af *AlertsFile) AddAlert(alert AlertConfig) error {
	// Check for duplicate name
	for _, existing := range af.Alerts {
		if strings.EqualFold(existing.Name, alert.Name) {
			return fmt.Errorf("alert with name '%s' already exists", alert.Name)
		}
	}

	af.Alerts = append(af.Alerts, alert)
	return nil
}

// RemoveAlert removes an alert by name
func (af *AlertsFile) RemoveAlert(name string) error {
	for i, alert := range af.Alerts {
		if strings.EqualFold(alert.Name, name) {
			af.Alerts = append(af.Alerts[:i], af.Alerts[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("alert with name '%s' not found", name)
}

// GetEnabledAlerts returns only enabled alerts
func (af *AlertsFile) GetEnabledAlerts() []AlertConfig {
	var enabled []AlertConfig
	for _, alert := range af.Alerts {
		if alert.Enabled {
			enabled = append(enabled, alert)
		}
	}
	return enabled
}
