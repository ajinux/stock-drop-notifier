package alphavantage

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// A minimal well-formed response, for tests that care about transport behaviour
// rather than parsing.
const validTimeSeriesJSON = `{
	"Meta Data": {
		"1. Information": "Daily Prices (open, high, low, close) and Volumes",
		"2. Symbol": "IBM",
		"3. Last Refreshed": "2024-01-15",
		"4. Output Size": "Compact",
		"5. Time Zone": "US/Eastern"
	},
	"Time Series (Daily)": {
		"2024-01-15": {
			"1. open": "150.00",
			"2. high": "155.00",
			"3. low": "149.00",
			"4. close": "153.50",
			"5. volume": "5000000"
		}
	}
}`

func TestParseTimeSeriesResponse(t *testing.T) {
	jsonResponse := `{
		"Meta Data": {
			"1. Information": "Daily Prices (open, high, low, close) and Volumes",
			"2. Symbol": "IBM",
			"3. Last Refreshed": "2024-01-15",
			"4. Output Size": "Compact",
			"5. Time Zone": "US/Eastern"
		},
		"Time Series (Daily)": {
			"2024-01-15": {
				"1. open": "150.00",
				"2. high": "155.00",
				"3. low": "149.00",
				"4. close": "153.50",
				"5. volume": "5000000"
			},
			"2024-01-14": {
				"1. open": "148.00",
				"2. high": "151.00",
				"3. low": "147.00",
				"4. close": "150.00",
				"5. volume": "4500000"
			}
		}
	}`

	response, err := ParseTimeSeriesResponse([]byte(jsonResponse))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.MetaData.Symbol != "IBM" {
		t.Errorf("expected symbol IBM, got %s", response.MetaData.Symbol)
	}

	if len(response.TimeSeries) != 2 {
		t.Errorf("expected 2 time series entries, got %d", len(response.TimeSeries))
	}

	// Check that data is sorted descending (most recent first)
	if response.TimeSeries[0].Date.Before(response.TimeSeries[1].Date) {
		t.Error("expected time series to be sorted descending")
	}

	// Check values
	if response.TimeSeries[0].Close != 153.50 {
		t.Errorf("expected close price 153.50, got %f", response.TimeSeries[0].Close)
	}
}

func TestParseTimeSeriesResponse_Error(t *testing.T) {
	jsonResponse := `{
		"Error Message": "Invalid API call. Please retry or visit the documentation."
	}`

	_, err := ParseTimeSeriesResponse([]byte(jsonResponse))
	if err == nil {
		t.Fatal("expected error for invalid API call")
	}
}

func TestParseTimeSeriesResponse_RateLimit(t *testing.T) {
	jsonResponse := `{
		"Note": "Thank you for using Alpha Vantage! Our standard API call frequency is 25 calls per day."
	}`

	_, err := ParseTimeSeriesResponse([]byte(jsonResponse))
	if err == nil {
		t.Fatal("expected error for rate limit")
	}

	_, ok := err.(*RateLimitError)
	if !ok {
		t.Errorf("expected RateLimitError, got %T", err)
	}
}

