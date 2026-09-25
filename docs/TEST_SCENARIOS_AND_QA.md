# 🧪 TEST_SCENARIOS_AND_QA.md — Test Senaryoları & Kalite Güvence (QA) Rehberi

> **Amaç:** Bu dosya, "Öncü Otogaz Kasa ve Defter-i Kebir Yönetim Platformu" projesinin son kullanıcıya hatasız, kuruşu kuruşuna doğru ve eksiksiz ulaşması için test edilmesi gereken tüm senaryoları canlı olarak listeler (SSOT).
> **Kullanım:** Karşılaşılan her uç durum, kullanıcı geri bildirimi ve geliştirilen her özellik bu listeye yeni bir test senaryosu olarak eklenir.

---

## 📌 Test Durum İmleri
- `[ ]` Henüz test edilmedi / Bekliyor
- `[x]` Başarıyla test edildi ve doğrulandı (PASS)
- `[!]` Hata tespit edildi / İnceleme altında (FAIL)

---

## 1. 📅 Dönem Yönetimi & Yaşam Döngüsü (Period Lifecycle)

### Senaryo 1.1 — Kilitli Dönemde "Yeni Dönem Aç" Butonunun Erişilebilirliği *(Kullanıcı Tespiti)*
- **Önkoşul:** Sistemde açık durumda bir dönem bulunmalı (`2026-08`, `status: open`).
- **İşlem Adımları:**
  1. "Dönemi Kilitle" butonuna basarak mevcut dönemi kapatın ve kilitleyin.
  2. Dönem durumu `locked` olduğunda üst kontrol panelini inceleyin.
- **Beklenen Sonuç (PASS):**
  - "Yeni Dönem Aç" (`tPeriod("openNextPeriod")`) butonu **asla kaybolmamalı**, daima tıklanabilir olmalıdır.
  - "Dönem Kilidini Aç" butonu görünür olmalıdır.
  - Yeni dönem açıldığında (`2026-09`), bir önceki dönemin kapanış bakiyesi otomatik devir bakiyesi olarak atanmalıdır.
- **Durum:** `[x]` (Düzeltildi ve doğrulandı - BUG-260904-21)

---

### Senaryo 1.2 — Kuruşu Kuruşuna Dönem Devir Bakiyesi Doğruluğu
- **Önkoşul:** Açık dönemde başlangıç bakiyesi 10.000,00 TL olsun.
- **İşlem Adımları:**
  1. 2.500,50 TL Gelir kaydı girin.
  2. 1.200,25 TL Gider kaydı girin.
  3. Net Kasa: `10.000,00 + 2.500,50 - 1.200,25 = 11.300,25 TL` olduğunu doğrulayın.
  4. Dönemi kilitleyin ve "Yeni Dönem Aç" butonuna basıp sonraki ayı açın.
- **Beklenen Sonuç (PASS):**
  - Yeni açılan dönemin "Devir / Açılış Bakiyesi" tam olarak `11.300,25 TL` olmalıdır (0.01 TL kuruş farkı dahi olamaz).
- **Durum:** `[ ]`

---

### Senaryo 1.3 — Kilitli Döneme Veri Girişi Engeli (Defense in Depth)
- **Önkoşul:** Dönem kilitli (`status = 'locked'`) olmalıdır.
- **İşlem Adımları:**
  1. Arayüzde "Yeni İşlem" ve "Hızlı Satır Girişi" butonlarının kilitli olduğunu veya gizlendiğini kontrol edin.
  2. API üzerinden doğrudan `POST /api/v1/transactions` isteği atarak kilitli döneme işlem eklemeyi deneyin.
- **Beklenen Sonuç (PASS):**
  - Hem Go backend servisi `422 Unprocessable Entity (Dönem Kilitli)` dönmeli, hem de PostgreSQL trigger'ı `trg_prevent_locked_period_insert` devreye girerek işlemi kesin olarak reddetmelidir.
- **Durum:** `[x]` (9/9 Integration PASS)

---

### Senaryo 1.4 — Kilitli Dönemin Kilidini Yeniden Açma (Unlock)
- **Önkoşul:** Kilitli bir dönem seçili olmalıdır. Kullanıcı `admin` rolüne sahip olmalıdır.
- **İşlem Adımları:**
  1. "Dönem Kilidini Aç" butonuna tıklayın.
- **Beklenen Sonuç (PASS):**
  - Dönem durumu anında `open` olmalı, "Dönemi Kilitle" butonu geri gelmeli ve işlem ekleme alanı tekrar aktifleşmelidir.
- **Durum:** `[ ]`

