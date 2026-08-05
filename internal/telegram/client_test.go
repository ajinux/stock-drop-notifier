package telegram

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_SendMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}

		// Parse request body
		body, _ := io.ReadAll(r.Body)
		var req sendMessageRequest
		json.Unmarshal(body, &req)

		if req.ChatID != 12345 {
			t.Errorf("expected chat_id 12345, got %d", req.ChatID)
		}

		if req.Text != "Test message" {
			t.Errorf("expected 'Test message', got %s", req.Text)
		}

		if req.ParseMode != "Markdown" {
			t.Errorf("expected Markdown parse mode, got %s", req.ParseMode)
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewClient("test-token", 12345).WithBaseURL(server.URL)
	err := client.SendMessage("Test message")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_SendMessage_MissingToken(t *testing.T) {
	client := NewClient("", 12345)
	err := client.SendMessage("Test message")

	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestClient_SendMessage_MissingChatID(t *testing.T) {
	client := NewClient("test-token", 0)
	err := client.SendMessage("Test message")

	if err == nil {
		t.Fatal("expected error for missing chat ID")
	}
}

func TestClient_SendMessage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok": false, "error_code": 401, "description": "Unauthorized"}`))
	}))
	defer server.Close()

	client := NewClient("invalid-token", 12345).WithBaseURL(server.URL)
	err := client.SendMessage("Test message")

	if err == nil {
		t.Fatal("expected error for API error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Code != 401 {
		t.Errorf("expected error code 401, got %d", apiErr.Code)
	}

	if !apiErr.IsAuthError() {
		t.Error("expected IsAuthError to return true")
	}
}

func TestClient_SendMessage_ChatNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok": false, "error_code": 400, "description": "Bad Request: chat not found"}`))
	}))
	defer server.Close()

	client := NewClient("test-token", 99999).WithBaseURL(server.URL)
	err := client.SendMessage("Test message")

	if err == nil {
		t.Fatal("expected error for chat not found")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if !apiErr.IsChatNotFoundError() {
		t.Error("expected IsChatNotFoundError to return true")
	}
}

func TestClient_SendBatch(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		// Fail on second message
		if callCount == 2 {
			w.Write([]byte(`{"ok": false, "error_code": 400, "description": "test error"}`))
		} else {
			w.Write([]byte(`{"ok": true}`))
		}
	}))
	defer server.Close()

	client := NewClient("test-token", 12345).WithBaseURL(server.URL)
	messages := []string{"Message 1", "Message 2", "Message 3"}

	sent, errors := client.SendBatch(messages)

	if sent != 2 {
		t.Errorf("expected 2 messages sent, got %d", sent)
	}

	if len(errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(errors))
	}

	if callCount != 3 {
		t.Errorf("expected 3 API calls, got %d", callCount)
	}
}

// A Markdown parse failure must not lose the notification: the client retries
// the same text without formatting.
func TestClient_SendMessage_FallsBackToPlainTextOnParseError(t *testing.T) {
	var parseModes []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req sendMessageRequest
		json.Unmarshal(body, &req)
		parseModes = append(parseModes, req.ParseMode)

		w.Header().Set("Content-Type", "application/json")
		if req.ParseMode == "Markdown" {
			w.Write([]byte(`{"ok": false, "error_code": 400, "description": "Bad Request: can't parse entities: Can't find end of the entity starting at byte offset 9"}`))
			return
		}
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewClient("test-token", 12345).WithBaseURL(server.URL)
	err := client.SendMessage("*AAPL_Drop* Alert")

	if err != nil {
		t.Fatalf("expected fallback to succeed, got: %v", err)
	}

	want := []string{"Markdown", ""}
	if len(parseModes) != len(want) || parseModes[0] != want[0] || parseModes[1] != want[1] {
		t.Errorf("expected parse modes %q, got %q", want, parseModes)
	}
}

// Errors unrelated to formatting must surface, not trigger a pointless retry.
func TestClient_SendMessage_DoesNotRetryOnAuthError(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok": false, "error_code": 401, "description": "Unauthorized"}`))
	}))
	defer server.Close()

	client := NewClient("bad-token", 12345).WithBaseURL(server.URL)
	err := client.SendMessage("Test message")

	if err == nil {
		t.Fatal("expected an error for an unauthorized token")
	}

	if calls != 1 {
		t.Errorf("expected 1 API call, got %d", calls)
	}
}
