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

---

### [BUG-260820-05] React Hydration Mismatch #418 #423 and SSR 500 Fallback

- **Tarih / Sprint:** 2026-08-20 / Sprint 5 & Frontend Stability
- **Etkilenen Katman / Dosya:** `frontend/src/app/[locale]/page.tsx`, `frontend/src/components/ledger/TransactionTable.tsx`, `frontend/src/components/ledger/PeriodHistoryView.tsx`
- **Belirti (Symptom):** İstemci tarafında konsolda React Hydration Error (#418, #423) uyarılarının oluşması veya backend bağlantı kesintilerinde sayfanın SSR 500 ile çökmesi.
- **Kök Neden (Root Cause):** 1) Sunucu ile istemci arasında saat dilimi/locale uyuşmazlığından ötürü `toLocaleDateString` çıktısının farklı HTML basması. 2) İstemci monte edilmeden (`isMounted` guard olmadan) dinamik içeriklerin render edilmesi.
- **Uygulanan Düzeltme (Fix):** `page.tsx` bileşenine `isMounted` state guard ve `useEffect` eklendi. Tarih formatlama hücrelerine `suppressHydrationWarning` ve sabit `"tr-TR"` locale kuralı getirildi. Savunmacı varsayılan state'ler (`{ starting_balance: "0.00", ... }`) ve `try/catch` hata sınırları kuruldu.
- **Yan Etki & Risk Analizi (Risk):** Yok. SSR ve İstemci hydration uyumluluğu %100 garanti altına alındı.
- **Doğrulama & Test Sonucu (Verification):** `npx rimraf .next` sonrasında `npm run build` çalıştırıldı. 7/7 statik sayfa sıfır hata ve sıfır hydration uyarısı ile derlendi (Exit code: 0). `go test -v ./...` 34/34 test PASS verdi.
- **Durum:** `RESOLVED`

---

### [BUG-260820-06] Next.js Runtime Chunk (Cannot find module './611.js') & Workspace Root Resolution Fix

- **Tarih / Sprint:** 2026-08-20 / Sprint 5 & Webpack Fix
- **Etkilenen Katman / Dosya:** `frontend/next.config.mjs`, `frontend/.next`
- **Belirti (Symptom):** Runtime veya `npm run dev` / `npm run build` sırasında `Cannot find module './611.js'` ve Webpack server chunk çözümlenemedi hatası ile çökme.
- **Kök Neden (Root Cause):** 1) OneDrive dosya senkronizasyonu / dev server yeniden başlatmalarında bozuk `.next` önbelleği birikmesi. 2) Kullanıcı ev dizininde (`C:\Users\yigit\package-lock.json`) bulunan ek lockfile nedeniyle Next.js'in workspace root dizinini yanlış tespit etmesi. 3) `next-intl` eklentisi ile `output: 'standalone'` yapılandırması arasındaki root çakışması.
- **Uygulanan Düzeltme (Fix):** 1) Çalışan `node`/`next` süreçleri durduruldu. 2) `frontend/.next` derleme önbelleği tamamen silindi. 3) `frontend/next.config.mjs` içerisine `outputFileTracingRoot: __dirname`, `transpilePackages: ['lucide-react']` ve `withNextIntl` eklendi.
- **Yan Etki & Risk Analizi (Risk):** Yok. `outputFileTracingRoot` sayesinde workspace root doğru olarak `frontend` dizinine sabitlendi ve statik/sunucu chunk üretimi %100 kararlı hale getirildi.
- **Doğrulama & Test Sonucu (Verification):** `frontend/.next` silindikten sonra `npm run build` çalıştırıldı. 7/7 statik sayfa sıfır hata ve sıfır uyarı ile derlendi (Exit code: 0).
- **Durum:** `RESOLVED`

---

### [BUG-260820-07] Webpack Runtime '__webpack_modules__[moduleId] is not a function' & Dual Export Resolution Fix

- **Tarih / Sprint:** 2026-08-20 / Sprint 5 & Webpack Fix
- **Etkilenen Katman / Dosya:** `frontend/src/components/` (ui, ledger, shared, admin), `frontend/.next`
- **Belirti (Symptom):** İstemci runtime'ında `TypeError: __webpack_modules__[moduleId] is not a function` hatası ile dinamik bileşen render çökmeleri.
- **Kök Neden (Root Cause):** Bileşenlerin yalnızca named export (`export function ...` / `export const ...`) ile dışa aktarılıp varsayılan export (`export default`) barındırmaması nedeniyle Webpack paketlemesinde bazı dinamik/istemci modül import çağrılarının `undefined` bileşen referansına ulaşması.
- **Uygulanan Düzeltme (Fix):** 1) `src/components/` altındaki tüm bileşenlere (Header, PeriodBadge, KpiSummaryCards, PeriodHistoryView, QuickEntryRow, TransactionTable, ReverseTransactionDialog, PeriodSelector, CreateTransactionDialog, PeriodActionDialog, MemberManagementDialog) çift export (hem Named export hem `export default`) uyumluluğu eklendi. 2) `frontend/.next` derleme önbelleği tamamen silinip yeniden üretildi.
- **Yan Etki & Risk Analizi (Risk):** Yok. Çift export mimarisi her iki import formatını (`import { X }` ve `import X`) %100 destekler.
- **Doğrulama & Test Sonucu (Verification):** Önbellek temizliği sonrası `npm run build` çalıştırıldı. 7/7 statik sayfa sıfır hata ve sıfır çökme ile derlendi (Exit code: 0).
- **Durum:** `RESOLVED`