---

### Senaryo 1.5 — Dönem Devri (Rollover) ve Yeni Döneme Sorunsuz İşlem Ekleme
- **Önkoşul:** Aktif dönem kilitlenmiş olmalıdır.
- **İşlem Adımları:**
  1. "Yeni Dönem Aç" butonuna tıklayıp yeni dönem etiketini (Örn: `2026-09`) onaylayın.
  2. Yeni dönemin başarıyla seçildiğini ve durumunun `Açık` olduğunu doğrulayın.
  3. Yeni döneme bir gelir veya gider işlemi ekleyin.
- **Beklenen Sonuç (PASS):**
  - İstek `422 (Unprocessable Entity)` almamalıdır.
  - Frontend, eski kilitli dönemin ID'sine geri düşmemeli (fallback tuzağı engellenmeli), veritabanında açılan yeni dönemin UUID'sini kullanmalıdır.
  - İşlem yeni döneme kuruşu kuruşuna kaydedilmelidir.
- **Durum:** `[x]` (Düzeltildi ve doğrulandı - BUG-260904-22)

---

## 2. 🔐 Kimlik Doğrulama & Kullanıcı Girişi (Auth & Security)

### Senaryo 2.1 — Yeni Kullanıcı Kaydı & Otomatik İşletme (Tenant) Eşleşmesi
- **Önkoşul:** Supabase bağlantısı aktif olmalıdır.
- **İşlem Adımları:**
  1. Giriş ekranında "Hesabınız yok mu? Yeni Kayıt Olun" bağlantısına tıklayın.
  2. Yeni bir e-posta ve şifre girip "Kayıt Ol" butonuna basın.
  3. Başarılı kayıttan sonra giriş yapın.
- **Beklenen Sonuç (PASS):**
  - Kullanıcı sisteme girdiğinde ekran boş kalmamalı; `on_auth_user_created` trigger'ı sayesinde otomatik olarak `Öncü Otogaz` işletmesine bağlanmalı ve dönem/işlem defterini görebilmelidir.
- **Durum:** `[x]` (Trigger ve UI entegre edildi - BUG-260904-19)

---

### Senaryo 2.2 — Hatalı Şifre / Bulunamayan Hesapta Zarif Hata Gösterimi
- **Önkoşul:** Giriş ekranında olunmalıdır.
- **İşlem Adımları:**
  1. Yanlış bir şifre veya kayıtlı olmayan bir e-posta yazıp "Giriş Yap" butonuna basın.
- **Beklenen Sonuç (PASS):**
  - Konsolda `IntlError` veya sayfa çökmesi yaşanmamalıdır.
  - Ekranda kırmızı şık bir kutu içinde Türkçe: *"E-posta adresi veya şifre hatalı."* uyarısı görünmelidir.
- **Durum:** `[x]` (Düzeltildi ve doğrulandı - BUG-260904-20)

---

### Senaryo 2.3 — Güvenlik Sorusu ile Şifre Sıfırlama
- **Önkoşul:** Kullanıcı daha önce güvenlik sorusu ve cevabını kaydetmiş olmalıdır.
- **İşlem Adımları:**
  1. Giriş ekranında "Şifremi Unuttum" linkine tıklayın.
  2. E-posta adresini girin, gelen güvenlik sorusunu doğru yanıtlayın ve yeni şifreyi belirleyin.
  3. Yeni şifreyle giriş yapmayı deneyin.
- **Beklenen Sonuç (PASS):**
  - Yeni şifre ile sisteme sorunsuz giriş yapılabilmelidir.
- **Durum:** `[ ]`

---

## 3. 💼 İşlem Defteri & Parasal Bütünlük (Ledger Integrity)

### Senaryo 3.1 — Klavye Odaklı Hızlı Satır Girişi (Excel Deneyimi)
- **İşlem Adımları:**
  1. Açık dönemde hızlı giriş barına gelin.
  2. Tutar yazıp `Tab` ile kanala geçin, yön seçip `Enter`'a basın.
- **Beklenen Sonuç (PASS):**
  - Fareye dokunmadan klavyeden ardışık işlemler saniyeler içinde deftere eklenebilmeli ve liste anında güncellenmelidir.
- **Durum:** `[ ]`

---

### Senaryo 3.2 — Çifte Tıklama / Idempotency Koruması
- **İşlem Adımları:**
  1. Hızlıca işlem kaydet butonuna arka arkaya 3-4 kez çift tıklayın (Double-click / Network lag simülasyonu).
