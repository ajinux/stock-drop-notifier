package alert

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajinux/stock-drop-notifier/internal/alphavantage"
)

func mockServer(response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
}

func TestEngine_EvaluateAlert_DropTriggered(t *testing.T) {
	// Mock response with 10% drop over 7 days
	server := mockServer(`{
		"Meta Data": {
			"1. Information": "Daily Prices",
			"2. Symbol": "IXIC",
			"3. Last Refreshed": "2024-01-15",
			"4. Output Size": "Compact",
			"5. Time Zone": "US/Eastern"
		},
		"Time Series (Daily)": {
			"2024-01-15": {"1. open": "90.00", "2. high": "91.00", "3. low": "89.00", "4. close": "90.00", "5. volume": "1000000"},
			"2024-01-08": {"1. open": "100.00", "2. high": "101.00", "3. low": "99.00", "4. close": "100.00", "5. volume": "1000000"}
		}
	}`)
	defer server.Close()

	client := alphavantage.NewClient("test-key").WithBaseURL(server.URL)
	engine := NewEngine(client)

	alert := Alert{
		Name:      "Test Drop",
		Symbol:    "IXIC",
		Threshold: -5.0, // Alert if drops more than 5%
		Period:    7,
		Enabled:   true,
	}

	result := engine.EvaluateAlert(alert)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}

	if !result.Triggered {
		t.Error("expected alert to be triggered (10% drop > 5% threshold)")
	}

	// Percent change should be around -10%
	if result.PercentChange > -9 || result.PercentChange < -11 {
		t.Errorf("expected ~-10%% change, got %f%%", result.PercentChange)
	}
}

func TestEngine_EvaluateAlert_DropNotTriggered(t *testing.T) {
	// Mock response with 3% drop over 7 days
	server := mockServer(`{
		"Meta Data": {
			"1. Information": "Daily Prices",
			"2. Symbol": "IXIC",
			"3. Last Refreshed": "2024-01-15",
			"4. Output Size": "Compact",
			"5. Time Zone": "US/Eastern"
		},
		"Time Series (Daily)": {
			"2024-01-15": {"1. open": "97.00", "2. high": "98.00", "3. low": "96.00", "4. close": "97.00", "5. volume": "1000000"},
			"2024-01-08": {"1. open": "100.00", "2. high": "101.00", "3. low": "99.00", "4. close": "100.00", "5. volume": "1000000"}
		}
	}`)
	defer server.Close()

	client := alphavantage.NewClient("test-key").WithBaseURL(server.URL)
	engine := NewEngine(client)

	alert := Alert{
		Name:      "Test Drop",
		Symbol:    "IXIC",
		Threshold: -5.0, // Alert if drops more than 5%
		Period:    7,
		Enabled:   true,
	}

	result := engine.EvaluateAlert(alert)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}

	if result.Triggered {
		t.Error("expected alert NOT to be triggered (3% drop < 5% threshold)")
	}
}

func TestEngine_EvaluateAlert_GainTriggered(t *testing.T) {
	// Mock response with 15% gain over 7 days
	server := mockServer(`{
		"Meta Data": {
			"1. Information": "Daily Prices",
			"2. Symbol": "AAPL",
			"3. Last Refreshed": "2024-01-15",
			"4. Output Size": "Compact",
			"5. Time Zone": "US/Eastern"
		},
		"Time Series (Daily)": {
			"2024-01-15": {"1. open": "115.00", "2. high": "116.00", "3. low": "114.00", "4. close": "115.00", "5. volume": "1000000"},
			"2024-01-08": {"1. open": "100.00", "2. high": "101.00", "3. low": "99.00", "4. close": "100.00", "5. volume": "1000000"}
		}
	}`)
	defer server.Close()

	client := alphavantage.NewClient("test-key").WithBaseURL(server.URL)
	engine := NewEngine(client)

	alert := Alert{
		Name:      "Test Gain",
		Symbol:    "AAPL",
		Threshold: 10.0, // Alert if gains more than 10%
		Period:    7,
		Enabled:   true,
	}

	result := engine.EvaluateAlert(alert)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}

	if !result.Triggered {
		t.Error("expected alert to be triggered (15% gain > 10% threshold)")
	}
}

func TestEngine_EvaluateAll_SkipsDisabled(t *testing.T) {
	server := mockServer(`{
		"Meta Data": {
			"1. Information": "Daily Prices",
			"2. Symbol": "IBM",
			"3. Last Refreshed": "2024-01-15",
			"4. Output Size": "Compact",
			"5. Time Zone": "US/Eastern"
		},
		"Time Series (Daily)": {
			"2024-01-15": {"1. open": "100.00", "2. high": "101.00", "3. low": "99.00", "4. close": "100.00", "5. volume": "1000000"}
		}
	}`)
	defer server.Close()

	client := alphavantage.NewClient("test-key").WithBaseURL(server.URL)
	engine := NewEngine(client).WithRateLimitDelay(0) // Disable delay for tests

	alerts := []Alert{
		{Name: "Enabled", Symbol: "IBM", Threshold: -5.0, Period: 7, Enabled: true},
		{Name: "Disabled", Symbol: "IBM", Threshold: -5.0, Period: 7, Enabled: false},
	}

	summary := engine.EvaluateAll(alerts)

	if summary.TotalAlerts != 2 {
		t.Errorf("expected 2 total alerts, got %d", summary.TotalAlerts)
	}

	if summary.SkippedAlerts != 1 {
		t.Errorf("expected 1 skipped alert, got %d", summary.SkippedAlerts)
	}

	if len(summary.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(summary.Results))
	}
}

