-- ==============================================================================
-- 17_supabase_jwt_custom_claims_hook.sql
-- ==============================================================================
-- Supabase Custom Access Token Hook (Sektör Standardı JWT Claims Mimarisi)
--
-- Amaç:
-- Kullanıcı Supabase Auth ile oturum açtığında veya token yenilediğinde,
-- kullanıcının aktif tenant_id ve role bilgilerini doğrudan JWT claims
-- (claims.app_metadata) içerisine enjekte eder.
--
-- Bu sayede backend API katmanı veritabanına ek sorgu atmadan doğrudan
-- kriptografik imza üzerinden kullanıcının işletme ve rol yetkisini doğrular.
-- ==============================================================================

-- 1. Custom Claims Hook Fonksiyonu
CREATE OR REPLACE FUNCTION public.custom_access_token_hook(event jsonb)
RETURNS jsonb
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    claims jsonb;
    user_tenant_id uuid;
    user_role text;
BEGIN
    -- Mevcut claims objesini al
    claims := event->'claims';

    -- Kullanıcının birincil işletme üyeliğini ve rolünü sorgula
    SELECT tm.tenant_id, tm.role
    INTO user_tenant_id, user_role
    FROM public.tenant_members tm
    WHERE tm.user_id = (event->>'user_id')::uuid
    ORDER BY tm.created_at ASC
    LIMIT 1;

    -- Eğer üyelik varsa claims objesine app_metadata olarak ekle
    IF user_tenant_id IS NOT NULL THEN
        claims := jsonb_set(
            claims,
            '{app_metadata,tenant_id}',
            to_jsonb(user_tenant_id::text)
        );
        claims := jsonb_set(
            claims,
            '{app_metadata,role}',
            to_jsonb(COALESCE(user_role, 'standart'))
        );
    END IF;

    -- Güncellenmiş claims ile eventi geri döndür
    event := jsonb_set(event, '{claims}', claims);
    RETURN event;
END;
$$;

-- 2. Hook İzinleri (Supabase Auth Server için - varsa atanır)
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'supabase_auth_admin') THEN
        GRANT EXECUTE ON FUNCTION public.custom_access_token_hook(jsonb) TO supabase_auth_admin;
    END IF;
END $$;

REVOKE EXECUTE ON FUNCTION public.custom_access_token_hook(jsonb) FROM authenticated, anon, public;

COMMENT ON FUNCTION public.custom_access_token_hook(jsonb) IS 'Supabase Auth Access Token Custom Claims Hook for Multi-Tenant RBAC';