- **Beklenen Sonuç (PASS):**
  - Sistemde yalnızca 1 adet kayıt oluşmalı, aynı tutar mükerrer olarak deftere 3-4 kez işlenmemelidir.
- **Durum:** `[x]` (Backend Idempotency-Key Middleware ile korumalı)

---

### Senaryo 3.3 — Append-Only Ters Kayıt (Reversal / İptal)
- **İşlem Adımları:**
  1. Defterdeki hatalı bir gelir işleminin yanındaki "Ters Kayıt (İptal)" butonuna basın.
  2. İptal gerekçesi yazıp onaylayın.
- **Beklenen Sonuç (PASS):**
  - Orijinal kayıt silinmemeli veya güncellenmemelidir (`UPDATE`/`DELETE` yasaktır).
  - Sisteme otomatik olarak aynı tutarda ters yönlü (Gider) bir iptal satırı eklenmeli ve bakiye kendini sıfırlamalıdır.
- **Durum:** `[ ]`

---

### Senaryo 3.4 — İptal Edilmiş İşlemin Tekrar İptal Edilememesi
- **İşlem Adımları:**
  1. Daha önce ters kayıt ile iptal edilmiş bir satırın üzerinde tekrar ters kayıt butonunu arayın veya API'den istek atın.
- **Beklenen Sonuç (PASS):**
  - Buton pasif (disabled) olmalı; API çağrısı `422: TRANSACTION_ALREADY_REVERSED` dönerek işlemi reddetmelidir.
- **Durum:** `[x]` (Backend domain kuralı test edildi)

---

### Senaryo 3.5 — Geçersiz / Negatif / Sıfır Tutar Girişinin Engellenmesi
- **İşlem Adımları:**
  1. Tutar alanına `-500`, `0` veya `abc` yazmayı deneyin.
- **Beklenen Sonuç (PASS):**
  - Hem arayüzde inline validasyon uyarmalı, hem de backend `400: INVALID_AMOUNT` hatası vererek veritabanına kaydetmemelidir.
- **Durum:** `[ ]`

---

## 4. 📊 Excel & CSV İçe / Dışa Aktarım (Import & Export)

### Senaryo 4.1 — Excel Türkçe Karakter Uyumluluğu (UTF-8 BOM)
- **İşlem Adımları:**
  1. İçinde "Öncü Otogaz, Maaş Elden, Yakıt, Çarşı Masrafı" gibi Türkçe karakterler olan işlemleri CSV olarak dışa aktarın.
  2. İndirilen `.csv` dosyasını doğrudan Microsoft Excel ile açın.
- **Beklenen Sonuç (PASS):**
  - Türkçe harfler (`ş, ğ, ı, ö, ü, ç`) bozulmadan, düzgün bir şekilde Excel sütunlarına ayrılmış olarak görünmelidir.
- **Durum:** `[ ]`

---

### Senaryo 4.2 — Hatalı Satırlı CSV Yükleme ve Hata Raporlama
- **İşlem Adımları:**
  1. 3. satırında tutarı boş veya harf olan bozuk bir CSV dosyasını "CSV İçe Aktar" modalından yükleyin.
- **Beklenen Sonuç (PASS):**
  - Sistem tüm dosyayı çöpe atmamalı veya sunucuyu çökertmemelidir.
  - Ekranda: *"3. Satırda geçersiz tutar formatı"* şeklinde spesifik satır hatası gösterilmelidir.
- **Durum:** `[ ]`

---

## 5. 👥 Rol Tabanlı Yetkilendirme (RBAC) & Güvenlik Savunması

### Senaryo 5.1 — Standart Kullanıcı Yetki Kısıtları
- **Önkoşul:** Kullanıcı rolü `standart` olmalıdır.
- **İşlem Adımları:**
  1. Standart kullanıcı hesabı ile giriş yapın.
  2. "Dönemi Kilitle" veya "Üye Yönetimi" alanlarına erişmeyi deneyin.
- **Beklenen Sonuç (PASS):**
  - Standart kullanıcılar şirketin dönemini kilitleyemez veya diğer üyeleri yönetemez. Bu butonlar ya gizlenmeli ya da işlem `403 Forbidden` almalıdır.
- **Durum:** `[ ]`

---

### Senaryo 5.2 — Backend Header Spoofing ve Sahte Rol (`X-User-Role: admin`) Yetki Yükseltme Saldırısı *(Acımasız QA)*
- **Saldırı Vektörü:** Bir saldırganın geçerli bir Supabase JWT token'ı olmadan doğrudan backend API'sine (`/api/v1/periods/`) `X-User-Role: admin` ve rastgele bir `X-User-ID` header'ı enjekte ederek yönetici yetkisi elde etmeye çalışması.
- **İşlem Adımları:**
  1. API'ye geçerli bir Supabase oturum token'ı göndermeyin.
  2. Header'lara `X-User-Role: admin` ve rastgele sahte bir `X-User-ID` ekleyin.
  3. `GET /api/v1/periods/` uç noktasına istek atın.
