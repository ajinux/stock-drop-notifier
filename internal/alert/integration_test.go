package alert_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajinux/stock-drop-notifier/internal/alert"
	"github.com/ajinux/stock-drop-notifier/internal/alphavantage"
	"github.com/ajinux/stock-drop-notifier/internal/config"
	"github.com/ajinux/stock-drop-notifier/internal/telegram"
)

// TestIntegration_FullCheckWorkflow tests the complete workflow:
// 1. Load alerts from config
// 2. Fetch stock data
// 3. Evaluate alerts
// 4. Send notifications
func TestIntegration_FullCheckWorkflow(t *testing.T) {
	// Mock Alpha Vantage server
	avServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		w.Header().Set("Content-Type", "application/json")

		// Return different responses based on symbol
		if symbol == "IXIC" {
			// NASDAQ with 6% drop - should trigger alert
			w.Write([]byte(`{
				"Meta Data": {
					"1. Information": "Daily Prices",
					"2. Symbol": "IXIC",
					"3. Last Refreshed": "2024-01-15",
					"4. Output Size": "Compact",
					"5. Time Zone": "US/Eastern"
				},
				"Time Series (Daily)": {
					"2024-01-15": {"1. open": "94.00", "2. high": "95.00", "3. low": "93.00", "4. close": "94.00", "5. volume": "1000000"},
					"2024-01-08": {"1. open": "100.00", "2. high": "101.00", "3. low": "99.00", "4. close": "100.00", "5. volume": "1000000"}
				}
			}`))
		} else if symbol == "AAPL" {
			// AAPL with 3% drop - should NOT trigger alert
			w.Write([]byte(`{
				"Meta Data": {
					"1. Information": "Daily Prices",
					"2. Symbol": "AAPL",
					"3. Last Refreshed": "2024-01-15",
					"4. Output Size": "Compact",
					"5. Time Zone": "US/Eastern"
				},
				"Time Series (Daily)": {
					"2024-01-15": {"1. open": "97.00", "2. high": "98.00", "3. low": "96.00", "4. close": "97.00", "5. volume": "1000000"},
					"2024-01-08": {"1. open": "100.00", "2. high": "101.00", "3. low": "99.00", "4. close": "100.00", "5. volume": "1000000"}
				}
			}`))
		}
	}))
	defer avServer.Close()

	// Mock Telegram server
	var notificationsSent []string
	tgServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notificationsSent = append(notificationsSent, "notification")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok": true}`))
	}))
	defer tgServer.Close()

	// Create alert configurations
	alertConfigs := []config.AlertConfig{
		{
			Name:   "NASDAQ Drop",
			Symbol: "IXIC",
			Condition: config.AlertCondition{
				Type:      "percent_change",
				Threshold: -5.0,
				Period:    "7d",
			},
			Enabled: true,
		},
		{
			Name:   "AAPL Drop",
			Symbol: "AAPL",
			Condition: config.AlertCondition{
				Type:      "percent_change",
				Threshold: -5.0,
				Period:    "7d",
			},
			Enabled: true,
		},
	}

	// Create clients with mock servers
	avClient := alphavantage.NewClient("test-key").WithBaseURL(avServer.URL)
	tgClient := telegram.NewClient("test-token", 12345).WithBaseURL(tgServer.URL)

	// Create alert engine and evaluate (disable rate limit delay for tests)
	engine := alert.NewEngine(avClient).WithRateLimitDelay(0)
	summary := engine.EvaluateFromConfig(alertConfigs)

	// Verify evaluation results
	if summary.TotalAlerts != 2 {
		t.Errorf("expected 2 total alerts, got %d", summary.TotalAlerts)
	}

	if summary.TriggeredAlerts != 1 {
		t.Errorf("expected 1 triggered alert, got %d", summary.TriggeredAlerts)
	}

	if summary.ErrorAlerts != 0 {
		t.Errorf("expected 0 error alerts, got %d", summary.ErrorAlerts)
	}

	// Send notifications for triggered alerts
	triggeredResults := summary.GetTriggeredResults()
	if len(triggeredResults) != 1 {
		t.Errorf("expected 1 triggered result, got %d", len(triggeredResults))
	}

	// Verify the triggered alert is NASDAQ
	if triggeredResults[0].Alert.Symbol != "IXIC" {
		t.Errorf("expected IXIC to be triggered, got %s", triggeredResults[0].Alert.Symbol)
	}

	// Send notifications
	var messages []string
	for _, r := range triggeredResults {
		messages = append(messages, r.TelegramMessage())
	}

	sent, errors := tgClient.SendBatch(messages)
	if sent != 1 {
		t.Errorf("expected 1 message sent, got %d", sent)
	}

	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errors))
	}

	if len(notificationsSent) != 1 {
		t.Errorf("expected 1 notification to be sent, got %d", len(notificationsSent))
	}
}

// TestIntegration_DisabledAlertsSkipped tests that disabled alerts are not evaluated
func TestIntegration_DisabledAlertsSkipped(t *testing.T) {
	requestCount := 0
	avServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"Meta Data": {"1. Information": "Daily", "2. Symbol": "IBM", "3. Last Refreshed": "2024-01-15", "4. Output Size": "Compact", "5. Time Zone": "US/Eastern"},
			"Time Series (Daily)": {
				"2024-01-15": {"1. open": "100.00", "2. high": "101.00", "3. low": "99.00", "4. close": "100.00", "5. volume": "1000000"},
				"2024-01-08": {"1. open": "100.00", "2. high": "101.00", "3. low": "99.00", "4. close": "100.00", "5. volume": "1000000"}
			}
		}`))
	}))
	defer avServer.Close()

	alertConfigs := []config.AlertConfig{
		{Name: "Enabled", Symbol: "IBM", Condition: config.AlertCondition{Type: "percent_change", Threshold: -5.0, Period: "7d"}, Enabled: true},
		{Name: "Disabled", Symbol: "IBM", Condition: config.AlertCondition{Type: "percent_change", Threshold: -5.0, Period: "7d"}, Enabled: false},
	}

	avClient := alphavantage.NewClient("test-key").WithBaseURL(avServer.URL)
	engine := alert.NewEngine(avClient).WithRateLimitDelay(0) // Disable delay for tests
	summary := engine.EvaluateFromConfig(alertConfigs)

	// Only 1 API call should be made (for enabled alert)
	if requestCount != 1 {
		t.Errorf("expected 1 API call, got %d", requestCount)
	}

	if summary.SkippedAlerts != 1 {
		t.Errorf("expected 1 skipped alert, got %d", summary.SkippedAlerts)
	}
}
