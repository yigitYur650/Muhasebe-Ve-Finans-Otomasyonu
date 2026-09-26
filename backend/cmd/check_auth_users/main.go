package main

import (
	"context"
	"fmt"
	"time"

	"deftersystem/backend/internal/repository"
)

func main() {
	dbURL := "postgres://postgres.lvsngrrdzjhbawhcuzqz:nptn0P5vEbyLm6iM@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil {
		fmt.Printf("Pool error: %v\n", err)
		return
	}
	defer pool.Close()

	fmt.Println("==========================================================================")
	fmt.Println("👥 CANLI SUPABASE AUTH KULLANICILARI (auth.users) VE ROLLERİ:")
	fmt.Println("==========================================================================")

	rows, err := pool.Query(ctx, `
		SELECT u.id, u.email, u.email_confirmed_at, u.last_sign_in_at, tm.role, tm.tenant_id
		FROM auth.users u
		LEFT JOIN public.tenant_members tm ON u.id = tm.user_id
	`)
	if err != nil {
		fmt.Printf("Query error: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, email string
		var emailConfirmedAt, lastSignInAt *time.Time
		var role, tenantID *string
		_ = rows.Scan(&id, &email, &emailConfirmedAt, &lastSignInAt, &role, &tenantID)

		roleStr := "<YOK>"
		if role != nil {
			roleStr = *role
		}
		tenantStr := "<YOK>"
		if tenantID != nil {
			tenantStr = *tenantID
		}
		confirmed := "HAYIR"
		if emailConfirmedAt != nil {
			confirmed = "EVET (" + emailConfirmedAt.Format("2006-01-02") + ")"
		}

		fmt.Printf("📧 Email: %-25s | ID: %s\n", email, id)
		fmt.Printf("   ├─ Rol: %-10s | Tenant: %s\n", roleStr, tenantStr)
		fmt.Printf("   └─ Email Onaylı mı: %s\n", confirmed)
		fmt.Println("--------------------------------------------------------------------------")
	}
	fmt.Println("==========================================================================")
}
