-- 18_recalculate_supplier_balances.sql
-- Amaç: Tedarikçi ve cari hareket bakiyelerini kuruşu kuruşuna denetleyen, 
-- ters kayıt (reversal) etkilerini filtreleyen ve bakiye raporunu üreten SQL fonksiyonları.

-- 1. Tedarikçi Bakiye Denetim ve Yeniden Hesaplama Fonksiyonu
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
        COALESCE(SUM(
            CASE 
                WHEN st.direction = 'purchase' 
                     AND st.reversed_by IS NULL 
                     AND NOT EXISTS (SELECT 1 FROM public.supplier_transactions rev WHERE rev.reversed_by = st.id)
                THEN st.amount 
                ELSE 0 
            END
        ), 0)::NUMERIC(15,2) AS total_purchases,
        COALESCE(SUM(
            CASE 
                WHEN st.direction = 'payment' 
                     AND st.reversed_by IS NULL 
                     AND NOT EXISTS (SELECT 1 FROM public.supplier_transactions rev WHERE rev.reversed_by = st.id)
                THEN st.amount 
                ELSE 0 
            END
        ), 0)::NUMERIC(15,2) AS total_payments,
        (
            COALESCE(SUM(
                CASE 
                    WHEN st.direction = 'purchase' 
                         AND st.reversed_by IS NULL 
                         AND NOT EXISTS (SELECT 1 FROM public.supplier_transactions rev WHERE rev.reversed_by = st.id)
                    THEN st.amount 
                    ELSE 0 
                END
            ), 0)
            -
            COALESCE(SUM(
                CASE 
                    WHEN st.direction = 'payment' 
                         AND st.reversed_by IS NULL 
                         AND NOT EXISTS (SELECT 1 FROM public.supplier_transactions rev WHERE rev.reversed_by = st.id)
                    THEN st.amount 
                    ELSE 0 
                END
            ), 0)
        )::NUMERIC(15,2) AS net_balance,
        COUNT(
            CASE 
                WHEN st.id IS NOT NULL 
                     AND st.reversed_by IS NULL 
                     AND NOT EXISTS (SELECT 1 FROM public.supplier_transactions rev WHERE rev.reversed_by = st.id)
                THEN st.id 
            END
        ) AS active_tx_count,
        COUNT(
            CASE 
                WHEN st.id IS NOT NULL 
                     AND (st.reversed_by IS NOT NULL OR EXISTS (SELECT 1 FROM public.supplier_transactions rev WHERE rev.reversed_by = st.id))
                THEN st.id 
            END
        ) AS reversed_tx_count
    FROM public.suppliers s
    LEFT JOIN public.supplier_transactions st ON s.id = st.supplier_id AND s.tenant_id = st.tenant_id
    WHERE (p_tenant_id IS NULL OR s.tenant_id = p_tenant_id)
    GROUP BY s.tenant_id, s.id, s.name, s.is_active
    ORDER BY s.name ASC;
END;
$$;

-- 2. İptal / Ters Kayıt Eşleştirme Onarım Yardımcısı (Legacy Reversal Linker)
-- Eski sistemde açıklamada [İPTAL/TERS KAYIT] veya benzeri yazan ama reversed_by kolonu boş kalmış kayıtları tespit eder.
CREATE OR REPLACE FUNCTION public.audit_unlinked_reversals(p_tenant_id UUID DEFAULT NULL)
RETURNS TABLE (
    tx_id UUID,
    tenant_id UUID,
    supplier_id UUID,
    tx_date DATE,
    direction VARCHAR(10),
    amount NUMERIC(15,2),
    description TEXT,
    issue_type TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    SELECT 
        st.id AS tx_id,
        st.tenant_id,
        st.supplier_id,
        st.tx_date,
        st.direction,
        st.amount,
        st.description,
        'Possible unlinked reversal notation in description'::TEXT AS issue_type
    FROM public.supplier_transactions st
    WHERE (p_tenant_id IS NULL OR st.tenant_id = p_tenant_id)
      AND st.reversed_by IS NULL
      AND (
          st.description ILIKE '%[İPTAL/TERS KAYIT]%' 
          OR st.description ILIKE '%[TERS KAYIT]%'
          OR st.description ILIKE '%TERS KAYIT%'
      )
      AND NOT EXISTS (
          SELECT 1 FROM public.supplier_transactions rev WHERE rev.reversed_by = st.id
      );
END;
$$;

-- 3. Yetkilendirmeler
GRANT EXECUTE ON FUNCTION public.recalculate_supplier_balances(UUID) TO authenticated;
GRANT EXECUTE ON FUNCTION public.audit_unlinked_reversals(UUID) TO authenticated;