func TestClient_GetDailyTimeSeries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		if r.URL.Query().Get("function") != "TIME_SERIES_DAILY" {
			t.Errorf("expected function TIME_SERIES_DAILY")
		}
		if r.URL.Query().Get("symbol") != "IBM" {
			t.Errorf("expected symbol IBM")
		}
		if r.URL.Query().Get("apikey") != "test-key" {
			t.Errorf("expected apikey test-key")
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"Meta Data": {
				"1. Information": "Daily Prices",
				"2. Symbol": "IBM",
				"3. Last Refreshed": "2024-01-15",
				"4. Output Size": "Compact",
				"5. Time Zone": "US/Eastern"
			},
			"Time Series (Daily)": {
				"2024-01-15": {
					"1. open": "150.00",
					"2. high": "155.00",
					"3. low": "149.00",
					"4. close": "153.50",
					"5. volume": "5000000"
				}
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-key").WithBaseURL(server.URL)
	response, err := client.GetDailyTimeSeries("IBM")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.MetaData.Symbol != "IBM" {
		t.Errorf("expected symbol IBM, got %s", response.MetaData.Symbol)
	}
}

func TestClient_GetDailyTimeSeries_MissingAPIKey(t *testing.T) {
	client := NewClient("")
	_, err := client.GetDailyTimeSeries("IBM")
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestClient_GetDailyTimeSeries_MissingSymbol(t *testing.T) {
	client := NewClient("test-key")
	_, err := client.GetDailyTimeSeries("")
	if err == nil {
		t.Fatal("expected error for missing symbol")
	}
}

func TestClient_CalculatePriceChange(t *testing.T) {
	client := NewClient("test-key")

	now := time.Now()
	timeSeries := []DailyPrice{
		{Date: now, Close: 110.0},                   // Current
		{Date: now.AddDate(0, 0, -1), Close: 108.0}, // 1 day ago
		{Date: now.AddDate(0, 0, -7), Close: 100.0}, // 7 days ago
		{Date: now.AddDate(0, 0, -14), Close: 95.0}, // 14 days ago
	}

	// Test 7-day change (100 -> 110 = 10%)
	change, err := client.CalculatePriceChange(timeSeries, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: (110 - 100) / 100 * 100 = 10%
	if change < 9.9 || change > 10.1 {
		t.Errorf("expected ~10%% change, got %f%%", change)
	}
}

func TestClient_CalculatePriceChange_InsufficientData(t *testing.T) {
	client := NewClient("test-key")

	now := time.Now()
	timeSeries := []DailyPrice{
		{Date: now, Close: 110.0},
		{Date: now.AddDate(0, 0, -1), Close: 108.0},
	}

	// Request 30 days but only have 2 days of data
	_, err := client.CalculatePriceChange(timeSeries, 30)
	if err == nil {
		t.Fatal("expected error for insufficient data")
	}
}

func TestClient_CalculatePriceChange_EmptyData(t *testing.T) {
	client := NewClient("test-key")

	_, err := client.CalculatePriceChange([]DailyPrice{}, 7)
	if err == nil {
		t.Fatal("expected error for empty data")
	}
}

func TestClient_CalculateExtremePriceChange(t *testing.T) {
	client := NewClient("test-key")

	now := time.Now()
	// Let's create a series where peak is higher than the start price, and current is lower than peak.
	// Scenario: Current is 105, Peak is 120 (5 days ago), Start (7 days ago) is 100.
	timeSeries := []DailyPrice{
		{Date: now, Close: 105.0}, // Current
		{Date: now.AddDate(0, 0, -1), Close: 115.0},
		{Date: now.AddDate(0, 0, -5), Close: 120.0}, // Peak (5 days ago)
		{Date: now.AddDate(0, 0, -7), Close: 100.0}, // Start (7 days ago)
	}

	// Test negative change (from peak: (105 - 120) / 120 * 100 = -12.5%)
	change, extreme, err := client.CalculateExtremePriceChange(timeSeries, 7, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if extreme != 120.0 {
		t.Errorf("expected extreme/peak price to be 120.0, got %f", extreme)
	}

	if change < -12.6 || change > -12.4 {
		t.Errorf("expected ~-12.5%% change, got %f%%", change)
	}

	// Scenario for trough: Current is 105, Trough is 90 (3 days ago), Start (7 days ago) is 100.
	timeSeriesTrough := []DailyPrice{
		{Date: now, Close: 105.0},                   // Current
		{Date: now.AddDate(0, 0, -3), Close: 90.0},  // Trough
		{Date: now.AddDate(0, 0, -7), Close: 100.0}, // Start
	}

	// Test positive change (from trough: (105 - 90) / 90 * 100 = 16.67%)
	changeT, extremeT, err := client.CalculateExtremePriceChange(timeSeriesTrough, 7, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if extremeT != 90.0 {
		t.Errorf("expected extreme/trough price to be 90.0, got %f", extremeT)
	}

	if changeT < 16.6 || changeT > 16.7 {
		t.Errorf("expected ~16.67%% change, got %f%%", changeT)
	}
}

// A stalled or 5xx response should be retried rather than failing the symbol on
// the first attempt — Alpha Vantage intermittently stops answering.
func TestClient_RetriesTransientFailures(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(validTimeSeriesJSON))
	}))
	defer server.Close()

	client := NewClient("test-key").WithBaseURL(server.URL).WithRetry(3, time.Millisecond)
	resp, err := client.GetDailyTimeSeries("IBM")

	if err != nil {
		t.Fatalf("expected success on the third attempt, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	if len(resp.TimeSeries) == 0 {
		t.Error("expected parsed time series data")
	}
}

// A 4xx is the caller's fault and will fail identically every time, so it must
// not burn attempts.
func TestClient_DoesNotRetryClientErrors(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient("test-key").WithBaseURL(server.URL).WithRetry(3, time.Millisecond)
	_, err := client.GetDailyTimeSeries("IBM")

	if err == nil {
		t.Fatal("expected an error for a 400 response")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

// Exhausting every attempt must report the underlying cause, not a bare count.
func TestClient_ReportsLastErrorAfterExhaustingAttempts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewClient("test-key").WithBaseURL(server.URL).WithRetry(2, time.Millisecond)
	_, err := client.GetDailyTimeSeries("IBM")

	if err == nil {
		t.Fatal("expected an error after exhausting attempts")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("expected the underlying 503 in the message, got: %v", err)
	}
}
