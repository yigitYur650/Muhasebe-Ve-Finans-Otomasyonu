-- 19_add_purchase_and_payment_returns.sql
-- Amaç: Muhasebe usulü 4 Temel İşlem Türü (purchase, payment, purchase_return, payment_return)
-- ve Net Alış / Net Ödeme audit trail mimarisi.

-- 1. supplier_transactions tablosundaki direction kısıtını 4 türe genişlet
ALTER TABLE public.supplier_transactions 
DROP CONSTRAINT IF EXISTS supplier_transactions_direction_check;

ALTER TABLE public.supplier_transactions 
ADD CONSTRAINT supplier_transactions_direction_check 
CHECK (direction IN ('purchase', 'payment', 'purchase_return', 'payment_return'));

-- 2. Eski sistemde alış iptali olup 'payment' olarak kaydedilmiş olan [İPTAL/TERS KAYIT] kayıtlarını 'purchase_return' tipine dönüştür
UPDATE public.supplier_transactions st
SET direction = 'purchase_return'
WHERE st.direction = 'payment'
  AND st.reversed_by IS NOT NULL
  AND EXISTS (
      SELECT 1 FROM public.supplier_transactions orig 
      WHERE orig.id = st.reversed_by AND orig.direction = 'purchase'
  );

-- 3. v_tedarikci_ozet Görünümü (View)
CREATE OR REPLACE VIEW public.v_tedarikci_ozet AS
SELECT 
    s.id AS supplier_id,
    s.tenant_id,
    s.name AS supplier_name,
    s.is_active,
    -- Toplam Net Alınan Mal = SUM(purchase) - SUM(purchase_return)
    COALESCE(SUM(CASE WHEN st.direction = 'purchase' THEN st.amount ELSE 0 END), 0) -
    COALESCE(SUM(CASE WHEN st.direction = 'purchase_return' THEN st.amount ELSE 0 END), 0) AS toplam_net_alis,
    
    -- Toplam Net Yapılan Ödeme = SUM(payment) - SUM(payment_return)
    COALESCE(SUM(CASE WHEN st.direction = 'payment' THEN st.amount ELSE 0 END), 0) -
    COALESCE(SUM(CASE WHEN st.direction = 'payment_return' THEN st.amount ELSE 0 END), 0) AS toplam_net_odeme,
    
    -- Kalan Borç = Net Alış - Net Ödeme
    (
        COALESCE(SUM(CASE WHEN st.direction = 'purchase' THEN st.amount ELSE 0 END), 0) -
        COALESCE(SUM(CASE WHEN st.direction = 'purchase_return' THEN st.amount ELSE 0 END), 0)
    ) - (
        COALESCE(SUM(CASE WHEN st.direction = 'payment' THEN st.amount ELSE 0 END), 0) -
        COALESCE(SUM(CASE WHEN st.direction = 'payment_return' THEN st.amount ELSE 0 END), 0)
    ) AS kalan_borc,

    COUNT(st.id) AS toplam_hareket_sayisi
FROM public.suppliers s
LEFT JOIN public.supplier_transactions st ON s.id = st.supplier_id AND s.tenant_id = st.tenant_id
GROUP BY s.id, s.tenant_id, s.name, s.is_active;

-- 4. recalculate_supplier_balances Fonksiyonunu 4 Muhasebe Türüyle Güncelle
CREATE OR REPLACE FUNCTION public.recalculate_supplier_balances(p_tenant_id UUID DEFAULT NULL)
RETURNS TABLE (
    tenant_id UUID,
    supplier_id UUID,
    supplier_name VARCHAR(150),
    is_active BOOLEAN,
    total_purchases NUMERIC(15,2),
    total_payments NUMERIC(15,2),
    net_balance NUMERIC(15,2),
    active_tx_count BIGINT,
    reversed_tx_count BIGINT
) 
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    SELECT 
        s.tenant_id,
        s.id AS supplier_id,
        s.name AS supplier_name,
        s.is_active,
        (
            COALESCE(SUM(CASE WHEN st.direction = 'purchase' THEN st.amount ELSE 0 END), 0) -
            COALESCE(SUM(CASE WHEN st.direction = 'purchase_return' THEN st.amount ELSE 0 END), 0)
        )::NUMERIC(15,2) AS total_purchases,
        (
            COALESCE(SUM(CASE WHEN st.direction = 'payment' THEN st.amount ELSE 0 END), 0) -
            COALESCE(SUM(CASE WHEN st.direction = 'payment_return' THEN st.amount ELSE 0 END), 0)
        )::NUMERIC(15,2) AS total_payments,
        (
            (
                COALESCE(SUM(CASE WHEN st.direction = 'purchase' THEN st.amount ELSE 0 END), 0) -
                COALESCE(SUM(CASE WHEN st.direction = 'purchase_return' THEN st.amount ELSE 0 END), 0)
            )
            -
            (
                COALESCE(SUM(CASE WHEN st.direction = 'payment' THEN st.amount ELSE 0 END), 0) -
                COALESCE(SUM(CASE WHEN st.direction = 'payment_return' THEN st.amount ELSE 0 END), 0)
            )
        )::NUMERIC(15,2) AS net_balance,
        COUNT(CASE WHEN st.direction IN ('purchase', 'payment') AND st.reversed_by IS NULL THEN st.id END) AS active_tx_count,
        COUNT(CASE WHEN st.direction IN ('purchase_return', 'payment_return') OR st.reversed_by IS NOT NULL THEN st.id END) AS reversed_tx_count
    FROM public.suppliers s
    LEFT JOIN public.supplier_transactions st ON s.id = st.supplier_id AND s.tenant_id = st.tenant_id
    WHERE (p_tenant_id IS NULL OR s.tenant_id = p_tenant_id)
    GROUP BY s.tenant_id, s.id, s.name, s.is_active
    ORDER BY s.name ASC;
END;
$$;
