# TASK.md — Kasa ve Defter-i Kebir Yönetim Platformu

> Kural: Bir mikro-prompt'un testleri geçmeden `[x]` işaretlenmez, sıradaki adıma geçilmez.
> Scope sprint başladıktan sonra dondurulur — mid-sprint ekleme yok (bkz. antigravityrules).

---

## Sprint 0 — Proje İskeleti ve Dokümantasyon

- [x] `docs/` klasörü ve tüm SSOT dosyaları oluşturulur (PROJECT_BRIEF.md ✅, TASK.md ✅, BUG_AND_FIX.md ✅, PROJECT_MAP_FOR_LLM.md ✅, SECURITY_AUDIT_REPORT.md ✅)
- [x] `antigravityrules` bu proje için uyarlanır (Kap-App şablonundan, stack referansları güncellenir)
- [x] Backend iskeleti: Go modül init, Fiber v2 kurulum, Clean Architecture klasörleri (`domain/repository/service/handler`)
- [x] Frontend iskeleti: Next.js 15 App Router init, shadcn/ui + Tailwind kurulum, TanStack Table bağımlılığı (PASS, 2026-08-20)
- [x] Supabase projesi oluşturulur, `.env.example` yazılır, `.gitignore` doğrulanır (`.env` asla commit edilmez) (PASS, 2026-08-20)
- [x] **AÇIK KARAR onaylanır:** Event sourcing / double-entry / append-only ledger seçimi netleşmeden Sprint 1'e geçilmez (Append-only ledger + period snapshot seçildi)

## Sprint 1 — Veritabanı Şeması ve Dönem Motoru

> Kap-App'teki gibi küçük, tek-sorumluluklu migration dosyaları — tek büyük şema dosyası YOK.
> Her dosya ayrı test edilir, hata çıkarsa hangi dosyada olduğu net anlaşılır (AI/insan için debug kolaylığı).

- [x] `01_create_tenants.sql` — tenants tablosu
- [x] `02_create_tenant_members.sql` — kullanıcı-tenant-rol eşlemesi
- [x] `03_create_current_tenant_fn.sql` — RLS'de tekrar kullanılan `current_tenant_ids()` helper fonksiyonu
- [x] `04_create_periods.sql` — periods tablosu (devir mantığı YOK, sadece şema)
- [x] `05_create_transactions.sql` — append-only işlem tablosu
- [x] `06_period_rollover_fn.sql` — `open_next_period()`: önceki dönem kapanış bakiyesini otomatik `starting_balance` yapar
- [x] `07_period_lock_and_append_only_triggers.sql` — transactions UPDATE/DELETE yasağı + locked period'a INSERT yasağı (DB seviyesi, defense in depth)
- [x] `08_rls_periods_and_transactions.sql` — tenant bazlı RLS (Kap-App SEC-010'daki `USING(true)` hatası burada YOK, FORCE RLS aktif)
- [x] `09_create_idempotency_keys.sql` — Idempotency-Key middleware için tekilleştirme tablosu
- [x] Her migration için ayrı SQL integrity testi: özellikle `06` (devir doğruluğu) ve `07` (kilitli döneme yazma reddi, append-only ihlali reddi) (9/9 PASS - `test_scenarios.sql`)
- [x] Migration dosyaları `PROJECT_MAP_FOR_LLM.md`'ye tek tek işlenir (Kap-App'teki Bölüm 4 formatı gibi)


## Sprint 2 — Go Backend Çekirdek Motor

- [x] Domain katmanı: `Transaction`, `Period` entity'leri, `shopspring/decimal` ile tutar tipi (6/6 PASS unit test)
- [x] Repository katmanı: Supabase/Postgres erişimi, service-role key sadece backend'de (PASS, 2026-08-20)
- [x] Service katmanı: işlem oluşturma, ters kayıt (reversal), dönem kapanış/açılış iş mantığı (PASS, 2026-08-20)
- [x] `Idempotency-Key` middleware: aynı key ile tekrar istek geldiğinde işlem tekrarlanmaz (PASS, 2026-08-20)
- [x] Recover + merkezi structured logging middleware (`console.log`/`fmt.Println` serpiştirme yasak) (PASS, 2026-08-20)
- [x] Hata yanıtları generic hale getirilir (iç detay istemciye dönmez — Kap-App SEC-007 dersinden) (PASS, 2026-08-20)
- [x] Go unit/edge-case testleri: negatif tutar, geçersiz tarih formatı, yetkisiz kullanıcı, kilitli döneme yazma (PASS, 2026-08-20)

