# 🚀 Release Notes — Sürüm v1.4.0

> **Sürüm Tarihi:** 2026-09-27  
> **Platform:** Öncü Otogaz — Kasa, Cari ve Defter-i Kebir Yönetim Platformu  
> **Mimari:** Go Fiber v2 Backend + Next.js 15 App Router Frontend + PostgreSQL / Supabase RLS  

---

### ✨ Sürüm v1.4.0 — Canlı Üretim (Vercel + Render) Stabilizasyonu & Tedarikçi Çift Defter Güçlendirmesi

#### 1. 🌐 CORS & Dinamik Preflight (OPTIONS) Geçişi
- Tarayıcı cross-origin `OPTIONS` (preflight) isteklerinin `AuthMiddleware` tarafından engellenmesi kalıcı olarak çözüldü (`if c.Method() == fiber.MethodOptions { return c.Next() }`).
- `router.go` içinde dinamik CORS eşleşmesi (`AllowOriginsFunc`) ile `*.vercel.app`, `onrender.com`, `oncuotogazmuhasebe.com.tr` ve `localhost` domainleri tam yetkilendirildi.

#### 2. 🔐 Supabase Asimetrik JWT (ES256 ECDSA P-256) JWKS Entegrasyonu
- Modern Supabase projelerinin P-256 asimetrik anahtarlarını önbelleğe alan `JWKSCache` mekanizması ve esnek `aud` (`string` / `[]string`) doğrulaması devreye alındı.
- Şifre sıfırlama ve güvenlik sorusu rotaları halka açık `publicAuth` grubuna ayrılarak sıfır engelle çalışma sağlandı.

#### 3. 📦 Tedarikçiler (Suppliers) int64 Scan & Migration Stabilizasyonu
- PostgreSQL `COUNT(...)` (bigint/int64) alanlarının Go `int` değişkenlerine dönüşümündeki tip uyuşmazlığı giderildi.
- Canlı Supabase üzerinde `15_create_suppliers.sql` ve `16_add_reversed_by_to_supplier_transactions.sql` uygulanarak sıfır veri kaybı ile tam kararlılık sağlandı.

---

# 🚀 Release Notes — Sürüm v1.3.0

> **Sürüm Tarihi:** 2026-09-25  
> **Platform:** Öncü Otogaz — Kasa, Cari ve Defter-i Kebir Yönetim Platformu  
> **Mimari:** Go Fiber v2 Backend + Next.js 15 App Router Frontend + PostgreSQL / Supabase RLS  

---

### ✨ Sürüm v1.3.0 — Telegram Anlık Alarm Motoru & 3-2-1 Google Drive Otomatik Yedekleme

#### 1. 🚨 Telegram Bot Anlık Alarm Sistemi (`pkg/telegram`)
- Sunucu çökmelerini (Panic) ve kritik `500 Internal Server Error` hatalarını anında yöneticinin Telegram hesabına formatlı HTML mesajı ile ileten asenkron non-blocking goroutine kuyruğu entegre edildi.
- Spam önleyici **Alert Throttling** mekanizması ile aynı hata dakikada en fazla 1 kez iletilir.

#### 2. ☁️ 3-2-1 Google Drive Otomatik Yedekleme Motoru (`pkg/backup`, `pkg/gdrive`)
- Veritabanından tüm aktif/kilitli dönemleri, kasa ve tedarikçi hareketlerini atomik snapshot olarak çıkaran ve `.sql.gz` olarak sıkıştıran motor yazıldı.
- Google Drive v3 REST API ile akış yüklemesi (streaming upload), 30 günlük otomatik retention (eski yedek temizliği) ve gece 03:00 cron zamanlayıcısı devreye alındı.
- Yedekleme sonuç raporları otomatik olarak Telegram sohbetine döküm dosyasıyla birlikte gönderilir.

#### 3. 🔄 Otomatik Dönem Kilitleme & JWT 401 Seamless Auto-Refresh
- `open_next_period()` içine yeni ay açılırken önceki açık dönemi atomik olarak kilitleme kuralı eklendi.
- Frontend `apiFetch` istemcisine 401/403 durumunda `refreshSession()` ile otomatik sessiz token yenileme ve isteği tekrarlama (seamless retry) yeteneği kazandırıldı.

---

# 🚀 Release Notes — Sürüm v1.2.0

