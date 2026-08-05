## ADDED Requirements

### Requirement: Load API keys from environment
The system SHALL load sensitive configuration (API keys, tokens) from environment variables or .env file.

#### Scenario: Load from .env file
- **WHEN** .env file contains ALPHAVANTAGE_API_KEY and TELEGRAM_BOT_TOKEN
- **THEN** system loads these values and makes them available to other components

#### Scenario: Load from environment variables
- **WHEN** ALPHAVANTAGE_API_KEY is set as system environment variable
- **AND** no .env file exists
- **THEN** system loads the value from the environment variable

#### Scenario: Environment variable overrides .env
- **WHEN** ALPHAVANTAGE_API_KEY is set in both .env and system environment
- **THEN** system uses the system environment variable value

#### Scenario: Missing required configuration
- **WHEN** ALPHAVANTAGE_API_KEY is not configured anywhere
- **THEN** system returns an error indicating the required configuration is missing

### Requirement: Load alert definitions from YAML
The system SHALL load alert configurations from an alerts.yaml file.

#### Scenario: Load valid alerts file
- **WHEN** alerts.yaml contains valid alert definitions
- **THEN** system parses and loads all alert configurations

#### Scenario: Handle missing alerts file
- **WHEN** alerts.yaml does not exist
- **THEN** system creates an empty alerts file or operates with no alerts

#### Scenario: Handle invalid YAML syntax
- **WHEN** alerts.yaml contains invalid YAML
- **THEN** system returns an error with line number and syntax issue

#### Scenario: Handle invalid alert definition
- **WHEN** alerts.yaml contains alert missing required field (e.g., symbol)
- **THEN** system returns validation error indicating the missing field

### Requirement: Save alert configurations
The system SHALL persist alert configurations to the alerts.yaml file.

#### Scenario: Save new alert
- **WHEN** user adds a new alert via CLI
- **THEN** system writes the updated alerts to alerts.yaml

#### Scenario: Remove alert and save
- **WHEN** user removes an alert via CLI
- **THEN** system updates alerts.yaml without the removed alert

### Requirement: Provide example configuration files
The system SHALL include example configuration files for user reference.

#### Scenario: .env.example file exists
- **WHEN** user clones the repository
- **THEN** .env.example file is present with placeholder values and comments

#### Scenario: alerts.yaml.example file exists
- **WHEN** user clones the repository
- **THEN** alerts.yaml.example file is present with sample alert definitions

### Requirement: Validate configuration on startup
The system SHALL validate all configuration on startup and report issues clearly.

#### Scenario: Valid configuration
- **WHEN** all required configuration is present and valid
- **THEN** system starts successfully

#### Scenario: Invalid Telegram chat ID format
- **WHEN** TELEGRAM_CHAT_ID contains non-numeric value
- **THEN** system returns validation error for chat ID format

#### Scenario: Report all validation errors
- **WHEN** multiple configuration issues exist
- **THEN** system reports all issues in a single error message
