-- 05_create_transactions.sql
-- Amaç: Append-only işlem defteri. UPDATE/DELETE yasağı 07'de trigger ile eklenecek.

CREATE TABLE public.transactions (
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
    created_by UUID NOT NULL REFERENCES auth.users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reversed_by UUID REFERENCES public.transactions(id)
);

ALTER TABLE public.transactions ENABLE ROW LEVEL SECURITY;

CREATE INDEX idx_transactions_period ON public.transactions(period_id);
CREATE INDEX idx_transactions_tenant ON public.transactions(tenant_id);