- **Beklenen Sonuç (PASS):**
  - Backend header'daki role asla körü körüne güvenmemelidir.
  - Veritabanı `tenant_members` tablosunda kullanıcının gerçek üyeliği ve aktifliği doğrulanmalıdır.
  - Üye olmayan veya sahte rol gönderen istek anında `403 Forbidden` / `401 Unauthorized` ile reddedilmelidir.
- **Durum:** `[x]` (Acımasız QA runner ile test edildi — HTTP 403 ile engellendi)

---

### Senaryo 5.3 — Çok Kiracılı İzolasyon ve Tenant Hopping (İşletmeler Arası Sızma) *(Acımasız QA)*
- **Saldırı Vektörü:** Sistemde kayıtlı meşru bir kullanıcının, ait olmadığı başka bir şirketin verilerine erişmek için istek başlığındaki `X-Tenant-ID` değerini farklı bir şirketin UUID'si ile değiştirmesi.
- **İşlem Adımları:**
  1. Gerçek ve meşru bir kullanıcı ID'si belirleyin.
  2. Header'daki `X-Tenant-ID` değerini kullanıcının üye olmadığı yabancı bir tenant UUID'si (`99999999-9999...`) yapın.
  3. `GET /api/v1/periods/` uç noktasına istek gönderin.
- **Beklenen Sonuç (PASS):**
  - Sistem kullanıcının o tenant üzerinde üyeliği olmadığını veritabanından doğrulamalı ve yabancı şirketin verilerini kesinlikle ifşa etmemelidir.
  - İstek `403 Forbidden` dönerek sonlandırılmalıdır.
- **Durum:** `[x]` (Acımasız QA runner ile test edildi — HTTP 403 ile engellendi)

---

## 6. 🛡️ Next.js SSR ve Oturum Doğrulaması (Middleware Defense)

### Senaryo 6.1 — Sahte `defter_session` Çerezi ile Yetkisiz Erişim Engeli (Bypass Önleme) *(Acımasız QA)*
- **Saldırı Vektörü:** Saldırganın tarayıcısına elle `defter_session=1` veya `defter_session=true` şeklinde basit bir sahte çerez ekleyerek kimlik doğrulama duvarını aşmaya çalışması.
- **İşlem Adımları:**
  1. Tarayıcı veya HTTP istemcisinde `Cookie: defter_session=1` başlığını ekleyin.
  2. Doğrudan korumalı panel rotasına (`http://localhost:3000/tr`) `GET` isteği atın.
- **Beklenen Sonuç (PASS):**
  - Next.js Edge Middleware sahte `defter_session` çerezini tamamen görmezden gelmeli; sunucu tarafında `@supabase/ssr` üzerinden kriptografik JWT doğrulaması (`getUser()`) yapmalıdır.
  - Yetkisiz istek anında `HTTP 307 Temporary Redirect` ile `/tr/login` sayfasına postalanmalıdır.
- **Durum:** `[x]` (Acımasız QA runner ile test edildi — HTTP 307 -> /tr/login yönlendirmesi doğrulandı)

---

### Senaryo 6.2 — Bozuk veya İptal Edilmiş Token ile Panel Erişimi Engeli *(Acımasız QA)*
- **Saldırı Vektörü:** Saldırganın imzasız veya kurcalanmış bir Supabase auth çerezi (`sb-access-token=malformed_fake_token`) göndermesi.
- **İşlem Adımları:**
  1. `Cookie: sb-access-token=tampered_xyz; defter_session=tampered` başlığı ile korumalı rotaya istek gönderin.
- **Beklenen Sonuç (PASS):**
  - Supabase SSR SDK'sı imza doğrulamasını başarısız saymalı ve kullanıcıyı `/tr/login` sayfasına yönlendirmelidir.
- **Durum:** `[x]` (Acımasız QA runner ile test edildi — Tampered token engellendi)

---

## 7. 📄 Sayfalama (Pagination) Sınır ve Veri Bütünlüğü

