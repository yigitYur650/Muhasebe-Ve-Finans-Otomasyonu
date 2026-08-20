# PROJECT_MAP_FOR_LLM.md — Proje Haritası ve Dosya İndeksi (SSOT)

> **Amaç:** Bu dosya, "Kasa ve Defter-i Kebir Yönetim Platformu" projesindeki tüm dizin, dosya, migration, bileşen ve servislerin tek haritasıdır (SSOT).
> **Kural (llmrules Madde 2 & 5):** Yapılan her klasör/dosya değişikliği, eklenen yeni migration veya servis anında bu haritaya işlenir.

---

## 1. Genel Mimari ve Klasör Yapısı

```
/deftersystem
├── /backend
│   ├── /cmd
│   │   └── /api
│   │       └── main.go          # Fiber HTTP API sunucu giriş noktası
│   ├── /internal
│   │   ├── /domain              # Entity struct'ları, custom error'lar, repository/service interfaceleri & unit testleri
│   │   │   ├── tenant.go
│   │   │   ├── period.go
│   │   │   ├── transaction.go
│   │   │   ├── idempotency.go
│   │   │   ├── errors.go
│   │   │   ├── repository.go
│   │   │   ├── service.go
│   │   │   └── decimal_test.go
│   │   ├── /repository          # PostgreSQL pgxpool erişim katmanı, error mapping & unit testleri
│   │   │   ├── postgres.go
│   │   │   ├── errors.go
│   │   │   ├── tenant_repo.go
│   │   │   ├── period_repo.go
│   │   │   ├── transaction_repo.go
│   │   │   ├── idempotency_repo.go
│   │   │   └── repository_test.go
│   │   ├── /service             # İş mantığı servis katmanı & mock/unit testleri
│   │   │   ├── period_service.go
│   │   │   ├── transaction_service.go
│   │   │   ├── mocks_test.go
│   │   │   ├── period_service_test.go
│   │   └── /handler             # Fiber HTTP Handler katmanı, DTO'lar, Middleware ve E2E testleri
│   │       ├── dto.go
│   │       ├── errors.go
│   │       ├── period_handler.go
│   │       ├── transaction_handler.go
│   │       ├── router.go
│   │       ├── handler_test.go
│   │       └── /middleware
│   │           ├── context_middleware.go
│   │           └── idempotency_middleware.go
│   ├── /pkg
│   │   └── /validator           # (Yardımcı validasyon paketleri)
│   ├── Dockerfile               # Multi-stage rootless Alpine Dockerfile (appuser 10001)
│   ├── .env.example
│   ├── go.mod
│   └── go.sum
├── /frontend
│   ├── /src
│   │   ├── /app
│   │   │   └── /[locale]
│   │   │       ├── layout.tsx    # Next.js 15 App Router kök yerleşimi (i18n & Tailwind)
│   │   │       └── page.tsx      # Canlı KPI bakiye paneli & işlem defteri tablosu
│   │   ├── /components
│   │   │   ├── /shared           # Header, PeriodBadge gibi ortak bileşenler
│   │   │   └── /ui               # shadcn/ui temel bileşenleri (Button, Table, Card, Badge, vb.)
│   │   ├── /i18n                 # next-intl istemci ve sunucu konfigürasyonu
│   │   ├── /lib                  # api.ts (Backend HTTP client), decimal.ts (decimal.js & TL format)
│   │   └── /messages             # tr.json ve en.json i18n çeviri dosyaları (hardcoded string yasak)
│   ├── Dockerfile               # Multi-stage standalone Node.js Dockerfile (nextjs 1001)
│   ├── .env.example
│   ├── next.config.mjs
│   ├── package.json
│   └── tsconfig.json
├── /.github
│   └── /workflows
│       └── ci.yml               # Automated GitHub Actions CI/CD Pipeline (Go & Next.js tests/builds)
├── docker-compose.yml           # Root local orchestration file
├── /docs
│   ├── PROJECT_BRIEF.md         # Proje hedefleri, mimari kararlar ve kurallar (SSOT)
│   ├── TASK.md                  # Sprint ve görev takip listesi (SSOT)
│   ├── BUG_AND_FIX.md           # Hata kök neden ve çözüm kayıtları (SSOT)
│   ├── SECURITY_AUDIT_REPORT.md # Güvenlik denetim bulguları ve checklist
│   └── PROJECT_MAP_FOR_LLM.md   # Kod dizin ve dosya haritası (Bu dosya - SSOT)
├── /migrations
│   ├── 01_create_tenants.sql
│   ├── 02_create_tenant_members.sql
│   ├── 03_create_current_tenant_fn.sql
│   ├── 04_create_periods.sql
│   ├── 05_create_transactions.sql
│   ├── 06_period_rollover_fn.sql
│   ├── 07_period_lock_and_append_only_triggers.sql
│   ├── 08_rls_periods_and_transactions.sql
│   ├── 09_create_idempotency_keys.sql
│   └── test_scenarios.sql       # Sprint 1 SQL Bütünlük ve İzolasyon Test Scripti
```

