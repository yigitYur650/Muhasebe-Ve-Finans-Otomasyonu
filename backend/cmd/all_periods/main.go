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

	rows, err := pool.Query(ctx, `SELECT id, tenant_id, label, starting_balance, status, opened_at, locked_at FROM public.periods`)
	if err != nil {
		log.Fatalf("Query err: %v", err)
	}
	defer rows.Close()

	fmt.Println("==========================================")
	fmt.Println("📌 ALL PERIODS IN SUPABASE DATABASE:")
	fmt.Println("==========================================")
	count := 0
	for rows.Next() {
		count++
		var id, tenantID, label, status string
		var startingBalance interface{}
		var openedAt interface{}
		var lockedAt interface{}
		_ = rows.Scan(&id, &tenantID, &label, &startingBalance, &status, &openedAt, &lockedAt)
		fmt.Printf("#%d | ID: %s | Label: %s | Status: '%s' | LockedAt: %v\n", count, id, label, status, lockedAt)
	}
	fmt.Printf("Total Periods Found: %d\n", count)
	fmt.Println("==========================================")
}
