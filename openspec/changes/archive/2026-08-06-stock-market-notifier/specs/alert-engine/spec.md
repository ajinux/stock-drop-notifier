## ADDED Requirements

### Requirement: Define alert conditions
The system SHALL support defining alerts with symbol, threshold percentage, and time period.

#### Scenario: Define percentage drop alert
- **WHEN** user defines an alert for "IXIC" with threshold -5.0% over 7 days
- **THEN** system stores the alert configuration with symbol "IXIC", threshold -5.0, and period 7 days

#### Scenario: Define percentage gain alert
- **WHEN** user defines an alert for "AAPL" with threshold +10.0% over 30 days
- **THEN** system stores the alert configuration with symbol "AAPL", threshold +10.0, and period 30 days

### Requirement: Evaluate alert conditions
The system SHALL evaluate alert conditions against current price data and determine if alerts should trigger.

#### Scenario: Alert triggers on threshold breach
- **WHEN** NASDAQ (IXIC) has dropped 6% over the last 7 days
- **AND** alert threshold is -5%
- **THEN** system marks the alert as triggered and returns alert details

#### Scenario: Alert does not trigger when below threshold
- **WHEN** NASDAQ (IXIC) has dropped 3% over the last 7 days
- **AND** alert threshold is -5%
- **THEN** system marks the alert as not triggered

#### Scenario: Alert triggers on positive threshold
- **WHEN** stock has gained 12% over the last 30 days
- **AND** alert threshold is +10%
- **THEN** system marks the alert as triggered

### Requirement: Support multiple alerts
The system SHALL evaluate multiple alert configurations in a single check run.

#### Scenario: Evaluate multiple alerts
- **WHEN** user has 3 alerts configured for different symbols
- **AND** user runs alert check
- **THEN** system evaluates all 3 alerts and returns results for each

#### Scenario: Continue on individual alert failure
- **WHEN** one alert fails to fetch data (e.g., invalid symbol)
- **AND** other alerts are valid
- **THEN** system continues evaluating remaining alerts and reports the failure separately

### Requirement: Generate alert messages
The system SHALL generate human-readable alert messages when conditions are met.

#### Scenario: Generate drop alert message
- **WHEN** alert triggers for NASDAQ dropping 5.2% over 7 days
- **THEN** system generates message: "NASDAQ (IXIC) is down 5.2% over the last 7 days (threshold: -5.0%)"

#### Scenario: Generate gain alert message
- **WHEN** alert triggers for AAPL gaining 11.5% over 30 days
- **THEN** system generates message: "AAPL is up 11.5% over the last 30 days (threshold: +10.0%)"

### Requirement: Support alert enable/disable
The system SHALL allow alerts to be enabled or disabled without removing them.

#### Scenario: Skip disabled alerts
- **WHEN** alert is marked as disabled
- **AND** user runs alert check
- **THEN** system skips the disabled alert and does not evaluate it