### Senaryo 7.1 — Sayfalar Arası Sıfır Mükerrerlik ve Sıfır Kayıp *(Acımasız QA)*
- **Önkoşul:** Dönemde tam 131 adet gerçek işlem bulunmalıdır (`2026-08` dönemi).
- **İşlem Adımları:**
  1. Sayfa 1: `limit=50, offset=0` ile verileri çekin (50 satır).
  2. Sayfa 2: `limit=50, offset=50` ile verileri çekin (50 satır).
  3. Sayfa 3: `limit=50, offset=100` ile verileri çekin (31 satır).
  4. Çekilen toplam 131 satırın UUID'lerini tekilleştirme kümesine (Set/Map) atın.
- **Beklenen Sonuç (PASS):**
  - Sayfa 1: 50 satır ve `X-Total-Count: 131`.
  - Sayfa 2: 50 satır.
  - Sayfa 3: 31 satır.
  - Toplam birleştirilen tekil kayıt tam olarak 131 olmalıdır.
  - Sayfalar arasında **0 mükerrer (duplicate)** kayıt ve **0 kayıp (dropped)** kayıt olmalıdır.
- **Durum:** `[x]` (PostgreSQL veritabanı üzerinde 131 satırın tamamı tek tek doğrulandı — 0 mükerrer, 0 kayıp)

---

### Senaryo 7.2 — Uç Değer ve Sınır Aşımı (Out-of-Bounds Offset) Güvenliği *(Acımasız QA)*
- **Saldırı Vektörü:** Kullanıcının veya istemcinin toplam kayıt sayısının çok üzerinde bir sayfa (Örn: `offset=5000`) talep etmesi.
- **İşlem Adımları:**
  1. `limit=50, offset=5000` parametreleri ile istek atın.
- **Beklenen Sonuç (PASS):**
  - Backend SQL hatası vermemeli veya panic yaşamamalıdır.
  - Sistem zarif bir şekilde `[]` (boş dizi) ve toplam kayıt sayısını (131) dönmelidir.
- **Durum:** `[x]` (Test edildi — Güvenli boş dizi dönüşü sağlandı)

---

## 8. 🌐 Çoklu Dil (i18n) Bütünlüğü ve Senkronizasyon

### Senaryo 8.1 — Türkçe ve İngilizce Dil Sözlüklerinin %100 Birebir Eşitliği *(Acımasız QA)*
- **Önkoşul:** `frontend/src/messages/tr.json` ve `frontend/src/messages/en.json` dosyaları.
- **İşlem Adımları:**
  1. İki sözlük dosyasındaki tüm iç içe geçmiş anahtarları (nested JSON keys) özyinelemeli (recursive) olarak tarayın.
  2. TR'de olup EN'de olmayan, EN'de olup TR'de olmayan anahtarları tespit edin.
- **Beklenen Sonuç (PASS):**
  - Hiçbir anahtar eksik olmamalı; dil değiştirildiğinde arayüzde eksik metin (fallback / missing key) hatası çıkmamalıdır.
- **Durum:** `[x]` (Test edildi — TR ve EN sözlükleri %100 birebir senkronize)

---

## 9. 🚚 Tedarikçiler & Parçacılar Çift Defter ve Ters Kayıt (Reversal)

### Senaryo 9.1 — Tedarikçi İşlem İptali ve Otomatik Zıt Yönlü Ters Kayıt
- **Önkoşul:** Tedarikçiye ait 15.000,00 TL tutarında bir `purchase` (Alınan Mal) kaydı bulunsun.
- **İşlem Adımları:**
  1. Tabloda işlemin yanındaki `İptal / Ters Kayıt` butonuna tıklayın.
  2. İptal gerekçesi girip onaylayın.
- **Beklenen Sonuç (PASS):**
  - Orijinal kayıt `[İPTAL EDİLDİ]` rozeti almalı ve `reversed_by` alanı güncellenmelidir.
  - Zıt yönlü (aynı tutarda ve `payment` yönünde) yeni bir `[TERS KAYIT]` satırı üretilmelidir.
  - Tedarikçinin cari bakiyesi kuruşu kuruşuna netlenmelidir (`15.000,00 - 15.000,00 = 0,00 TL`).
- **Durum:** `[x]` (Veritabanı transaction ve Go unit/handler testleri PASS)

### Senaryo 9.2 — Ters Kaydın Tekrar İptal Edilmesi Yasağı (Double Reversal Block)
- **Önkoşul:** İptal edilmiş bir tedarikçi işlemi veya `[TERS KAYIT]` satırı seçilsin.
- **İşlem Adımları:**
  1. API üzerinden doğrudan `POST /api/v1/suppliers/transactions/:id/reverse` isteği atın.
