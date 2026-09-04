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

	var id, periodID, dir, amount, desc string
	err = pool.QueryRow(ctx, `SELECT id, period_id, direction, amount, description FROM public.transactions WHERE id = '4b0f4ed9-dd8d-40d2-9af2-a68c4c32e82e'`).Scan(&id, &periodID, &dir, &amount, &desc)
	if err != nil {
		fmt.Printf("Tx not found: %v\n", err)
		return
	}
	fmt.Printf("Tx %s period_id is: %s\n", id, periodID)

	var pLabel, pStatus string
	err = pool.QueryRow(ctx, `SELECT label, status FROM public.periods WHERE id = $1`, periodID).Scan(&pLabel, &pStatus)
	if err != nil {
		fmt.Printf("Period %s not found: %v\n", periodID, err)
		return
	}
	fmt.Printf("Period %s Label: %s | Status: %s\n", periodID, pLabel, pStatus)
}