> **Sürüm Tarihi:** 2026-09-24  
> **Platform:** Öncü Otogaz — Kasa, Cari ve Defter-i Kebir Yönetim Platformu  
> **Mimari:** Go Fiber v2 Backend + Next.js 15 App Router Frontend + PostgreSQL / Supabase RLS  

---

### ✨ Sürüm v1.2.0 — Tedarikçi Çift Defter & Ters Kayıt, Sektör Standardı JWT Claims ve Geliştirici Çıkışı

#### 1. 🔄 Tedarikçiler & Parçacılar Çift Defter / Ters Kayıt (Reversal) Mimarisi
- Kasa Defteri ile birebir uyumlu **Append-Only Ters Kayıt (Reversal)** mekanizması Tedarikçi & Parçacı modülüne taşındı.
- `migrations/16_add_reversed_by_to_supplier_transactions.sql` ile `supplier_transactions` tablosuna `reversed_by` self-reference sütunu ve endeksi eklendi.
- `POST /api/v1/suppliers/transactions/:id/reverse` uç noktası eklendi; veritabanı seviyesinde atomik işlem (`pgx.Tx`), ters bakiye güncellemeleri ve mükerrer iptal engeli (`422 Unprocessable Entity`) sağlandı.
- Frontend tarafında `ReverseSupplierTransactionDialog.tsx` modalı, `[İPTAL EDİLDİ]` ve `[TERS KAYIT]` durum rozetleri ile aktif/iptal edilmiş işlem filtreleme seçenekleri sunuldu.

#### 2. 🔐 Sektör Standardı Supabase JWT Custom Claims (App Metadata)
- `AuthMiddleware` mimarisi kurumsal SaaS standartlarına yükseltildi.
- Supabase Custom Access Token Hook standardı ile uyumlu `claims.app_metadata.tenant_id` ve `claims.app_metadata.role` doğrudan kriptografik imzalı token'dan sıfır DB gecikmesiyle (stateless) çözümlenir.
- `migrations/17_supabase_jwt_custom_claims_hook.sql` oluşturularak Supabase Auth Server için `custom_access_token_hook` fonksiyonu ve yetkilendirmesi tanımlandı.

#### 3. 🔑 Base64 HMAC Secret & Çift Katmanlı İmza Doğrulama
- Supabase Dashboard tarafından üretilen 64-baytlık Base64 kodlu JWT anahtarları (`Lw7HcEdLRo+...==`) ve düz UTF-8 string anahtarlar için çift katmanlı imza çözücü entegre edildi.
- `jwt.Parse` seviyesinde imza uyuşmazlığı ve 403 engeli giderildi.

#### 4. ⚡ Geliştirici Modu (Local Dev Auto-Login & Debug Logs)
- `ENVIRONMENT=development` modunda, tarayıcıdan token gelmediğinde dahi yerel veritabanının birincil işletmesine otomatik admin oturumu sağlayan developer bypass desteği getirildi.
- Olası token doğrulama hatalarında terminale detaylı ve anlaşılır Türkçe debug logları basılması sağlandı.

#### 5. 📊 Akıllı Excel İçe Aktarımı ve Format Desteği
- Tedarikçi Excel yükleme motoru (`matchSheetForPeriod`) tek sayfalı dosyaları (örn. `MAYIS25`, `KASA`) otomatik algılayacak ve Türkçe ay isimlerini haritalayacak şekilde güçlendirildi.
- 10MB boyut sınırı (`io.LimitReader`) ve kullanıcı dostu Türkçe hata açıklamaları eklendi.

---

# 🚀 Release Notes — Sürüm v1.1.0

> **Sürüm Tarihi:** 2026-09-04  
> **Platform:** Öncü Otogaz — Kasa ve Defter-i Kebir Yönetim Platformu  
> **Mimari:** Go Fiber v2 Backend + Next.js 15 App Router Frontend + PostgreSQL / Supabase RLS  

---

### ✨ Sürüm v1.1.0 — Supabase Bulut Entegrasyonu, Tip Güvenliği & Otomatik Üyelik

#### 1. 🛡️ Supabase TypeScript Tip Güvenliği (End-to-End Type Safety)
- `frontend/src/types/database.types.ts` oluşturuldu; `public.tenants`, `public.periods`, `public.transactions`, `public.idempotency_keys` ve `public.user_security` şeması tam olarak modellendi.
- Supabase browser ve SSR server istemcilerine `<Database>` tipi bağlandı. `npx tsc --noEmit` ile 0 tip hatası doğrulandı.