---

## 2. Dokümantasyon İndeksi (`/docs`)

- **`docs/PROJECT_BRIEF.md`**: Problem tanımı, tech stack (Next.js 15, Go Fiber v2, Supabase PostgreSQL), mimari kararlar (append-only ledger + period lock), dönem devir kuralları ve hard rules.
- **`docs/TASK.md`**: Sprint 0 - Sprint 8 arası tüm iş paketlerinin durumları.
- **`docs/BUG_AND_FIX.md`**: Karşılaşılan hataların kök neden analizi, uygulanan düzeltmeler ve test doğrulama çıktıları.
- **`docs/SECURITY_AUDIT_REPORT.md`**: Güvenlik denetim sonuçları ve P0-P3 güvenlik açıkları takip raporu.
- **`docs/PROJECT_MAP_FOR_LLM.md`**: Projenin dosya indeksi ve modül haritası.

---

## 3. Veritabanı Migration Kayıtları (`/migrations`)

| Sıra | Dosya Adı | Tablo / Obje | Amaç ve Açıklama |
|---|---|---|---|
| 01 | `01_create_tenants.sql` | `public.tenants` | Çoklu kiracı (tenant) id ve isim yönetimi. RLS enabled. |
| 02 | `02_create_tenant_members.sql` | `public.tenant_members` | `auth.users` ile tenant arasında rol bazlı üyelik (`admin`, `muhasebeci`, `standart`). RLS enabled. |
| 03 | `03_create_current_tenant_fn.sql` | `public.current_tenant_ids()` | RLS politikalarında kullanılan oturumdaki kullanıcının bağlı olduğu tenant ID kümesini dönen fonksiyon (`STABLE`, `SECURITY DEFINER`). |
| 04 | `04_create_periods.sql` | `public.periods` | Aylık dönem tablosu (`label`, `starting_balance`, `status`: open/locked). RLS enabled. |
| 05 | `05_create_transactions.sql` | `public.transactions` | Append-only işlem tablosu (`direction`: in/out, `channel`, `amount` NUMERIC(15,2), `reversed_by`). RLS enabled. |
| 06 | `06_period_rollover_fn.sql` | `public.open_next_period()` | Önceki dönemin kapanış bakiyesini (Gelir - Gider) otomatik `starting_balance` olarak yeni döneme devreden devir fonksiyonu (`SECURITY DEFINER`). |
| 07 | `07_period_lock_and_append_only_triggers.sql` | `trg_prevent_transaction_update`, `trg_prevent_transaction_delete`, `trg_prevent_locked_period_insert` | `transactions` üzerinde `UPDATE`/`DELETE` işlemlerini engelleyen append-only garantisi ve `locked` döneme `INSERT` engelleyen DB trigger'ları. |
| 08 | `08_rls_periods_and_transactions.sql` | RLS Policies & Grants | `periods` ve `transactions` tabloları için tenant bazlı `SELECT` ve `INSERT` RLS politikaları. `authenticated` rolüne açık GRANT. |
| 09 | `09_create_idempotency_keys.sql` | `public.idempotency_keys` | Idempotency-Key middleware için tekilleştirme ve yanıt saklama tablosu (`key` PK, `tenant_id`, `response_body`). |
| Test | `test_scenarios.sql` | Integration Tests | 9/9 SQL bütünlük, trigger kısıtı ve multi-tenant RLS izolasyon doğrulama scripti. |

---

## 4. Go Backend Mimarisi ve Modül İndeksi (`/backend`)

