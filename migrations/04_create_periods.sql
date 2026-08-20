-- 04_create_periods.sql
-- Amaç: Aylık dönem izolasyonu. Devir mantığı burada DEĞİL, 06'da (ayrı fonksiyon).

CREATE TABLE public.periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    label TEXT NOT NULL,                          -- ör. "2025-05"
    starting_balance NUMERIC(15,2) NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'locked')),
    opened_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at TIMESTAMPTZ,
    UNIQUE (tenant_id, label)
);

ALTER TABLE public.periods ENABLE ROW LEVEL SECURITY;
