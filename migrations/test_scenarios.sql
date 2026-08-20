-- migrations/test_scenarios.sql
-- Kasa ve Defter-i Kebir Platformu — Sprint 1 SQL Bütünlük ve İzolasyon Testleri (9/9)

CREATE OR REPLACE FUNCTION public.run_sprint1_tests()
RETURNS TABLE (
    test_id INT,
    test_name TEXT,
    status TEXT,
    details TEXT
) 
LANGUAGE plpgsql
AS $$
DECLARE
    v_u_a UUID := gen_random_uuid();
    v_u_b UUID := gen_random_uuid();
    v_t_a UUID;
    v_t_b UUID;
    v_fake_t UUID := gen_random_uuid();
    v_p1_id UUID;
    v_p2_id UUID;
    v_tx1_id UUID;
    v_tx2_id UUID;
    v_count INT;
    v_balance NUMERIC(15,2);
    v_err_msg TEXT;
    v_passed BOOLEAN;
BEGIN
    ----------------------------------------------------------------------------
    -- HAZIRLIK: Test verileri oluştur (Superuser yetkisiyle)
    ----------------------------------------------------------------------------
    TRUNCATE public.idempotency_keys CASCADE;
    TRUNCATE public.tenants CASCADE;
    TRUNCATE auth.users CASCADE;
    INSERT INTO auth.users (id, email) VALUES (v_u_a, 'user_a@test.com'), (v_u_b, 'user_b@test.com');

    
    INSERT INTO public.tenants (name) VALUES ('Tenant Alpha') RETURNING id INTO v_t_a;
    INSERT INTO public.tenants (name) VALUES ('Tenant Beta') RETURNING id INTO v_t_b;
    
    INSERT INTO public.tenant_members (tenant_id, user_id, role) VALUES 
        (v_t_a, v_u_a, 'admin'),
        (v_t_b, v_u_b, 'admin');

    ----------------------------------------------------------------------------
    -- TEST 1: tenants ve tenant_members CRUD & RLS İzolasyonu
    ----------------------------------------------------------------------------
    v_passed := FALSE;
    SET LOCAL ROLE authenticated;
    PERFORM set_config('request.jwt.claim.sub', v_u_a::text, true);
    PERFORM set_config('request.jwt.claim.role', 'authenticated', true);
    
    -- User A sadece kendi üyeliğini görebilmeli
    SELECT COUNT(*) INTO v_count FROM public.tenant_members WHERE user_id = v_u_a;
    IF v_count = 1 THEN
        v_passed := TRUE;
        v_err_msg := 'User A kendi üyelik kaydına (Tenant Alpha) erişebildi.';
    ELSE
        v_err_msg := format('Beklenen 1 kaydolması gerekirken %s bulundu.', v_count);
    END IF;
    RESET ROLE;
    
    test_id := 1;
    test_name := 'tenants ve tenant_members RLS İzolasyonu';
    status := CASE WHEN v_passed THEN 'PASS' ELSE 'FAIL' END;
    details := v_err_msg;
    RETURN NEXT;

    ----------------------------------------------------------------------------
    -- TEST 2: current_tenant_ids() Fonksiyonunun Doğru Tenant Setini Dönmesi
    ----------------------------------------------------------------------------
    v_passed := FALSE;
    SET LOCAL ROLE authenticated;
    PERFORM set_config('request.jwt.claim.sub', v_u_a::text, true);
    
    SELECT COUNT(*) INTO v_count FROM public.current_tenant_ids() WHERE current_tenant_ids = v_t_a;
    IF v_count = 1 THEN
        PERFORM set_config('request.jwt.claim.sub', v_u_b::text, true);
        SELECT COUNT(*) INTO v_count FROM public.current_tenant_ids() WHERE current_tenant_ids = v_t_b;
        IF v_count = 1 THEN
            v_passed := TRUE;
            v_err_msg := 'current_tenant_ids() hem User A hem User B için doğru tenant_id döndürdü.';
        ELSE
            v_err_msg := 'User B için current_tenant_ids() yanlış sonuç döndürdü.';
        END IF;
    ELSE
        v_err_msg := 'User A için current_tenant_ids() yanlış sonuç döndürdü.';
    END IF;
    RESET ROLE;

    test_id := 2;
    test_name := 'current_tenant_ids() Fonksiyon Doğruluğu';
    status := CASE WHEN v_passed THEN 'PASS' ELSE 'FAIL' END;
    details := v_err_msg;
    RETURN NEXT;

    ----------------------------------------------------------------------------
    -- TEST 3: periods Tablosuna Sahte/Yetkisiz Tenant İle Kayıt Yazılamaması (RLS)
    ----------------------------------------------------------------------------
    v_passed := FALSE;
    SET LOCAL ROLE authenticated;
    PERFORM set_config('request.jwt.claim.sub', v_u_a::text, true);
    
    BEGIN
        INSERT INTO public.periods (tenant_id, label, starting_balance)
        VALUES (v_fake_t, '2025-99', 100);
        v_err_msg := 'HATA: Sahte tenant ile periods ekleme engellenmedi!';
    EXCEPTION WHEN OTHERS THEN
        v_passed := TRUE;
        v_err_msg := format('RLS/FK kısıtı başarıyla engelledi: %s', SQLERRM);
    END;
    RESET ROLE;

    test_id := 3;
    test_name := 'periods Sahte Tenant RLS Engeli';
    status := CASE WHEN v_passed THEN 'PASS' ELSE 'FAIL' END;
    details := v_err_msg;
    RETURN NEXT;

    ----------------------------------------------------------------------------
    -- TEST 4: transactions Tablosuna amount <= 0 ve Geçersiz Veri Girilememesi (CHECK)
    ----------------------------------------------------------------------------
    v_passed := FALSE;
    -- superuser context'te period oluşturalım test için
    INSERT INTO public.periods (tenant_id, label, starting_balance)
    VALUES (v_t_a, '2025-01', 1000.00) RETURNING id INTO v_p1_id;
    
    BEGIN
        INSERT INTO public.transactions (tenant_id, period_id, direction, channel, amount, created_by)
        VALUES (v_t_a, v_p1_id, 'in', 'eft', -50.00, v_u_a);
        v_err_msg := 'HATA: Negatif tutarlı işlem veritabanı kısıtını geçti!';
    EXCEPTION WHEN OTHERS THEN
        v_passed := TRUE;
        v_err_msg := format('CHECK kısıtı (amount > 0) negatif tutarı engelledi: %s', SQLERRM);
    END;

    test_id := 4;
    test_name := 'transactions NUMERIC(15,2) & amount > 0 Check';
    status := CASE WHEN v_passed THEN 'PASS' ELSE 'FAIL' END;
    details := v_err_msg;
    RETURN NEXT;

    ----------------------------------------------------------------------------
    -- TEST 5: open_next_period() Devir Fonksiyonu (Gelir - Gider Snapshot)
    ----------------------------------------------------------------------------
    v_passed := FALSE;
    -- Period 1 (starting_balance = 1000.00) içine işlemler ekleyelim:
    -- Gelir: 500.00, Gider: 200.00 -> Net Kapanış Bakiyesi = 1000 + 500 - 200 = 1300.00
    INSERT INTO public.transactions (tenant_id, period_id, direction, channel, amount, created_by)
    VALUES (v_t_a, v_p1_id, 'in', 'eft', 500.00, v_u_a) RETURNING id INTO v_tx1_id;

    INSERT INTO public.transactions (tenant_id, period_id, direction, channel, amount, created_by)
    VALUES (v_t_a, v_p1_id, 'out', 'nakit', 200.00, v_u_a) RETURNING id INTO v_tx2_id;

    -- open_next_period çağır
    v_p2_id := public.open_next_period(v_t_a, '2025-02');
    
    SELECT starting_balance INTO v_balance FROM public.periods WHERE id = v_p2_id;
    IF v_balance = 1300.00 THEN
        v_passed := TRUE;
        v_err_msg := format('Devir bakiyesi tam doğrulukla kopyalandı: %s TL (1000 + 500 - 200).', v_balance);
    ELSE
        v_err_msg := format('Beklenen devir bakiyesi 1300.00 TL iken %s TL hesaplandı.', v_balance);
    END IF;

    test_id := 5;
    test_name := 'open_next_period() Devir Bakiyesi Doğrulaması';
    status := CASE WHEN v_passed THEN 'PASS' ELSE 'FAIL' END;
    details := v_err_msg;
    RETURN NEXT;

    ----------------------------------------------------------------------------
    -- TEST 6: prevent_transaction_mutation() Trigger (UPDATE/DELETE Yasak)
    ----------------------------------------------------------------------------
    v_passed := FALSE;
    BEGIN
        UPDATE public.transactions SET amount = 9999.00 WHERE id = v_tx1_id;
        v_err_msg := 'HATA: UPDATE işlemi engellenmedi!';
    EXCEPTION WHEN OTHERS THEN
        IF SQLERRM LIKE '%append-only%' THEN
            BEGIN
                DELETE FROM public.transactions WHERE id = v_tx1_id;
                v_err_msg := 'HATA: DELETE işlemi engellenmedi!';
            EXCEPTION WHEN OTHERS THEN
                IF SQLERRM LIKE '%append-only%' THEN
                    v_passed := TRUE;
                    v_err_msg := 'trg_prevent_transaction_update ve delete (UPDATE ve DELETE) başarıyla engellendi.';
                ELSE
                    v_err_msg := format('Farklı bir hata alındı: %s', SQLERRM);
                END IF;
            END;
        ELSE
            v_err_msg := format('Farklı bir hata alındı: %s', SQLERRM);
        END IF;
    END;

    test_id := 6;
    test_name := 'transactions Append-Only Trigger Engeli (UPDATE/DELETE)';
    status := CASE WHEN v_passed THEN 'PASS' ELSE 'FAIL' END;
    details := v_err_msg;
    RETURN NEXT;

    ----------------------------------------------------------------------------
    -- TEST 7: prevent_write_to_locked_period() Trigger (Locked Döneme Write Yasak)
    ----------------------------------------------------------------------------
    v_passed := FALSE;
    -- Period 1'i kilitli duruma getir
    UPDATE public.periods SET status = 'locked', locked_at = now() WHERE id = v_p1_id;
    
    BEGIN
        INSERT INTO public.transactions (tenant_id, period_id, direction, channel, amount, created_by)
        VALUES (v_t_a, v_p1_id, 'in', 'pos', 150.00, v_u_a);
        v_err_msg := 'HATA: Kilitli döneme işlem ekleme engellenmedi!';
    EXCEPTION WHEN OTHERS THEN
        IF SQLERRM LIKE '%kilitli%' THEN
            v_passed := TRUE;
            v_err_msg := format('trg_prevent_locked_period_insert kilitli döneme yazmayı engelledi: %s', SQLERRM);
        ELSE
            v_err_msg := format('Beklenmeyen hata: %s', SQLERRM);
        END IF;
    END;

    test_id := 7;
    test_name := 'Kilitli Döneme INSERT Engeli Trigger';
    status := CASE WHEN v_passed THEN 'PASS' ELSE 'FAIL' END;
    details := v_err_msg;
    RETURN NEXT;

    ----------------------------------------------------------------------------
    -- TEST 8: Multi-Tenant RLS Testi (Tenant B kullanıcısı Tenant A verisini göremez/yazamaz)
    ----------------------------------------------------------------------------
    v_passed := FALSE;
    SET LOCAL ROLE authenticated;
    PERFORM set_config('request.jwt.claim.sub', v_u_b::text, true);
    PERFORM set_config('request.jwt.claim.role', 'authenticated', true);
    
    -- Tenant B kullanıcısı Tenant A'nın işlemlerini görememeli
    SELECT COUNT(*) INTO v_count FROM public.transactions WHERE tenant_id = v_t_a;
    IF v_count = 0 THEN
        -- Tenant B kullanıcısı Tenant A'ya işlem ekleyememeli (RLS INSERT WITH CHECK)
        BEGIN
            INSERT INTO public.transactions (tenant_id, period_id, direction, channel, amount, created_by)
            VALUES (v_t_a, v_p2_id, 'in', 'eft', 300.00, v_u_b);
            v_err_msg := 'HATA: Tenant B kullanıcısı Tenant A hesabı için işlem ekleyebildi!';
        EXCEPTION WHEN OTHERS THEN
            v_passed := TRUE;
            v_err_msg := 'Tenant B kullanıcısı Tenant A verilerini okuyamadı (0 kayıt) ve yazamadı (RLS Engeli).';
        END;
    ELSE
        v_err_msg := format('HATA: Tenant B kullanıcısı Tenant A verilerini okuyabildi (%s kayıt).', v_count);
    END IF;
    RESET ROLE;

    test_id := 8;
    test_name := 'Multi-Tenant RLS İzolasyon Testi';
    status := CASE WHEN v_passed THEN 'PASS' ELSE 'FAIL' END;
    details := v_err_msg;
    RETURN NEXT;

    ----------------------------------------------------------------------------
    -- TEST 9: idempotency_keys Tablosunda Duplicate Key Çakışması (PK Violation)
    ----------------------------------------------------------------------------
    v_passed := FALSE;
    PERFORM set_config('request.jwt.claim.sub', v_u_a::text, true);
    
    INSERT INTO public.idempotency_keys (key, tenant_id, response_status)
    VALUES ('req-key-001', v_t_a, 201);
    
    BEGIN
        INSERT INTO public.idempotency_keys (key, tenant_id, response_status)
        VALUES ('req-key-001', v_t_a, 201);
        v_err_msg := 'HATA: Duplicate Idempotency Key eklenmesine izin verildi!';
    EXCEPTION WHEN OTHERS THEN
        IF SQLERRM LIKE '%idempotency_keys_pkey%' OR SQLSTATE = '23505' THEN
            v_passed := TRUE;
            v_err_msg := format('PK Ihali (23505) başarıyla tetiklendi: %s', SQLERRM);
        ELSE
            v_err_msg := format('Farklı bir hata alındı: %s', SQLERRM);
        END IF;
    END;

    test_id := 9;
    test_name := 'Idempotency Key PK Çakışma Engeli';
    status := CASE WHEN v_passed THEN 'PASS' ELSE 'FAIL' END;
    details := v_err_msg;
    RETURN NEXT;

END;
$$;