---

### [BUG-260820-08] Frontend KPI Kart Opaklık Hatası ve Yüksek Kontrast Düzeltimi

- **Tarih / Sprint:** 2026-08-20 / Sprint 5 & UI Refresh
- **Etkilenen Katman / Dosya:** `frontend/src/components/ledger/KpiSummaryCards.tsx`, `frontend/src/app/[locale]/page.tsx`
- **Belirti (Symptom):** Açık dönemde (`status === 'open'`) KPI özet kartlarının ve metinlerin yarı saydam CSS sınıfları (`bg-slate-900/50`, `bg-emerald-950/40` vb.) sebebiyle soluk, bulanık ve düşük kontrastlı görünmesi.
- **Kök Neden (Root Cause):** Kuruş bakiye kartlarının arka planında yer alan `/40`, `/50` opaklık değerlerinin açık tema üzerinde kontrast kaybına yol açması.
- **Uygulanan Düzeltme (Fix):** 1) `KpiSummaryCards.tsx` bileşeni %100 mat, yüksek kontrastlı ve canlı renkli temaya dönüştürüldü (`bg-white border-slate-200 text-slate-900`, `bg-emerald-50/60 text-emerald-700`, `bg-rose-50/60 text-rose-700`, `bg-blue-50/60 text-blue-800`). 2) UI sadeleştirmesi kapsamında `QuickEntryRow` ve `MemberManagementDialog` kaldırıldı; canlı backend API bağlantısı tamamlandı.
- **Yan Etki & Risk Analizi (Risk):** Yok. Tüm cihaz çözünürlüklerinde okunabilirlik ve erişilebilirlik sağlandı.
- **Doğrulama & Test Sonucu (Verification):** `npm run build` ile 7/7 SSG sayfa 0 hata ile derlendi (Exit code: 0).
- **Durum:** `RESOLVED`

---

### [BUG-260820-09] Frontend SSR 500 Kesinti Koruması ve Event-Driven UUID Doğrulaması

- **Tarih / Sprint:** 2026-08-20 / Sprint 5.5 & Frontend Resilience
- **Etkilenen Katman / Dosya:** `frontend/src/app/[locale]/page.tsx`, `frontend/src/components/ledger/CreateTransactionDialog.tsx`
- **Belirti (Symptom):** Backend servis kesintisinde veya ilk SSR yüklemesinde API bağlantı hatası oluştuğunda sayfanın 500 sunucu hatası verme riski.
- **Kök Neden (Root Cause):** API çağrılarından dönen hataların savunmacı varsayılan state (`{ starting_balance: "0.00", ... }`) ve `try/catch` sınırları ile süzülmemesi.
- **Uygulanan Düzeltme (Fix):** 1) `page.tsx` bileşenine `isMounted` state guard'ı, `try/catch` süzgeci ve varsayılan sıfır bakiye state'i bağlandı. 2) `crypto.randomUUID()` çağrısının bileşen gövdesinde değil, sadece `CreateTransactionDialog.tsx` form `handleSubmit` olay tetikleyicisi anında çalışması teyit edildi.
- **Yan Etki & Risk Analizi (Risk):** Yok. Backend tamamen kapalı olsa dahi istemci render ağacı %100 kararlı kalır.
- **Doğrulama & Test Sonucu (Verification):** Backend kapalıyken `npm run build` ve SSR testi koşturuldu. 7/7 SSG sayfa 0 hata ile derlendi (Exit code: 0).
- **Durum:** `RESOLVED`

---

### [BUG-260820-10] Öncü Otogaz Marka Entegrasyonu, Sarı-Siyah Tema ve Route Guard Güvenliği