func TestEngine_EvaluateAll_ContinuesOnError(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		// First request fails, second succeeds
		if requestCount == 1 {
			w.Write([]byte(`{"Error Message": "Invalid symbol"}`))
		} else {
			w.Write([]byte(`{
				"Meta Data": {"1. Information": "Daily", "2. Symbol": "IBM", "3. Last Refreshed": "2024-01-15", "4. Output Size": "Compact", "5. Time Zone": "US/Eastern"},
				"Time Series (Daily)": {
					"2024-01-15": {"1. open": "100.00", "2. high": "101.00", "3. low": "99.00", "4. close": "100.00", "5. volume": "1000000"},
					"2024-01-08": {"1. open": "100.00", "2. high": "101.00", "3. low": "99.00", "4. close": "100.00", "5. volume": "1000000"}
				}
			}`))
		}
	}))
	defer server.Close()

	client := alphavantage.NewClient("test-key").WithBaseURL(server.URL)
	engine := NewEngine(client).WithRateLimitDelay(0) // Disable delay for tests

	alerts := []Alert{
		{Name: "Invalid", Symbol: "INVALID", Threshold: -5.0, Period: 7, Enabled: true},
		{Name: "Valid", Symbol: "IBM", Threshold: -5.0, Period: 7, Enabled: true},
	}

	summary := engine.EvaluateAll(alerts)

	if summary.ErrorAlerts != 1 {
		t.Errorf("expected 1 error alert, got %d", summary.ErrorAlerts)
	}

	if len(summary.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(summary.Results))
	}

	// First should have error
	if summary.Results[0].Error == nil {
		t.Error("expected first result to have error")
	}

	// Second should not have error
	if summary.Results[1].Error != nil {
		t.Errorf("expected second result to succeed, got error: %v", summary.Results[1].Error)
	}
}

func TestAlertResult_Message(t *testing.T) {
	result := AlertResult{
		Alert: Alert{
			Name:      "NASDAQ Drop",
			Symbol:    "IXIC",
			Threshold: -5.0,
			Period:    7,
		},
		Triggered:     true,
		PercentChange: -6.5,
	}

	msg := result.Message()
	if msg == "" {
		t.Error("expected non-empty message")
	}

	// Should contain key info
	if !containsAll(msg, "NASDAQ Drop", "IXIC", "down", "6.50%", "7 days") {
		t.Errorf("message missing expected content: %s", msg)
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func TestEngine_EvaluateAlert_CalculationMethods(t *testing.T) {
	// Stock starts at 100, peaks at 120, ends at 105
	server := mockServer(`{
		"Meta Data": {
			"1. Information": "Daily Prices",
			"2. Symbol": "AAPL",
			"3. Last Refreshed": "2024-01-15",
			"4. Output Size": "Compact",
			"5. Time Zone": "US/Eastern"
		},
		"Time Series (Daily)": {
			"2024-01-15": {"1. open": "105.00", "2. high": "106.00", "3. low": "104.00", "4. close": "105.00", "5. volume": "1000000"},
			"2024-01-12": {"1. open": "120.00", "2. high": "121.00", "3. low": "119.00", "4. close": "120.00", "5. volume": "1000000"},
			"2024-01-08": {"1. open": "100.00", "2. high": "101.00", "3. low": "99.00", "4. close": "100.00", "5. volume": "1000000"}
		}
	}`)
	defer server.Close()

	client := alphavantage.NewClient("test-key").WithBaseURL(server.URL)
	engine := NewEngine(client)

	// Case 1: from_peak (default/explicit)
	// Peak is 120, current is 105. Drop = (105 - 120)/120 = -12.5%
	alertFromPeak := Alert{
		Name:              "From Peak Drop",
		Symbol:            "AAPL",
		Threshold:         -10.0,
		Period:            7,
		CalculationMethod: "from_peak",
		Enabled:           true,
	}

	resultFromPeak := engine.EvaluateAlert(alertFromPeak)
	if resultFromPeak.Error != nil {
		t.Fatalf("unexpected error: %v", resultFromPeak.Error)
	}
	if !resultFromPeak.Triggered {
		t.Error("expected from_peak alert to trigger (12.5% drop > 10% threshold)")
	}
	if resultFromPeak.ExtremePrice != 120.0 {
		t.Errorf("expected extreme price to be 120.0, got %f", resultFromPeak.ExtremePrice)
	}

	// Case 2: period_to_period
	// Start is 100, current is 105. Change = +5% (no drop)
	alertPeriodToPeriod := Alert{
		Name:              "Period to Period Drop",
		Symbol:            "AAPL",
		Threshold:         -10.0,
		Period:            7,
		CalculationMethod: "period_to_period",
		Enabled:           true,
	}

	resultPeriodToPeriod := engine.EvaluateAlert(alertPeriodToPeriod)
	if resultPeriodToPeriod.Error != nil {
		t.Fatalf("unexpected error: %v", resultPeriodToPeriod.Error)
	}
	if resultPeriodToPeriod.Triggered {
		t.Error("expected period_to_period alert NOT to trigger (price went up overall)")
	}
}
