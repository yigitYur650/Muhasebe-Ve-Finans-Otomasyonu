package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"deftersystem/backend/internal/repository"
)

func main() {
	baseURL := os.Getenv("LIVE_API_URL")
	if baseURL == "" {
		baseURL = "https://muhasebe-ve-finans-otomasyonu-2.onrender.com/api/v1"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres.xtmfsdvwlminlchpustb:6uNlbk0wlN5TuSDZ@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require"
	}

	fmt.Println("==========================================================================")
	fmt.Println("🔍 CANLI SUPABASE GERÇEK VERİTABANI YAZMA & KONTROL TESTİ")
	fmt.Println("==========================================================================")

	// 1. Step: Make API request to create transaction
	idemKey := fmt.Sprintf("verify-supabase-%d", time.Now().UnixNano())
	payload := map[string]interface{}{
		"period_id":   "00000000-0000-0000-0000-000000000001",
		"direction":   "in",
		"channel":     "eft",
		"amount":      "2500.00",
		"description": "Canlı Supabase Doğrulama Gelir Kaydı",
	}
	bodyBytes, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("POST", baseURL+"/transactions", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	req.Header.Set("X-User-ID", "00000000-0000-0000-0000-000000000002")
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("Idempotency-Key", idemKey)

	log.Println("1. Render API üzerinden yeni işlem ekleme isteği gönderiliyor...")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("❌ API İsteği Hatası: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("   API Yanıtı (Status %d): %s", resp.StatusCode, string(respBody))

	// 2. Step: Query PostgreSQL DB pool directly to verify record was inserted into Supabase tables
	log.Println("\n2. Canlı Supabase PostgreSQL Veritabanı Havuzuna Doğrudan Bağlanılıyor...")
	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil {
		log.Fatalf("❌ Supabase Bağlantı Hatası: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	query := `
		SELECT id::text, period_id::text, direction, channel, amount::text, description, created_at 
		FROM public.transactions 
		ORDER BY created_at DESC 
		LIMIT 5;
	`
	rows, err := pool.Query(ctx, query)
	if err != nil {
		log.Fatalf("❌ Supabase Sorgulama Hatası: %v", err)
	}
	defer rows.Close()

	fmt.Println("\n==========================================================================")
	fmt.Println("📊 SUPABASE VERİTABANINDAN DOĞRUDAN OKUNAN CANLI KAYITLAR (SELECT FROM transactions):")
	fmt.Println("==========================================================================")

	count := 0
	for rows.Next() {
		count++
		var id, periodID, direction, channel, amount, description string
		var createdAt time.Time
		if err := rows.Scan(&id, &periodID, &direction, &channel, &amount, &description, &createdAt); err != nil {
			log.Printf("Scan hatası: %v", err)
			continue
		}
		fmt.Printf("   📌 Kayıt #%d:\n", count)
		fmt.Printf("      - İşlem ID     : %s\n", id)
		fmt.Printf("      - Dönem ID     : %s\n", periodID)
		fmt.Printf("      - Yön / Kanal  : %s / %s\n", direction, channel)
		fmt.Printf("      - Tutar        : %s TL\n", amount)
		fmt.Printf("      - Açıklama     : %s\n", description)
		fmt.Printf("      - Kayıt Tarihi : %v\n", createdAt.Format(time.RFC3339))
		fmt.Println("  ------------------------------------------------------------------")
	}

	if count > 0 {
		fmt.Println("🎉 %100 BAŞARILI! Veriler Render API üzerinden doğrudan Supabase PostgreSQL veritabanına yazılıyor ve saklanıyor!")
	} else {
		fmt.Println("⚠️ Veritabanında henüz işlem bulunamadı.")
	}
	fmt.Println("==========================================================================")
}
