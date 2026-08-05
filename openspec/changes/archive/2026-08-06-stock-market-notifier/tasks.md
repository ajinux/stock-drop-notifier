## 1. Project Setup

- [x] 1.1 Initialize Go module with `go mod init stock-market-notifier`
- [x] 1.2 Create project directory structure (cmd/, internal/, etc.)
- [x] 1.3 Create .env.example with placeholder values for ALPHAVANTAGE_API_KEY, TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID
- [x] 1.4 Create .env file with actual API key for testing (J0CG6EYWJMNBLRVL)
- [x] 1.5 Create alerts.yaml.example with sample alert definitions
- [x] 1.6 Add .gitignore for .env and other generated files

## 2. Configuration Management

- [x] 2.1 Create internal/config/config.go with Config struct and Load function
- [x] 2.2 Implement .env file loading using godotenv library
- [x] 2.3 Implement environment variable override logic
- [x] 2.4 Implement validation for required configuration fields
- [x] 2.5 Create internal/config/alerts.go for loading alerts from YAML
- [x] 2.6 Implement alert validation (required fields, valid threshold/period)
- [x] 2.7 Implement saving alerts back to YAML file

## 3. Alpha Vantage Client

- [x] 3.1 Create internal/alphavantage/types.go with response structs (DailyPrice, TimeSeriesResponse)
- [x] 3.2 Create internal/alphavantage/client.go with Client struct
- [x] 3.3 Implement GetDailyTimeSeries method to fetch daily data
- [x] 3.4 Implement JSON response parsing into typed structs
- [x] 3.5 Implement error handling for API errors (rate limit, invalid symbol)
- [x] 3.6 Implement CalculatePriceChange method for percentage calculations
- [x] 3.7 Write unit tests for Alpha Vantage client

## 4. Alert Engine

- [x] 4.1 Create internal/alert/types.go with Alert and AlertResult structs
- [x] 4.2 Create internal/alert/engine.go with Engine struct
- [x] 4.3 Implement EvaluateAlert method for single alert evaluation
- [x] 4.4 Implement EvaluateAll method for batch alert evaluation
- [x] 4.5 Implement alert message generation with formatted output
- [x] 4.6 Handle disabled alerts (skip evaluation)
- [x] 4.7 Implement graceful error handling (continue on individual failures)
- [x] 4.8 Write unit tests for alert engine

## 5. Telegram Notifier

- [x] 5.1 Create internal/telegram/client.go with Client struct
- [x] 5.2 Implement SendMessage method using Bot API
- [x] 5.3 Implement Markdown formatting for alert messages
- [x] 5.4 Add timestamp to notification messages
- [x] 5.5 Implement error handling for Telegram API errors
- [x] 5.6 Implement batch notification sending
- [x] 5.7 Write unit tests for Telegram client (with mocked HTTP)

## 6. CLI Commands

- [x] 6.1 Add Cobra dependency and create cmd/notifier/main.go
- [x] 6.2 Implement root command with version flag
- [x] 6.3 Implement `check` command to evaluate alerts and send notifications
- [x] 6.4 Add --verbose flag to check command for detailed output
- [x] 6.5 Implement `list` command to display configured alerts
- [x] 6.6 Implement `add` command with --symbol, --threshold, --period, --name flags
- [x] 6.7 Implement `remove` command to delete alerts by name
- [x] 6.8 Implement `version` command with build info
- [x] 6.9 Add comprehensive help text for all commands

## 7. Integration and Testing

- [x] 7.1 Write integration test for full check workflow (fetch -> evaluate -> notify)
- [x] 7.2 Test with real Alpha Vantage API call (using test API key)
- [x] 7.3 Create README.md with setup instructions and usage examples
- [x] 7.4 Document Telegram bot creation steps in README
- [x] 7.5 Build and test CLI end-to-end with sample alerts

## 8. Polish and Documentation

- [x] 8.1 Add proper logging throughout the application
- [x] 8.2 Improve error messages for user-friendliness
- [x] 8.3 Add Makefile with build, test, and run targets
- [x] 8.4 Create example cron job configuration in README
- [x] 8.5 Final review and code cleanup
