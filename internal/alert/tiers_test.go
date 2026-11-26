package alert

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ajinux/stock-drop-notifier/internal/alphavantage"
)

const tierSeries = `{
	"Meta Data": {"2. Symbol": "SPY"},
	"Time Series (Daily)": {
		"2024-01-15": {"1. open": "87.00", "2. high": "87.00", "3. low": "87.00", "4. close": "87.00", "5. volume": "1"},
		"2024-01-08": {"1. open": "100.00", "2. high": "100.00", "3. low": "100.00", "4. close": "100.00", "5. volume": "1"}
	}
}`

func TestEngine_EvaluateAll_SharesFetchAcrossSameSymbol(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Write([]byte(tierSeries))
	}))
	defer server.Close()

	engine := NewEngine(alphavantage.NewClient("k").WithBaseURL(server.URL)).WithRateLimitDelay(0)
	alerts := []Alert{
		{Name: "SPY -10", Symbol: "SPY", Threshold: -10, Period: 7, Enabled: true},
		{Name: "SPY -12", Symbol: "SPY", Threshold: -12, Period: 7, Enabled: true},
		{Name: "SPY -15", Symbol: "SPY", Threshold: -15, Period: 7, Enabled: true},
		{Name: "AAPL", Symbol: "AAPL", Threshold: -5, Period: 7, Enabled: true},
	}
	summary := engine.EvaluateAll(alerts)

	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("expected 2 HTTP requests (SPY once, AAPL once), got %d", got)
	}
	if summary.APIRequests != 2 {
		t.Errorf("expected APIRequests=2, got %d", summary.APIRequests)
	}
	if summary.TriggeredAlerts != 3 { // -13% move: -10, -12 and AAPL(-5) trigger; -15 does not
		t.Errorf("expected 3 triggered, got %d", summary.TriggeredAlerts)
	}
}

func TestEvaluationSummary_GetNotifiableResults_KeepsDeepestTier(t *testing.T) {
	mk := func(name, sym string, th float64, period int, trig bool) AlertResult {
		return AlertResult{Alert: Alert{Name: name, Symbol: sym, Threshold: th, Period: period, CalculationMethod: "from_peak"}, Triggered: trig, PercentChange: -16}
	}
	s := EvaluationSummary{Results: []AlertResult{
		mk("SPY -10", "SPY", -10, 90, true),
		mk("AAPL", "AAPL", -12, 90, true),
		mk("SPY -15", "SPY", -15, 90, true),
		mk("SPY -12", "SPY", -12, 90, true),
		mk("SPY -20", "SPY", -20, 90, false),
		mk("SPY 30d", "SPY", -5, 30, true), // different period: separate group
	}}

	got := s.GetNotifiableResults()
	var names []string
	for _, r := range got {
		names = append(names, r.Alert.Name)
	}
	want := []string{"SPY -15", "AAPL", "SPY 30d"}
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("got %v, want %v", names, want)
		}
	}
}