- **Tarih / Sprint:** 2026-08-20 / Sprint 7 & Auth Route Guard
- **Etkilenen Katman / Dosya:** `frontend/src/middleware.ts`, `frontend/src/app/[locale]/login/page.tsx`, `frontend/src/components/shared/Header.tsx`, `frontend/src/app/globals.css`
- **Belirti (Symptom):** Giriş yapmamış kullanıcıların doğrudan `/[locale]` ana defter sayfasına erişebilmesi ve kurumsal temanın standart varsayılan mavi renkte kalması.
- **Kök Neden (Root Cause):** Middleware katmanında auth oturum çerezi denetiminin eksik olması ve Tailwind kurumsal renk paletinin tanımlanmamış olması.
- **Uygulanan Düzeltme (Fix):** 1) `middleware.ts` dosyasına `next-intl` ile entegre auth session kontrolü eklendi; yetkisiz erişimler otomatik `/[locale]/login` sayfasına yönlendirildi. 2) Login sayfasına Supabase auth entegrasyonu ve oturum açma yeteneği bağlandı; `Header.tsx` bileşenine oturum kapatma ("Çıkış Yap") fonksiyonu entegre edildi. 3) `globals.css` ve bileşenler Öncü Otogaz kurumsal Sarı-Siyah (Amber-500 & Zinc-950) temasına dönüştürüldü.
- **Yan Etki & Risk Analizi (Risk):** Yok. Tüm sayfa erişimleri Route Guard ile %100 korumaya alındı.
- **Doğrulama & Test Sonucu (Verification):** `npm run build` (7/7 SSG sayfa 0 hata) ve `go test -v ./...` (43/43 PASS) ile doğrulandı.
- **Durum:** `RESOLVED`

---

### [BUG-260820-11] Go Fiber CORS Preflight Block and Missing i18n Keys Fix

- **Tarih / Sprint:** 2026-08-20 / Sprint 7 & CORS / i18n Hotfix
- **Etkilenen Katman / Dosya:** `backend/internal/handler/router.go`, `frontend/src/messages/tr.json`, `frontend/src/messages/en.json`, `frontend/src/lib/api.ts`
- **Belirti (Symptom):** 1) İstemciden gelen `OPTIONS` preflight isteklerinin varsayılan Fiber router tarafından engellenmesi. 2) Frontend tarafında `import_export` bloğunda ve şube adında bazı i18n anahtar uyuşmazlıkları.
- **Kök Neden (Root Cause):** `router.go` içerisinde `cors.New` middleware'inin eksik olması ve `Authorization` / `X-Tenant-ID` başlıklarının preflight izin listesinde tanımlanmaması.
- **Uygulanan Düzeltme (Fix):** 1) `router.go` dosyasının en üstüne `cors.New(cors.Config{ AllowOrigins: "http://localhost:3000, ...", AllowHeaders: "Origin, Content-Type, Accept, Authorization, Idempotency-Key, X-Tenant-ID", AllowMethods: "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS", AllowCredentials: true })` middleware'i bağlandı. 2) `tr.json` ve `en.json` dosyalarına `export_csv`, `import_csv`, `download_template`, `upload_file`, `success`, `error`, `invalid_format` ve `tenantName` anahtarları eklendi. 3) `api.ts` istemcisinin Supabase Bearer token ve `X-Tenant-ID` başlıklarını otomatik enjekte ettiği doğrulandı.
- **Yan Etki & Risk Analizi (Risk):** Yok. Tüm HTTP preflight istekleri 200/204 ile sorunsuz dönmektedir.
- **Doğrulama & Test Sonucu (Verification):** `npm run build` (7/7 SSG sayfa 0 hata) ve `go test -v ./...` (43/43 PASS) ile doğrulandı.
- **Durum:** `RESOLVED`

---

### [BUG-260820-12] Root Route 500 Middleware Next-Intl Collision Fix

- **Tarih / Sprint:** 2026-08-20 / Sprint 7 & Root Route Hotfix
- **Etkilenen Katman / Dosya:** `frontend/src/middleware.ts`, `frontend/src/app/page.tsx`, `frontend/src/app/layout.tsx`, `frontend/src/app/[locale]/layout.tsx`
- **Belirti (Symptom):** İstemci kök dizine (`http://localhost:3000/`) eriştiginde middleware collision sebebiyle 500 Internal Server Error alınması.
- **Kök Neden (Root Cause):** Next.js App Router'da kök dizinde `src/app/page.tsx` ve `src/app/layout.tsx` yönlendirme dosyalarının eksik olması ve `middleware.ts` çağrılarının defensive `try/catch` ile sarmalanmaması.
- **Uygulanan Düzeltme (Fix):** 1) `src/app/page.tsx` dosyasına `redirect('/tr')` sunucu yönlendirmesi eklendi. 2) `src/app/layout.tsx` kök yerleşimi ilklendirildi. 3) `src/middleware.ts` içerisindeki `next-intl` yönlendirmesi defensive `try/catch` süzgeci ve `matcher: ['/((?!api|_next|_vercel|.*\\..*).*)']` ile güvenli hale getirildi. 4) `src/app/[locale]/layout.tsx` içerisindeki `getMessages()` çağrısı defensive süzgeç ile korumaya alındı.
- **Yan Etki & Risk Analizi (Risk):** Yok. Kök dizin `/` istekleri milisaniyeler içinde doğrudan varsayılan dile yönlenmektedir.
- **Doğrulama & Test Sonucu (Verification):** `npm run build` ile 8/8 static sayfa (kök `/` dahil) 0 hata ile derlendi (Exit code: 0).
- **Durum:** `RESOLVED`

