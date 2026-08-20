# TASK.md — Kasa ve Defter-i Kebir Yönetim Platformu

> Kural: Bir mikro-prompt'un testleri geçmeden `[x]` işaretlenmez, sıradaki adıma geçilmez.
> Scope sprint başladıktan sonra dondurulur — mid-sprint ekleme yok (bkz. antigravityrules).

---

## Sprint 0 — Proje İskeleti ve Dokümantasyon

- [x] `docs/` klasörü ve tüm SSOT dosyaları oluşturulur (PROJECT_BRIEF.md ✅, TASK.md ✅, BUG_AND_FIX.md ✅, PROJECT_MAP_FOR_LLM.md ✅, SECURITY_AUDIT_REPORT.md ✅)
- [x] `antigravityrules` bu proje için uyarlanır (Kap-App şablonundan, stack referansları güncellenir)
- [x] Backend iskeleti: Go modül init, Fiber v2 kurulum, Clean Architecture klasörleri (`domain/repository/service/handler`)
- [x] Frontend iskeleti: Next.js 15 App Router init, shadcn/ui + Tailwind kurulum, TanStack Table bağımlılığı (PASS, 2026-08-20)
- [ ] Supabase projesi oluşturulur, `.env.example` yazılır, `.gitignore` doğrulanır (`.env` asla commit edilmez)
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
- [ ] CORS: production'da açık origin listesi, wildcard yasak

## Sprint 4 — Klavye Odaklı Veri Girişi Arayüzü

- [x] TanStack Table ile hızlı satır ekleme/düzenleme (klavye kısayolları: Enter ile yeni satır, Tab ile alan geçişi, G/C kısayolları) (PASS, 2026-08-20)
- [x] Kanal seçimi (EFT/POS/nakit/kredi/kira/maaş/kredi kartı/kartuş/yemek/yakıt) — i18n JSON'dan (PASS, 2026-08-20)
- [x] shadcn/ui form bileşenleri, hardcoded metin yok (PASS, 2026-08-20)
- [x] Frontend edge-case: negatif/hatalı tutar girişinde inline validasyon (PASS, 2026-08-20)

## Sprint 5 — Canlı KPI ve Bakiye Paneli

- [ ] Seçili döneme ait toplam gelir/gider/net bakiye canlı hesaplama (backend endpoint + frontend özet kart)
- [ ] Dönem geçmişi görünümü (kapanmış dönemlerin salt-okunur listesi)

## Sprint 6 — Excel Import/Export ve Tenant İzolasyon E2E

- [x] Uçtan Uca Entegrasyon ve Güvenlik/İzolasyon Testleri (5 Kritik E2E & Güvenlik Matrisi Senaryosu) (PASS, 2026-08-20)
- [ ] Mevcut Excel geçmişinin sisteme aktarımı (import) — format doğrulama ve hata raporlama
- [ ] Excel/PDF dışa aktarım
- [x] Playwright / Go Integration E2E: dönemler arası bakiye devri doğruluğu (PASS, 2026-08-20)
- [x] Playwright / Go Integration E2E: tenant izolasyonu (bir tenant diğerinin verisini göremiyor) (PASS, 2026-08-20)
- [ ] Playwright E2E: Excel import sonrası hesaplamaların doğruluğu

## Sprint 7 — Güvenlik Denetimi ve Sertleştirme

- [x] Güvenlik Sıkılaştırma, SQL SETOF Tenant Düzeltmesi ve Idempotency Kompozit Key Güvenliği (PASS, 2026-08-20)
- [ ] `SECURITY_AUDIT_REPORT.md` bu proje için doldurulur (Kap-App raporundaki formatla)
- [ ] Güvenlik headerları eklenir (`X-Content-Type-Options`, `X-Frame-Options`, HSTS, CSP)
- [x] Dockerfile: non-root kullanıcı (backend appuser 10001, frontend nextjs 1001) (PASS, 2026-08-20)
- [ ] Rate limiter kalıcılığı (in-memory yerine, restart'ta sıfırlanmayan bir çözüm) değerlendirilir

## Sprint 8 — Deployment ve Release

- [x] Docker Containerization, GitHub Actions CI/CD ve Production Deployment Hazırlığı (PASS, 2026-08-20)
- [ ] `render.yaml` (veya seçilen platform) — tüm env var'lar eksiksiz tanımlı
- [ ] `RELEASE_NOTES.md` ilk sürüm için doldurulur
- [ ] Canlıya hazır demo doğrulaması

---

## Not
Sprint 0'daki "AÇIK KARAR" maddesi onaylanmadan Sprint 1 şema tasarımına başlanmamalı — bkz. PROJECT_BRIEF.md Bölüm 3.
