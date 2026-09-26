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

	rows, err := pool.Query(ctx, `
		SELECT id, direction, channel, amount, description, created_at 
		FROM public.transactions 
		ORDER BY created_at DESC 
		LIMIT 20
	`)
	if err != nil {
		fmt.Printf("Query error: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Println("==========================================================================")
	fmt.Println("📌 SON EKLENEN 20 İŞLEM KAYDI:")
	fmt.Println("==========================================================================")
	for rows.Next() {
		var id, dir, chanName, amt, desc string
		var createdAt time.Time
		_ = rows.Scan(&id, &dir, &chanName, &amt, &desc, &createdAt)
		fmt.Printf("[%s] %s | %s | %10s TL | %-35s | ID: %s\n",
			createdAt.Format("2006-01-02 15:04:05"), dir, chanName, amt, desc, id)
	}
	fmt.Println("==========================================================================")
}
