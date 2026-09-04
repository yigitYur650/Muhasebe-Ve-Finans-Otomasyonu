package main

import (
	"context"
	"fmt"
	"log"

	"deftersystem/backend/internal/repository"
)

func main() {
	dbURL := "postgres://postgres.xtmfsdvwlminlchpustb:6uNlbk0wlN5TuSDZ@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require"
	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil {
		log.Fatalf("Pool err: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// 1. Inspect periods
	rows, err := pool.Query(ctx, `SELECT id, label, status FROM public.periods`)
	if err != nil {
		log.Fatalf("Periods query err: %v", err)
	}
	defer rows.Close()

	fmt.Println("=== PERIODS IN SUPABASE ===")
	for rows.Next() {
		var id, label, status string
		_ = rows.Scan(&id, &label, &status)
		fmt.Printf("ID: %s | Label: %s | Status: %s\n", id, label, status)
	}

	// 2. Inspect target tx
	txID := "4b0f4ed9-dd8d-40d2-9af2-a68c4c32e82e"
	var id, periodID, dir, amount, desc string
	var reversedBy *string
	err = pool.QueryRow(ctx, `SELECT id, period_id, direction, amount, description, reversed_by FROM public.transactions WHERE id = $1`, txID).Scan(&id, &periodID, &dir, &amount, &desc, &reversedBy)
	if err != nil {
		fmt.Printf("\nTarget Tx %s query error: %v\n", txID, err)
	} else {
		revStr := "nil"
		if reversedBy != nil {
			revStr = *reversedBy
		}
		fmt.Printf("\nTarget Tx Found: ID=%s | PeriodID=%s | Dir=%s | Amt=%s | Desc=%s | ReversedBy=%s\n", id, periodID, dir, amount, desc, revStr)
	}
}
