## ADDED Requirements

### Requirement: Send notification via Telegram Bot
The system SHALL send alert notifications to a configured Telegram chat using the Bot API.

#### Scenario: Send alert notification successfully
- **WHEN** alert triggers for NASDAQ dropping 5%
- **AND** Telegram bot token and chat ID are configured
- **THEN** system sends message to the configured Telegram chat

#### Scenario: Missing bot token
- **WHEN** system attempts to send notification
- **AND** TELEGRAM_BOT_TOKEN is not configured
- **THEN** system returns an error indicating bot token is required

#### Scenario: Missing chat ID
- **WHEN** system attempts to send notification
- **AND** TELEGRAM_CHAT_ID is not configured
- **THEN** system returns an error indicating chat ID is required

### Requirement: Format notification messages
The system SHALL format alert messages with Markdown for better readability in Telegram.

#### Scenario: Format drop alert with markdown
- **WHEN** sending alert for NASDAQ down 5.2%
- **THEN** message includes bold symbol name and formatted percentage

#### Scenario: Include timestamp in notification
- **WHEN** sending any alert notification
- **THEN** message includes the current date and time of the alert

### Requirement: Handle Telegram API errors
The system SHALL handle Telegram API errors gracefully and report them.

#### Scenario: Invalid bot token
- **WHEN** system sends message with invalid bot token
- **THEN** system returns an error indicating authentication failed

#### Scenario: Invalid chat ID
- **WHEN** system sends message to non-existent chat ID
- **THEN** system returns an error indicating chat was not found

#### Scenario: Network failure to Telegram
- **WHEN** network connection to Telegram API fails
- **THEN** system returns an error indicating network connectivity issue

### Requirement: Support batch notifications
The system SHALL support sending multiple alert notifications in sequence.

#### Scenario: Send multiple alerts
- **WHEN** 3 alerts trigger simultaneously
- **THEN** system sends 3 separate messages to Telegram, one for each alert

#### Scenario: Continue on individual send failure
- **WHEN** one notification fails to send
- **AND** other notifications are pending
- **THEN** system continues sending remaining notifications and reports the failure
