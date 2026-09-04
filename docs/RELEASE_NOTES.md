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
