-- 07_period_lock_and_append_only_triggers.sql
-- Amaç: (a) transactions tablosunda UPDATE/DELETE'i tamamen engelle (append-only garanti).
--       (b) locked durumundaki bir period'a INSERT denemesini engelle.
-- Bunlar RLS'nin yanında DB seviyesinde de garanti (defense in depth).

CREATE OR REPLACE FUNCTION public.prevent_transaction_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'transactions tablosu append-only: UPDATE/DELETE yasak. Düzeltme için reversed_by kullanın.';
END;
$$;

CREATE TRIGGER trg_prevent_transaction_update
BEFORE UPDATE ON public.transactions
FOR EACH ROW EXECUTE FUNCTION public.prevent_transaction_mutation();

CREATE TRIGGER trg_prevent_transaction_delete
BEFORE DELETE ON public.transactions
FOR EACH ROW EXECUTE FUNCTION public.prevent_transaction_mutation();

CREATE OR REPLACE FUNCTION public.prevent_write_to_locked_period()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    v_status TEXT;
BEGIN
    SELECT status INTO v_status FROM public.periods WHERE id = NEW.period_id;
    IF v_status = 'locked' THEN
        RAISE EXCEPTION 'Bu dönem kilitli, yeni işlem eklenemez (period_id: %)', NEW.period_id;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_prevent_locked_period_insert
BEFORE INSERT ON public.transactions
FOR EACH ROW EXECUTE FUNCTION public.prevent_write_to_locked_period();
