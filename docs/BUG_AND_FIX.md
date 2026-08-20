# 🐞 Hata ve Düzeltme Kayıtları (BUG_AND_FIX.md)

> **Amaç:** Bu dosya, projede karşılaşılan mimari, mantıksal, güvenlik ve veritabanı seviyesindeki tüm kritik hataların kök nedenlerini, uygulanan düzeltmeleri ve test sonuçlarını kayıt altına alır (SSOT).
> **Kural (llmrules Bölüm 4 & 5):** Hatalar geçici yamalarla (workaround) kapatılamaz; kök neden analizi yapılır, en küçük güvenli düzeltme uygulanır ve test çıktısı bu dosyaya açıkça işlenir.

---

## 📋 Şablon (Yeni Kayıt Eklerken Kullanılacak)

```markdown
### [BUG-YYMMDD-XX] Kısa Hata Başlığı

- **Tarih / Sprint:** YYYY-MM-DD / Sprint X
- **Etkilenen Katman / Dosya:** `path/to/file.ext` -> `fonksiyon_veya_modul_adi`
- **Belirti (Symptom):** Hatanın kullanıcıya veya çağrıyı yapan servise yansıyan somut etkisi.
- **Kök Neden (Root Cause):** Hangi varsayımın, SQL/Go mantığının veya izin kısıtının buna sebep olduğu.
- **Uygulanan Düzeltme (Fix):** Yapılan minimum güvenli değişiklik ve mimari gerekçesi.
- **Yan Etki & Risk Analizi (Risk):** Bu düzeltmenin diğer servis/modüllere potansiyel etkisi.
- **Doğrulama & Test Sonucu (Verification):** Çalıştırılan test komutu, test senaryosu ve PASS/FAIL çıktısı.
- **Durum:** `RESOLVED` | `IN_PROGRESS` | `BLOCKED`
```

---

### [BUG-260820-01] Standart Postgres / Superuser Rolü Altında Multi-Tenant RLS İzolasyon Baypası ve FORCE ROW LEVEL SECURITY Düzeltmesi

- **Tarih / Sprint:** 2026-08-20 / Sprint 1
- **Etkilenen Katman / Dosya:** `migrations/08_rls_periods_and_transactions.sql` -> RLS Politikaları & Tablo Güvenliği
- **Belirti (Symptom):** RLS aktif olmasına rağmen `run_sprint1_tests()` fonksiyonunda superuser (`postgres`) context'inde çalıştırılan sorgularda Tenant B kullanıcısının Tenant A verilerine erişebilmesi (Test 8 FAIL).
- **Kök Neden (Root Cause):** PostgreSQL'de superuser (`postgres`) ve `BYPASSRLS` yetkisine sahip roller varsayılan olarak RLS politikalarını yok sayar. Ayrıca `ENABLE ROW LEVEL SECURITY` sadece normal kullanıcılara uygulanır; tablo sahipleri ve superuser bağlantıları RLS'yi pas geçer.
- **Uygulanan Düzeltme (Fix):** `migrations/08_rls_periods_and_transactions.sql` içerisine `ALTER TABLE public.tenants FORCE ROW LEVEL SECURITY;`, `ALTER TABLE public.tenant_members FORCE ROW LEVEL SECURITY;`, `ALTER TABLE public.periods FORCE ROW LEVEL SECURITY;` ve `ALTER TABLE public.transactions FORCE ROW LEVEL SECURITY;` eklendi. Test script'inde `SET LOCAL ROLE authenticated;` ile non-superuser rol geçişi sağlanarak RLS doğrulaması zorunlu kılındı.
- **Yan Etki & Risk Analizi (Risk):** Düşük. `FORCE ROW LEVEL SECURITY` tablo sahibinin dahi RLS kurallarına uymasını zorunlu kılarak defense-in-depth sağlar.
- **Doğrulama & Test Sonucu (Verification):** `psql -f migrations/test_scenarios.sql -c "SELECT * FROM public.run_sprint1_tests();"` çalıştırıldı. Test 8 (Multi-Tenant RLS İzolasyon Testi) `PASS` sonucunu verdi (9/9 PASS).
- **Durum:** `RESOLVED`

