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

## 5. 👥 Rol Tabanlı Yetkilendirme (RBAC)

### Senaryo 5.1 — Standart Kullanıcı Yetki Kısıtları
- **Önkoşul:** Kullanıcı rolü `standart` olmalıdır.
- **İşlem Adımları:**
  1. Standart kullanıcı hesabı ile giriş yapın.
  2. "Dönemi Kilitle" veya "Üye Yönetimi" alanlarına erişmeyi deneyin.
- **Beklenen Sonuç (PASS):**
  - Standart kullanıcılar şirketin dönemini kilitleyemez veya diğer üyeleri yönetemez. Bu butonlar ya gizlenmeli ya da işlem `403 Forbidden` almalıdır.
- **Durum:** `[ ]`

---

## 📝 Not Defteri ve Gelecek Eklemeler
*Kullanıcı olarak uygulamayı test ettikçe aklınıza gelen tüm senaryoları, şüpheli durumları veya özel müşteri isteklerini buraya madde madde ekleyeceğiz.*