#### 2. 👥 Otomatik Kullanıcı-İşletme (Tenant) Bağlama Motoru
- `13_auto_assign_tenant_on_signup.sql` migration'ı ile `auth.users` üzerinde `on_auth_user_created` trigger'ı kuruldu.
- Yeni kayıt olan her kullanıcının `tenant_members` tablosuna `admin` rolüyle otomatik eklenmesi sağlandı; böylece RLS kısıtlamasından kaynaklanan boş ekran ve 403 erişim engelleri kökten çözüldü.

#### 3. 🔐 Giriş ve Kayıt Portalı İyileştirmesi
- `frontend/src/app/[locale]/login/page.tsx` arayüzüne "Yeni Kayıt Ol" modu (`supabase.auth.signUp`) entegre edildi.
- Giriş ve kayıt modları arasında anlık geçiş ve bilgilendirici durum mesajları sağlandı.

#### 4. 🗄️ Konsolide Şema ve CLI Migration Aracı
- Supabase Dashboard SQL Editor üzerinden tek tıkla çalıştırılabilir `migrations/supabase_combined_schema.sql` hazırlandı.
- Terminalden otomatik çalıştırma için `backend/cmd/migrate/main.go` aracı eklendi.

---

# 🚀 Release Notes — Sürüm v1.0.0

> **Sürüm Tarihi:** 2026-08-20  
> **Platform:** Öncü Otogaz — Kasa ve Defter-i Kebir Yönetim Platformu  
> **Mimari:** Go Fiber v2 Backend + Next.js 15 App Router Frontend + PostgreSQL / Supabase RLS  

---

### ✨ Öne Çıkan Özellikler ve Yenilikler

#### 1. 🏢 Öncü Otogaz Kurumsal Kimlik & Sarı-Siyah Tema
- Kurumsal endüstriyel **Amber-500 & Zinc-950** renk paleti entegrasyonu.
- Çoklu dil desteği (`tr`/`en` i18n JSON) ile sıfır hardcoded metin garantisi.

#### 2. 📊 Append-Only Defter ve Devir Garantili Dönem Motoru
- `shopspring/decimal` ile kuruşu kuruşuna (0.01 TL) ondalık hassasiyet.
- Önceki dönem net bakiyesini otomatik yeni dönem devir bakiyesi yapan `open_next_period()` veritabanı fonksiyonu.
- Kilitlenen (`status = 'locked'`) finansal dönemlerde veritabanı seviyesinde `UPDATE`/`DELETE` yasağı ve immutability (değişmezlik).

#### 3. 🔒 Güvenlik & İzolasyon
- Tenant bazlı **Row Level Security (RLS)** (`USING (tenant_id = ANY(current_tenant_ids()))`).
- `Idempotency-Key` middleware ile eşzamanlı isteklerde mükerrer işlem engeli.
- Next.js **Route Guard Middleware** ile yetkisiz erişimlerin otomatik `/login` sayfasına yönlendirilmesi.
- `bcrypt` şifrelenmiş cevaba dayalı **Güvenlik Sorulu Şifre Sıfırlama** modülü (`user_security`).
- Güvenlik headerları (`X-Content-Type-Options`, `X-Frame-Options`, HSTS) ve non-root Docker container yapılandırması.

#### 4. 📁 Excel/CSV İçe ve Dışa Aktarım Engine
- Dönemsel verileri Microsoft Excel uyumlu **UTF-8 BOM** (`\xEF\xBB\xBF`) formatında indiren CSV Export handler.
- Hatalı satır numarası raporlayan toplu CSV Import modülü.

#### 5. 🛠️ CI/CD & Deployment
- GitHub Actions CI Pipeline (`.github/workflows/ci.yml`) ile otomatik `go test -v -race` ve `npm run build` doğrulaması.
- Render platform yayın yapılandırması (`render.yaml`).

---

### 🧪 Test & Kalite Metrikleri
- **Go Unit & Integration Tests:** 45/45 PASS (%100)
- **Next.js Production Build:** 8/8 Static Pages Exit Code 0 (%100)
- **SQL Integrity Scenarios:** 9/9 Scenarios PASS (%100)