- **Beklenen Sonuç (PASS):**
  - Backend `422 Unprocessable Entity` ("bu işlem zaten ters kayıt yapılmış veya kendisi bir ters kayıttır") hatası dönerek işlemi reddetmelidir.
- **Durum:** `[x]` (`TestReverseTransaction_DoubleReversalBlocked` PASS)

---

## 10. 🔐 Supabase JWT Custom Claims & Geliştirici Çıkışı Doğrulaması

### Senaryo 10.1 — Base64 Kodlu HMAC Secret ile Kriptografik İmza Doğrulaması
- **Önkoşul:** `SUPABASE_JWT_SECRET` ortam değişkeni Base64 formatında (`Lw7HcEdLRo+...==`) tanımlanmış olsun.
- **İşlem Adımları:**
  1. Supabase Auth tarafından imzalanmış geçerli bir kullanıcı Bearer token'ı ile korumalı uç noktalara istek atın.
- **Beklenen Sonuç (PASS):**
  - Backend token imzasını Base64 decoded bytes ile doğrulamalı ve `HTTP 200 OK` dönmelidir.
- **Durum:** `[x]` (`TestAuthMiddleware_ValidToken` PASS)

### Senaryo 10.2 — Geliştirici Modunda Token Olmadan Yerel Çalışma (Developer Bypass)
- **Önkoşul:** `ENVIRONMENT=development` ayarlı olsun, istemciden hiçbir auth token gönderilmesin.
- **İşlem Adımları:**
  1. Tarayıcıdan veya `curl` ile `GET /api/v1/periods/` isteği atın.
- **Beklenen Sonuç (PASS):**
  - Backend `403` hatası yerine yerel veritabanındaki aktif işletmeye otomatik `Admin` rolü bağlayarak `HTTP 200 OK` dönmelidir.
- **Durum:** `[x]` (`TestAuthMiddleware_DynamicTenantAutoProvision` PASS)

---

## 11. 🚀 Y-Serisi: Yüksek Hacim, Dayanıklılık ve Operasyonel Canlıya Geçiş Testleri

### Y.1 — Yıl Sonu Veri Hacmi Stres Testi (10.000 İşlem Kaydı & Excel Export)
- **Önkoşul:** Test ortamında 12 aylık, günde ortalama 20-30 işlem varsayımıyla ~10.000 kayıt üretilmiş olması.
- **İşlem Adımları:**
  1. KPI panelini (`GET /api/v1/periods/:id/summary`), işlem tablosunu (`GET /api/v1/periods/:id/transactions`) ve dönem geçmişini bu hacimde açın.
  2. Sayfalama (25 / 50 / 100) ile gezinme hızını ölçün.
  3. 10.000 satırlık Excel export'unu (`GET /api/v1/periods/:id/export/excel`) çalıştırın.
- **Test Ölçüm Sonuçları (PASS):**
  - 10.000 Kayıt Bellek Yükleme: **6.03 ms**
  - KPI Panel Özeti (Summary) Yanıt Süresi: **2.00 ms** (Hedef: <50ms)
  - Sayfalama (Limit 25 / Offset 5000) Yanıt Süresi: **15.69 ms** (Hedef: <30ms)
  - Sayfalama (Limit 50 / Offset 5000) Yanıt Süresi: **12.93 ms**
  - Sayfalama (Limit 100 / Offset 5000) Yanıt Süresi: **13.61 ms**
  - 10.000 Satır Excel Export Üretimi: **159.45 ms** (290 KB akış, Hedef: <5s)
  - Kuruş Hassasiyeti: Bakiye ve toplamlar kuruşu kuruşuna tam doğru.
- **Gecikmeyi (Latency / MS) Daha Da Düşürme Stratejisi (Optimizasyon Yol Haritası):**
  1. **Bileşik İndeks (Composite Index):** `CREATE INDEX idx_transactions_period_created ON transactions(period_id, created_at DESC);`
  2. **Streaming Excel Export:** Veriler bellek tamponu yerine doğrudan HTTP chunked stream olarak istemciye iletilir.
  3. **DB Connection Pool Tuning:** `max_conns=25, min_conns=5, max_conn_idle_time=5m` ile pgx havuzu sıcak tutulur.
- **Durum:** `[x]` (`TestStress_Y1_YearEndVolumeBenchmark` PASS)

---

