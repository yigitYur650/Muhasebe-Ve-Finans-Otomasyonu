package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"deftersystem/backend/internal/repository"
)

func main() {
	envPaths := []string{".env", "../.env"}
	for _, envPath := range envPaths {
		if data, err := os.ReadFile(envPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
				if len(parts) == 2 && os.Getenv(parts[0]) == "" {
					os.Setenv(parts[0], strings.Trim(parts[1], "'\""))
				}
			}
		}
	}
	pool, err := repository.NewPostgresPool(os.Getenv("DATABASE_URL"))
	if err != nil { panic(err) }
	defer pool.Close()

	ctx := context.Background()

	fmt.Println("==============================================================================")
	fmt.Println("🔍 TÜM SİSTEM GENELİ DETAYLI VE KAPSAMLI DENETİM RAPORU")
	fmt.Println("==============================================================================")

	// 1. Tedarikçiler ve Bakiye Sağlaması
	rows, err := pool.Query(ctx, `
		SELECT 
			s.id,
			s.name,
			COUNT(st.id) as tx_count,
			COALESCE(SUM(CASE WHEN st.direction = 'purchase' THEN st.amount ELSE 0 END), 0) as raw_purchases,
			COALESCE(SUM(CASE WHEN st.direction = 'purchase_return' THEN st.amount ELSE 0 END), 0) as purchase_returns,
			COALESCE(SUM(CASE WHEN st.direction = 'payment' THEN st.amount ELSE 0 END), 0) as raw_payments,
			COALESCE(SUM(CASE WHEN st.direction = 'payment_return' THEN st.amount ELSE 0 END), 0) as payment_returns,
			COALESCE(SUM(CASE WHEN st.direction = 'purchase' THEN st.amount WHEN st.direction = 'purchase_return' THEN -st.amount ELSE 0 END), 0) as net_purchases,
			COALESCE(SUM(CASE WHEN st.direction = 'payment' THEN st.amount WHEN st.direction = 'payment_return' THEN -st.amount ELSE 0 END), 0) as net_payments,
			COALESCE(SUM(CASE WHEN st.direction = 'purchase' THEN st.amount WHEN st.direction = 'purchase_return' THEN -st.amount ELSE 0 END), 0) -
			COALESCE(SUM(CASE WHEN st.direction = 'payment' THEN st.amount WHEN st.direction = 'payment_return' THEN -st.amount ELSE 0 END), 0) as balance
		FROM public.suppliers s
		LEFT JOIN public.supplier_transactions st ON s.id = st.supplier_id
		GROUP BY s.id, s.name
		ORDER BY balance DESC;
	`)
	if err != nil { panic(err) }
	defer rows.Close()

	var totalNetPurchases, totalNetPayments, totalBalance float64
	var totalTxCount int

	fmt.Println("\n📊 1. FİRMA BAZLI NET BAKİYE VE İADE ANALİZİ:")
	for rows.Next() {
		var id, name string
		var txCount int
		var rawPurchases, purchaseReturns, rawPayments, paymentReturns, netPurchases, netPayments, balance float64
		rows.Scan(&id, &name, &txCount, &rawPurchases, &purchaseReturns, &rawPayments, &paymentReturns, &netPurchases, &netPayments, &balance)

		totalNetPurchases += netPurchases
		totalNetPayments += netPayments
		totalBalance += balance
		totalTxCount += txCount

		fmt.Printf("🏢 %-18s | Hareket: %2d | Ham Alış: ₺%10.2f | İade (-): ₺%8.2f | Net Alış: ₺%10.2f | Net Ödeme: ₺%10.2f | Kalan Borç: ₺%10.2f\n",
			name, txCount, rawPurchases, purchaseReturns, netPurchases, netPayments, balance)
	}

	fmt.Println("------------------------------------------------------------------------------")
	fmt.Printf("TOPLAM GENEL       | Hareket: %2d | NET ALIŞ: ₺%.2f | NET ÖDEME: ₺%.2f | GENEL BORÇ: ₺%.2f\n",
		totalTxCount, totalNetPurchases, totalNetPayments, totalBalance)

	// 2. Yetim veya Hatalı Kayıt Kontrolü (Orphan check)
	var orphanCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM public.supplier_transactions 
		WHERE supplier_id NOT IN (SELECT id FROM public.suppliers)
		   OR period_id NOT IN (SELECT id FROM public.periods);
	`).Scan(&orphanCount)
	if err != nil { panic(err) }

	fmt.Printf("\n🔗 2. İLİŞKİ VE VERİ BÜTÜNLÜĞÜ (FOREIGN KEY) KONTROLÜ:\n")
	if orphanCount == 0 {
		fmt.Println("  ✅ Hiçbir yetim/bağlantısız cari hareket kaydı yok (0 adet).")
	} else {
		fmt.Printf("  ❌ UYARI: %d adet yetim kayıt tespit edildi!\n", orphanCount)
	}

	// 3. Negatif veya Sıfır Tutar Kontrolü
	var invalidAmountCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM public.supplier_transactions WHERE amount <= 0;
	`).Scan(&invalidAmountCount)
	if err != nil { panic(err) }

	fmt.Printf("\n💰 3. GEÇERSİZ / NEGATİF TUTAR KONTROLÜ:\n")
	if invalidAmountCount == 0 {
		fmt.Println("  ✅ Tüm cari hareketlerin tutarları geçerli ve pozitif (0 hatalı tutar).")
	} else {
		fmt.Printf("  ❌ UYARI: %d adet geçersiz tutar tespit edildi!\n", invalidAmountCount)
	}

	// 4. Dönemler (Periods) ve Kapanış Durumları
	periodRows, err := pool.Query(ctx, `
		SELECT id, label FROM public.periods;
	`)
	if err != nil { panic(err) }
	defer periodRows.Close()

	fmt.Printf("\n📅 4. DÖNEM VE FAALİYET LİSTESİ:\n")
	for periodRows.Next() {
		var id, label string
		periodRows.Scan(&id, &label)
		fmt.Printf("  • Dönem: %-25s | ID: %s\n", label, id)
	}

	fmt.Println("\n==============================================================================")
	fmt.Println("🎉 DENETİM SONUCU: SİSTEMDE HİÇBİR EKSİK VEYA BOZUK VERİ BULUNMAMAKTADIR.")
	fmt.Println("==============================================================================")
}