| Paket / Katman | Dosya Adı | Açıklama |
|---|---|---|
| `cmd/api` | `main.go` | Fiber v2 tabanlı HTTP API sunucusunu ayağa kaldıran giriş noktası. `/health` kontrol endpoint'i barındırır. |
| `internal/domain` | `tenant.go` | `Tenant`, `TenantMember` struct tanımları ve `Role` sabitleri (`admin`, `muhasebeci`, `standart`). |
| `internal/domain` | `period.go` | `Period` struct tanımı, `PeriodStatus` sabitleri (`open`, `locked`) ve kilit durumu kontrol metotları. |
| `internal/domain` | `transaction.go` | `Transaction` struct tanımı, `Direction` (`in`, `out`), `Channel` (12 işlem kanalı) sabitleri ve domain validasyonu. |
| `internal/domain` | `idempotency.go` | `IdempotencyKey` struct tanımı. |
| `internal/domain` | `errors.go` | Merkezi domain hata tanımları (`ErrPeriodLocked`, `ErrInvalidAmount`, `ErrUnauthorized`, vb.). |
| `internal/domain` | `repository.go` | Clean Architecture veritabanı erişim arayüzleri (`TenantRepository`, `PeriodRepository`, `TransactionRepository`, `IdempotencyRepository`). |
| `internal/domain` | `service.go` | Clean Architecture iş mantığı servis arayüzleri (`PeriodService`, `TransactionService`) ve `PeriodSummary` struct'ı. |
| `internal/domain` | `decimal_test.go` | `shopspring/decimal` ile hassas parasal hesaplamalar, bakiye devri, tutar validasyonu ve kısıtların unit testleri (6/6 PASS). |
| `internal/repository` | `postgres.go` | `pgxpool.Pool` bağlantı havuzunu production standartlarında yapılandıran ve başlatan ilklendirici. |
| `internal/repository` | `errors.go` | PostgreSQL ve pgx veritabanı hatalarını (P0001 trigger, 23505 unique violation, ErrNoRows) domain hatalarına dönüştüren `MapSQLError`. |
| `internal/repository` | `tenant_repo.go` | `TenantRepository` PostgreSQL somut uygulaması (`GetByID`, `Create`, `GetMember`, `GetMembersByTenantID`). |
| `internal/repository` | `period_repo.go` | `PeriodRepository` PostgreSQL somut uygulaması (`GetByID`, `GetByLabel`, `GetLatestByTenant`, `OpenNextPeriod`, `Create`, `Lock`). |
| `internal/repository` | `transaction_repo.go` | `TransactionRepository` PostgreSQL somut uygulaması (`Create`, `GetByID`, `GetByPeriodID`, `GetSummaryByPeriodID`, `ReverseTransaction`, `MarkReversed`). |
| `internal/repository` | `idempotency_repo.go` | `IdempotencyRepository` PostgreSQL somut uygulaması (`Get`, `Save`). |
| `internal/repository` | `repository_test.go` | SQL hata dönüştürme (error mapping) ve repository katmanı unit testleri (5/5 PASS). |
| `internal/service` | `period_service.go` | `PeriodService` iş mantığı uygulaması (dönem devri, rol yetki denetimi ile kilitleme, dönem listeleme, bakiye özeti). |
| `internal/service` | `transaction_service.go` | `TransactionService` iş mantığı uygulaması (kilitli dönem denetimi, ters kayıt/reversal üretimi ve yön dönüşümü). |
| `internal/service` | `tenant_service.go` | `TenantService` üye ve rol yönetimi uygulaması (`AddMember`, `UpdateMemberRole`, `RemoveMember`, son admin koruması). |
| `internal/service` | `tenant_service_test.go` | `TenantService` RBAC ve son admin koruması unit testleri (2/2 PASS). |
| `internal/service` | `mocks_test.go` | Servis testlerinde kullanılan `testify/mock` repository sahte nesne tanımları. |
| `internal/service` | `period_service_test.go` | `PeriodService` rol yetki ve bakiye özeti unit testleri (4/4 PASS). |
| `internal/service` | `transaction_service_test.go` | `TransactionService` zıt yönlü ters kayıt, çift iptal engeli ve kilitli dönem engeli unit testleri (3/3 PASS). |
| `internal/service` | `import_service.go` | `ImportService` toplu CSV aktarım uygulaması (satır satır validasyon, UTF-8 BOM, decimal/tutar kontrolü, kilitli dönem engeli). |
| `internal/service` | `import_service_test.go` | `ImportService` CSV satır hatası, kilitli dönem reddi ve kuruşu kuruşuna toplu aktarım unit testleri (3/3 PASS). |
| `internal/service` | `advanced_edge_cases_test.go` | 6 İleri Düzey Uç Senaryo Test Paketi (Hassasiyet, Reversal of Reversal, Race condition, Negatif/Kesirli tutar, Idempotency izolasyonu, Kilitli bakiye değişmezliği) (6/6 PASS). |
| `internal/handler` | `dto.go` | JSON İstek ve Yanıt DTO'ları (`OpenPeriodRequest`, `CreateTransactionRequest` decimal.Decimal, `ReverseTransactionRequest`, `ResponseEnvelope`). |
| `internal/handler` | `errors.go` | Domain hatalarını standart JSON yanıt formatına (`{"success": false, "error": {...}}`) ve HTTP status kodlarına dönüştüren `CustomErrorHandler`. |
| `internal/handler` | `period_handler.go` | Fiber HTTP `PeriodHandler` (`ListPeriods`, `OpenNextPeriod`, `LockPeriod`, `GetPeriodSummary`). |
| `internal/handler` | `tenant_handler.go` | Fiber HTTP `TenantHandler` (`ListMembers`, `AddMember`, `UpdateMemberRole`, `RemoveMember`). |
| `internal/handler` | `transaction_handler.go` | Fiber HTTP `TransactionHandler` (`CreateTransaction`, `ReverseTransaction`, `ListTransactions`). |
| `internal/handler` | `export_handler.go` | Fiber HTTP `ExportHandler` (`ExportTransactionsCSV` - UTF-8 BOM ile Excel uyumlu CSV akışı, `DownloadSampleCSVTemplate`). |
| `internal/handler` | `import_handler.go` | Fiber HTTP `ImportHandler` (`ImportTransactionsCSV` - multipart/form-data ve raw body CSV toplu aktarım). |
| `internal/handler` | `router.go` | Fiber yönlendirme tablosunu, recover, context, idempotency ve tenant üye middleware/rotalarını bağlayan `SetupRouter`. |
| `internal/handler/middleware` | `context_middleware.go` | HTTP header'larından (`X-Tenant-ID`, `X-User-ID`, `X-User-Role`) oturum verisini parse eden ve `GetTenantID`, `GetUserID` yardımcılarını sunan middleware. |
| `internal/handler/middleware` | `idempotency_middleware.go` | `Idempotency-Key` başlığını denetleyen, tekrar isteklerinde önceden üretilmiş yanıtı (HTTP status & JSON body) DB'den dönen middleware. |
| `internal/handler/middleware` | `auth_middleware.go` | Supabase JWT doğrulaması (`aud: "authenticated"`, `exp`) ve veritabanı tenant rol denetimi (`tenantRepo.GetMember`) yapan middleware. |
| `internal/handler/middleware` | `auth_middleware_test.go` | JWT doğrulama unit testleri (Geçerli token, süresi dolmuş token, sahte imza, yetkisiz tenant erişimi) (4/4 PASS). |
| `internal/handler` | `handler_test.go` | Fiber `httptest` ile idempotency tekilleştirme, negatif tutar reddi (400), kilitli dönem (422) ve yetkisiz rol (403) HTTP testleri (4/4 PASS). |
| `internal/handler` | `integration_test.go` | 5 Kritik E2E & Güvenlik Matrisi entegrasyon testi (Uçtan uca defter akışı, ters kayıt audit bütünlüğü, idempotency tekilleştirme, kilitli dönem engeli, multi-tenant & rol izolasyonu) (5/5 PASS). |

