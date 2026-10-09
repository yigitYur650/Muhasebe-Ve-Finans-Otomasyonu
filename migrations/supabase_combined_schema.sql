-- ==============================================================================
-- DEFSYSTEM - SUPABASE FULL SCHEMA MIGRATION & SEED
-- ==============================================================================
-- Bu dosyayı Supabase Dashboard -> SQL Editor içerisine yapıştırıp "Run" 
-- butonuna basarak tüm veritabanı şemasını ve ilk verileri tek seferde kurabilirsiniz.
-- ==============================================================================

BEGIN;

-- 1. TENANTS (Çoklu İşletme Temel Tablosu)
CREATE TABLE IF NOT EXISTS public.tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.tenants ENABLE ROW LEVEL SECURITY;

-- 2. TENANT MEMBERS (Kullanıcı-İşletme Rol Eşleşmesi)
CREATE TABLE IF NOT EXISTS public.tenant_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('admin', 'muhasebeci', 'standart')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, user_id)
);

ALTER TABLE public.tenant_members ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Kullanıcı kendi üyeliklerini görür" ON public.tenant_members;
CREATE POLICY "Kullanıcı kendi üyeliklerini görür"
ON public.tenant_members FOR SELECT
TO authenticated
USING (user_id = auth.uid());

-- 3. CURRENT TENANT IDS YARDIMCI FONKSİYONU
CREATE OR REPLACE FUNCTION public.current_tenant_ids()
RETURNS SETOF UUID
LANGUAGE sql
STABLE
SECURITY DEFINER
AS $$
    SELECT tenant_id FROM public.tenant_members WHERE user_id = auth.uid();
$$;

-- 4. PERIODS (Muhasebe Dönemleri)
CREATE TABLE IF NOT EXISTS public.periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    label TEXT NOT NULL,                          -- ör. "2026-08"
    starting_balance NUMERIC(15,2) NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'locked')),
    opened_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at TIMESTAMPTZ,
    UNIQUE (tenant_id, label)
);

ALTER TABLE public.periods ENABLE ROW LEVEL SECURITY;

