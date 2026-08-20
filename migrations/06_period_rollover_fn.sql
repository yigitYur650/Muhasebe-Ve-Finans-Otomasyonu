-- 06_period_rollover_fn.sql
-- Amaç: Yeni dönem açılırken starting_balance'ı ELLE DEĞİL,
-- önceki dönemin kapanış bakiyesinden otomatik hesapla.
-- Bu fonksiyon, "manuel devir → bir sonraki ay bozulur" sorununun çözümüdür.

CREATE OR REPLACE FUNCTION public.open_next_period(p_tenant_id UUID, p_label TEXT)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_prev_period RECORD;
    v_closing_balance NUMERIC(15,2);
    v_new_period_id UUID;
BEGIN
    -- En son (tarihe göre) dönemi bul
    SELECT * INTO v_prev_period
    FROM public.periods
    WHERE tenant_id = p_tenant_id
    ORDER BY opened_at DESC
    LIMIT 1;

    IF v_prev_period IS NULL THEN
        v_closing_balance := 0;
    ELSE
        SELECT v_prev_period.starting_balance
            + COALESCE(SUM(CASE WHEN direction = 'in' THEN amount ELSE 0 END), 0)
            - COALESCE(SUM(CASE WHEN direction = 'out' THEN amount ELSE 0 END), 0)
        INTO v_closing_balance
        FROM public.transactions
        WHERE period_id = v_prev_period.id
          AND reversed_by IS NULL;
    END IF;

    INSERT INTO public.periods (tenant_id, label, starting_balance, status)
    VALUES (p_tenant_id, p_label, v_closing_balance, 'open')
    RETURNING id INTO v_new_period_id;

    RETURN v_new_period_id;
END;
$$;

-- Test notu: Sprint 1 SQL integrity testi bu fonksiyonu, önceki dönem kilitliyken
-- ve reversal içerirken ayrı ayrı çalıştırıp doğrulamalı.
