## ADDED Requirements

### Requirement: Fetch daily time series data
The system SHALL fetch daily OHLCV (Open, High, Low, Close, Volume) time series data from Alpha Vantage API for a given stock symbol.

#### Scenario: Successful data fetch for US stock
- **WHEN** client requests daily data for symbol "IBM"
- **THEN** system returns a list of daily price records with date, open, high, low, close, and volume fields

#### Scenario: Successful data fetch for index
- **WHEN** client requests daily data for NASDAQ composite symbol "IXIC"
- **THEN** system returns a list of daily price records for the index

#### Scenario: API key is missing
- **WHEN** client attempts to fetch data without an API key configured
- **THEN** system returns an error indicating the API key is required

### Requirement: Handle API errors gracefully
The system SHALL handle Alpha Vantage API errors and return meaningful error messages.

#### Scenario: Invalid symbol
- **WHEN** client requests data for an invalid symbol "NOTASTOCK"
- **THEN** system returns an error indicating the symbol was not found

#### Scenario: Rate limit exceeded
- **WHEN** client exceeds the API rate limit
- **THEN** system returns an error indicating rate limit was exceeded with retry guidance

#### Scenario: Network failure
- **WHEN** network connection fails during API call
- **THEN** system returns an error indicating network connectivity issue

### Requirement: Parse API response into structured data
The system SHALL parse the JSON response from Alpha Vantage into strongly-typed Go structs.

#### Scenario: Parse daily time series response
- **WHEN** API returns valid JSON with "Time Series (Daily)" data
- **THEN** system parses response into slice of DailyPrice structs with Date, Open, High, Low, Close, Volume fields

#### Scenario: Handle empty response
- **WHEN** API returns empty time series data
- **THEN** system returns empty slice without error

### Requirement: Calculate price change percentage
The system SHALL calculate the percentage change in closing price between two dates.

#### Scenario: Calculate 7-day price change
- **WHEN** given price data spanning at least 7 days
- **AND** user requests 7-day percentage change
- **THEN** system returns the percentage difference between today's close and the close 7 days ago

#### Scenario: Insufficient data for period
- **WHEN** given price data with fewer days than requested period
- **THEN** system returns an error indicating insufficient historical data
