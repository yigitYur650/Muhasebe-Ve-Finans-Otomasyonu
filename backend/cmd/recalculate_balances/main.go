package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"

	"deftersystem/backend/internal/repository"
)

func main() {
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

	tenantFlag := flag.String("tenant", "", "Tenant UUID to audit/recalculate (optional)")
	periodFlag := flag.String("period", "", "Period UUID to filter (optional)")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Println("⚠️  DATABASE_URL is not set. Exiting.")
		return
	}

	ctx := context.Background()
	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil || pool == nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer pool.Close()

	var tenantID uuid.UUID
	if *tenantFlag != "" {
		parsed, err := uuid.Parse(*tenantFlag)
		if err != nil {
			log.Fatalf("❌ Invalid tenant UUID: %v", err)
		}
		tenantID = parsed
	}

	var periodID *uuid.UUID
	if *periodFlag != "" {
		parsed, err := uuid.Parse(*periodFlag)
		if err != nil {
			log.Fatalf("❌ Invalid period UUID: %v", err)
		}
		periodID = &parsed
	}

	repo := repository.NewPostgresSupplierRepository(pool)

	fmt.Println("================================================================================")
	fmt.Println("📊 TEDARİKÇİ & CARİ HAREKET BAKİYE DENETİMİ (RECALCULATE BALANCES)")
	fmt.Println("================================================================================")

	suppliers, err := repo.GetSuppliersWithBalances(ctx, tenantID, periodID)
	if err != nil {
		log.Fatalf("❌ Error fetching supplier balances: %v", err)
	}

	fmt.Printf("%-25s | %-16s | %-16s | %-16s | %-10s\n",
		"Firma Adı", "Toplam Alış", "Toplam Ödeme", "Net Bakiye", "Aktif Hareket")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, s := range suppliers {
		fmt.Printf("%-25s | ₺%-15s | ₺%-15s | ₺%-15s | %-10d\n",
			s.Name,
			s.TotalPurchase.StringFixed(2),
			s.TotalPayment.StringFixed(2),
			s.Balance.StringFixed(2),
			s.TransactionCount,
		)
	}

	summary, err := repo.GetSummary(ctx, tenantID, periodID)
	if err != nil {
		log.Fatalf("❌ Error fetching summary: %v", err)
	}

	fmt.Println("================================================================================")
	fmt.Printf("Genel Toplam Alış   : ₺%s\n", summary.TotalPurchases.StringFixed(2))
	fmt.Printf("Genel Toplam Ödeme  : ₺%s\n", summary.TotalPayments.StringFixed(2))
	fmt.Printf("Genel Net Borç      : ₺%s\n", summary.NetBalance.StringFixed(2))
	fmt.Printf("Toplam Aktif Satır  : %d\n", summary.TotalTransactionRows)
	fmt.Printf("Aktif Tedarikçi     : %d\n", summary.ActiveSupplierCount)
	fmt.Println("================================================================================")
	fmt.Println("✅ Bakiye denetimi başarıyla tamamlandı.")
}
