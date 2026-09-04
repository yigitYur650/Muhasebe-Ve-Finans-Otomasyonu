-- 08_rls_periods_and_transactions.sql
-- Amaç: Tenant izolasyonu. Kap-App denetimindeki SEC-010 (USING(true) ile herkese
-- yazma izni) hatası burada BİLİNÇLİ olarak tekrarlanmıyor — her policy tenant'a daralıyor.

CREATE POLICY "Tenant üyeleri kendi dönemlerini görür"
ON public.periods FOR SELECT
TO authenticated
USING (tenant_id IN (SELECT public.current_tenant_ids()));

CREATE POLICY "Sadece admin/muhasebeci dönem oluşturur"
ON public.periods FOR INSERT
TO authenticated
WITH CHECK (
    tenant_id IN (
        SELECT tenant_id FROM public.tenant_members
        WHERE user_id = auth.uid() AND role IN ('admin', 'muhasebeci')
    )
);

CREATE POLICY "Tenant üyeleri kendi işlemlerini görür"
ON public.transactions FOR SELECT
TO authenticated
USING (tenant_id IN (SELECT public.current_tenant_ids()));

CREATE POLICY "Tenant üyeleri kendi tenant'ına işlem ekler"
ON public.transactions FOR INSERT
TO authenticated
WITH CHECK (tenant_id IN (SELECT public.current_tenant_ids()));

-- UPDATE/DELETE için ayrıca policy YAZILMIYOR — zaten 07'deki trigger DB seviyesinde
-- engelliyor. Policy eklemek yanlış bir "izin var" izlenimi yaratabilir, bilinçli olarak atlandı.

ALTER TABLE public.tenants FORCE ROW LEVEL SECURITY;
ALTER TABLE public.tenant_members FORCE ROW LEVEL SECURITY;
ALTER TABLE public.periods FORCE ROW LEVEL SECURITY;
ALTER TABLE public.transactions FORCE ROW LEVEL SECURITY;

GRANT USAGE ON SCHEMA public TO authenticated;
GRANT SELECT ON public.tenants TO authenticated;
GRANT SELECT ON public.tenant_members TO authenticated;
GRANT SELECT, INSERT ON public.periods TO authenticated;
GRANT SELECT, INSERT ON public.transactions TO authenticated;