-- 5. TRANSACTIONS (Append-Only İşlem Defteri)
CREATE TABLE IF NOT EXISTS public.transactions (
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

ALTER TABLE public.transactions ENABLE ROW LEVEL SECURITY;

CREATE INDEX IF NOT EXISTS idx_transactions_period ON public.transactions(period_id);
CREATE INDEX IF NOT EXISTS idx_transactions_tenant ON public.transactions(tenant_id);

-- Dış anahtar esnekliği (API servis istekleri için)
ALTER TABLE public.transactions DROP CONSTRAINT IF EXISTS transactions_created_by_fkey;

-- 6. APPEND-ONLY & PERIOD LOCK TRIGGER'LARI
CREATE OR REPLACE FUNCTION public.prevent_transaction_mutation()
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

CREATE OR REPLACE FUNCTION public.prevent_write_to_locked_period()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    v_status TEXT;
BEGIN
    SELECT status INTO v_status FROM public.periods WHERE id = NEW.period_id;
    IF v_status = 'locked' THEN
        RAISE EXCEPTION 'Bu dönem kilitli, yeni işlem eklenemez (period_id: %)', NEW.period_id;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_prevent_locked_period_insert ON public.transactions;
CREATE TRIGGER trg_prevent_locked_period_insert
BEFORE INSERT ON public.transactions
FOR EACH ROW EXECUTE FUNCTION public.prevent_write_to_locked_period();

-- 7. OPEN_NEXT_PERIOD (Kapanış Bakiyesini Doğru Devreden Fonksiyon)
CREATE OR REPLACE FUNCTION public.open_next_period(p_tenant_id UUID, p_label TEXT)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
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

        -- Önceki açık dönemi otomatik kilitle (Sistemde daima tek açık dönem kuralı)
        IF v_prev_period.status = 'open' THEN
            UPDATE public.periods
            SET status = 'locked', locked_at = now()
            WHERE id = v_prev_period.id;
        END IF;
    END IF;

    INSERT INTO public.periods (tenant_id, label, starting_balance, status)
    VALUES (p_tenant_id, p_label, COALESCE(v_closing_balance, 0), 'open')
    RETURNING id INTO v_new_period_id;

    RETURN v_new_period_id;
END;
$$;

-- 8. ROW LEVEL SECURITY (RLS) POLİTİKALARI
DROP POLICY IF EXISTS "Tenant üyeleri kendi dönemlerini görür" ON public.periods;
CREATE POLICY "Tenant üyeleri kendi dönemlerini görür"
ON public.periods FOR SELECT
TO authenticated
USING (tenant_id IN (SELECT public.current_tenant_ids()));

DROP POLICY IF EXISTS "Sadece admin/muhasebeci dönem oluşturur" ON public.periods;
CREATE POLICY "Sadece admin/muhasebeci dönem oluşturur"
ON public.periods FOR INSERT
TO authenticated
WITH CHECK (
    tenant_id IN (
        SELECT tenant_id FROM public.tenant_members
        WHERE user_id = auth.uid() AND role IN ('admin', 'muhasebeci')
    )
);

DROP POLICY IF EXISTS "Tenant üyeleri kendi işlemlerini görür" ON public.transactions;
CREATE POLICY "Tenant üyeleri kendi işlemlerini görür"
ON public.transactions FOR SELECT
TO authenticated
USING (tenant_id IN (SELECT public.current_tenant_ids()));

DROP POLICY IF EXISTS "Tenant üyeleri kendi tenant'ına işlem ekler" ON public.transactions;
CREATE POLICY "Tenant üyeleri kendi tenant'ına işlem ekler"
ON public.transactions FOR INSERT
TO authenticated
WITH CHECK (tenant_id IN (SELECT public.current_tenant_ids()));

ALTER TABLE public.tenants FORCE ROW LEVEL SECURITY;
ALTER TABLE public.tenant_members FORCE ROW LEVEL SECURITY;
ALTER TABLE public.periods FORCE ROW LEVEL SECURITY;
ALTER TABLE public.transactions FORCE ROW LEVEL SECURITY;

GRANT USAGE ON SCHEMA public TO authenticated;
GRANT SELECT ON public.tenants TO authenticated;
GRANT SELECT ON public.tenant_members TO authenticated;
GRANT SELECT, INSERT ON public.periods TO authenticated;
GRANT SELECT, INSERT ON public.transactions TO authenticated;

-- 9. IDEMPOTENCY KEYS (Mükerrer İstek Önleme)
CREATE TABLE IF NOT EXISTS public.idempotency_keys (
    key TEXT NOT NULL,
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    response_body JSONB,
    response_status INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (key, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_idempotency_created_at ON public.idempotency_keys(created_at);

CREATE OR REPLACE FUNCTION public.cleanup_expired_idempotency_keys(p_ttl_hours INT DEFAULT 24)
RETURNS INT
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_deleted_count INT;
BEGIN
    DELETE FROM public.idempotency_keys
    WHERE created_at < (now() - (p_ttl_hours || ' hours')::INTERVAL);
    
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$;

GRANT SELECT, INSERT ON public.idempotency_keys TO authenticated;

-- 10. USER SECURITY (Güvenlik Sorusu ve Şifre Sıfırlama)
CREATE TABLE IF NOT EXISTS public.user_security (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    security_question TEXT NOT NULL,
    security_answer_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_security_email ON public.user_security(email);

ALTER TABLE public.user_security ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_security FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS user_security_owner_policy ON public.user_security;
CREATE POLICY user_security_owner_policy ON public.user_security
    FOR ALL
    USING (user_id = auth.uid())
    WITH CHECK (user_id = auth.uid());

-- 11. STRICT DATA LOSS PREVENTION & TRUNCATE PROTECTION (Migration 14)
DROP TRIGGER IF EXISTS trg_prevent_transaction_truncate ON public.transactions;
CREATE TRIGGER trg_prevent_transaction_truncate
BEFORE TRUNCATE ON public.transactions
FOR EACH STATEMENT EXECUTE FUNCTION public.prevent_transaction_mutation();

ALTER TABLE public.transactions DROP CONSTRAINT IF EXISTS transactions_tenant_id_fkey;
ALTER TABLE public.transactions ADD CONSTRAINT transactions_tenant_id_fkey
FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE RESTRICT;

ALTER TABLE public.periods DROP CONSTRAINT IF EXISTS periods_tenant_id_fkey;
ALTER TABLE public.periods ADD CONSTRAINT periods_tenant_id_fkey
FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE RESTRICT;

-- 12. SUPPLIERS & SUPPLIER TRANSACTIONS (DUAL-LEDGER) (Migration 15)
CREATE TABLE IF NOT EXISTS public.suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_suppliers_tenant_name UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_suppliers_tenant ON public.suppliers(tenant_id);

CREATE TABLE IF NOT EXISTS public.supplier_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    supplier_id UUID NOT NULL REFERENCES public.suppliers(id) ON DELETE CASCADE,
    period_id UUID NOT NULL REFERENCES public.periods(id) ON DELETE CASCADE,
    invoice_no VARCHAR(100),
    customer_name VARCHAR(150),
    document_status VARCHAR(50),
    tx_date DATE NOT NULL,
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('purchase', 'payment')),
    amount NUMERIC(15,2) NOT NULL CHECK (amount > 0),
    description TEXT,
    created_by UUID REFERENCES auth.users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_supplier_txs_tenant ON public.supplier_transactions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_supplier_txs_supplier ON public.supplier_transactions(supplier_id);
CREATE INDEX IF NOT EXISTS idx_supplier_txs_period ON public.supplier_transactions(period_id);
CREATE INDEX IF NOT EXISTS idx_supplier_txs_date ON public.supplier_transactions(tx_date);

ALTER TABLE public.suppliers ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.suppliers FORCE ROW LEVEL SECURITY;

ALTER TABLE public.supplier_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.supplier_transactions FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Tenant üyeleri kendi tedarikçilerini görür" ON public.suppliers;
CREATE POLICY "Tenant üyeleri kendi tedarikçilerini görür"
ON public.suppliers FOR SELECT
TO authenticated
USING (tenant_id IN (SELECT public.current_tenant_ids()));

DROP POLICY IF EXISTS "Tenant üyeleri kendi tenant'ına tedarikçi ekler" ON public.suppliers;
CREATE POLICY "Tenant üyeleri kendi tenant'ına tedarikçi ekler"
ON public.suppliers FOR INSERT
TO authenticated
WITH CHECK (tenant_id IN (SELECT public.current_tenant_ids()));

DROP POLICY IF EXISTS "Tenant üyeleri kendi tedarikçilerini günceller" ON public.suppliers;
CREATE POLICY "Tenant üyeleri kendi tedarikçilerini günceller"
ON public.suppliers FOR UPDATE
TO authenticated
USING (tenant_id IN (SELECT public.current_tenant_ids()))
WITH CHECK (tenant_id IN (SELECT public.current_tenant_ids()));

DROP POLICY IF EXISTS "Tenant üyeleri kendi tedarikçilerini siler" ON public.suppliers;
CREATE POLICY "Tenant üyeleri kendi tedarikçilerini siler"
ON public.suppliers FOR DELETE
TO authenticated
USING (tenant_id IN (SELECT public.current_tenant_ids()));

DROP POLICY IF EXISTS "Tenant üyeleri kendi tedarikçi hareketlerini görür" ON public.supplier_transactions;
CREATE POLICY "Tenant üyeleri kendi tedarikçi hareketlerini görür"
ON public.supplier_transactions FOR SELECT
TO authenticated
USING (tenant_id IN (SELECT public.current_tenant_ids()));

DROP POLICY IF EXISTS "Tenant üyeleri kendi tenant'ına tedarikçi hareketi ekler" ON public.supplier_transactions;
CREATE POLICY "Tenant üyeleri kendi tenant'ına tedarikçi hareketi ekler"
ON public.supplier_transactions FOR INSERT
TO authenticated
WITH CHECK (tenant_id IN (SELECT public.current_tenant_ids()));

DROP POLICY IF EXISTS "Tenant üyeleri kendi tedarikçi hareketlerini günceller" ON public.supplier_transactions;
CREATE POLICY "Tenant üyeleri kendi tedarikçi hareketlerini günceller"
ON public.supplier_transactions FOR UPDATE
TO authenticated
USING (tenant_id IN (SELECT public.current_tenant_ids()))
WITH CHECK (tenant_id IN (SELECT public.current_tenant_ids()));

DROP POLICY IF EXISTS "Tenant üyeleri kendi tedarikçi hareketlerini siler" ON public.supplier_transactions;
CREATE POLICY "Tenant üyeleri kendi tedarikçi hareketlerini siler"
ON public.supplier_transactions FOR DELETE
TO authenticated
USING (tenant_id IN (SELECT public.current_tenant_ids()));

GRANT SELECT, INSERT, UPDATE, DELETE ON public.suppliers TO authenticated;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.supplier_transactions TO authenticated;

-- 13. BAŞLANGIÇ TOHUM VERİLERİ (SEED)
INSERT INTO public.tenants (id, name, created_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Öncü Otogaz Muhasebe ve Finans',
    NOW()
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO public.periods (id, tenant_id, label, starting_balance, status, opened_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    '2026-08',
    0.00,
    'open',
    NOW()
)
ON CONFLICT (tenant_id, label) DO NOTHING;

-- 14. OTOMATİK KULLANICI-İŞLETME (TENANT) BAĞLAMA TRİGGER'I
CREATE OR REPLACE FUNCTION public.handle_new_user()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    INSERT INTO public.tenant_members (tenant_id, user_id, role)
    VALUES (
        '00000000-0000-0000-0000-000000000001',
        NEW.id,
        'admin'
    )
    ON CONFLICT (tenant_id, user_id) DO NOTHING;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS on_auth_user_created ON auth.users;
CREATE TRIGGER on_auth_user_created
    AFTER INSERT ON auth.users
    FOR EACH ROW EXECUTE FUNCTION public.handle_new_user();

-- Halihazırda auth.users içinde olan kullanıcıları da geriye dönük bağla
INSERT INTO public.tenant_members (tenant_id, user_id, role)
SELECT '00000000-0000-0000-0000-000000000001', id, 'admin'
FROM auth.users
ON CONFLICT (tenant_id, user_id) DO NOTHING;

-- 15. ÇİFT DEFTER (DUAL-LEDGER) TEDARİKÇİ & CARİ TAKİP ŞEMASI
CREATE TABLE IF NOT EXISTS public.suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_suppliers_tenant_name UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_suppliers_tenant ON public.suppliers(tenant_id);

CREATE TABLE IF NOT EXISTS public.supplier_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    supplier_id UUID NOT NULL REFERENCES public.suppliers(id) ON DELETE CASCADE,
    period_id UUID NOT NULL REFERENCES public.periods(id) ON DELETE CASCADE,
    invoice_no VARCHAR(100),
    customer_name VARCHAR(150),
    document_status VARCHAR(50),
    tx_date DATE NOT NULL,
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('purchase', 'payment')),
    amount NUMERIC(15,2) NOT NULL CHECK (amount > 0),
    description TEXT,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reversed_by UUID REFERENCES public.supplier_transactions(id)
);

CREATE INDEX IF NOT EXISTS idx_supplier_txs_tenant ON public.supplier_transactions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_supplier_txs_supplier ON public.supplier_transactions(supplier_id);
CREATE INDEX IF NOT EXISTS idx_supplier_txs_period ON public.supplier_transactions(period_id);
CREATE INDEX IF NOT EXISTS idx_supplier_txs_date ON public.supplier_transactions(tx_date);
CREATE INDEX IF NOT EXISTS idx_supplier_txs_reversed_by ON public.supplier_transactions(reversed_by);

ALTER TABLE public.suppliers ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.suppliers FORCE ROW LEVEL SECURITY;

ALTER TABLE public.supplier_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.supplier_transactions FORCE ROW LEVEL SECURITY;

DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'suppliers' AND policyname = 'Tenant üyeleri kendi tedarikçilerini görür') THEN
        CREATE POLICY "Tenant üyeleri kendi tedarikçilerini görür"
        ON public.suppliers FOR SELECT
        TO authenticated
        USING (tenant_id IN (SELECT public.current_tenant_ids()));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'suppliers' AND policyname = 'Tenant üyeleri kendi tenant''ına tedarikçi ekler') THEN
        CREATE POLICY "Tenant üyeleri kendi tenant'ına tedarikçi ekler"
        ON public.suppliers FOR INSERT
        TO authenticated
        WITH CHECK (tenant_id IN (SELECT public.current_tenant_ids()));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'suppliers' AND policyname = 'Tenant üyeleri kendi tedarikçilerini günceller') THEN
        CREATE POLICY "Tenant üyeleri kendi tedarikçilerini günceller"
        ON public.suppliers FOR UPDATE
        TO authenticated
        USING (tenant_id IN (SELECT public.current_tenant_ids()))
        WITH CHECK (tenant_id IN (SELECT public.current_tenant_ids()));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'suppliers' AND policyname = 'Tenant üyeleri kendi tedarikçilerini siler') THEN
        CREATE POLICY "Tenant üyeleri kendi tedarikçilerini siler"
        ON public.suppliers FOR DELETE
        TO authenticated
        USING (tenant_id IN (SELECT public.current_tenant_ids()));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'supplier_transactions' AND policyname = 'Tenant üyeleri kendi tedarikçi hareketlerini görür') THEN
        CREATE POLICY "Tenant üyeleri kendi tedarikçi hareketlerini görür"
        ON public.supplier_transactions FOR SELECT
        TO authenticated
        USING (tenant_id IN (SELECT public.current_tenant_ids()));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'supplier_transactions' AND policyname = 'Tenant üyeleri kendi tenant''ına tedarikçi hareketi ekler') THEN
        CREATE POLICY "Tenant üyeleri kendi tenant'ına tedarikçi hareketi ekler"
        ON public.supplier_transactions FOR INSERT
        TO authenticated
        WITH CHECK (tenant_id IN (SELECT public.current_tenant_ids()));
    END IF;
