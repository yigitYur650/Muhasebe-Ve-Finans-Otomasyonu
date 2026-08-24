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
		dbURL = "postgres://postgres.xtmfsdvwlminlchpustb:6uNlbk0wlN5TuSDZ@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require"
	}

	fmt.Println("==========================================================================")
	fmt.Println("🧹 SUPABASE VERİTABANI TESLİMAT SIFIRLAMA VE SEED ARACI")
	fmt.Println("==========================================================================")
	log.Printf("Canlı Supabase veritabanına bağlanılıyor...")

	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil {
		log.Fatalf("❌ Veritabanı bağlantı hatası: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	resetSQL := `
	BEGIN;
	TRUNCATE TABLE 
		public.transactions,
		public.idempotency_keys,
		public.user_security_questions,
		public.periods,
		public.tenant_members,
		public.tenants
	RESTART IDENTITY CASCADE;

	INSERT INTO public.tenants (id, name, created_at)
	VALUES (
		'00000000-0000-0000-0000-000000000001',
		'Öncü Otogaz Muhasebe ve Finans',
		NOW()
	);

	INSERT INTO public.tenant_members (tenant_id, user_id, role, joined_at)
	VALUES (
		'00000000-0000-0000-0000-000000000001',
		'00000000-0000-0000-0000-000000000002',
		'admin',
		NOW()
	);

	INSERT INTO public.periods (id, tenant_id, label, starting_balance, status, opened_at)
	VALUES (
		'00000000-0000-0000-0000-000000000001',
		'00000000-0000-0000-0000-000000000001',
		'2026-08',
		0.00,
		'open',
		NOW()
	);
	COMMIT;
	`

	_, err = pool.Exec(ctx, resetSQL)
	if err != nil {
		log.Fatalf("❌ Sıfırlama işlemi sırasında hata oluştu: %v", err)
	}

	fmt.Println("==========================================================================")
	fmt.Println("🎉 SUPABASE VERİTABANI KUSURSUZ ŞEKİLDE SIFIRLANDI VE İLK SEED YÜKLENDİ!")
	fmt.Println("   - Tüm test ve yük verileri temizlendi.")
	fmt.Println("   - İlk Şirket (Tenant) ve İlk Dönem (2026-08) 0.00 TL bakiye ile açıldı.")
	fmt.Println("   - Sistem canlı teslime hazır!")
	fmt.Println("==========================================================================")
}