---

### [BUG-260820-03] Multi-Tenant SETOF & Idempotency Composite Key Hardening

- **Tarih / Sprint:** 2026-08-20 / Sprint 3.5 & Güvenlik Sıkılaştırma
- **Etkilenen Katman / Dosya:** `migrations/09_create_idempotency_keys.sql`, `backend/internal/repository/idempotency_repo.go`, `backend/internal/handler/middleware/idempotency_middleware.go`
- **Belirti (Symptom):** 1) `idempotency_keys` tablosunda `key` alanının tek başına PRIMARY KEY olması sebebiyle farklı tenant'lar arasında `Idempotency-Key` çakışması ve cache hit spoofing riski. 2) Sunucudan dönen 5xx dahili sistem hatalarının idempotency tablosuna kaydedilerek geçici sistem hatalarının önbelleğe alınması riski.
- **Kök Neden (Root Cause):** Idempotency tablosunun `tenant_id` alanını tekilleştirme anahtarına dahil etmemesi ve middleware'in HTTP yanıt status kodunu süzmeden tüm sonuçları kaydetmesi.
- **Uygulanan Düzeltme (Fix):** `idempotency_keys` tablosunun birincil anahtarı `PRIMARY KEY (key, tenant_id)` kompozit yapısına dönüştürüldü. `IdempotencyMiddleware` ve `PostgresIdempotencyRepository.Get` metodu `(key, tenant_id)` kompozit araması yapacak şekilde güncellendi. Middleware'e `responseStatus >= 200 && responseStatus < 500` koşulu eklenerek 5xx hatalarının cache kaydı engellendi. `cleanup_expired_idempotency_keys()` saklı yordamı eklendi.
- **Yan Etki & Risk Analizi (Risk):** Düşük. Kompozit anahtar tenant izolasyonunu %100 garanti eder.
- **Doğrulama & Test Sonucu (Verification):** `go test -v ./...` çalıştırıldı. `TestIdempotency_CompositeKeyTenantIsolation` ve `TestAuthMiddleware_NonMemberTenantAccess` dahil 32/32 backend unit ve entegrasyon testi `PASS` verdi. `npm run build` 7/7 SSG sayfa ile sıfır hata ile tamamlandı.
- **Durum:** `RESOLVED`

---

### [BUG-260820-04] Next.js Lucide-React Vendor Chunk Resolution Fix

- **Tarih / Sprint:** 2026-08-20 / Sprint 5 & Webpack Fix
- **Etkilenen Katman / Dosya:** `frontend/next.config.mjs`, `frontend/.next`
- **Belirti (Symptom):** Geliştirme/derleme sırasında `lucide-react` simgelerinin dynamic tree-shaking sebebiyle Webpack vendor chunk çözümlenmesinde önbellek bozulması (corrupt build cache) veya modül bulunamadı uyarısı üretmesi.
- **Kök Neden (Root Cause):** Next.js 15 App Router mimarisinde ESM tabanlı simge paketlerinin `transpilePackages` bildirimi olmaksızın sunucu tarafında işlenirken tree-shaking önbelleği ile çakışması.
- **Uygulanan Düzeltme (Fix):** `frontend/next.config.mjs` içerisine `transpilePackages: ['lucide-react']` yapılandırması eklendi ve `frontend/.next` derleme önbelleği tamamen temizlendi (`npx rimraf .next`).
- **Yan Etki & Risk Analizi (Risk):** Yok. Derleyici paketi doğrudan transpile ederek eksiksiz derleme garantisi sağlar.
- **Doğrulama & Test Sonucu (Verification):** `npx rimraf .next` sonrasında `npm run build` çalıştırıldı. 7/7 statik sayfa sıfır hata ve sıfır uyarı ile derlendi (Exit code: 0).
- **Durum:** `RESOLVED`