END $$;

GRANT ALL ON public.suppliers TO authenticated;
GRANT ALL ON public.supplier_transactions TO authenticated;

-- 16. TEDARİKÇİ & CARİ HAREKET BAKİYE YENİDEN HESAPLAMA (RECALCULATE) FONKSİYONLARI
CREATE OR REPLACE FUNCTION public.recalculate_supplier_balances(p_tenant_id UUID DEFAULT NULL)
RETURNS TABLE (
    tenant_id UUID,
    supplier_id UUID,
    supplier_name VARCHAR(150),
    is_active BOOLEAN,
    total_purchases NUMERIC(15,2),
    total_payments NUMERIC(15,2),
    net_balance NUMERIC(15,2),
    active_tx_count BIGINT,
    reversed_tx_count BIGINT
) 
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    SELECT 
        s.tenant_id,
        s.id AS supplier_id,
        s.name AS supplier_name,
        s.is_active,
        COALESCE(SUM(
            CASE 
                WHEN st.direction = 'purchase' 
                     AND st.reversed_by IS NULL 
                     AND NOT EXISTS (SELECT 1 FROM public.supplier_transactions rev WHERE rev.reversed_by = st.id)
                THEN st.amount 
                ELSE 0 
            END
        ), 0)::NUMERIC(15,2) AS total_purchases,
        COALESCE(SUM(
            CASE 
                WHEN st.direction = 'payment' 
                     AND st.reversed_by IS NULL 
                     AND NOT EXISTS (SELECT 1 FROM public.supplier_transactions rev WHERE rev.reversed_by = st.id)
                THEN st.amount 
                ELSE 0 
            END
        ), 0)::NUMERIC(15,2) AS total_payments,
        (
            COALESCE(SUM(
                CASE 
                    WHEN st.direction = 'purchase' 
                         AND st.reversed_by IS NULL 
                         AND NOT EXISTS (SELECT 1 FROM public.supplier_transactions rev WHERE rev.reversed_by = st.id)
                    THEN st.amount 
                    ELSE 0 
                END
            ), 0)
            -
            COALESCE(SUM(
                CASE 
                    WHEN st.direction = 'payment' 
                         AND st.reversed_by IS NULL 
                         AND NOT EXISTS (SELECT 1 FROM public.supplier_transactions rev WHERE rev.reversed_by = st.id)
                    THEN st.amount 
                    ELSE 0 
                END
            ), 0)
        )::NUMERIC(15,2) AS net_balance,
        COUNT(
            CASE 
                WHEN st.id IS NOT NULL 
                     AND st.reversed_by IS NULL 
                     AND NOT EXISTS (SELECT 1 FROM public.supplier_transactions rev WHERE rev.reversed_by = st.id)
                THEN st.id 
            END
        ) AS active_tx_count,
        COUNT(
            CASE 
                WHEN st.id IS NOT NULL 
                     AND (st.reversed_by IS NOT NULL OR EXISTS (SELECT 1 FROM public.supplier_transactions rev WHERE rev.reversed_by = st.id))
                THEN st.id 
            END
        ) AS reversed_tx_count
    FROM public.suppliers s
    LEFT JOIN public.supplier_transactions st ON s.id = st.supplier_id AND s.tenant_id = st.tenant_id
    WHERE (p_tenant_id IS NULL OR s.tenant_id = p_tenant_id)
    GROUP BY s.tenant_id, s.id, s.name, s.is_active
    ORDER BY s.name ASC;
END;
$$;

GRANT EXECUTE ON FUNCTION public.recalculate_supplier_balances(UUID) TO authenticated;

COMMIT;

