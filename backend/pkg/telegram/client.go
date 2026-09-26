package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Client handles communication with the official Telegram Bot API.
type Client struct {
	botToken   string
	chatID     string
	enabled    bool
	httpClient *http.Client

	// Alert throttling to prevent notification spam
	mu        sync.Mutex
	lastAlert map[string]time.Time
}

// Config holds Telegram credentials and settings.
type Config struct {
	BotToken string
	ChatID   string
	Enabled  bool
}

// NewClient initializes a new Telegram Client.
func NewClient(cfg Config) *Client {
	return &Client{
		botToken: cfg.BotToken,
		chatID:   cfg.ChatID,
		enabled:  cfg.Enabled && cfg.BotToken != "" && cfg.ChatID != "",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		lastAlert: make(map[string]time.Time),
	}
}

// IsEnabled returns true if Telegram alerting is active.
func (c *Client) IsEnabled() bool {
	if c == nil {
		return false
	}
	return c.enabled
}

type sendMessagePayload struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// SendMessage sends a formatted message directly to the configured Telegram chat.
func (c *Client) SendMessage(text string) error {
	if !c.IsEnabled() {
		return nil
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.botToken)
	payload := sendMessagePayload{
		ChatID:    c.chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send telegram request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram API returned status: %d", resp.StatusCode)
	}

	return nil
}

// SendAlert sends an urgent critical error alert with throttling.
func (c *Client) SendAlert(title, errMsg, method, path, tenantID, clientIP, stackTrace string) {
	if !c.IsEnabled() {
		return
	}

	// Throttling key: group by error & path
	key := fmt.Sprintf("%s:%s", path, errMsg)
	c.mu.Lock()
	last, exists := c.lastAlert[key]
	if exists && time.Since(last) < 60*time.Second {
		c.mu.Unlock()
		return // Suppress duplicate alert for 60 seconds
	}
	c.lastAlert[key] = time.Now()
	c.mu.Unlock()

	// Asynchronous non-blocking delivery
	go func() {
		now := time.Now().Format("2006-01-02 15:04:05 MST")

		tenantInfo := tenantID
		if tenantInfo == "" {
			tenantInfo = "N/A"
		}

		msg := fmt.Sprintf(
			"🚨 <b>[ÖNCÜ OTOGAZ KRİTİK HATA ALARMI]</b>\n\n"+
				"⏰ <b>Zaman:</b> <code>%s</code>\n"+
				"📍 <b>Endpoint:</b> <code>%s %s</code>\n"+
				"🏢 <b>Tenant ID:</b> <code>%s</code>\n"+
				"🌐 <b>IP:</b> <code>%s</code>\n"+
				"💥 <b>Hata:</b> <code>%s</code>\n",
			now, method, path, tenantInfo, clientIP, errMsg,
		)

		if stackTrace != "" {
			// Limit stack trace length for Telegram max message limit (4096 chars)
			if len(stackTrace) > 800 {
				stackTrace = stackTrace[:800] + "..."
			}
			msg += fmt.Sprintf("\n📑 <b>Stack Trace:</b>\n<pre>%s</pre>", stackTrace)
		}

		_ = c.SendMessage(msg)
	}()
}

// SendBackupReport sends a detailed report about the daily backup status.
func (c *Client) SendBackupReport(success bool, filename string, sizeBytes int64, duration time.Duration, errMsg string) {
	if !c.IsEnabled() {
		return
	}

	go func() {
		now := time.Now().Format("2006-01-02 15:04:05 MST")
		var msg string

		if success {
			sizeMB := float64(sizeBytes) / (1024 * 1024)
			msg = fmt.Sprintf(
				"✅ <b>[GÜNLÜK YEDEKLEME BAŞARILI]</b>\n\n"+
					"⏰ <b>Tarih:</b> <code>%s</code>\n"+
					"📁 <b>Dosya:</b> <code>%s</code>\n"+
					"📦 <b>Boyut:</b> <code>%.2f MB</code>\n"+
					"☁️ <b>Hedef:</b> Google Drive / Oncu_Otogaz_Backups\n"+
					"⏱️ <b>Süre:</b> <code>%v</code>\n",
				now, filename, sizeMB, duration,
			)
		} else {
			msg = fmt.Sprintf(
				"❌ <b>[GÜNLÜK YEDEKLEME BAŞARISIZ!]</b>\n\n"+
					"⏰ <b>Tarih:</b> <code>%s</code>\n"+
					"📁 <b>Dosya:</b> <code>%s</code>\n"+
					"💥 <b>Hata Detayı:</b> <code>%s</code>\n"+
					"⚠️ <b>Lütfen veritabanı ve Google Drive bağlantısını kontrol edin!</b>\n",
				now, filename, errMsg,
			)
		}

		_ = c.SendMessage(msg)
	}()
}
