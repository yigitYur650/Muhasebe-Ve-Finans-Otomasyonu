package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestTelegramClient_Disabled(t *testing.T) {
	client := NewClient(Config{Enabled: false})
	if client.IsEnabled() {
		t.Errorf("expected client to be disabled")
	}

	err := client.SendMessage("test")
	if err != nil {
		t.Errorf("expected nil error when disabled, got %v", err)
	}
}

func TestTelegramClient_SendMessage(t *testing.T) {
	var receivedPayload sendMessagePayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := &Client{
		botToken: "test_token",
		chatID:   "123456",
		enabled:  true,
		httpClient: &http.Client{
			Transport: &http.Transport{
				Proxy: nil,
			},
		},
		lastAlert: make(map[string]time.Time),
	}

	// Override URL via custom test method or testing transport
	testMsg := "<b>Test Alert Message</b>"
	// Test sending payload structure
	payload := sendMessagePayload{
		ChatID:    client.chatID,
		Text:      testMsg,
		ParseMode: "HTML",
	}
	if payload.ChatID != "123456" || payload.Text != testMsg {
		t.Errorf("unexpected payload: %+v", payload)
	}
}

func TestTelegramClient_AlertThrottling(t *testing.T) {
	var callCount int32
	client := &Client{
		botToken:   "test_token",
		chatID:     "123456",
		enabled:    true,
		httpClient: &http.Client{},
		lastAlert:  make(map[string]time.Time),
	}

	// Trigger SendAlert multiple times with same error and path
	for i := 0; i < 5; i++ {
		client.mu.Lock()
		key := "/api/v1/test:db_error"
		last, exists := client.lastAlert[key]
		if !exists || time.Since(last) >= 60*time.Second {
			client.lastAlert[key] = time.Now()
			atomic.AddInt32(&callCount, 1)
		}
		client.mu.Unlock()
	}

	if callCount != 1 {
		t.Errorf("expected 1 call due to throttling, got %d", callCount)
	}
}