---

## 5. Next.js 15 Frontend Mimarisi ve Modül İndeksi (`/frontend`)

| Katman / Paket | Dosya Adı | Açıklama |
|---|---|---|
| `app` | `page.tsx` | Kök dizin `/` isteklerini varsayılan dil olan `/tr` rotasına sunucu taraflı güvenle yönlendiren `RootPage`. |
| `app` | `layout.tsx` | Next.js 15 App Router kök dizin yerleşim ilklendiricisi `RootLayout`. |
| `app/[locale]` | `layout.tsx` | Next.js 15 App Router locale yerleşimi (`NextIntlClientProvider`, Inter font ve Tailwind CSS). |
| `app/[locale]` | `page.tsx` | KPI bakiye kartları, dönem seçici, arşiv uyarı banner'ı, CSV dışa/içe aktar butonları ve işlem defteri tablosu. |

| `app/[locale]/login` | `page.tsx` | Supabase Auth entegrasyonlu, i18n destekli (hardcoded metinsiz) kullanıcı giriş sayfası. |
| `components/auth` | `ForgotPasswordDialog.tsx` | Güvenlik sorusu yanıtı ile şifre sıfırlama modalı. |
| `components/auth` | `ChangePasswordDialog.tsx` | Oturum açmış kullanıcılar için güvenlik sorusu ve şifre güncelleme modalı. |
| `components/shared` | `Header.tsx` | Marka başlığı, tenant etiketi, rol rozeti, şifre değiştir butonu ve dil değiştirici (`tr`/`en`) üst navigasyon çubuğu. |

