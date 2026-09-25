-- 15_create_suppliers.sql
-- Amaç: Çift Defter (Dual-Ledger) modeli için Tedarikçi ve Cari Hareketler tabloları.
-- Kasa defterinden (transactions) bağımsız olarak tedarikçilere olan borç ve ödeme takibini sağlar.

-- 0. Bağımsız PostgreSQL & Supabase Rol Uyumluluğu
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'authenticated') THEN
        CREATE ROLE authenticated NOLOGIN;
    END IF;
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'anon') THEN
        CREATE ROLE anon NOLOGIN;
    END IF;
END $$;

-- 1. Tedarikçiler Tablosu (Suppliers)
CREATE TABLE IF NOT EXISTS public.suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_suppliers_tenant_name UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_suppliers_tenant ON public.suppliers(tenant_id);

-- 2. Tedarikçi Cari Hareketleri Tablosu (Supplier Transactions)
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

-- 3. Row Level Security (RLS) & Tenant İzolasyonu
ALTER TABLE public.suppliers ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.suppliers FORCE ROW LEVEL SECURITY;

ALTER TABLE public.supplier_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.supplier_transactions FORCE ROW LEVEL SECURITY;

-- Suppliers RLS Politikaları
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
END $$;

-- Supplier Transactions RLS Politikaları
DO $$ 
BEGIN
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

    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'supplier_transactions' AND policyname = 'Tenant üyeleri kendi tedarikçi hareketlerini günceller') THEN
        CREATE POLICY "Tenant üyeleri kendi tedarikçi hareketlerini günceller"
        ON public.supplier_transactions FOR UPDATE
        TO authenticated
        USING (tenant_id IN (SELECT public.current_tenant_ids()))
        WITH CHECK (tenant_id IN (SELECT public.current_tenant_ids()));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'supplier_transactions' AND policyname = 'Tenant üyeleri kendi tedarikçi hareketlerini siler') THEN
        CREATE POLICY "Tenant üyeleri kendi tedarikçi hareketlerini siler"
        ON public.supplier_transactions FOR DELETE
        TO authenticated
        USING (tenant_id IN (SELECT public.current_tenant_ids()));
    END IF;
END $$;

-- 4. Yetkilendirmeler
GRANT SELECT, INSERT, UPDATE, DELETE ON public.suppliers TO authenticated;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.supplier_transactions TO authenticated;