## Sprint 3 — Auth ve Yetkilendirme

- [x] Supabase Auth entegrasyonu (kullanıcı/rol modeli: admin, muhasebeci, standart kullanıcı) (PASS, 2026-08-20)
- [x] JWT doğrulama: `iss`/`aud` claim kontrolü dahil (PASS, 2026-08-20)
- [x] Admin rotaları hem frontend guard hem backend middleware ile korunur (PASS, 2026-08-20)
- [x] Tenant Üye ve Rol Yönetimi ile Geçmiş Dönem Arşiv İnceleme Görünümü (PASS, 2026-08-20)
- [x] CORS: production'da açık origin listesi, wildcard yasak (PASS, 2026-08-20)

## Sprint 4 — Klavye Odaklı Veri Girişi Arayüzü

- [x] TanStack Table ile hızlı satır ekleme/düzenleme (klavye kısayolları: Enter ile yeni satır, Tab ile alan geçişi, G/C kısayolları) (PASS, 2026-08-20)
- [x] Kanal seçimi (EFT/POS/nakit/kredi/kira/maaş/kredi kartı/kartuş/yemek/yakıt) — i18n JSON'dan (PASS, 2026-08-20)
- [x] shadcn/ui form bileşenleri, hardcoded metin yok (PASS, 2026-08-20)
- [x] Frontend edge-case: negatif/hatalı tutar girişinde inline validasyon (PASS, 2026-08-20)

## Sprint 5 — Canlı KPI ve Bakiye Paneli

- [x] Seçili döneme ait toplam gelir/gider/net bakiye canlı hesaplama (backend endpoint + frontend özet kart) (PASS, 2026-08-20)
- [x] Dönem geçmişi görünümü (kapanmış dönemlerin salt-okunur listesi) (PASS, 2026-08-20)
- [x] KPI kart opaklık düzeltimi, UI sadeleştirilmesi ve canlı API entegrasyonu (PASS, 2026-08-20)
- [x] 6 İleri Düzey Uç Durum (Edge-Case) Test Paketi (Çift Kuruş, Reversal of Reversal, Concurrency, Sub-penny bounds, Idempotency Cross-tenant, Balance Immutability) (PASS, 2026-08-20)


## Sprint 6 — Excel Import/Export ve Tenant İzolasyon E2E

- [x] Uçtan Uca Entegrasyon ve Güvenlik/İzolasyon Testleri (5 Kritik E2E & Güvenlik Matrisi Senaryosu) (PASS, 2026-08-20)
- [x] Mevcut Excel geçmişinin sisteme aktarımı (import) — format doğrulama ve hata raporlama (PASS, 2026-08-20)
- [x] Excel/CSV dışa aktarım (UTF-8 BOM desteği ile Excel uyumluluğu) (PASS, 2026-08-20)
- [x] Playwright / Go Integration E2E: dönemler arası bakiye devri doğruluğu (PASS, 2026-08-20)
- [x] Playwright / Go Integration E2E: tenant izolasyonu (bir tenant diğerinin verisini göremiyor) (PASS, 2026-08-20)
- [x] Playwright / Service E2E: Excel import sonrası hesaplamaların kuruşu kuruşuna doğruluğu (PASS, 2026-08-20)


## Sprint 7 — Güvenlik Denetimi ve Sertleştirme

