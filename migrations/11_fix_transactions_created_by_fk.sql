-- 11_fix_transactions_created_by_fk.sql
-- Amaç: API üzerinden veya dış entegrasyonlar ile yapılan işlem kaydında
--       foreign key kısıtlamasının (transactions_created_by_fkey) engel oluşturmasını önlemek.

ALTER TABLE public.transactions DROP CONSTRAINT IF EXISTS transactions_created_by_fkey;
