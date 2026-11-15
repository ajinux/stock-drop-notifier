package alert

import (
	"strings"
	"testing"
	"time"
)

func TestCurrencySymbol(t *testing.T) {
	cases := map[string]string{
		"AAPL":          "$",
		"SPY":           "$",
		"NIFTYBEES.BSE": "₹",
		"reliance.bse":  "₹",
		"INFY.NSE":      "₹",
	}
	for symbol, want := range cases {
		if got := CurrencySymbol(symbol); got != want {
			t.Errorf("CurrencySymbol(%q) = %q, want %q", symbol, got, want)
		}
	}
}

func TestTelegramMessageUsesRupeesForIndianSymbols(t *testing.T) {
	result := &AlertResult{
		Alert: Alert{
			Name:              "Nifty 50 Monthly Drop",
			Symbol:            "NIFTYBEES.BSE",
			Threshold:         -5,
			Period:            30,
			CalculationMethod: "from_peak",
		},
		Triggered:     true,
		CurrentPrice:  270.5,
		ExtremePrice:  290,
		PercentChange: -6.72,
		EvaluatedAt:   time.Now(),
	}

	for _, msg := range []string{result.TelegramMessage(), result.Message()} {
		if !strings.Contains(msg, "₹290.00") {
			t.Errorf("expected rupee-denominated peak price, got: %s", msg)
		}
		if strings.Contains(msg, "$") {
			t.Errorf("expected no dollar sign for an Indian symbol, got: %s", msg)
		}
	}
}
