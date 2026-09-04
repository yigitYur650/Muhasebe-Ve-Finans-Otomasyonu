package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	regions := []string{
		"aws-0-eu-central-1",
		"aws-0-eu-west-1",
		"aws-0-eu-west-2",
		"aws-0-eu-west-3",
		"aws-0-eu-north-1",
		"aws-0-us-east-1",
		"aws-0-us-east-2",
		"aws-0-us-west-1",
		"aws-0-us-west-2",
		"aws-0-sa-east-1",
		"aws-0-ap-southeast-1",
		"aws-0-ap-northeast-1",
		"aws-0-ap-south-1",
	}

	for _, region := range regions {
		dbURL := fmt.Sprintf("postgres://postgres.xtmfsdvwlminlchpustb:6uNlbk0wlN5TuSDZ@%s.pooler.supabase.com:6543/postgres?sslmode=require", region)
		fmt.Printf("Testing region %s...\n", region)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		pool, err := pgxpool.New(ctx, dbURL)
		if err != nil {
			cancel()
			continue
		}

		err = pool.Ping(ctx)
		pool.Close()
		cancel()

		if err != nil {
			errStr := err.Error()
			if !contains(errStr, "ENOTFOUND") && !contains(errStr, "no such host") && !contains(errStr, "deadline exceeded") {
				fmt.Printf("🎉 REGION MATCH OR DIFFERENT ERROR on %s: %v\n", region, err)
			} else {
				fmt.Printf("   Not %s (%v)\n", region, errStr)
			}
		} else {
			fmt.Printf("🎉🎉🎉 SUCCESS! MATCHED REGION: %s\n", region)
			fmt.Printf("Full Connection String: %s\n", dbURL)
			return
		}
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
