-- ==============================================================================
-- 99_reset_and_seed.sql
-- Amaç: Müşteriye teslim etmeden önce Supabase veritabanındaki tüm test verilerini
--       (test işlemlerini, kilitleri, idempotency kayıtlarını) sıfırlamak ve
--       temiz canlı başlangıç (seed) verilerini yüklemek.
-- ==============================================================================

BEGIN;

-- 1. Tüm Test ve İşlem Verilerini Güvenle Temizle (CASCADE)
TRUNCATE TABLE 
    public.transactions,
    public.idempotency_keys,
    public.user_security_questions,
    public.periods,
    public.tenant_members,
    public.tenants
RESTART IDENTITY CASCADE;

-- 2. Canlı İlk İşletmeyi (Default Tenant) Oluştur
INSERT INTO public.tenants (id, name, created_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Öncü Otogaz Muhasebe ve Finans',
    NOW()
);

-- 3. İlk Yönetici Kullanıcı Eşleşmesini Ekle
INSERT INTO public.tenant_members (tenant_id, user_id, role, joined_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000002',
    'admin',
    NOW()
);

-- 4. Canlı İlk Açık Muhasebe Dönemini (00000000-0000-0000-0000-000000000001) Oluştur
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