### Y.4 — Migration Aracının Tekrar Çalıştırılmaya Dayanıklılığı (Idempotency)
- **İşlem Adımları:** `backend/cmd/migrate/main.go` aracını art arda iki kez çalıştırın.
- **Beklenen Sonuç (PASS):** İkinci çalıştırma, `schema_migrations` tablosu sayesinde zaten uygulanmış migration'ları atlamalı; hata vermemeli, mükerrer `ALTER TABLE/CREATE` denemesiyle çökmemelidir.
- **Durum:** `[x]` (Art arda çalıştırma test edildi: Run 1 -> Migration 17 uygulandı; Run 2 -> "Veritabanı tamamen güncel! Çalıştırılacak yeni migration bulunmuyor" PASS)

---

### Y.5 — Yavaş/Kesintili İnternet Bağlantısında İşlem Girişi (Slow 3G & Offline)
- **İşlem Adımları:**
  1. Chrome DevTools -> Network -> "Slow 3G" simülasyonunu açın.
  2. Yeni bir işlem kaydedin, kayıt sırasında bağlantıyı "Offline" moduna alın.
- **Beklenen Sonuç (PASS):** Kullanıcıya net hata/yeniden deneme mesajı gösterilir; çift kayıt oluşmaz (`Idempotency-Key` korur); sayfa donmaz veya sonsuz spinner'da kalmaz.
- **Durum:** `[x]` (Frontend `api.ts` retry & toast bildirimleri + backend idempotency test edildi)

---

### Y.6 — Sayfa Yenileme (F5) ve Tarayıcı Geri Tuşu Sırasında Form Durumu
- **İşlem Adımları:** İşlem/tedarikçi kaydı formunu doldurup gönderin; tamamlanır tamamlanmaz F5'e veya geri tuşuna basın.
- **Beklenen Sonuç (PASS):** Form tekrar gönderilmez (mükerrer kayıt riski sıfır); sayfa tutarlı state'e döner, bozuk/yarım veri görünmez.
- **Durum:** `[x]` (Next.js React state ve React Hook Form reset akışı doğrulandı)

---

### Y.7 — "Kaydet" Butonuna Hızlı Art Arda Çoklu Tıklama (Double-Click Defense)
- **İşlem Adımları:** İşlem kaydederken "Kaydet" butonuna 4-5 kez art arda hızlıca tıklayın.
- **Beklenen Sonuç (PASS):** Frontend butonu tıklandığı ilk anda `disabled` / `isSubmitting` yapar (UX katmanı); arka planda backend `Idempotency-Key` ile aynı isteği tekilleştirir (Defense-in-Depth).
- **Durum:** `[x]` (`TestIdempotencySecurity_DuplicateInterception` & `TransactionDialog.tsx` PASS)

---

### Y.8 — Düşük Teknik Yetkinlikli Kullanıcı / Eski Cihaz Ortamı (UI Erişilebilirlik)
- **İşlem Adımları:** Sistemi eski bir tarayıcı/düşük çözünürlüklü monitör (1366x768) ortamında test edin.
- **Beklenen Sonuç (PASS):** Arayüz kırılmaz, butonlar ve metinler okunaklı kalır, kritik işlevler modern JS polyfill gereksinimi olmadan çalışır.
- **Durum:** `[x]` (Tailwind responsive grid ve SVG ikon fallbacks ile doğrulandı)

---

### Y.9 — Excel Import Boyut Sınırına Yakın Gerçek Dosya (10MB LimitReader)
- **İşlem Adımları:** 10MB sınırına yakın veya üzerinde (örn. 11MB) bir dosya ile import deneyin.
- **Beklenen Sonuç (PASS):** `io.LimitReader` sınırı düzgün çalışır; sınır aşımında sistem çökmeden anlaşılır Türkçe hata mesajı gösterilir.
- **Durum:** `[x]` (`TestOperational_Y9_ExcelImportSizeLimits` PASS)

---

### Y.11 — Rate Limiter Restart Sonrası Davranışı
- **İşlem Adımları:** Sunucu yeniden başlatıldıktan sonra rate limiter durumunu gözlemleyin.
- **Beklenen Sonuç (PASS/RİSK DEĞERLENDİRMESİ):** In-memory rate limiter sunucu restart edildiğinde sayaçları sıfırlar. Tek instance KOBİ mimarisinde bu kabul edilebilir bir davranıştır; dağıtık çoklu sunucuya geçildiğinde Redis tabanlı rate limiting entegre edilecektir.
- **Durum:** `[x]` (Dokümante edildi ve risk profili onaylandı)

---

### Y.12 — Veritabanı Bağlantı Kopmasında Kullanıcı Deneyimi
- **İşlem Adımları:** Backend çalışırken PostgreSQL bağlantısını kesin.
- **Beklenen Sonuç (PASS):** Backend loglara güvenli hata yazar; frontend kullanıcıya beyaz ekran yerine anlaşılır "Sunucuya / Veritabanına ulaşılamıyor, lütfen bağlantınızı kontrol edin" modal/banner'ı sunar.
- **Durum:** `[x]` (Frontend Global Error Boundary & Toast mekanizması doğrulandı)

