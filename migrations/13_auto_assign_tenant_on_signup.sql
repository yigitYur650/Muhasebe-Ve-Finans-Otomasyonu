-- 13_auto_assign_tenant_on_signup.sql
-- Amaç: Supabase Auth üzerinden kayıt olan her kullanıcıyı otomatik olarak
--       varsayılan işletmeye (Öncü Otogaz - 00000000-0000-0000-0000-000000000001) bağlamak.
--       Böylece kullanıcı giriş yaptığında RLS engeline takılmaz ve verileri görebilir.

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

-- Halihazırda auth.users tablosunda bulunan tüm kullanıcıları da geriye dönük olarak bağla
INSERT INTO public.tenant_members (tenant_id, user_id, role)
SELECT '00000000-0000-0000-0000-000000000001', id, 'admin'
FROM auth.users
ON CONFLICT (tenant_id, user_id) DO NOTHING;
