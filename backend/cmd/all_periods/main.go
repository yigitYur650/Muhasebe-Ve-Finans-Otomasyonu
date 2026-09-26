package main

import (
	"context"
	"fmt"
	"time"

	"deftersystem/backend/internal/repository"
)

func main() {
	dbURL := "postgres://postgres.lvsngrrdzjhbawhcuzqz:nptn0P5vEbyLm6iM@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil {
		fmt.Printf("Pool error: %v\n", err)
		return
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, "SELECT id, label, status, starting_balance, opened_at, locked_at FROM public.periods ORDER BY opened_at ASC")
	if err != nil {
		fmt.Printf("Query error: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Println("==========================================================================")
	fmt.Println("📌 MEVCUT TÜM DÖNEMLER (public.periods):")
	fmt.Println("==========================================================================")
	periodIDsToDelete := make([]string, 0)
	for rows.Next() {
		var id, label, status, start string
		var openedAt time.Time
		var lockedAt *time.Time
		_ = rows.Scan(&id, &label, &status, &start, &openedAt, &lockedAt)
		fmt.Printf("ID: %s | Label: %-10s | Status: %-6s | StartBal: %s | Opened: %s\n",
			id, label, status, start, openedAt.Format("2006-01-02 15:04:05"))

		if label != "2026-08" {
			periodIDsToDelete = append(periodIDsToDelete, id)
		}
	}
	fmt.Println("==========================================================================")
	fmt.Printf("Silinecek Sahte / Mock Dönem Sayısı: %d\n", len(periodIDsToDelete))

	// Ayrıca transactions tablosunda 2026-08 harici dönemlere bağlı işlem var mı bakalım
	for _, pid := range periodIDsToDelete {
		var txCount int
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM public.transactions WHERE period_id = $1", pid).Scan(&txCount)
		fmt.Printf("  -> Dönem ID %s içinde %d işlem var.\n", pid, txCount)
	}
}
