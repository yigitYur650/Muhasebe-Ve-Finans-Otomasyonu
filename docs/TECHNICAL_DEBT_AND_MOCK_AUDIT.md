# 📋 Teknik Borçlar, Mock Veriler ve Mimari İnceleme Raporu
**Proje:** Muhasebe ve Finans Otomasyonu (Defter-i Kebir & Kasa Sistemi)  
**Tarih:** 2026-09-04  
**Kapsam:** Backend (Go Fiber v2), Frontend (Next.js 15), Veritabanı (Supabase PostgreSQL), Güvenlik & DevOps  
**Amaç:** Sistemdeki geçici kodlar, mock (sahte) yapılar, hardcoded değerler ve prodüksiyona geçiş öncesi çözülmesi gereken mimari teknik borçların eksiksiz envanteri.

---

## 📌 İçindekiler
1. [Özet Değerlendirme](#1-özet-değerlendirme)
2. [Mock Veriler ve Bellek İçi (In-Memory) Kalıntılar](#2-mock-veriler-ve-bellek-içi-in-memory-kalıntılar)
3. [Hardcoded Değerler ve Sabit Tanımlar](#3-hardcoded-değerler-ve-sabit-tanımlar)
4. [Kimlik Doğrulama ve Güvenlik Açıkları (Auth & Security Debt)](#4-kimlik-doğrulama-ve-güvenlik-açıkları-auth--security-debt)
5. [Mimari & Veri Bütünlüğü Borçları (Backend & Database)](#5-mimari--veri-bütünlüğü-borçları-backend--database)
6. [Frontend & Kullanıcı Deneyimi Teknik Borçları](#6-frontend--kullanıcı-deneyimi-teknik-borçları)
7. [DevOps, Migration ve Test Altyapısı Borçları](#7-devops-migration-ve-test-altyapısı-borçları)
8. [Önceliklendirilmiş Çözüm Yol Haritası (Priority Action Matrix)](#8-önceliklendirilmiş-çözüm-yol-haritası-priority-action-matrix)

---

## 1. Özet Değerlendirme

Yapılan derin statik ve dinamik kod analizinde; temel çift taraflı kayıt kuralları, append-only defter mimarisi, dönem kilitleme (`lock`) ve devir bakiyesi (`rollover`) fonksiyonlarının veritabanı seviyesinde sağlam kurulduğu tespit edilmiştir. 

Ancak geliştirme sürecinde hız kazanmak amacıyla eklenen **mock repository'ler**, **doğrulanmayan HTTP başlıkları (headers)**, **hardcoded UUID'ler** ve **Next.js arayüzündeki yerel bypass kontrolleri** üretim (production) ortamı öncesinde çözülmesi gereken ciddi teknik borçlar oluşturmaktadır.
| Kategori | Kritiklik Seviyesi | Tespit Sayısı | Ana Risk |
| :--- | :---: | :---: | :--- |
| **Güvenlik & Auth** | 🔴 YÜKSEK | 4 | JWT doğrulanmaması, sahte session cookie'si, rol sahteciliği |
| **Mock & Hardcoded Veriler** | 🟠 ORTA / YÜKSEK | 6 | Mock repo fallback'leri, hardcoded UUID ve kullanıcı adları |
| **Mimari & Performans** | 🟠 ORTA | 5 | Sayfalama (Pagination) yokluğu, bellek içi CSV oluşturma |
| **Frontend Kod Hijyeni** | 🟡 ORTA | 4 | Monolitik `page.tsx` (630+ satır), prop-drilling, eksik i18n |
| **DevOps & Migration** | 🟢 DÜŞÜK / ORTA | 3 | Migration sürüm takip tablosunun olmaması |

---

## 2. Mock Veriler ve Bellek İçi (In-Memory) Kalıntılar

### 2.1. 🟢 ÇÖZÜLDÜ: `PostgresUserSecurityRepository` Canlı PostgreSQL'e Bağlandı
- **Dosyalar:** `backend/internal/handler/router.go`, `backend/cmd/api/main.go`
- **Durum:** `secRepo := repository.NewPostgresUserSecurityRepository(pool)` canlı havuza bağlandı ve `SetupRouter` üzerinden handler'a enjekte edildi. Şifre sıfırlama güvenlik soruları artık kalıcı olarak Supabase `public.user_security` tablosuna yazılmaktadır.

### 2.2. 🟢 ÇÖZÜLDÜ: Sessiz In-Memory Mock Fallback Kaldırıldı (Fail-Fast)
- **Dosya:** `backend/cmd/api/main.go`
- **Durum:** PostgreSQL bağlantısı koptuğunda veya havuz başlatılamadığında sistemin sessizce RAM sahte kasasına geçmesi engellendi. `log.Fatalf("FATAL: Failed to connect to PostgreSQL database pool: %v")` ile sunucu güvenli şekilde durmaktadır.

### 2.3. 🟢 ÇÖZÜLDÜ: Frontend Dönem Seçimi Dinamik Hale Getirildi
- **Dosya:** `frontend/src/app/[locale]/page.tsx`
- **Durum:** `fetchPeriods` fonksiyonunda aktif (`status === "open"`) dönem otomatik olarak önceliklendirildi. Kullanıcı arayüzünde sabit statik kilitlenme giderildi.

---

## 3. Hardcoded Değerler ve Sabit Tanımlar

### 3.1. Sabit Varsayılan Tenant ve Kullanıcı UUID'leri
- **Kullanılan Değerler:**
  - `00000000-0000-0000-0000-000000000001` (Varsayılan Öncü Otogaz Tenant ID)
  - `149c91f0-0d03-4e3a-81d7-0bc5688c01b0` (Test Kullanıcı ID)
- **Tespit Edilen Dosyalar:**
  - `frontend/src/lib/api.ts` (Satır 74): `defaultHeaders['X-Tenant-ID'] = tenantId || '00000000-0000-0000-0000-000000000001';`
  - `backend/cmd/breakpoint/main.go`, `cmd/capacity/main.go`, `cmd/megastress/main.go`, `cmd/stress/main.go`, `cmd/run_yaml_tests/main.go`
- **Etki:** Çok kiracılı (Multi-tenant SaaS) mimaride tenant ID asla kod içine gömülü (hardcoded) olmamalıdır. Kullanıcı giriş yaptığında Supabase JWT'sinden veya `tenant_members` tablosundan dinamik okunmalıdır.

### 3.2. 🟢 ÇÖZÜLDÜ: İşlem Tablosunda Dinamik Kullanıcı Profili Entegrasyonu
- **Dosya:** `frontend/src/app/[locale]/page.tsx`
- **Durum:** Statik `"Admin"` metni kaldırıldı. Supabase oturumundan dinamik kullanıcı adı (`currentUserLabel`) okunarak optimistik ve gerçek işlemler bu kimlikle etiketlendi.

### 3.3. 🟢 ÇÖZÜLDÜ: Sabit Admin Giriş Bilgileri Temizlendi
- **Dosya:** `frontend/src/app/[locale]/login/page.tsx`
- **Durum:** İstemci tarafında `NEXT_PUBLIC_` ile sızan sabit admin e-posta ve şifresi temizlendi; oturum açma doğrudan Supabase Auth API (`signInWithPassword`) üzerinden güvenli yürütülmektedir.

---

## 4. Kimlik Doğrulama ve Güvenlik Açıkları (Auth & Security Debt)

### 4.1. 🟢 ÇÖZÜLDÜ: Backend'in HTTP Başlıklarına Doğrulamasız Güvenmesi (Spoofing) Engellendi
- **Dosyalar:** `backend/internal/handler/middleware/auth_middleware.go`, `backend/cmd/api/main.go`
- **Durum:** `.env` dosyasındaki tüm çevre değişkenleri (`SUPABASE_JWT_SECRET`) artık uygulama ayağa kalkarken runtime'a eksiksiz yüklenmektedir. `AuthMiddleware` içerisinde geliştirici modu olsa dahi `tenantRepo.GetMember` kontrolü zorunlu tutularak dışarıdan `X-User-Role: admin` başlığıyla yetki sahteciliği yapılması engellendi. Kullanıcı rolü ve yetkisi doğrudan veritabanı `tenant_members` tablosundan atanmaktadır.

### 4.2. 🟢 ÇÖZÜLDÜ: Next.js Middleware Sahte Cookie ile Yetkilendirme Bypass'ı Kaldırıldı
- **Dosya:** `frontend/src/middleware.ts`
- **Durum:** Düz string cookie varlık kontrolü (`defter_session`) tamamen kaldırıldı. Next.js Edge Middleware içerisinde `@supabase/ssr` `createServerClient` entegre edilerek her korumalı sayfa isteğinde gerçek `supabase.auth.getUser()` kriptografik doğrulaması bağlandı. Geçersiz veya süresi dolmuş oturumlarda kullanıcı doğrudan `/${locale}/login` sayfasına yönlendirilmektedir.

### 4.3. 🟢 ÇÖZÜLDÜ: Giriş Sayfasında Sahte Oturum (Bypass Session) Açılması Temizlendi
- **Dosya:** `frontend/src/app/[locale]/login/page.tsx`
- **Durum:** Kullanıcı girişi doğrudan Supabase Auth API (`signInWithPassword`) üzerinden güvenli yürütülmekte, gerçek JWT oturumu oluşturulmaktadır.

---

## 5. Mimari & Veri Bütünlüğü Borçları (Backend & Database)

### 5.1. 🟢 ÇÖZÜLDÜ: Sayfalama (Pagination) Altyapısı Eklendi
- **Dosyalar:** `backend/internal/domain/repository.go`, `backend/internal/repository/transaction_repo.go` (`GetByPeriodIDPaginated`), `backend/internal/handler/transaction_handler.go`, `frontend/src/components/ledger/TransactionTable.tsx`
- **Durum:** Backend katmanında `GetByPeriodIDPaginated` tanımlandı; `page` ve `limit` parametreleri ile `X-Total-Count`, `X-Page`, `X-Limit` başlıkları eklendi. Frontend işlem tablosuna TanStack Table `getPaginationRowModel` ile sayfa başı (25/50/100) kayıt seçimi ve akıcı sayfalama çubuğu entegre edildi.

### 5.2. 🟢 ÇÖZÜLDÜ: Ters Kayıtta (Reversal) Veritabanı Transaction (ACID) Güvencesi
- **Dosya:** `backend/internal/repository/transaction_repo.go` -> `ReverseTransaction()`
- **Durum:** `ReverseTransaction()` metodu `dbTx, err := r.pool.Begin(ctx)` bloğu içinde çalıştırılmakta; orijinal işlemin varlığı, mükerrer iptal kontrolü ve yeni ters kaydın eklenmesi atomik olarak `dbTx.Commit(ctx)` ile tamamlanmaktadır. Zero-UPDATE append-only mimarisi tam güvencededir.

### 5.3. 🟢 ÇÖZÜLDÜ: Otomatik Biçimlendirilmiş Native Excel (.xlsx) Dışa Aktarımı
- **Dosyalar:** `backend/internal/handler/export_handler.go`, `backend/internal/handler/export_excel.go`, `frontend/src/components/ledger/ExportCsvButton.tsx`
- **Durum:** `excelize/v2` entegre edildi. Tarih sütunu `20`, kanallar `22+`, tutar `18` (`#,##0.00`), açıklamalar `40-75` dinamik genişlik ile üretilmektedir. Lisanssız Office görüntüleme modunda dahi kullanıcı müdahalesine gerek kalmaksızın mükemmel sütun genişlikleri, koyu başlık, dondurulmuş satır ve otomatik filtrelerle açılmaktadır.

### 5.4. 🟢 ÇÖZÜLDÜ: SQL `open_next_period()` Saklı Yordamı Go Katmanıyla Eşitlendi
- **Dosyalar:** `migrations/06_period_rollover_fn.sql`, Canlı Supabase PostgreSQL
- **Durum:** Saklı yordam güncellendi. `v_closing_balance` hesaplanırken hem ters kaydın kendisi (`t.reversed_by IS NULL`) hem de orijinal iptal edilmiş işlem (`NOT EXISTS (SELECT 1 FROM transactions rev WHERE rev.reversed_by = t.id)`) süzülerek Go repo mantığıyla kuruşu kuruşuna eşitlendi. Devir bakiyesi `684.047,00 TL` olarak canlı veritabanında doğrulandı.

---

## 6. Frontend & Kullanıcı Deneyimi Teknik Borçları

### 6.1. 🟢 ÇÖZÜLDÜ: Monolitik Bileşen Yapısı (`page.tsx` - 630+ Satırdan Modüler Hook'lara Bölündü)
- **Dosyalar:** `frontend/src/hooks/usePeriods.ts`, `frontend/src/hooks/useTransactions.ts`, `frontend/src/app/[locale]/page.tsx`
- **Durum:** 646 satırlık monolitik sayfa `usePeriods` ve `useTransactions` custom hook'larına ayrıldı. `page.tsx` boyutu ~250 satırlık temiz ve okunabilir bir orkestratöre indirgendi.

### 6.2. 🟢 ÇÖZÜLDÜ: Çoklu Dil (i18n) Eksikleri ve Hardcoded Türkçe Metinler Temizlendi
- **Dosyalar:** `frontend/src/messages/tr.json`, `frontend/src/messages/en.json`, `frontend/src/app/[locale]/page.tsx`, `frontend/src/components/ledger/PeriodHistoryView.tsx`
- **Durum:** Hardcoded metinler (`"İşlem Defteri"`, `"İşlem Kaydı"`, `"Dönem Kilidini Aç"`, `"İşlemler"`) `next-intl` sözlük anahtarlarına bağlanarak İngilizce dil desteğiyle tam senkronize edildi.

### 6.3. Eksik Error Boundary ve Ağ Hatası Geri Bildirimleri
- **Durum:** API istekleri başarısız olduğunda `alert()` fonksiyonu veya sessiz `catch {}` blokları kullanılmaktadır. Toast bildirim sistemi (`sonner` veya `react-hot-toast`) eksiktir.

---

## 7. DevOps, Migration ve Test Altyapısı Borçları

### 7.1. 🟢 ÇÖZÜLDÜ: Akıllı Migration Motoru ve Sürüm Takip Tablosu (`schema_migrations`)
- **Dosyalar:** `backend/cmd/migrate/main.go`, `migrations/14_strict_data_loss_prevention.sql`, `backend/cmd/backup/main.go`
- **Durum:** PostgreSQL veritabanında `public.schema_migrations` tablosu oluşturuldu. `backend/cmd/migrate/main.go` aracı güncellenerek akıllı baseline tespiti, sıralı SQL çalıştırma (`BEGIN...COMMIT`) ve sürüm kaydı yapısı kuruldu.
- **Veri Güvenliği Takviyesi (Migration 14):** `transactions` tablosuna `TRUNCATE` yasağı koyan statement-level trigger (`trg_prevent_transaction_truncate`) ve tenant silindiğinde defterin silinmesini engelleyen `ON DELETE RESTRICT` kuralı başarıyla canlıya uygulandı.
- **Yerel Soğuk Yedek (Offline Backup):** `backend/cmd/backup/main.go` aracı geliştirilerek tek komutla tüm canlı verilerin şifreli/temiz `.json` ve `.sql` dökümünü yerel `backups/` klasörüne indiren sistem kuruldu.

### 7.2. Test Kapsamı ve CI Entegrasyonu
- **Durum:** `backend/cmd/run_yaml_tests/main.go` gibi harika bir uçtan uca senaryo koşucusu yazılmış olsa da bu test GitHub Actions CI pipeline'ına bağlı değildir. Her push işleminde otomatik olarak test container'ı üzerinde koşulmamaktadır.

### 7.3. `seed_donem.bat` Komut Dosyasında Parametre Kontrolü
- **Dosya:** `seed_donem.bat`
- **Durum:** `go run cmd/run_yaml_tests/main.go %1` satırında eğer parametre verilmezse bazı terminal ortamlarında `%1` string olarak Go'ya iletilmektedir. Dosyaya parametre varlık kontrolü eklenmelidir.

---

## 8. Önceliklendirilmiş Çözüm Yol Haritası (Priority Action Matrix)

Sistem canlıya çıkmadan önce tamamlanması önerilen adımlar aciliyet sırasına göre aşağıda listelenmiştir:

```
[ACİL - GÜVENLİK]  ✅ 1. Supabase JWT doğrulama ve Header Spoofing engeli (ÇÖZÜLDÜ - 2026-09-13)
                    ✅ 2. login/page.tsx içindeki bypass kodları tamamen kaldırıldı (ÇÖZÜLDÜ - 2026-09-13)
                    ✅ 3. Next.js middleware @supabase/ssr doğrulamasına geçirildi (ÇÖZÜLDÜ - 2026-09-13)

[ÖNEMLİ - VERİ]    ✅ 4. PostgresUserSecurityRepository canlı havuza bağlandı (ÇÖZÜLDÜ - 2026-09-13)
                    ✅ 5. Transaction ters kaydında (reversal) pgx.Tx ACID güvencesi sağlandı (ÇÖZÜLDÜ - 2026-09-13)
                    ✅ 6. Fail-fast DB pool & sessiz mock fallback temizlendi (ÇÖZÜLDÜ - 2026-09-13)

[OPTİMİZASYON]     ✅ 7. Backend & Frontend sayfalama (Pagination: 50'şer kayıt) eklendi (ÇÖZÜLDÜ - 2026-09-13)
                    ✅ 8. page.tsx custom hook'lara bölündü (usePeriods, useTransactions) (ÇÖZÜLDÜ - 2026-09-13)
                    ✅ 9. messages/tr.json ve en.json sözlükleri %100 senkronize edildi (ÇÖZÜLDÜ - 2026-09-13)

[ALTYAPI & DEVOPS] ✅ 10. Native Excel motoru (.xlsx otomatik sütun/filtre/format) (ÇÖZÜLDÜ - 2026-09-13)
                    ✅ 11. schema_migrations takip tablosu ve akıllı migration motoru (ÇÖZÜLDÜ - 2026-09-13)
                    ✅ 12. TRUNCATE yasağı trigger'ı ve ON DELETE RESTRICT veri zırhı (ÇÖZÜLDÜ - 2026-09-13)
                    ✅ 13. Çevrimdışı yerel yedekleme aracı (JSON & SQL dump) (ÇÖZÜLDÜ - 2026-09-13)
```

---

## 9. 🚀 Canlıya Geçiş ve Aslının Yerine Koyma Adımları (Production Cutover)

1. **Dinamik Tenant Entegrasyonu:** `frontend/src/lib/api.ts` içindeki varsayılan hardcoded UUID'nin giriş yapan kullanıcının gerçek `tenant_id`'sine dinamik bağlanması.
2. **Kullanıcı & Personel Hesapları:** Şirket yetkilileri ve muhasebe personeli için Supabase Auth hesaplarının açılması ve rollerinin (`admin`, `muhasebeci`, `standart`) atanması.
3. **Canlı Sunucuya Yayınlama (Deployment):**
   - Frontend: Vercel / Cloudflare / Render üzerine canlıya alma (SSL/HTTPS).
   - Backend: Docker container olarak bir sunucuya (Render, Railway veya şirket VPS'i) yerleştirme.
4. **Resmi Açılış Bakiyesi (Cutover):** Sisteme geçiş gününün son net kasa bakiyesinin yeni döneme açılış devri olarak işlenmesi.

---
*Bu doküman projenin teknik sağlığını korumak ve güvenli bir üretim ortamı sürümüne (v1.1+) zemin hazırlamak amacıyla hazırlanmıştır.*
