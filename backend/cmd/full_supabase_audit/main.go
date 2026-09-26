package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"deftersystem/backend/internal/repository"
)

type AuditReport struct {
	Timestamp          string                   `json:"timestamp"`
	TotalTenants       int                      `json:"total_tenants"`
	TotalMembers       int                      `json:"total_members"`
	TotalPeriods       int                      `json:"total_periods"`
	TotalTransactions  int                      `json:"total_transactions"`
	TotalSuppliers     int                      `json:"total_suppliers"`
	TotalSupplierTxs   int                      `json:"total_supplier_transactions"`
	SuspectRecords     []map[string]interface{} `json:"suspect_records"`
	AllPeriods         []map[string]interface{} `json:"all_periods"`
	AllTransactions    []map[string]interface{} `json:"all_transactions"`
}

func main() {
	dbURL := "postgres://postgres.lvsngrrdzjhbawhcuzqz:nptn0P5vEbyLm6iM@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil {
		fmt.Printf("❌ Veritabanı bağlantı hatası: %v\n", err)
		return
	}
	defer pool.Close()

	report := AuditReport{
		Timestamp:       time.Now().Format(time.RFC3339),
		SuspectRecords:  make([]map[string]interface{}, 0),
		AllPeriods:      make([]map[string]interface{}, 0),
		AllTransactions: make([]map[string]interface{}, 0),
	}

	// 1. Tenants
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM public.tenants").Scan(&report.TotalTenants)
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM public.tenant_members").Scan(&report.TotalMembers)

	// 2. Periods
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM public.periods").Scan(&report.TotalPeriods)
	pRows, err := pool.Query(ctx, "SELECT id, tenant_id, label, status, starting_balance, opened_at, locked_at FROM public.periods ORDER BY opened_at ASC")
	if err == nil {
		for pRows.Next() {
			var id, tenantID, label, status, start string
			var openedAt time.Time
			var lockedAt *time.Time
			_ = pRows.Scan(&id, &tenantID, &label, &status, &start, &openedAt, &lockedAt)
			report.AllPeriods = append(report.AllPeriods, map[string]interface{}{
				"id":               id,
				"tenant_id":        tenantID,
				"label":            label,
				"status":           status,
				"starting_balance": start,
				"opened_at":        openedAt.Format("2006-01-02 15:04:05"),
				"locked_at":        lockedAt,
			})
		}
		pRows.Close()
	}

	// 3. Transactions
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM public.transactions").Scan(&report.TotalTransactions)
	tRows, err := pool.Query(ctx, `
		SELECT id, tenant_id, period_id, direction, channel, amount, description, created_by, created_at, reversed_by 
		FROM public.transactions 
		ORDER BY created_at ASC
	`)
	if err == nil {
		for tRows.Next() {
			var id, tenantID, periodID, dir, chanName, amt, desc, createdBy string
			var createdAt time.Time
			var revBy *string
			_ = tRows.Scan(&id, &tenantID, &periodID, &dir, &chanName, &amt, &desc, &createdBy, &createdAt, &revBy)

			txData := map[string]interface{}{
				"id":          id,
				"tenant_id":   tenantID,
				"period_id":   periodID,
				"direction":   dir,
				"channel":     chanName,
				"amount":      amt,
				"description": desc,
				"created_by":  createdBy,
				"created_at":  createdAt.Format("2006-01-02 15:04:05"),
				"reversed_by": revBy,
			}
			report.AllTransactions = append(report.AllTransactions, txData)

			// Mock / Test Verisi Taraması (Test kelimeleri veya sentetik kalıplar)
			descLower := strings.ToLower(desc)
			if strings.Contains(descLower, "test") || strings.Contains(descLower, "mock") || strings.Contains(descLower, "yaml") || strings.Contains(descLower, "sehven") || strings.Contains(descLower, "yazilim hizmet bedeli") {
				report.SuspectRecords = append(report.SuspectRecords, txData)
			}
		}
		tRows.Close()
	}

	// 4. Suppliers (Varsa)
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM public.suppliers").Scan(&report.TotalSuppliers)
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM public.supplier_transactions").Scan(&report.TotalSupplierTxs)

	// Dosyaya yazalım
	outBytes, _ := json.MarshalIndent(report, "", "  ")
	outFile := "supabase_live_data_audit.json"
	_ = os.WriteFile(outFile, outBytes, 0644)

	fmt.Println("==========================================================================")
	fmt.Printf("🔍 CANLI SUPABASE GERÇEK VERİ DENETİM VE ANALİZ RAPORU:\n")
	fmt.Printf("   🏢 Şirket/Tenant Sayısı    : %d\n", report.TotalTenants)
	fmt.Printf("   👥 Üye Sayısı              : %d\n", report.TotalMembers)
	fmt.Printf("   📅 Muhasebe Dönemi Sayısı  : %d\n", report.TotalPeriods)
	fmt.Printf("   📝 Toplam Gerçek İşlem     : %d\n", report.TotalTransactions)
	fmt.Printf("   🚚 Tedarikçi Sayısı        : %d\n", report.TotalSuppliers)
	fmt.Printf("   📦 Tedarikçi Hareketi      : %d\n", report.TotalSupplierTxs)
	fmt.Printf("   ⚠️ Şüpheli / Mock İşlem    : %d adet tespit edildi\n", len(report.SuspectRecords))
	fmt.Println("==========================================================================")
	fmt.Printf("📁 Tüm veriler '%s' dosyasına eksiksiz olarak çekildi.\n", outFile)
}
