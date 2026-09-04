-- 03_create_current_tenant_fn.sql
-- Amaç: RLS politikalarında tekrar tekrar yazılan tenant kontrolünü
-- tek bir fonksiyonda topla (ileride mantık değişirse tek yerden düzelt).
-- NOT: Bu basit hali "kullanıcının tek tenant'ı var" varsayar.
-- Çoklu tenant üyeliği senaryosu netleşirse bu fonksiyon güncellenecek (bkz. bug-and-fix.md).

CREATE OR REPLACE FUNCTION public.current_tenant_ids()
RETURNS SETOF UUID
LANGUAGE sql
STABLE
SECURITY DEFINER
AS $$
    SELECT tenant_id FROM public.tenant_members WHERE user_id = auth.uid();
$$;
