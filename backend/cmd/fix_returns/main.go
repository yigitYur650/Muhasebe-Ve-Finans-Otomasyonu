package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"deftersystem/backend/internal/repository"
)

func main() {
	fmt.Println("🚀 [ÖNCÜ OTOGAZ] Geçmiş İptal/Ters Kayıtları Muhasebe Standartlarına (purchase_return/payment_return) Dönüştürme Başlatılıyor...")

	envPaths := []string{".env", "../.env", ".env.local", "../.env.local", "backend/.env"}
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

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatalf("❌ HATA: DATABASE_URL çevre değişkeni bulunamadı!")
	}

	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil || pool == nil {
		log.Fatalf("❌ HATA: Veritabanı havuzu başlatılamadı: %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Güncellenecek kayıtları tespit et
	rows, err := pool.Query(ctx, `
		SELECT id, supplier_id, invoice_no, amount, direction, description
		FROM public.supplier_transactions
		WHERE description ILIKE '%[İPTAL/TERS KAYIT]%' OR description ILIKE '%TERS KAYIT%' OR description ILIKE '%İPTAL%'
		ORDER BY tx_date DESC;
	`)
	if err != nil {
		log.Fatalf("Sorgu hatası: %v", err)
	}
	defer rows.Close()

	type TxToFix struct {
		ID          string
		SupplierID  string
		InvoiceNo   *string
		Amount      float64
		Direction   string
		Description string
	}

	var toFix []TxToFix
	for rows.Next() {
		var t TxToFix
		if err := rows.Scan(&t.ID, &t.SupplierID, &t.InvoiceNo, &t.Amount, &t.Direction, &t.Description); err != nil {
			log.Fatalf("Scan hatası: %v", err)
		}
		toFix = append(toFix, t)
	}

	fmt.Printf("🔍 Bulunan Ters Kayıt / İptal Satır Sayısı: %d\n", len(toFix))
	for _, t := range toFix {
		inv := "-"
		if t.InvoiceNo != nil {
			inv = *t.InvoiceNo
		}
		fmt.Printf("  • ID: %s | Fatura: %-18s | Tutar: %10.2f TL | Mevcut Yön: %-10s | Açıklama: %s\n",
			t.ID, inv, t.Amount, t.Direction, t.Description)
	}

	// 2. Kolon uzunluğunu ve kısıtları genişlet (VARCHAR(30))
	_, err = pool.Exec(ctx, `
		ALTER TABLE public.supplier_transactions ALTER COLUMN direction TYPE VARCHAR(30);
		ALTER TABLE public.supplier_transactions DROP CONSTRAINT IF EXISTS supplier_transactions_direction_check;
		ALTER TABLE public.supplier_transactions ADD CONSTRAINT supplier_transactions_direction_check 
			CHECK (direction IN ('purchase', 'payment', 'purchase_return', 'payment_return'));
	`)
	if err != nil {
		log.Printf("⚠️ Constraint/Kolon uyarısı (zaten uygun olabilir): %v\n", err)
	}

	tag, err := pool.Exec(ctx, `
		UPDATE public.supplier_transactions
		SET direction = 'purchase_return'
		WHERE (description ILIKE '%[İPTAL/TERS KAYIT]%' OR description ILIKE '%TERS KAYIT%' OR description ILIKE '%İPTAL%')
		  AND direction = 'payment';
	`)
	if err != nil {
		log.Fatalf("Güncelleme hatası: %v", err)
	}
	fmt.Printf("\n✅ %d adet satır 'purchase_return' (Alış İadesi) olarak güncellendi!\n", tag.RowsAffected())

	// 3. Güncel Tedarikçi Bakiye Özeti
	summaryRows, err := pool.Query(ctx, `
		SELECT 
			s.name,
			COALESCE(SUM(CASE WHEN st.direction = 'purchase' THEN st.amount WHEN st.direction = 'purchase_return' THEN -st.amount ELSE 0 END), 0) as total_purchases,
			COALESCE(SUM(CASE WHEN st.direction = 'payment' THEN st.amount WHEN st.direction = 'payment_return' THEN -st.amount ELSE 0 END), 0) as total_payments,
			COALESCE(SUM(CASE WHEN st.direction = 'purchase' THEN st.amount WHEN st.direction = 'purchase_return' THEN -st.amount ELSE 0 END), 0) -
			COALESCE(SUM(CASE WHEN st.direction = 'payment' THEN st.amount WHEN st.direction = 'payment_return' THEN -st.amount ELSE 0 END), 0) as balance
		FROM public.suppliers s
		LEFT JOIN public.supplier_transactions st ON s.id = st.supplier_id
		GROUP BY s.id, s.name
		ORDER BY balance DESC;
	`)
	if err != nil {
		log.Fatalf("Özet sorgusu hatası: %v", err)
	}
	defer summaryRows.Close()

	fmt.Println("\n==============================================================================")
	fmt.Println("📊 YENİ VE DOĞRULANMIŞ FİRMA BAKİYE ÇETELESİ:")
	fmt.Println("==============================================================================")
	for summaryRows.Next() {
		var name string
		var purchases, payments, balance float64
		if err := summaryRows.Scan(&name, &purchases, &payments, &balance); err != nil {
			log.Fatalf("Özet scan hatası: %v", err)
		}
		fmt.Printf("🏢 %-20s | Net Alış: ₺%12.2f | Net Ödeme: ₺%12.2f | Kalan Borç: ₺%12.2f\n",
			name, purchases, payments, balance)
	}
	fmt.Println("==============================================================================")
}
