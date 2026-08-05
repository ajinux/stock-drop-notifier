package alphavantage

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// BaseURL is the Alpha Vantage API base URL
	BaseURL = "https://www.alphavantage.co/query"

	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second

	// CompactOutputMaxDays is the maximum period (in calendar days) that compact output can cover
	// Compact returns ~100 trading days, which is roughly 140 calendar days (~5 months)
	// We use 120 days as a safe threshold to ensure we have enough data
	CompactOutputMaxDays = 120
)

// Client is the Alpha Vantage API client
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Alpha Vantage API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		baseURL: BaseURL,
	}
}

// WithHTTPClient sets a custom HTTP client (useful for testing)
func (c *Client) WithHTTPClient(client *http.Client) *Client {
	c.httpClient = client
	return c
}

// WithBaseURL sets a custom base URL (useful for testing)
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.baseURL = baseURL
	return c
}

// OutputSize represents the amount of data to fetch
type OutputSize string

const (
	// OutputSizeCompact returns the latest 100 data points (~5 months)
	OutputSizeCompact OutputSize = "compact"
	// OutputSizeFull returns 20+ years of historical data
	OutputSizeFull OutputSize = "full"
)

// GetDailyTimeSeries fetches daily time series data for a symbol using compact output
func (c *Client) GetDailyTimeSeries(symbol string) (*TimeSeriesResponse, error) {
	return c.GetDailyTimeSeriesWithSize(symbol, OutputSizeCompact)
}

// GetDailyTimeSeriesWithSize fetches daily time series data with specified output size
func (c *Client) GetDailyTimeSeriesWithSize(symbol string, outputSize OutputSize) (*TimeSeriesResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	// Build URL with query parameters
	params := url.Values{}
	params.Set("function", "TIME_SERIES_DAILY")
	params.Set("symbol", symbol)
	params.Set("apikey", c.apiKey)
	params.Set("outputsize", string(outputSize))

	reqURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	// Make HTTP request
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	tsResponse, err := ParseTimeSeriesResponse(body)
	if err != nil {
		// Check if it's a symbol not found error
		if strings.Contains(err.Error(), "Invalid API call") ||
			strings.Contains(err.Error(), "Error Message") {
			return nil, &SymbolNotFoundError{Symbol: symbol}
		}
		return nil, err
	}

	// Check if we got any data
	if len(tsResponse.TimeSeries) == 0 {
		return nil, &SymbolNotFoundError{Symbol: symbol}
	}

	return tsResponse, nil
}

// GetDailyTimeSeriesForPeriod fetches daily time series with appropriate output size for the period
// For periods <= 120 days, uses compact output (faster, less data)
// For periods > 120 days, uses full output (slower, more data)
func (c *Client) GetDailyTimeSeriesForPeriod(symbol string, days int) (*TimeSeriesResponse, error) {
	outputSize := OutputSizeCompact
	if days > CompactOutputMaxDays {
		outputSize = OutputSizeFull
	}
	return c.GetDailyTimeSeriesWithSize(symbol, outputSize)
}

// CalculatePriceChange calculates the percentage change in closing price over a period
func (c *Client) CalculatePriceChange(timeSeries []DailyPrice, days int) (float64, error) {
	if len(timeSeries) == 0 {
		return 0, fmt.Errorf("no price data available")
	}

	if days <= 0 {
		return 0, fmt.Errorf("days must be positive")
	}

	// Time series is sorted by date descending (most recent first)
	// Find the most recent price
	currentPrice := timeSeries[0].Close
	currentDate := timeSeries[0].Date

	// Find the price from 'days' ago
	targetDate := currentDate.AddDate(0, 0, -days)
	var pastPrice float64
	var foundPast bool

	for _, price := range timeSeries {
		// Find the closest date on or before the target date
		if price.Date.Before(targetDate) || price.Date.Equal(targetDate) {
			pastPrice = price.Close
			foundPast = true
			break
		}
		// Also keep track in case we need the oldest available
		pastPrice = price.Close
	}

	if !foundPast && len(timeSeries) < days {
		return 0, fmt.Errorf("insufficient historical data: need %d days, have %d", days, len(timeSeries))
	}

	// Calculate percentage change: ((current - past) / past) * 100
	if pastPrice == 0 {
		return 0, fmt.Errorf("past price is zero, cannot calculate percentage change")
	}

	percentChange := ((currentPrice - pastPrice) / pastPrice) * 100

	return percentChange, nil
}

// CalculateExtremePriceChange calculates price change from peak (for negative changes/drops) or from trough (for positive changes/gains) over a period
func (c *Client) CalculateExtremePriceChange(timeSeries []DailyPrice, days int, isNegative bool) (float64, float64, error) {
	if len(timeSeries) == 0 {
		return 0, 0, fmt.Errorf("no price data available")
	}

	if days <= 0 {
		return 0, 0, fmt.Errorf("days must be positive")
	}

	currentPrice := timeSeries[0].Close
	currentDate := timeSeries[0].Date

	// Find the price from 'days' ago to define our window
	targetDate := currentDate.AddDate(0, 0, -days)
	var windowPrices []DailyPrice
	var foundPast bool

	for _, price := range timeSeries {
		if price.Date.After(targetDate) {
			windowPrices = append(windowPrices, price)
		} else if price.Date.Equal(targetDate) {
			windowPrices = append(windowPrices, price)
			foundPast = true
			break
		} else {
			// Include the closest date on or before targetDate to capture the full period
			windowPrices = append(windowPrices, price)
			foundPast = true
			break
		}
	}

	if !foundPast && len(timeSeries) < days {
		return 0, 0, fmt.Errorf("insufficient historical data: need %d days, have %d", days, len(timeSeries))
	}

	if len(windowPrices) == 0 {
		return 0, 0, fmt.Errorf("no price data within target window")
	}

	// Find the extreme price (max for drop/isNegative, min for gain/!isNegative)
	extremePrice := windowPrices[0].Close
	for _, price := range windowPrices {
		if isNegative {
			// Looking for the peak (maximum close price)
			if price.Close > extremePrice {
				extremePrice = price.Close
			}
		} else {
			// Looking for the trough (minimum close price)
			if price.Close < extremePrice {
				extremePrice = price.Close
			}
		}
	}

	if extremePrice == 0 {
		return 0, 0, fmt.Errorf("extreme price is zero, cannot calculate percentage change")
	}

	percentChange := ((currentPrice - extremePrice) / extremePrice) * 100

	return percentChange, extremePrice, nil
}

// GetPriceChangeForSymbol is a convenience method that fetches data and calculates price change
func (c *Client) GetPriceChangeForSymbol(symbol string, days int) (float64, float64, float64, error) {
	response, err := c.GetDailyTimeSeries(symbol)
	if err != nil {
		return 0, 0, 0, err
	}

	percentChange, err := c.CalculatePriceChange(response.TimeSeries, days)
	if err != nil {
		return 0, 0, 0, err
	}

	// Return current price, past price (approximate), and percent change
	currentPrice := response.TimeSeries[0].Close
	pastPrice := currentPrice / (1 + percentChange/100)

	return currentPrice, pastPrice, percentChange, nil
}
