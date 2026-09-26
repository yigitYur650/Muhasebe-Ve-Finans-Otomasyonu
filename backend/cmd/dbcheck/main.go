package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbURL := "postgres://postgres.lvsngrrdzjhbawhcuzqz:nptn0P5vEbyLm6iM@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Printf("Pool error: %v\n", err)
		return
	}
	defer pool.Close()

	var tID, tName, tSlug string
	err = pool.QueryRow(ctx, "SELECT id, name, slug FROM public.tenants ORDER BY created_at ASC LIMIT 1").Scan(&tID, &tName, &tSlug)
	if err != nil {
		fmt.Printf("❌ Tenants tablosunda kayıt yok veya sorgu hatası: %v\n", err)
	} else {
		fmt.Printf("✅ Birincil Tenant Bulundu: ID=%s | Name=%s | Slug=%s\n", tID, tName, tSlug)
	}

	rows, err := pool.Query(ctx, "SELECT id, email FROM auth.users")
	if err != nil {
		fmt.Printf("Auth users query error: %v\n", err)
	} else {
		fmt.Println("\n👥 SUPABASE AUTH KULLANICILARI (auth.users):")
		for rows.Next() {
			var uID, email string
			_ = rows.Scan(&uID, &email)
			fmt.Printf("   User: ID=%s | Email=%s\n", uID, email)
		}
		rows.Close()
	}
}