---

### Y.13 — Eşzamanlı İki Cihazdan Aynı Dönemi Kilitleme/Açma (Race Condition)
- **İşlem Adımları:** İki farklı cihazdan eşzamanlı olarak aynı anda aynı dönemi kilitlemeyi deneyin.
- **Beklenen Sonuç (PASS):** Sadece bir istek başarılı olur (`HTTP 200 OK`), diğeri düzgün çakışma hatası alır (`HTTP 422 Period Already Locked`); dönem durumu yarı-kilitli tutarsız duruma düşmez.
- **Durum:** `[x]` (`TestOperational_Y13_ConcurrentPeriodLockRaceCondition` PASS)

---

### Y.14 — Yanlış Tutar Girişi Sonrası Düzeltme Akışının Kullanılabilirliği (Ters Kayıt)
- **İşlem Adımları:** Sehven 1.000 TL yerine 10.000 TL girin; ardından tek tıkla ters kayıt (iptal) yapıp doğru 1.000 TL'yi kaydedin.
- **Beklenen Sonuç (PASS):** 10.000 TL ters kayıt ile netleşir (0 TL net etki); 1.000 TL doğru kayıt ile nihai bakiye kuruşu kuruşuna 1.000 TL olur.
- **Durum:** `[x]` (`TestOperational_Y14_CorrectionFlowAndReversal` PASS)

---

### Y.15 — Doğrudan SQL ile Silme/Truncate Denemesi (Trigger Koruması)
- **İşlem Adımları:** Doğrudan veritabanı bağlantısı ile `TRUNCATE transactions;` ve `DELETE FROM transactions WHERE ...;` komutlarını çalıştırın.
- **Beklenen Sonuç (PASS):** Migration 07 ve Migration 14'teki veritabanı trigger'ları (`trg_prevent_transaction_truncate` ve `trg_prevent_transaction_mutation`) her iki denemeyi de SQL seviyesinde engelleyerek veri kaybını önler.
- **Durum:** `[x]` (Veritabanı trigger testleri ile doğrulandı)

---

## 12. Gerçek Kullanıcı Excel Dosyaları Doğrulama Testi (Real-World Excel Test)

### 12.1 — Dosya 1: `defter-2026-08-aktif-kayitlar (1).xlsx`
- **Dosya Boyutu:** 11 KB
- **Sayfalar:** `[İşlem Defteri]` (153 satır)
- **Başlık Yapısı:** `[Tarih, Yön, Kanal, Tutar (TL), Açıklama, Durum]`
- **Sonuç (PASS):** Tüm işlem satırları, tarih formatları (`YYYY-MM-DD HH:mm`), yönler (Gelir/Gider), kanallar (EFT/Havale, POS, Elden Nakit, Yemek Masrafı) ve kuruşlu tutarlar sıfır veri kaybıyla başarıyla parse edildi.

### 12.2 — Dosya 2: `KASA DEFTERİM 2026.xlsx`
- **Dosya Boyutu:** 370 KB
- **Sayfalar:** 36 Sayfa (Aylık Kasa sayfaları: `AĞUSTOS26`, `OCAK26`, `ŞUBAT26`, `MART26`, `MAYIS26` vb. + Tedarikçi Takip Tabloları)
- **Yapılan Test:**
  - `AĞUSTOS26` sayfası import edildi: 41 işlem, 4 Tedarikçi (`ATİKER`, `PRİNS`, `ASİL GRUP`, `PARÇACI UĞUR ABİ`).
  - Toplam Alınan Mal: **1.259.703,58 ₺**
  - Toplam Yapılan Ödeme: **1.626.451,79 ₺**
  - Net Cari Bakiye: **-366.748,21 ₺** (kuruşu kuruşuna tam mutabakat)
  - `OCAK26` (71 satır), `ŞUBAT26` (69 satır), `MART26` (54 satır), `MAYIS26` (61 satır) sayfaları çoklu sayfa tarama algoritmasıyla sıfır hata ile test edildi.
- **Durum:** `[x]` (`backend/internal/service/user_real_excel_test.go` %100 PASS)

---

## 📝 Not Defteri ve Gelecek Eklemeler
*Kullanıcı olarak uygulamayı test ettikçe aklınıza gelen tüm senaryoları, şüpheli durumları veya özel müşteri isteklerini buraya madde madde ekleyeceğiz.*



