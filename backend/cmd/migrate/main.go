package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"deftersystem/backend/internal/repository"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Try reading from .env or backend/.env
		data, err := os.ReadFile(".env")
		if err != nil {
			data, err = os.ReadFile("backend/.env")
		}
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "DATABASE_URL=") {
					dbURL = strings.TrimPrefix(line, "DATABASE_URL=")
					dbURL = strings.Trim(dbURL, `"'`)
					break
				}
			}
		}
	}

	if dbURL == "" {
		log.Fatalf("❌ DATABASE_URL bulunamadı. Lütfen .env dosyasını kontrol edin veya ortam değişkeni olarak ayarlayın.")
	}

	fmt.Println("==========================================================================")
	fmt.Println("🚀 SUPABASE VERİTABANI MİGRATİON ÇALIŞTIRICI")
	fmt.Println("==========================================================================")
	log.Println("1. Supabase PostgreSQL bağlantısı kuruluyor...")

	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil {
		log.Fatalf("❌ Veritabanı bağlantı hatası: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Locate combined schema file
	possiblePaths := []string{
		"migrations/supabase_combined_schema.sql",
		"../migrations/supabase_combined_schema.sql",
		filepath.Join(".", "migrations", "supabase_combined_schema.sql"),
	}

	var schemaPath string
	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			schemaPath = p
			break
		}
	}

	if schemaPath == "" {
		log.Fatalf("❌ migrations/supabase_combined_schema.sql dosyası bulunamadı!")
	}

	log.Printf("2. Migration dosyası okunuyor: %s", schemaPath)
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Fatalf("❌ Migration dosyası okunamadı: %v", err)
	}

	log.Println("3. Migration SQL komutları Supabase üzerinde çalıştırılıyor...")
	_, err = pool.Exec(ctx, string(content))
	if err != nil {
		log.Fatalf("❌ Migration çalıştırma hatası: %v", err)
	}

	fmt.Println("==========================================================================")
	fmt.Println("🎉 TEBRİKLER! TÜM SUPABASE TABLOLARI, TRIGGERLAR VE SEED BAŞARIYLA OLUŞTURULDU!")
	fmt.Println("   - public.tenants")
	fmt.Println("   - public.tenant_members")
	fmt.Println("   - public.periods")
	fmt.Println("   - public.transactions")
	fmt.Println("   - public.idempotency_keys")
	fmt.Println("   - public.user_security")
	fmt.Println("   - Tüm RLS kuralları ve fonksiyonlar")
	fmt.Println("==========================================================================")
}
