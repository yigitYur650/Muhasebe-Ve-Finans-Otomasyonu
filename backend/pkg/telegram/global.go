package telegram

import (
	"os"
	"strings"
	"sync"
)

var (
	globalClient *Client
	globalMu     sync.RWMutex
)

// InitGlobal initializes the global Telegram notifier from environment variables.
func InitGlobal() *Client {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	enabledStr := strings.ToLower(os.Getenv("TELEGRAM_ALERTS_ENABLED"))
	enabled := enabledStr == "true" || enabledStr == "1" || enabledStr == "yes"

	client := NewClient(Config{
		BotToken: token,
		ChatID:   chatID,
		Enabled:  enabled,
	})

	globalMu.Lock()
	globalClient = client
	globalMu.Unlock()

	return client
}

// Global returns the singleton Telegram client instance.
func Global() *Client {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if globalClient == nil {
		return NewClient(Config{Enabled: false})
	}
	return globalClient
}

// SetGlobal overrides the global Telegram client (useful for unit testing).
func SetGlobal(c *Client) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalClient = c
}
