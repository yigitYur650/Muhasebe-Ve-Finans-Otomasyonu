package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"deftersystem/backend/internal/repository"
	"deftersystem/backend/pkg/scheduler"
	"deftersystem/backend/pkg/telegram"
)

func main() {
	fmt.Println("🚀 [ÖNCÜ OTOGAZ] Otomatik 3-2-1 Yedekleme Motoru Başlatılıyor...")

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

	telegram.InitGlobal()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatalf("❌ HATA: DATABASE_URL çevre değişkeni bulunamadı!")
	}

	fmt.Println("📡 PostgreSQL veritabanına bağlanılıyor...")
	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil || pool == nil {
		log.Fatalf("❌ HATA: Veritabanı havuzu başlatılamadı: %v", err)
	}
	defer pool.Close()

	orchestrator := scheduler.NewBackupOrchestrator(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	summary, err := orchestrator.RunBackup(ctx)
	if err != nil {
		log.Fatalf("❌ Yedekleme başarısız oldu: %v", err)
	}

	fmt.Println("\n==============================================================================")
	fmt.Println("🎉 TEBRİKLER! YEDEKLEME BAŞARIYLA TAMAMLANDI")
	fmt.Println("==============================================================================")
	fmt.Printf("📁 Dosya:      %s\n", summary.Filename)
	fmt.Printf("📦 Boyut:      %.2f KB\n", float64(summary.SizeBytes)/1024)
	fmt.Printf("📊 Toplam Kayıt: %d\n", summary.RecordCount)
	fmt.Printf("⏱️ Süre:       %v\n", summary.Duration)
	fmt.Println("------------------------------------------------------------------------------")
	for tbl, count := range summary.Tables {
		fmt.Printf("  • %-22s : %d kayıt\n", tbl, count)
	}
	fmt.Println("==============================================================================")
	fmt.Println("📱 Telegram kanalınıza/sohbetinize .sql.gz yedek dosyası ve raporu gönderildi.")
	fmt.Println("==============================================================================")
}
