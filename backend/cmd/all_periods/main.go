package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"deftersystem/backend/internal/repository"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres.lvsngrrdzjhbawhcuzqz:nptn0P5vEbyLm6iM@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require"
	}
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
		
		var txCount int
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM public.transactions WHERE period_id = $1", id).Scan(&txCount)
		
		fmt.Printf("#%d | ID: %s | Label: %s | Status: '%s' | StartingBalance: %v | TxCount: %d | LockedAt: %v\n", 
			count, id, label, status, startingBalance, txCount, lockedAt)
	}
	fmt.Printf("Total Periods Found: %d\n", count)
	fmt.Println("==========================================")
}
