package telegram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// BaseURL is the Telegram Bot API base URL
	BaseURL = "https://api.telegram.org"

	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second
)

// Client is the Telegram Bot API client
type Client struct {
	botToken   string
	chatID     int64
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Telegram Bot API client
func NewClient(botToken string, chatID int64) *Client {
	return &Client{
		botToken: botToken,
		chatID:   chatID,
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

// sendMessageRequest is the request payload for sendMessage API
type sendMessageRequest struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// apiResponse is the response from Telegram API
type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	ErrorCode   int    `json:"error_code,omitempty"`
}

// SendMessage sends a message to the configured chat using Markdown formatting.
// Alert names are user-supplied and may contain characters Telegram's Markdown
// parser rejects, so a parse failure retries as plain text rather than losing
// the notification entirely.
func (c *Client) SendMessage(message string) error {
	if c.botToken == "" {
		return fmt.Errorf("bot token is required")
	}

	if c.chatID == 0 {
		return fmt.Errorf("chat ID is required")
	}

	err := c.sendMessage(message, "Markdown")
	if err != nil && isParseError(err) {
		return c.SendPlainMessage(message)
	}
	return err
}

// isParseError reports whether Telegram rejected the message's Markdown, as
// opposed to rejecting the request itself (bad token, unknown chat, rate limit).
func isParseError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Code == http.StatusBadRequest &&
		strings.Contains(strings.ToLower(apiErr.Description), "parse")
}

// SendPlainMessage sends a message without Markdown formatting
func (c *Client) SendPlainMessage(message string) error {
	if c.botToken == "" {
		return fmt.Errorf("bot token is required")
	}

	if c.chatID == 0 {
		return fmt.Errorf("chat ID is required")
	}

	return c.sendMessage(message, "")
}

func (c *Client) sendMessage(message string, parseMode string) error {
	url := fmt.Sprintf("%s/bot%s/sendMessage", c.baseURL, c.botToken)

	reqBody := sendMessageRequest{
		ChatID:    c.chatID,
		Text:      message,
		ParseMode: parseMode,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.OK {
		return &APIError{
			Code:        apiResp.ErrorCode,
			Description: apiResp.Description,
		}
	}

	return nil
}

// SendBatch sends multiple messages in sequence
// Returns the number of successfully sent messages and any errors encountered
func (c *Client) SendBatch(messages []string) (int, []error) {
	var errors []error
	sent := 0

	for _, msg := range messages {
		if err := c.SendMessage(msg); err != nil {
			errors = append(errors, err)
		} else {
			sent++
		}
	}

	return sent, errors
}

// APIError represents a Telegram API error
type APIError struct {
	Code        int
	Description string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Telegram API error %d: %s", e.Code, e.Description)
}

// IsAuthError returns true if the error is an authentication error
func (e *APIError) IsAuthError() bool {
	return e.Code == 401 || e.Code == 403
}

// IsChatNotFoundError returns true if the chat was not found
func (e *APIError) IsChatNotFoundError() bool {
	return e.Code == 400 && (e.Description == "Bad Request: chat not found" ||
		e.Description == "Bad Request: CHAT_ID_INVALID")
}
