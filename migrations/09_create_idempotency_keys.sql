-- 09_create_idempotency_keys.sql
-- Amaç: Go backend'in Idempotency-Key middleware'i için tekilleştirme tablosu.
-- Aynı key ile ikinci istek geldiğinde ilk sonucun tekrar döndürülmesini sağlar.
-- Güvenlik Sıkılaştırması: Key ve tenant_id kompozit birincil anahtar (PRIMARY KEY) yapılarak multi-tenant key spoofing engellenmiştir.

CREATE TABLE public.idempotency_keys (
    key TEXT NOT NULL,
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    response_body JSONB,
    response_status INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (key, tenant_id)
);

CREATE INDEX idx_idempotency_created_at ON public.idempotency_keys(created_at);

-- 24 saatten eski idempotency kayıtlarını temizleyen saklı yordam
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
