package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	// Alpha Vantage API configuration
	AlphaVantageAPIKey string

	// Telegram Bot configuration
	TelegramBotToken string
	TelegramChatID   int64
}

// Load loads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	cfg := &Config{}

	// Load Alpha Vantage API key
	cfg.AlphaVantageAPIKey = os.Getenv("ALPHAVANTAGE_API_KEY")

	// Load Telegram configuration
	cfg.TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	if chatIDStr != "" {
		chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid TELEGRAM_CHAT_ID: %w", err)
		}
		cfg.TelegramChatID = chatID
	}

	return cfg, nil
}

// Validate checks that all required configuration is present
func (c *Config) Validate() error {
	var errs []string

	if c.AlphaVantageAPIKey == "" {
		errs = append(errs, "ALPHAVANTAGE_API_KEY is required")
	}

	if c.TelegramBotToken == "" {
		errs = append(errs, "TELEGRAM_BOT_TOKEN is required")
	}

	if c.TelegramChatID == 0 {
		errs = append(errs, "TELEGRAM_CHAT_ID is required")
	}

	if len(errs) > 0 {
		return errors.New("configuration errors: " + joinErrors(errs))
	}

	return nil
}

// ValidateForCheck validates configuration required for the check command
// This allows checking alerts even without Telegram configured
func (c *Config) ValidateForCheck() error {
	if c.AlphaVantageAPIKey == "" {
		return errors.New("ALPHAVANTAGE_API_KEY is required")
	}
	return nil
}

// ValidateForNotify validates configuration required for sending notifications
func (c *Config) ValidateForNotify() error {
	var errs []string

	if c.TelegramBotToken == "" {
		errs = append(errs, "TELEGRAM_BOT_TOKEN is required")
	}

	if c.TelegramChatID == 0 {
		errs = append(errs, "TELEGRAM_CHAT_ID is required")
	}

	if len(errs) > 0 {
		return errors.New("configuration errors: " + joinErrors(errs))
	}

	return nil
}

func joinErrors(errs []string) string {
	result := ""
	for i, err := range errs {
		if i > 0 {
			result += "; "
		}
		result += err
	}
	return result
}
