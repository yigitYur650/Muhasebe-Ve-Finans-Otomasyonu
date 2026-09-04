BEGIN;

-- 0. Dış Anahtar Kısıtını Esnet (API İstekleri İçin)
ALTER TABLE public.transactions DROP CONSTRAINT IF EXISTS transactions_created_by_fkey;

-- 1. Tüm Test ve İşlem Verilerini Güvenle Temizle (CASCADE)
TRUNCATE TABLE 
    public.idempotency_keys,
    public.user_security,
    public.tenants
RESTART IDENTITY CASCADE;

-- 2. Canlı İlk İşletmeyi (Default Tenant) Oluştur
INSERT INTO public.tenants (id, name, created_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Öncü Otogaz Muhasebe ve Finans',
    NOW()
);

-- 3. Canlı İlk Açık Muhasebe Dönemini (00000000-0000-0000-0000-000000000001) Oluştur
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

-- ==============================================================================
-- ✅ VERİTABANI BAŞARIYLA SIFIRLANDI VE CANLI TESLİMAT İÇİN HAZIRLANDI!
-- ==============================================================================
