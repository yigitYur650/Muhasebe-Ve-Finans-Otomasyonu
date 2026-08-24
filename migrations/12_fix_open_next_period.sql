-- 12_fix_open_next_period.sql
-- Fix open_next_period function to accurately calculate closing balance in pure append-only ledger

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
    -- Find latest period by opened_at
    SELECT * INTO v_prev_period
    FROM public.periods
    WHERE tenant_id = p_tenant_id
    ORDER BY opened_at DESC
    LIMIT 1;

    IF v_prev_period IS NULL THEN
        v_closing_balance := 0;
    ELSE
        -- In an append-only ledger, sum ALL transactions in the period.
        -- Original entries and offsetting reversal entries net out to 0 TL,
        -- producing 100% penny-accurate starting balance for the next period.
        SELECT v_prev_period.starting_balance
            + COALESCE(SUM(CASE WHEN direction = 'in' THEN amount ELSE 0 END), 0)
            - COALESCE(SUM(CASE WHEN direction = 'out' THEN amount ELSE 0 END), 0)
        INTO v_closing_balance
        FROM public.transactions
        WHERE period_id = v_prev_period.id;
    END IF;

    INSERT INTO public.periods (tenant_id, label, starting_balance, status)
    VALUES (p_tenant_id, p_label, COALESCE(v_closing_balance, 0), 'open')
    RETURNING id INTO v_new_period_id;

    RETURN v_new_period_id;
END;
$$;
