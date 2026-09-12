package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"deftersystem/backend/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BackupPayload struct {
	ExportedAt     string                   `json:"exported_at"`
	Environment    string                   `json:"environment"`
	TotalPeriods   int                      `json:"total_periods"`
	TotalTxs       int                      `json:"total_transactions"`
	TotalIn        string                   `json:"total_in"`
	TotalOut       string                   `json:"total_out"`
	ClosingBalance string                   `json:"closing_balance"`
	Tenants        []map[string]interface{} `json:"tenants"`
	TenantMembers  []map[string]interface{} `json:"tenant_members"`
	Periods        []map[string]interface{} `json:"periods"`
	Transactions   []map[string]interface{} `json:"transactions"`
	UserSecurity   []map[string]interface{} `json:"user_security"`
}

func main() {
	dbURL := getDatabaseURL()
	if dbURL == "" {
		log.Fatalf("❌ DATABASE_URL bulunamadı.")
	}

	fmt.Println("==========================================================================")
	fmt.Println("💾 DEFTER-İ KEBİR — GÜVENLİ VERİTABANI YEDEKLEME MOTORU (OFFLINE BACKUP)")
	fmt.Println("==========================================================================")

	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil {
		log.Fatalf("❌ Veritabanı bağlantı hatası: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// 1. backups/ klasörünün varlığını garantiye al
	backupDir := "backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		backupDir = "../backups"
		_ = os.MkdirAll(backupDir, 0755)
	}

	timestamp := time.Now().Format("2006-01-02_150405")
	jsonFile := filepath.Join(backupDir, fmt.Sprintf("defter_backup_%s.json", timestamp))
	sqlFile := filepath.Join(backupDir, fmt.Sprintf("defter_backup_%s.sql", timestamp))

	payload := BackupPayload{
		ExportedAt:  time.Now().Format(time.RFC3339),
		Environment: "Supabase Production Cloud",
	}

	// 2. Tabloları oku (UUID ve Numeric tiplerini temiz metin formatında çek)
	payload.Tenants = fetchTableRows(ctx, pool, "SELECT id::text, name, created_at::text FROM public.tenants ORDER BY created_at ASC")
	payload.TenantMembers = fetchTableRows(ctx, pool, "SELECT id::text, tenant_id::text, user_id::text, role, created_at::text FROM public.tenant_members ORDER BY created_at ASC")
	payload.Periods = fetchTableRows(ctx, pool, "SELECT id::text, tenant_id::text, label, starting_balance::text, status, opened_at::text, locked_at::text FROM public.periods ORDER BY label ASC")
	payload.Transactions = fetchTableRows(ctx, pool, "SELECT id::text, tenant_id::text, period_id::text, direction, channel, amount::text, description, created_by::text, created_at::text, reversed_by::text FROM public.transactions ORDER BY created_at ASC")
	payload.UserSecurity = fetchTableRows(ctx, pool, "SELECT user_id::text, security_question, updated_at::text FROM public.user_security")

	payload.TotalPeriods = len(payload.Periods)
	payload.TotalTxs = len(payload.Transactions)

	// Toplam bakiye hesabı
	var sumIn, sumOut float64
	for _, tx := range payload.Transactions {
		revBy, hasRev := tx["reversed_by"]
		if hasRev && revBy != nil && revBy != "" {
			continue // İptal edilmiş işlemleri net toplama katma
		}
		desc, _ := tx["description"].(string)
		if strings.HasPrefix(desc, "[İPTAL/TERS KAYIT]") {
			continue // Ters kayıt satırını katma
		}

		amtStr := fmt.Sprintf("%v", tx["amount"])
		var amt float64
		_, _ = fmt.Sscanf(amtStr, "%f", &amt)

		dir, _ := tx["direction"].(string)
		if dir == "in" {
			sumIn += amt
		} else if dir == "out" {
			sumOut += amt
		}
	}

	payload.TotalIn = fmt.Sprintf("%.2f TL", sumIn)
	payload.TotalOut = fmt.Sprintf("%.2f TL", sumOut)
	payload.ClosingBalance = fmt.Sprintf("%.2f TL", sumIn-sumOut)

	// 3. JSON formatında kaydet
	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Fatalf("❌ JSON formatlama hatası: %v", err)
	}

	if err := os.WriteFile(jsonFile, jsonData, 0644); err != nil {
		log.Fatalf("❌ JSON yedek dosyası yazılamadı: %v", err)
	}

	// 4. Doğrudan psql / Supabase ile geri yüklenebilir SQL dosyası üret
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString(fmt.Sprintf("-- DEFTER-İ KEBİR YEDEKLEME DÖKÜMÜ\n-- Tarih: %s\n-- Toplam İşlem: %d | Net Kasa: %s\n\nBEGIN;\n\n",
		time.Now().Format("2006-01-02 15:04:05"), payload.TotalTxs, payload.ClosingBalance))

	// Tenants
	for _, t := range payload.Tenants {
		sqlBuilder.WriteString(fmt.Sprintf("INSERT INTO public.tenants (id, name, created_at) VALUES ('%v', '%v', '%v') ON CONFLICT (id) DO NOTHING;\n",
			t["id"], escapeSQL(fmt.Sprintf("%v", t["name"])), t["created_at"]))
	}
	sqlBuilder.WriteString("\n")

	// Periods
	for _, p := range payload.Periods {
		sqlBuilder.WriteString(fmt.Sprintf("INSERT INTO public.periods (id, tenant_id, label, starting_balance, status, opened_at) VALUES ('%v', '%v', '%v', %v, '%v', '%v') ON CONFLICT (id) DO NOTHING;\n",
			p["id"], p["tenant_id"], p["label"], p["starting_balance"], p["status"], p["opened_at"]))
	}
	sqlBuilder.WriteString("\n")

	// Transactions
	for _, tx := range payload.Transactions {
		revVal := "NULL"
		if tx["reversed_by"] != nil && fmt.Sprintf("%v", tx["reversed_by"]) != "" {
			revVal = fmt.Sprintf("'%v'", tx["reversed_by"])
		}
		desc := escapeSQL(fmt.Sprintf("%v", tx["description"]))
		sqlBuilder.WriteString(fmt.Sprintf("INSERT INTO public.transactions (id, tenant_id, period_id, direction, channel, amount, description, created_by, created_at, reversed_by) VALUES ('%v', '%v', '%v', '%v', '%v', %v, '%v', '%v', '%v', %s) ON CONFLICT (id) DO NOTHING;\n",
			tx["id"], tx["tenant_id"], tx["period_id"], tx["direction"], tx["channel"], tx["amount"], desc, tx["created_by"], tx["created_at"], revVal))
	}

	sqlBuilder.WriteString("\nCOMMIT;\n")

	if err := os.WriteFile(sqlFile, []byte(sqlBuilder.String()), 0644); err != nil {
		log.Fatalf("❌ SQL yedek dosyası yazılamadı: %v", err)
	}

	fmt.Printf("✅ YEDEKLEME BAŞARIYLA TAMAMLANDI!\n")
	fmt.Printf("   📊 Toplam Dönem: %d\n", payload.TotalPeriods)
	fmt.Printf("   📝 Toplam İşlem: %d\n", payload.TotalTxs)
	fmt.Printf("   💰 Toplam Gelir: %s\n", payload.TotalIn)
	fmt.Printf("   💸 Toplam Gider: %s\n", payload.TotalOut)
	fmt.Printf("   ⚖️  Net Kasa:     %s\n", payload.ClosingBalance)
	fmt.Printf("   📁 JSON Yedeği: %s\n", jsonFile)
	fmt.Printf("   📜 SQL Yedeği:  %s\n", sqlFile)
	fmt.Println("==========================================================================")
}

func fetchTableRows(ctx context.Context, pool *pgxpool.Pool, query string) []map[string]interface{} {
	rows, err := pool.Query(ctx, query)
	if err != nil {
		log.Printf("⚠️ Tablo sorgulanamadı (%s): %v", query, err)
		return nil
	}
	defer rows.Close()

	var result []map[string]interface{}
	fieldDescs := rows.FieldDescriptions()

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			continue
		}
		rowMap := make(map[string]interface{})
		for i, fd := range fieldDescs {
			rowMap[string(fd.Name)] = values[i]
		}
		result = append(result, rowMap)
	}
	return result
}

func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func getDatabaseURL() string {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		return dbURL
	}
	envPaths := []string{".env", "../.env", "backend/.env"}
	for _, p := range envPaths {
		data, err := os.ReadFile(p)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "DATABASE_URL=") {
					dbURL = strings.TrimPrefix(line, "DATABASE_URL=")
					return strings.Trim(dbURL, `"'`)
				}
			}
		}
	}
	return ""
}
