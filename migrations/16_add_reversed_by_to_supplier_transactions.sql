-- 16_add_reversed_by_to_supplier_transactions.sql
-- Amaç: Tedarikçi hareketlerinde kasa defterindeki gibi append-only ters kayıt (reversal) mekanizmasını desteklemek.

ALTER TABLE public.supplier_transactions 
ADD COLUMN IF NOT EXISTS reversed_by UUID REFERENCES public.supplier_transactions(id);

CREATE INDEX IF NOT EXISTS idx_supplier_txs_reversed_by ON public.supplier_transactions(reversed_by);

-- Dış/servis çağrılarında foreign key kısıtının engel oluşturmaması için
ALTER TABLE public.supplier_transactions DROP CONSTRAINT IF EXISTS supplier_transactions_created_by_fkey;
