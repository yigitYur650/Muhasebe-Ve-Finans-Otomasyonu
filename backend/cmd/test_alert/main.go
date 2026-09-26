package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"deftersystem/backend/pkg/telegram"
)

func main() {
	fmt.Println("🚀 [ÖNCÜ OTOGAZ] Telegram Alarm Test Aracı Başlatılıyor...")

	envPaths := []string{".env.local", "../.env.local", ".env", "../.env", "backend/.env"}
	for _, envPath := range envPaths {
		if data, err := os.ReadFile(envPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
					if os.Getenv(key) == "" {
						_ = os.Setenv(key, val)
					}
				}
			}
			break
		}
	}

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if botToken == "" || chatID == "" {
		log.Fatalf("\n❌ HATA: TELEGRAM_BOT_TOKEN veya TELEGRAM_CHAT_ID bulunamadı!\n" +
			"👉 Lütfen docs/TELEGRAM_AND_DRIVE_BACKUP_GUIDE.md dosyasındaki adımları izleyerek .env dosyanızı doldurun.")
	}

	client := telegram.NewClient(telegram.Config{
		BotToken: botToken,
		ChatID:   chatID,
		Enabled:  true,
	})

	testMessage := fmt.Sprintf(
		"🚨 <b>[ÖNCÜ OTOGAZ — TEST ALARMI]</b>\n\n"+
			"✅ Telegram Alarm & Bildirim Sistemi başarıyla aktif edildi!\n"+
			"⏰ <b>Zaman:</b> <code>%s</code>\n"+
			"🖥️ <b>Ortam:</b> <code>%s</code>\n"+
			"📌 <b>Durum:</b> Sunucu çökme, 500 hataları ve günlük Drive yedekleme raporları bu kanala iletilecektir.",
		time.Now().Format("2006-01-02 15:04:05 MST"),
		os.Getenv("ENVIRONMENT"),
	)

	fmt.Printf("📡 Telegram Bot API'sine test mesajı gönderiliyor (Chat ID: %s)...\n", chatID)
	err := client.SendMessage(testMessage)
	if err != nil {
		log.Fatalf("❌ Mesaj gönderilemedi: %v\n👉 Bot Token ve Chat ID'nizin doğruluğunu, botu Telegram'da /start ile başlattığınızı kontrol edin.", err)
	}

	fmt.Println("🎉 TEBRİKLER! Test mesajı Telegram hesabınıza/kanalınıza başarıyla ulaştı.")
}
