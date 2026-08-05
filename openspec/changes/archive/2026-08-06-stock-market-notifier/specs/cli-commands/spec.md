## ADDED Requirements

### Requirement: Check command runs alert evaluation
The system SHALL provide a `check` command that evaluates all enabled alerts and sends notifications for triggered alerts.

#### Scenario: Run check with triggered alerts
- **WHEN** user runs `notifier check`
- **AND** 2 out of 5 alerts trigger
- **THEN** system sends 2 Telegram notifications and displays summary

#### Scenario: Run check with no triggered alerts
- **WHEN** user runs `notifier check`
- **AND** no alerts trigger
- **THEN** system displays message indicating no alerts were triggered

#### Scenario: Run check with verbose output
- **WHEN** user runs `notifier check --verbose`
- **THEN** system displays detailed evaluation results for each alert

### Requirement: List command shows configured alerts
The system SHALL provide a `list` command that displays all configured alerts.

#### Scenario: List all alerts
- **WHEN** user runs `notifier list`
- **AND** 3 alerts are configured
- **THEN** system displays table with symbol, threshold, period, and enabled status for each alert

#### Scenario: List with no alerts configured
- **WHEN** user runs `notifier list`
- **AND** no alerts are configured
- **THEN** system displays message indicating no alerts are configured

### Requirement: Add command creates new alert
The system SHALL provide an `add` command to create new alert configurations.

#### Scenario: Add alert with all parameters
- **WHEN** user runs `notifier add --symbol IXIC --threshold -5 --period 7d --name "NASDAQ Weekly Drop"`
- **THEN** system adds the alert to configuration and confirms creation

#### Scenario: Add alert with missing required parameter
- **WHEN** user runs `notifier add --symbol IXIC` without threshold
- **THEN** system displays error indicating threshold is required

### Requirement: Remove command deletes alert
The system SHALL provide a `remove` command to delete alert configurations.

#### Scenario: Remove alert by name
- **WHEN** user runs `notifier remove "NASDAQ Weekly Drop"`
- **AND** alert with that name exists
- **THEN** system removes the alert and confirms deletion

#### Scenario: Remove non-existent alert
- **WHEN** user runs `notifier remove "NonExistent"`
- **AND** no alert with that name exists
- **THEN** system displays error indicating alert was not found

### Requirement: Version command shows version info
The system SHALL provide a `version` command that displays the application version.

#### Scenario: Display version
- **WHEN** user runs `notifier version`
- **THEN** system displays version number and build information

### Requirement: Help command shows usage
The system SHALL provide help text for all commands.

#### Scenario: Display main help
- **WHEN** user runs `notifier --help`
- **THEN** system displays list of available commands with descriptions

#### Scenario: Display command-specific help
- **WHEN** user runs `notifier check --help`
- **THEN** system displays detailed help for the check command including flags
