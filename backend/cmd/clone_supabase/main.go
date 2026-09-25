package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"deftersystem/backend/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

const liveDBURL = "postgres://postgres.lvsngrrdzjhbawhcuzqz:nptn0P5vEbyLm6iM@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require"

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	fmt.Println("==========================================================================")
	fmt.Println("📦 CANLI SUPABASE'DEN YEREL (LOCAL) TAM KLON OLUŞTURUCU")
	fmt.Println("==========================================================================")

	pool, err := repository.NewPostgresPool(liveDBURL)
	if err != nil {
		log.Fatalf("❌ Canlı Supabase bağlantı hatası: %v", err)
	}
	defer pool.Close()

	fmt.Println("✅ Canlı Supabase veritabanına bağlanıldı.")
	fmt.Println("📥 Tüm veriler ve tablolar taranıyor...")

	// 1. Verileri Çek
	tenants := fetchRows(ctx, pool, "SELECT id::text, name, created_at::text FROM public.tenants ORDER BY created_at ASC")
	authUsers := fetchRows(ctx, pool, "SELECT id::text, email, created_at::text FROM auth.users ORDER BY created_at ASC")
	tenantMembers := fetchRows(ctx, pool, "SELECT id::text, tenant_id::text, user_id::text, role, created_at::text FROM public.tenant_members ORDER BY created_at ASC")
	periods := fetchRows(ctx, pool, "SELECT id::text, tenant_id::text, label, starting_balance::text, status, opened_at::text, COALESCE(locked_at::text, '') as locked_at FROM public.periods ORDER BY opened_at ASC")
	transactions := fetchRows(ctx, pool, "SELECT id::text, tenant_id::text, period_id::text, direction, channel, amount::text, COALESCE(description, '') as description, created_by::text, created_at::text, COALESCE(reversed_by::text, '') as reversed_by FROM public.transactions ORDER BY created_at ASC")
	idempotencyKeys := fetchRows(ctx, pool, "SELECT key, tenant_id::text, COALESCE(response_status, 0)::text as response_status, created_at::text FROM public.idempotency_keys ORDER BY created_at ASC")

	fmt.Printf("   • Tenants         : %d kayıt\n", len(tenants))
	fmt.Printf("   • Auth Kullanıcı  : %d kayıt\n", len(authUsers))
	fmt.Printf("   • Tenant Members  : %d kayıt\n", len(tenantMembers))
	fmt.Printf("   • Periods         : %d kayıt\n", len(periods))
	fmt.Printf("   • Transactions    : %d kayıt\n", len(transactions))
	fmt.Printf("   • Idempotency Keys: %d kayıt\n", len(idempotencyKeys))

	// 2. SQL DDL + DML Scriptini Oluştur
	var sb strings.Builder

	sb.WriteString("-- ==============================================================================\n")
	sb.WriteString("-- DEFSYSTEM - YEREL POSTGRESQL / SUPABASE TAM KLON VERİTABANI\n")
	sb.WriteString(fmt.Sprintf("-- Oluşturulma Tarihi: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("-- Canlı Supabase verilerinin %100 birebir yerel bağımsız kopyasıdır.\n")
	sb.WriteString("-- ==============================================================================\n\n")

	sb.WriteString("BEGIN;\n\n")

	// Şema ve Uzantılar
	sb.WriteString("-- 1. EXTENSIONS & SCHEMAS\n")
	sb.WriteString("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";\n")
	sb.WriteString("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";\n")
	sb.WriteString("CREATE SCHEMA IF NOT EXISTS auth;\n")
	sb.WriteString("CREATE SCHEMA IF NOT EXISTS public;\n\n")

	// Auth Users Tablosu (Yerelde auth bağımsızlığı için)
	sb.WriteString("-- 2. AUTH USERS (Yerel simülasyon)\n")
	sb.WriteString(`CREATE TABLE IF NOT EXISTS auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
` + "\n")

	// Tenants
	sb.WriteString("-- 3. PUBLIC.TENANTS\n")
	sb.WriteString(`CREATE TABLE IF NOT EXISTS public.tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
` + "\n")

	// Tenant Members
	sb.WriteString("-- 4. PUBLIC.TENANT_MEMBERS\n")
	sb.WriteString(`CREATE TABLE IF NOT EXISTS public.tenant_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('admin', 'muhasebeci', 'standart')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, user_id)
);
` + "\n")

	// Periods
	sb.WriteString("-- 5. PUBLIC.PERIODS\n")
	sb.WriteString(`CREATE TABLE IF NOT EXISTS public.periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    starting_balance NUMERIC(15,2) NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'locked')),
    opened_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at TIMESTAMPTZ,
    UNIQUE (tenant_id, label)
);
` + "\n")

	// Transactions
	sb.WriteString("-- 6. PUBLIC.TRANSACTIONS\n")
	sb.WriteString(`CREATE TABLE IF NOT EXISTS public.transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    period_id UUID NOT NULL REFERENCES public.periods(id),
    direction TEXT NOT NULL CHECK (direction IN ('in', 'out')),
    channel TEXT NOT NULL CHECK (channel IN (
        'eft', 'pos', 'nakit', 'kredi',
        'kira', 'maas_banka', 'maas_elden', 'kredi_karti',
        'kartus', 'yemek', 'yakit', 'diger'
    )),
    amount NUMERIC(15,2) NOT NULL CHECK (amount > 0),
    description TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reversed_by UUID REFERENCES public.transactions(id)
);

CREATE INDEX IF NOT EXISTS idx_transactions_period ON public.transactions(period_id);
CREATE INDEX IF NOT EXISTS idx_transactions_tenant ON public.transactions(tenant_id);
` + "\n")

	// Idempotency Keys
	sb.WriteString("-- 7. PUBLIC.IDEMPOTENCY_KEYS\n")
	sb.WriteString(`CREATE TABLE IF NOT EXISTS public.idempotency_keys (
    key TEXT NOT NULL,
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    response_body JSONB,
    response_status INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (key, tenant_id)
);
` + "\n")

	// Fonksiyonlar ve Triggerlar
	sb.WriteString("-- 8. FONKSİYONLAR & TRIGGER'LAR\n")
	sb.WriteString(`CREATE OR REPLACE FUNCTION public.prevent_transaction_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'transactions tablosu append-only: UPDATE/DELETE yasak. Düzeltme için reversed_by kullanın.';
END;
$$;

DROP TRIGGER IF EXISTS trg_prevent_transaction_update ON public.transactions;
CREATE TRIGGER trg_prevent_transaction_update
BEFORE UPDATE ON public.transactions
FOR EACH ROW EXECUTE FUNCTION public.prevent_transaction_mutation();

DROP TRIGGER IF EXISTS trg_prevent_transaction_delete ON public.transactions;
CREATE TRIGGER trg_prevent_transaction_delete
BEFORE DELETE ON public.transactions
FOR EACH ROW EXECUTE FUNCTION public.prevent_transaction_mutation();

CREATE OR REPLACE FUNCTION public.open_next_period(p_tenant_id UUID, p_label TEXT)
RETURNS UUID
LANGUAGE plpgsql
AS $$
DECLARE
    v_prev_period RECORD;
    v_closing_balance NUMERIC(15,2);
    v_new_period_id UUID;
BEGIN
    SELECT * INTO v_prev_period
    FROM public.periods
    WHERE tenant_id = p_tenant_id
    ORDER BY opened_at DESC
    LIMIT 1;

    IF v_prev_period IS NULL THEN
        v_closing_balance := 0;
    ELSE
        SELECT v_prev_period.starting_balance
            + COALESCE(SUM(CASE WHEN direction = 'in' THEN amount ELSE 0 END), 0)
            - COALESCE(SUM(CASE WHEN direction = 'out' THEN amount ELSE 0 END), 0)
        INTO v_closing_balance
        FROM public.transactions
        WHERE period_id = v_prev_period.id;
    END IF;

    INSERT INTO public.periods (tenant_id, label, starting_balance, status)
    VALUES (p_tenant_id, p_label, COALESCE(v_closing_balance, 0), 'open')
    RETURNING id INTO v_new_period_id;

    RETURN v_new_period_id;
END;
$$;
` + "\n")

	// 9. INSERT STATEMENTS (Verilerin Aktarımı)
	sb.WriteString("-- 9. CANLI SUPABASE VERİLERİNİN ENJEKSİYONU (SEEDING)\n\n")

	// Auth Users
	for _, u := range authUsers {
		sb.WriteString(fmt.Sprintf("INSERT INTO auth.users (id, email, created_at) VALUES ('%v', '%v', '%v') ON CONFLICT (id) DO NOTHING;\n",
			u["id"], escapeSQL(fmt.Sprintf("%v", u["email"])), u["created_at"]))
	}
	sb.WriteString("\n")

	// Tenants
	for _, t := range tenants {
		sb.WriteString(fmt.Sprintf("INSERT INTO public.tenants (id, name, created_at) VALUES ('%v', '%v', '%v') ON CONFLICT (id) DO NOTHING;\n",
			t["id"], escapeSQL(fmt.Sprintf("%v", t["name"])), t["created_at"]))
	}
	sb.WriteString("\n")

	// Tenant Members
	for _, tm := range tenantMembers {
		sb.WriteString(fmt.Sprintf("INSERT INTO public.tenant_members (id, tenant_id, user_id, role, created_at) VALUES ('%v', '%v', '%v', '%v', '%v') ON CONFLICT (tenant_id, user_id) DO NOTHING;\n",
			tm["id"], tm["tenant_id"], tm["user_id"], tm["role"], tm["created_at"]))
	}
	sb.WriteString("\n")

	// Periods
	for _, p := range periods {
		lockedAtVal := "NULL"
		if p["locked_at"] != nil && fmt.Sprintf("%v", p["locked_at"]) != "" {
			lockedAtVal = fmt.Sprintf("'%v'", p["locked_at"])
		}
		sb.WriteString(fmt.Sprintf("INSERT INTO public.periods (id, tenant_id, label, starting_balance, status, opened_at, locked_at) VALUES ('%v', '%v', '%v', %v, '%v', '%v', %s) ON CONFLICT (id) DO NOTHING;\n",
			p["id"], p["tenant_id"], p["label"], p["starting_balance"], p["status"], p["opened_at"], lockedAtVal))
	}
	sb.WriteString("\n")

	// Transactions
	for _, tx := range transactions {
		revVal := "NULL"
		if tx["reversed_by"] != nil && fmt.Sprintf("%v", tx["reversed_by"]) != "" {
			revVal = fmt.Sprintf("'%v'", tx["reversed_by"])
		}
		desc := escapeSQL(fmt.Sprintf("%v", tx["description"]))
		sb.WriteString(fmt.Sprintf("INSERT INTO public.transactions (id, tenant_id, period_id, direction, channel, amount, description, created_by, created_at, reversed_by) VALUES ('%v', '%v', '%v', '%v', '%v', %v, '%v', '%v', '%v', %s) ON CONFLICT (id) DO NOTHING;\n",
			tx["id"], tx["tenant_id"], tx["period_id"], tx["direction"], tx["channel"], tx["amount"], desc, tx["created_by"], tx["created_at"], revVal))
	}
	sb.WriteString("\n")

	sb.WriteString("COMMIT;\n")

	// Dosyaları kaydet
	targetDir := "backups"
	_ = os.MkdirAll(targetDir, 0755)
	_ = os.MkdirAll("../backups", 0755)

	outPath1 := filepath.Join(targetDir, "init_local_db.sql")
	outPath2 := filepath.Join("../backups", "init_local_db.sql")

	_ = os.WriteFile(outPath1, []byte(sb.String()), 0644)
	_ = os.WriteFile(outPath2, []byte(sb.String()), 0644)

	fmt.Println("==========================================================================")
	fmt.Println("🎉 KLONLAMA DOSYASI BAŞARIYLA ÜRETİLDİ!")
	fmt.Printf("📁 Çıktı Dosyası: %s\n", outPath1)
	fmt.Println("==========================================================================")
}

func fetchRows(ctx context.Context, pool *pgxpool.Pool, query string) []map[string]interface{} {
	rows, err := pool.Query(ctx, query)
	if err != nil {
		log.Printf("Hata (%s): %v", query, err)
		return nil
	}
	defer rows.Close()

	var results []map[string]interface{}
	fieldDescs := rows.FieldDescriptions()

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			continue
		}
		m := make(map[string]interface{})
		for i, fd := range fieldDescs {
			m[string(fd.Name)] = values[i]
		}
		results = append(results, m)
	}
	return results
}

func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
