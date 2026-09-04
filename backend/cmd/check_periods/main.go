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

	_, _ = pool.Exec(ctx, `
		INSERT INTO public.tenant_members (tenant_id, user_id, role)
		VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', 'admin')
		ON CONFLICT (tenant_id, user_id) DO NOTHING;
	`)

	mRows, err := pool.Query(ctx, `SELECT tenant_id, user_id, role FROM public.tenant_members`)
	if err == nil {
		defer mRows.Close()
		fmt.Println("👥 CANLI TENANT MEMBERS:")
		for mRows.Next() {
			var tID, uID, role string
			_ = mRows.Scan(&tID, &uID, &role)
			fmt.Printf("Tenant: %s | User: %s | Role: %s\n", tID, uID, role)
		}
		fmt.Println("==========================================")
	}
}