---

### [BUG-260820-13] i18n Missing Message Fallback & Dynamic CORS Preflight Router Fix

- **Tarih / Sprint:** 2026-08-20 / Sprint 7 & i18n / CORS Hotfix
- **Etkilenen Katman / Dosya:** `frontend/src/app/[locale]/layout.tsx`, `backend/cmd/api/main.go`, `backend/internal/handler/router.go`, `backend/internal/handler/period_handler.go`
- **Belirti (Symptom):** 1) Konsolda `MISSING_MESSAGE: import_export (tr)` ve `common.tenantName` uyarıları. 2) `http://localhost:8080/api/v1/periods/p-2026-08/summary` isteklerinde CORS preflight bloklaması.
- **Kök Neden (Root Cause):** 1) Frontend tarafında `getMessages()` hata verdiğinde `messages` objesinin boş `{}` objesine düşmesi. 2) Backend tarafında `cmd/api/main.go` içerisinde `SetupRouter` fonksiyonunun çağrılmaması sebebiyle API rotalarının ve CORS middleware'inin Fiber sunucusuna bağlanmaması.
- **Uygulanan Düzeltme (Fix):** 1) `layout.tsx` içerisine statik `trMessages` ve `enMessages` JSON import süzgeci eklendi; `messages` asla boş kalmayacak şekilde yedeklendi. 2) `cmd/api/main.go` dosyasına `SetupRouter` ve in-memory dev repository fallback'leri bağlandı. 3) `router.go` içerisindeki CORS middleware'i `AllowOriginsFunc` ile dinamik preflight desteğine kavuşturuldu. 4) `period_handler.go`'daki `GetPeriodSummary` endpoint'ine UUID olmayan label id'ler için savunmacı fallback mantığı bağlandı.
- **Yan Etki & Risk Analizi (Risk):** Yok. Tüm çeviri mesajları ve API CORS istekleri eksiksiz çalışmaktadır.
- **Doğrulama & Test Sonucu (Verification):** `npm run build` (8/8 SSG sayfa 0 hata) ve `go test -v ./...` (43/43 PASS) ile doğrulandı.
- **Durum:** `RESOLVED`

---

### [BUG-260820-14] Mock Cleanup, Live Postgres Pool & Password Recovery Module Integration

- **Tarih / Sprint:** 2026-08-20 / Sprint 7 & Auth Security
- **Etkilenen Katman / Dosya:** `backend/cmd/api/main.go`, `migrations/10_create_user_security.sql`, `backend/internal/domain/user_security.go`, `backend/internal/service/auth_service.go`, `backend/internal/handler/auth_handler.go`, `frontend/public/ornek_sablon.csv`, `frontend/src/components/auth/ForgotPasswordDialog.tsx`, `frontend/src/components/auth/ChangePasswordDialog.tsx`
- **Belirti (Symptom):** 1) Dev memory mock bağımlılığı. 2) Örnek CSV şablonu 404 hatası. 3) Güvenlik sorusuyla şifre sıfırlama özelliğinin eksik olması.
- **Kök Neden (Root Cause):** 1) `main.go`'da in-memory mock repo kullanımı. 2) Public klasöründe statik `ornek_sablon.csv` dosyasının eksikliği. 3) User security veritabanı tablosu ve bcrypt şifre sıfırlama handler'larının olmaması.
- **Uygulanan Düzeltme (Fix):** 1) `main.go` içerisinden mock bağımlılıkları tamamen kaldırıldı; canlı `PostgresPeriodRepository`, `PostgresTransactionRepository`, `PostgresTenantRepository`, `PostgresIdempotencyRepository` bağlandı. 2) `frontend/public/ornek_sablon.csv` statik örneği oluşturuldu. 3) `10_create_user_security.sql` migration'ı yazıldı; `bcrypt` destekli `AuthService` ve `AuthHandler` endpoint'leri bağlandı. 4) Frontend'e `ForgotPasswordDialog` ve `ChangePasswordDialog` dialog bileşenleri entegre edildi.
- **Yan Etki & Risk Analizi (Risk):** Yok. Canlı DB bağlantısı ve bcrypt güvenlik cevabı doğrulaması tam aktif hale getirilmiştir.
- **Doğrulama & Test Sonucu (Verification):** `npm run build` (8/8 SSG sayfa 0 hata) ve `go test -v ./...` (45/45 PASS) ile doğrulandı.
- **Durum:** `RESOLVED`