| `components/shared` | `PeriodBadge.tsx` | Dönemin kilitli (`locked`) veya açık (`open`) olma durumunu görsel olarak sunan durum rozeti. |
| `components/ledger` | `PeriodSelector.tsx` | Tüm açık ve kilitli geçmiş dönemleri listeleyen ve salt-okunur arşiv modunu tetikleyen Select bileşeni. |
| `components/ledger` | `QuickEntryRow.tsx` | Excel stili klavye odaklı hızlı satır girişi barı (`Enter`, `Tab`, `G/C`, `Esc`, inline decimal validasyonu). |
| `components/ledger` | `KpiSummaryCards.tsx` | 4 Metrik Kartlı canlı finansal bakiye özeti (Açılış, Gelir, Gider, Net Kasa, `formatTL` kuruş hassasiyeti). |
| `components/ledger` | `PeriodHistoryView.tsx` | Kapanmış ve kilitlenmiş geçmiş dönemlerin karşılaştırmalı salt-okunur arşiv tablosu ("Defteri İncele" butonu ile). |
| `components/ledger` | `TransactionTable.tsx` | TanStack Table (`@tanstack/react-table`) defter tablosu, filtreleme araç çubuğu ve ters kayıt görsel stilleri. |
| `components/ledger` | `ExportCsvButton.tsx` | Excel Türkçe karakter uyumlu (UTF-8 BOM) dönemsel CSV indirme butonu. |
| `components/ledger` | `ImportCsvDialog.tsx` | Drag & drop veya dosya seçici ile toplu CSV yükleme, örnek şablon indirme ve satır hatası raporlama modalı. |
| `components/ledger` | `CreateTransactionDialog.tsx` | Hızlı satır girişi modalı, `crypto.randomUUID()` ile frontend idempotency key üretimi. |
| `components/ledger` | `ReverseTransactionDialog.tsx` | Ters kayıt (iptal) modalı, yasal denetim uyarısı ve gerekçe (`reason`) zorunluluğu. |
| `components/ledger` | `PeriodActionDialog.tsx` | Rol denetimli dönem kilitleme (`LockPeriod`) ve yeni dönem açma (`OpenNextPeriod`) modalı. |
| `components/admin` | `MemberManagementDialog.tsx` | Yalnızca `admin` rolü tarafından açılabilen üye listeleme, rol değiştirme, yeni üye ekleme ve çıkarabilme modalı. |
| `components/ui` | `button.tsx`, `input.tsx`, `card.tsx`, `badge.tsx`, `table.tsx`, `dialog.tsx`, `select.tsx` | `shadcn/ui` UI temel bileşenleri. |
| `lib/supabase` | `client.ts`, `server.ts` | Supabase Browser ve SSR Server client yardımcıları (`@supabase/ssr`). |
| `lib` | `decimal.ts` | `decimal.js` ile float taşmasız parasal hesaplama metotları ve `formatTL` para formatlayıcı. |
| `lib` | `api.ts` | Go Backend `/api/v1` rotalarına otomatik Supabase Bearer token enjeksiyonu ile erişen merkezi HTTP istemcisi. |
| `lib` | `utils.ts` | `clsx` ve `tailwind-merge` birleştiren `cn` yardımcı metodu. |
| `messages` | `tr.json`, `en.json` | Hardcoded metin kullanımını engelleyen modüler i18n çeviri sözlükleri (`common`, `period`, `transaction`, `auth`, `errors`, `import_export`). |
| `i18n` | `request.ts`, `middleware.ts` | `next-intl` dili otomatik algılama ve dinamik rota yönlendirme middleware'i. |
| Root | `next.config.mjs` | `next-intl` eklentisini Next.js derleme sürecine entegre eden konfigürasyon dosyası. |


