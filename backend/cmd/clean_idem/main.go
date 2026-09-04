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

	cmd, err := pool.Exec(ctx, `TRUNCATE TABLE public.idempotency_keys;`)
	if err != nil {
		log.Fatalf("Truncate err: %v", err)
	}
	fmt.Printf("Cleared idempotency_keys table! Rows affected: %d\n", cmd.RowsAffected())
}
