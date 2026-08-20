-- 01_create_tenants.sql
-- Amaç: Çoklu işletme (tenant) izolasyonunun temel tablosu.

CREATE TABLE public.tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE public.tenants ENABLE ROW LEVEL SECURITY;

-- Not: Kullanıcı-tenant eşlemesi 02'de (users/auth ile birlikte) değil,
-- ayrı bir migration'da (bkz. 03_create_tenant_members.sql) yapılacak —
-- tek dosya tek sorumluluk.