- [x] Güvenlik Sıkılaştırma, SQL SETOF Tenant Düzeltmesi ve Idempotency Kompozit Key Güvenliği (PASS, 2026-08-20)
- [x] "Öncü Otogaz" Marka & Sarı-Siyah Tema Entegrasyonu, Supabase Login Ekranı ve Next.js Route Guard Middleware (PASS, 2026-08-20)
- [x] Go Fiber CORS Preflight Yapılandırması, i18n Sözlük İyileştirmeleri ve Supabase Auth Hata Yönetimi (PASS, 2026-08-20)
- [x] Kök Dizin (http://localhost:3000/) 500 Hatasının Giderilmesi ve Middleware İzolasyonu (PASS, 2026-08-20)
- [x] i18n Statik JSON Fallback ve Go API Main Server SetupRouter Entegrasyonu (PASS, 2026-08-20)
- [x] Mock Repo Temizliği (Canlı Postgres Bağlantısı), CSV Export Handler ve Güvenlik Sorusu ile Şifre Yönetimi (PASS, 2026-08-20)
- [x] `SECURITY_AUDIT_REPORT.md` bu proje için doldurulur (Kap-App raporundaki formatla) (PASS, 2026-08-20)
- [x] Güvenlik headerları eklenir (`X-Content-Type-Options`, `X-Frame-Options`, HSTS, CSP) (PASS, 2026-08-20)
- [x] Dockerfile: non-root kullanıcı (backend appuser 10001, frontend nextjs 1001) (PASS, 2026-08-20)
- [ ] Rate limiter kalıcılığı (in-memory yerine, restart'ta sıfırlanmayan bir çözüm) değerlendirilir






## Sprint 8 — Deployment ve Release

- [x] Docker Containerization, GitHub Actions CI/CD ve Production Deployment Hazırlığı (PASS, 2026-08-20)
- [x] Docker Build Context Sabitlemesi & `backend/cmd/api/main.go` PORT Entegrasyonu (PASS, 2026-08-23)
- [x] `render.yaml` (veya seçilen platform) — tüm env var'lar eksiksiz tanımlı (PASS, 2026-08-20)
- [x] `RELEASE_NOTES.md` ilk sürüm için doldurulur (PASS, 2026-08-20)
- [x] Canlıya hazır demo doğrulaması (PASS, 2026-08-20)

## Sprint 9 — Supabase Canlı Kurulum, Tip Güvenliği & Otomatik Üyelik Entegrasyonu

- [x] `feat/supabase-setup` branch'i açılır ve paket bağımlılıkları (`@supabase/supabase-js`, `@supabase/ssr`) kurulur (PASS, 2026-09-04)
- [x] Git güvenlik sertleştirmesi: `.env` ve `.env.local` sır sızıntıları `.gitignore` ile engellenir, güvenli `.env.example` şablonları hazırlanır (PASS, 2026-09-04)
- [x] Supabase SQL Editor için tekil konsolide kurulum şeması (`migrations/supabase_combined_schema.sql`) hazırlanır (PASS, 2026-09-04)
- [x] Otomatik CLI migration aracı (`backend/cmd/migrate/main.go`) geliştirilir (PASS, 2026-09-04)
- [x] `frontend/src/types/database.types.ts` ile tam tip güvenlikli (`Database['public']['Tables']`) TypeScript şeması oluşturulur (PASS, 2026-09-04)
- [x] Supabase browser ve SSR server istemcilerine `<Database>` tipi bağlanır (0 type error, `npx tsc` PASS, 2026-09-04)
- [x] Yeni kullanıcı kaydında RLS engeline takılmayı önleyen otomatik tenant bağlama trigger'ı (`migrations/13_auto_assign_tenant_on_signup.sql`) yazılır (PASS, 2026-09-04)
- [x] Giriş ekranına (`frontend/src/app/[locale]/login/page.tsx`) "Yeni Kullanıcı Kayıt Ol" modu (`supabase.auth.signUp`) entegre edilir (PASS, 2026-09-04)
- [x] Dokümantasyon (`PROJECT_MAP_FOR_LLM.md`, `TASK.md`, `BUG_AND_FIX.md`, `RELEASE_NOTES.md`) güncellenir (PASS, 2026-09-04)

## Sprint 9.5 — Excel Motoru (.xlsx), Fiber Tampon İyileştirmesi ve Canlı Veri Bütünlüğü

- [x] Go Fiber HTTP 431 hatası çözümü: `ReadBufferSize: 16384` ile JWT ve auth çerezleri güvenceye alınır (PASS, 2026-09-12)
- [x] Yeni dönem açma modalında Idempotency-Key çakışması (HTTP 409) giderilir, taze UUID üretimi sağlanır (PASS, 2026-09-12)
- [x] Supabase veritabanındaki 32 adet test işlemi ve mock dönemler temizlenir; 2026-08 dönemindeki 131 adet gerçek işlem kuruşu kuruşuna doğrulanır (PASS, 2026-09-12)
- [x] Native Excel motoru (`github.com/xuri/excelize/v2`) entegre edilir; otomatik sütun genişlikleri, koyu başlık, dondurulmuş satır, otomatik filtre ve para birimi formatlaması eklenir (`export_excel.go`) (PASS, 2026-09-13)
- [x] Frontend `ExportCsvButton.tsx` güncellenerek lisanssız/görüntüleme modundaki Excel'lerde dahi bozulmayan hazır `.xlsx` indirme aktif edilir (PASS, 2026-09-13)
- [x] `BUG_AND_FIX.md` ve `TECHNICAL_DEBT_AND_MOCK_AUDIT.md` güncel denetim bulgularıyla senkronize edilir (PASS, 2026-09-13)

## Sprint 11 — Çift Defter (Dual-Ledger) Tedarikçi & Cari Takip ve Excel İçe Aktarma Motoru

- [x] `migrations/15_create_suppliers.sql` oluşturulur ve yerel PostgreSQL üzerinde uygulanır: `suppliers` ve `supplier_transactions` tabloları, `FORCE ROW LEVEL SECURITY` politikaları ve `tenant_id IN (SELECT public.current_tenant_ids())` tam tenant izolasyonu (PASS, 2026-09-22)
- [x] Parasal float yasağına titizlikle uyulur: Go tarafında `shopspring/decimal.Decimal`, veritabanında `NUMERIC(15,2)` kullanılır (PASS, 2026-09-22)
- [x] Go Clean Architecture katmanları: `domain/supplier.go`, `repository/supplier_repo.go`, `service/supplier_service.go`, `handler/supplier_handler.go` ve Fiber router entegrasyonu (PASS, 2026-09-22)
- [x] Native Excel İçe Aktarma Motoru (`github.com/xuri/excelize/v2`): 36 sayfalık Kasa Defterindeki çoklu firma bloklarını (`ATİKER`, `PRİNS`, `ASİL GRUP`, `PARÇACI UĞUR ABİ`, `FİLTRECİ`) dinamik algılayıp `purchase` (Alınan Mal) ve `payment` (Geçilen Ödeme) olarak kuruş hassasiyetinde `pgx.Tx` transaction içinde toplu kaydetme (PASS, 2026-09-22)
- [x] Frontend Çift Defter arayüzü: Üstte `[ 💰 Kasa Defteri ]` ve `[ 🚚 Tedarikçiler & Parçacılar ]` sekmeleri, tedarikçi bakiye özet kartları, detaylı cari listesi ve `ImportSupplierExcelDialog.tsx` modalı (PASS, 2026-09-22)
- [x] Tüm kelimeler `tr.json` ve `en.json` dosyalarına `suppliers` ve `navigation` namespace'leri altında sıfır hardcoded metin kuralıyla eklenir (PASS, 2026-09-22)
- [x] Terminal doğrulaması: `cd backend && go test -v ./...` ve `frontend` dizininde `npm run build` 0 hata ile başarıyla geçer (PASS, 2026-09-22)

## Sprint 12 — Tedarikçiler & Parçacılar Ters Kayıt (Reversal), Sektör Standardı JWT Claims ve Geliştirici Bypass Modu

- [x] Tedarikçi & Parçacı modülünde Append-Only Ters Kayıt (Reversal) mekanizması: `migrations/16_add_reversed_by_to_supplier_transactions.sql` ile `reversed_by` sütunu ve self-reference endeksi oluşturulur (PASS, 2026-09-24)
- [x] Go Backend Tedarikçi Ters Kayıt atomik transaction (`pgx.Tx`), çift iptal engeli (`422 Unprocessable Entity`) ve bakiye mutasyonu `domain/supplier.go`, `repository/supplier_repo.go`, `service/supplier_service.go`, `handler/supplier_handler.go` katmanlarında uygulanır (PASS, 2026-09-24)
- [x] Frontend `ReverseSupplierTransactionDialog.tsx` modalı, `SupplierLedgerTable.tsx` üzerinde `[TERS KAYIT]` / `[İPTAL EDİLDİ]` rozetleri, filtreleme ve iptal aksiyonu entegre edilir (PASS, 2026-09-24)
- [x] Akıllı Excel eşleme (`matchSheetForPeriod`) tek sayfalı dosyalar (`MAYIS25`, `KASA`) ve Türkçe ay isimleri desteği ile güçlendirilir; 10MB `io.LimitReader` boyutu sınırlanır (PASS, 2026-09-24)
- [x] Sektör Standardı Supabase JWT Custom Claims / App Metadata (`claims.app_metadata.tenant_id` / `claims.app_metadata.role`) desteği `auth_middleware.go` içine eklenir; `migrations/17_supabase_jwt_custom_claims_hook.sql` tanımlanır (PASS, 2026-09-24)
- [x] Base64 kodlu 64-baytlık Supabase JWT secret anahtarları (`Lw7HcEdLRo+...==`) ve düz UTF-8 anahtarlar için çift katmanlı çözücü eklenir (PASS, 2026-09-24)
- [x] `ENVIRONMENT=development` geliştirici modu (Developer Bypass) eklenir; token bulunmadığında yerel admin oturumuyla istekler kesintisiz karşılanır (PASS, 2026-09-24)
- [x] Test doğrulamaları: `go test -v ./...` %100 PASS (Tüm birim, entegrasyon ve custom claim testleri dahil) (PASS, 2026-09-24)
- [x] Dokümantasyon (`RELEASE_NOTES.md`, `PROJECT_MAP_FOR_LLM.md`, `BUG_AND_FIX.md`, `SECURITY_AUDIT_REPORT.md`, `TASK.md`) güncellenir (PASS, 2026-09-24)

## Sprint 13 — Telegram Anlık Alarm & Hata Bildirim Motoru

- [x] `backend/pkg/telegram/client.go`: Hafif ve sıfır harici bağımlılıklı Telegram Bot API HTTP istemcisi (PASS, 2026-09-25)
- [x] `backend/pkg/telegram/global.go`: Asenkron non-blocking goroutine kuyruğu, Markdown/HTML şablonları ve alert throttling (spam önleme) (PASS, 2026-09-25)
- [x] `backend/internal/handler/router.go` (recover) ve Fiber `CustomErrorHandler` entegrasyonu (Panic ve 500+ alarmları) (PASS, 2026-09-25)
- [x] `backend/cmd/test_alert/main.go` test ve doğrulama CLI aracı (PASS, 2026-09-25)
- [x] `.env.example` ve `backend/.env` içine Telegram konfigürasyon parametrelerinin eklenmesi (PASS, 2026-09-25)


## Sprint 14 — Google Drive & 3-2-1 Otomatik Günlük Yedekleme & Retention Motoru

- [x] `backend/pkg/backup/exporter.go`: Atomik veritabanı dump/snapshot ve `.sql.gz` sıkıştırma motoru (PASS, 2026-09-25)
- [x] `backend/pkg/gdrive/uploader.go`: Native Google Drive REST API istemcisi, streaming upload ve 30 günlük retention temizliği (PASS, 2026-09-25)
- [x] `backend/pkg/telegram/document.go`: Yedeklenen `.sql.gz` dosyasını Telegram sohbetine otomatik gönderme (PASS, 2026-09-25)
- [x] `backend/pkg/scheduler/scheduler.go`: Gece 03:00 otomatik 3-2-1 cron zamanlayıcısı ve Telegram raporlama (PASS, 2026-09-25)
- [x] `backend/cmd/backup/main.go`: Manuel yedekleme tetikleme CLI aracı (PASS, 2026-09-25)

## Sprint 15 — Otomatik Dönem Kilitleme & JWT 401 Seamless Auto-Refresh Güçlendirmesi

- [x] `migrations/06_period_rollover_fn.sql` & `supabase_combined_schema.sql`: `open_next_period()` içine yeni ay açılırken önceki açık dönemi atomik olarak kilitleme kuralı eklendi (PASS, 2026-09-25)
- [x] `frontend/src/lib/api.ts`: 401 Unauthorized durumunda `supabase.auth.refreshSession()` ile otomatik sessiz token yenileme ve isteği tekrarlama (seamless retry) entegrasyonu (PASS, 2026-09-25)
- [x] `backend/internal/service/period_service_test.go`: `TestPeriodService_OpenNextPeriod_Validation` unit testi yazıldı (PASS, 2026-09-25)
- [x] Test doğrulamaları: `go test ./...` %100 PASS ve `npx tsc --noEmit` 0 hata ile doğrulandı (PASS, 2026-09-25)



## Sprint 16 — İptal/Ters Kayıt Mantığı İyileştirmesi, Sayfalama (Pagination) ve Bakiye Yeniden Hesaplama

- [x] **Task 1: İptal / Ters Kayıtların Muhasebe Yönü & Tip Mimarisi Düzeltmesi (Kritik)** (PASS, 2026-10-09)
  - [x] Alış faturası iptal edildiğinde "Ödeme (+)" olarak değil, "Alış Tutarı (-)" olarak netleşmesini sağlayan SQL ve Go aggregation kuralları güncellendi
  - [x] `GetSuppliersWithBalances`, `GetSupplierByID` ve `GetSummary` SQL sorgularında iptal/ters kayıtlar (`reversed_by IS NULL AND NOT EXISTS (SELECT 1 ... rev.reversed_by = st.id)`) filtresiyle ödeme toplamına yanlış yazılması engellendi
  - [x] 4 adet alış faturası (₺50.467,56) iptal edildiğinde ödeme toplamının sıfır kaldığını ve net bakiyenin kuruşu kuruşuna sıfırlandığını doğrulayan birim testi yazıldı (`TestSupplierService_ReversalDoesNotInflatePaymentSummary` PASS)

- [x] **Task 2: Cari Hareketler Sayfalama (Pagination) ve Satır Sayısı Senkronizasyonu** (PASS, 2026-10-09)
  - [x] Backend `ResponseEnvelope` içine `total`, `page`, `limit` alanları eklendi ve `ListSupplierTransactions` / `ListAllTransactions` yanıtlarında döndürüldü
  - [x] `frontend/src/lib/api.ts` ve `useSuppliers.ts` içine `page`, `pageSize`, `totalCount`, `setPage`, `setPageSize` pagination entegrasyonu sağlandı
  - [x] `SupplierLedgerTable.tsx` altına modern sayfalama çubuğu (Önceki/Sonraki, sayfa numarası, sayfa başı `25/50/100/200` seçici ve `"Gösterilen: X-Y / Toplam Z kayıt"`) eklendi

- [x] **Task 3: Geçmişe Dönük Veri Onarımı & Bakiye Yeniden Hesaplama Scripti (Recalculate)** (PASS, 2026-10-09)
  - [x] Geçmişte hatalı işlenen ₺50.467,56'lık ters kayıtların ve firma bakiye toplamlarının düzeltilmesi için SQL onarım migration'ı (`migrations/18_recalculate_supplier_balances.sql`) ve `supabase_combined_schema.sql` hazırlandı
  - [x] Tüm firmaların `total_purchases`, `total_payments` ve `balance` değerlerini hareket geçmişine göre sıfırdan hesaplayan `recalculate_supplier_balances` SQL fonksiyonu ve CLI aracı (`backend/cmd/recalculate_balances/main.go`) geliştirildi
  - [x] Kuruş doğruluğu ve dönem kilitleri ile çelişmeyen atomik veri denetim testleri yazıldı (`recalculate_balances_test.go` %100 PASS)

- [x] **Task 4: Doğrulama Sonrası Test Veri Seti Temizliği (Cleanup)** (PASS, 2026-10-09)
  - [x] Tüm test ve doğrulamalar başarıyla tamamlandı, `testveriseti/` klasörü ve geçici test SQL dosyaları yerel dizinden güvenle silindi


---

## Not
Sprint 0'daki "AÇIK KARAR" maddesi onaylanmadan Sprint 1 şema tasarımına başlanmamalı — bkz. PROJECT_BRIEF.md Bölüm 3.



