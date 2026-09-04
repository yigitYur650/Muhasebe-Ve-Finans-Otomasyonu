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

	rows, err := pool.Query(ctx, `SELECT id, label, status FROM public.periods`)
	if err != nil {
		log.Fatalf("Query err: %v", err)
	}
	defer rows.Close()

	fmt.Println("==========================================")
	fmt.Println("📌 CANLI SUPABASE DÖNEM DURUMLARI:")
	fmt.Println("==========================================")
	for rows.Next() {
		var id, label, status string
		_ = rows.Scan(&id, &label, &status)
		fmt.Printf("Period ID: %s | Label: %s | Status: %s\n", id, label, status)
	}
	fmt.Println("==========================================")
}
