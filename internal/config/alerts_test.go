package config

import (
	"testing"
)

func TestParsePeriod(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasError bool
	}{
		// Days
		{"7d", 7, false},
		{"30d", 30, false},
		{"90d", 90, false},
		{"365d", 365, false},

		// Weeks
		{"1w", 7, false},
		{"2w", 14, false},
		{"4w", 28, false},

		// Months
		{"1m", 30, false},
		{"3m", 90, false},
		{"6m", 180, false},
		{"12m", 360, false},

		// Years
		{"1y", 365, false},
		{"2y", 730, false},

		// Case insensitive
		{"7D", 7, false},
		{"1M", 30, false},
		{"1Y", 365, false},

		// With whitespace
		{" 7d ", 7, false},

		// Invalid formats
		{"", 0, true},
		{"7", 0, true},
		{"d", 0, true},
		{"7x", 0, true},
		{"-7d", 0, true},
		{"0d", 0, true},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParsePeriod(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("ParsePeriod(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("ParsePeriod(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("ParsePeriod(%q) = %d, expected %d", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestFormatPeriod(t *testing.T) {
	tests := []struct {
		days     int
		expected string
	}{
		{7, "1w"},
		{14, "2w"},
		{30, "1m"},
		{60, "2m"},
		{90, "3m"},
		{365, "1y"},
		{730, "2y"},
		{10, "10d"},   // Not divisible by week
		{45, "45d"},   // Not divisible by month
		{100, "100d"}, // Not divisible by anything
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatPeriod(tt.days)
			if result != tt.expected {
				t.Errorf("FormatPeriod(%d) = %q, expected %q", tt.days, result, tt.expected)
			}
		})
	}
}

func TestValidateAlert(t *testing.T) {
	tests := []struct {
		name     string
		alert    AlertConfig
		hasError bool
	}{
		{
			name: "valid alert with days",
			alert: AlertConfig{
				Name:   "Test",
				Symbol: "IBM",
				Condition: AlertCondition{
					Type:      "percent_change",
					Threshold: -5.0,
					Period:    "7d",
				},
				Enabled: true,
			},
			hasError: false,
		},
		{
			name: "valid alert with months",
			alert: AlertConfig{
				Name:   "Test",
				Symbol: "IBM",
				Condition: AlertCondition{
					Type:      "percent_change",
					Threshold: -5.0,
					Period:    "3m",
				},
				Enabled: true,
			},
			hasError: false,
		},
		{
			name: "valid alert with year",
			alert: AlertConfig{
				Name:   "Test",
				Symbol: "IBM",
				Condition: AlertCondition{
					Type:      "percent_change",
					Threshold: -20.0,
					Period:    "1y",
				},
				Enabled: true,
			},
			hasError: false,
		},
		{
			name: "missing name",
			alert: AlertConfig{
				Symbol: "IBM",
				Condition: AlertCondition{
					Type:      "percent_change",
					Threshold: -5.0,
					Period:    "7d",
				},
			},
			hasError: true,
		},
		{
			name: "invalid period",
			alert: AlertConfig{
				Name:   "Test",
				Symbol: "IBM",
				Condition: AlertCondition{
					Type:      "percent_change",
					Threshold: -5.0,
					Period:    "invalid",
				},
			},
			hasError: true,
		},
		{
			name: "valid calculation method from_peak",
			alert: AlertConfig{
				Name:   "Test",
				Symbol: "IBM",
				Condition: AlertCondition{
					Type:              "percent_change",
					Threshold:         -5.0,
					Period:            "7d",
					CalculationMethod: "from_peak",
				},
			},
			hasError: false,
		},
		{
			name: "valid calculation method period_to_period",
			alert: AlertConfig{
				Name:   "Test",
				Symbol: "IBM",
				Condition: AlertCondition{
					Type:              "percent_change",
					Threshold:         -5.0,
					Period:            "7d",
					CalculationMethod: "period_to_period",
				},
			},
			hasError: false,
		},
		{
			name: "invalid calculation method",
			alert: AlertConfig{
				Name:   "Test",
				Symbol: "IBM",
				Condition: AlertCondition{
					Type:              "percent_change",
					Threshold:         -5.0,
					Period:            "7d",
					CalculationMethod: "invalid_method",
				},
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAlert(&tt.alert)
			if tt.hasError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.hasError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
