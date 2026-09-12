-- 14_strict_data_loss_prevention.sql
-- Amaç: 
-- 1. TRUNCATE komutunu veritabanı seviyesinde yasaklamak (statement-level trigger).
-- 2. Tenant silindiğinde bağlı dönemlerin ve işlemlerin basamaklı (CASCADE) silinmesini engellemek (RESTRICT).
--    Böylece bünyesinde işlem veya dönem barındıran bir işletme ASLA silinemez.

BEGIN;

-- 1. TRUNCATE Yasağı Trigger'ı (Statement-level)
DROP TRIGGER IF EXISTS trg_prevent_transaction_truncate ON public.transactions;
CREATE TRIGGER trg_prevent_transaction_truncate
BEFORE TRUNCATE ON public.transactions
FOR EACH STATEMENT EXECUTE FUNCTION public.prevent_transaction_mutation();

-- 2. transactions tablosunun tenant_id foreign key'ini RESTRICT yap
ALTER TABLE public.transactions
DROP CONSTRAINT IF EXISTS transactions_tenant_id_fkey;

ALTER TABLE public.transactions
ADD CONSTRAINT transactions_tenant_id_fkey
FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE RESTRICT;

-- 3. periods tablosunun tenant_id foreign key'ini RESTRICT yap
ALTER TABLE public.periods
DROP CONSTRAINT IF EXISTS periods_tenant_id_fkey;

ALTER TABLE public.periods
ADD CONSTRAINT periods_tenant_id_fkey
FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE RESTRICT;

COMMIT;
