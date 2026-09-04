package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	urls := []string{
		"postgres://postgres.lvsngrrdzjhbawhcuzqz:nptn0P5vEbyLm6iM@aws-0-ap-northeast-1.pooler.supabase.com:5432/postgres?sslmode=require",
		"postgres://postgres.lvsngrrdzjhbawhcuzqz:nptn0P5vEbyLm6iM@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require",
	}

	for _, dbURL := range urls {
		fmt.Printf("Testing connection: %s\n", dbURL)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool, err := pgxpool.New(ctx, dbURL)
		if err != nil {
			fmt.Printf("  Pool init error: %v\n", err)
			cancel()
			continue
		}

		err = pool.Ping(ctx)
		if err != nil {
			fmt.Printf("  Ping error: %v\n", err)
		} else {
			fmt.Printf("  🎉 SUCCESS! Connected successfully!\n")
			var count int
			_ = pool.QueryRow(ctx, "SELECT count(*) FROM public.periods").Scan(&count)
			fmt.Printf("  Periods count in Supabase: %d\n", count)
		}
		pool.Close()
		cancel()
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSub(s, substr))
}

func containsSub(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
