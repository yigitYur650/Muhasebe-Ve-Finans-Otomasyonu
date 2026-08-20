-- 09_create_idempotency_keys.sql
-- Amaç: Go backend'in Idempotency-Key middleware'i için tekilleştirme tablosu.
-- Aynı key ile ikinci istek geldiğinde ilk sonucun tekrar döndürülmesini sağlar.

CREATE TABLE public.idempotency_keys (
    key TEXT PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    response_body JSONB,
    response_status INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 24 saatten eski kayıtları temizlemek için cron/scheduled job önerilir (Sprint 8'de değerlendirilecek).
CREATE INDEX idx_idempotency_created_at ON public.idempotency_keys(created_at);

GRANT SELECT, INSERT ON public.idempotency_keys TO authenticated;

