# 🚨 Telegram Alarm Sistemi & ☁️ Google Drive Yedekleme Rehberi

**Proje:** Öncü Otogaz — Kasa, Cari ve Defter-i Kebir Yönetim Platformu  
**Tarih:** 2026-09-25  
**Kapsam:** Go Fiber v2 Backend, Error Handler Middleware, Google Drive API v3, Cron Scheduler  

---

## 📌 İçindekiler
1. [Sistemin Genel Mimarisi](#1-sistemin-genel-mimarisi)
2. [Kullanıcının (Sizin) Yapacağı Adımlar](#2-kullanıcının-sizin-yapacağı-adımlar)
   - [A. Telegram Botu & Chat ID Alma](#a-telegram-botu--chat-id-alma)
   - [B. Google Drive API & Servis Hesabı Kurulumu](#b-google-drive-api--servis-hesabı-kurulumu)
3. [Geliştiricinin (Ajanın) Kodlayacağı Katmanlar](#3-geliştiricinin-ajanın-kodlayacağı-katmanlar)
   - [Faz 1: Telegram Alarm Motoru (Sprint 13)](#faz-1-telegram-alarm-motoru-sprint-13)
   - [Faz 2: Google Drive Günlük Yedekleme Motoru (Sprint 14)](#faz-2-google-drive-günlük-yedekleme-motoru-sprint-14)
4. [Çevre Değişkenleri (.env) Matrisi](#4-çevre-değişkenleri-env-matrisi)
5. [Test ve Doğrulama Adımları](#5-test-ve-doğrulama-adımları)

---

## 1. Sistemin Genel Mimarisi

```mermaid
flowchart TD
    subgraph Go Fiber API Core
        API[HTTP Request] -->|Panic / 500 Hatası| RecoverMid[Recover & Error Middleware]
        Cron[Gece 03:00 Cron Scheduler] -->|Tetikle| BackupWorker[Backup Engine]
    end

    subgraph Telegram Bildirim Servisi (Non-Blocking Goroutine)
        RecoverMid -->|Kritik Hata Özeti| TgService[pkg/telegram Service]
        BackupWorker -->|Yedekleme Durum Raporu| TgService
        TgService -->|HTTPS POST| TgAPI[Telegram Bot API]
        TgAPI -->|Anlık Alarm| AdminPhone[Yönetici Telegram Hesabı]
    end

    subgraph Google Drive Yedekleme Deposu
        BackupWorker -->|PostgreSQL Dump & Gzip| LocalDump[Geçici .sql.gz / Snapshot]
        LocalDump -->|Drive API v3 OAuth2| GDrive[Google Drive / Oncu_Otogaz_Backups]
        GDrive -->|Eski Dosyaları Temizle| Retention[30 Günlük Retention Policy]
    end
```

---

## 2. Kullanıcının (Sizin) Yapacağı Adımlar

### A. Telegram Botu & Chat ID Alma (2 Dakika)
1. **Telegram'ı açın** ve arama çubuğuna `@BotFather` yazın (mavi onay rozetli resmi bot).
2. BotFather'a `/newbot` komutunu gönderin.
3. Bot için bir isim yazın (Örn: `Öncü Otogaz Alarm`).
4. Bot için sonu `bot` ile biten bir kullanıcı adı yazın (Örn: `oncu_otogaz_alarm_bot`).
5. BotFather size bir **HTTP API Token** verecektir (Örn: `7891234567:AAHfdk_abcdef123456789`). Bu anahtarı kopyalayın.
6. Kendi oluşturduğunuz bota gidin ve **`/start`** butonuna basın (Mesaj atabilmesi için başlatmanız şarttır).
7. Kendi **Telegram Chat ID**'nizi öğrenmek için Telegram'da `@userinfobot` botuna `/start` yazın veya [Telegram Get Updates URL](https://api.telegram.org/bot<TOKEN>/getUpdates) adresine tarayıcıdan girip `"chat":{"id": 123456789}` değerini alın.
8. Bu iki değeri `backend/.env` dosyasına ekleyin:
   ```env
   TELEGRAM_BOT_TOKEN=7891234567:AAHfdk_abcdef123456789
   TELEGRAM_CHAT_ID=123456789
   TELEGRAM_ALERTS_ENABLED=true
   ```

---

### B. Google Drive API & Servis Hesabı Kurulumu (3 Dakika)
1. [Google Cloud Console](https://console.cloud.google.com/)'a gidin ve yeni bir proje oluşturun (Örn: `Oncu-Otogaz-Finance`).
2. **APIs & Services > Library** sayfasına girip **Google Drive API**'yi aratın ve **Enable (Etkinleştir)** butonuna basın.
3. **IAM & Admin > Service Accounts** sayfasına gidin, **Create Service Account** deyin (Örn: `drive-backup-sa`).
4. Oluşturulan servis hesabının üstüne tıklayıp **Keys > Add Key > Create New Key > JSON** seçerek `credentials.json` anahtar dosyasını bilgisayarınıza indirin.
5. Bu indirilen JSON dosyasını `backend/credentials.json` olarak proje içine kaydedin (Bu dosya `.gitignore` içinde korunur, repoya sızmaz).
6. Kişisel Google Drive'ınızı açın, `Oncu_Otogaz_Backups` adında yeni bir klasör oluşturun.
7. Klasöre sağ tıklayıp **Paylaş (Share)** deyin ve Servis Hesabının e-posta adresini (Örn: `drive-backup-sa@oncu-otogaz-finance.iam.gserviceaccount.com`) **Düzenleyen (Editor)** olarak ekleyin.
8. Klasörün tarayıcıdaki linkinden `folders/1AbC-xYz...` kısmındaki **Folder ID**'yi alıp `backend/.env` içine yazın:
   ```env
   GDRIVE_BACKUP_FOLDER_ID=1AbC-xYz123456789ABC
   GDRIVE_CREDENTIALS_FILE=credentials.json
   GDRIVE_BACKUP_ENABLED=true
   GDRIVE_RETENTION_DAYS=30
   ```

---

## 3. Geliştiricinin (Ajanın) Kodlayacağı Katmanlar

### Faz 1: Telegram Alarm Motoru (Sprint 13)
- [x] `backend/pkg/telegram/client.go`: Standart Go HTTP kütüphanesiyle sıfır harici bağımlılıklı, yüksek performanslı Telegram API istemcisi. (TAMAMLANDI)
- [x] `backend/pkg/telegram/global.go`: Formatlı HTML/Markdown mesaj şablonları, non-blocking `goroutine` kuyruğu ve spam önleyici **Rate Limiter / Alert Throttling** (aynı hatayı dakikada 1 kez gönderir). (TAMAMLANDI)
- [x] `backend/internal/handler/router.go` (recover): Sunucu çökmesini önleyen ve yakalanan Panic'leri anında Telegram'a bildiren mekanizma. (TAMAMLANDI)
- [x] Fiber merkezi `CustomErrorHandler` entegrasyonu: Sadece 500+ Internal Server Error ve DB kopmalarında otomatik alarm tetikleme. (TAMAMLANDI)
- [x] `backend/cmd/test_alert/main.go`: Terminalden tek komutla test alarmı gönderme aracı. (TAMAMLANDI)

### Faz 2: Google Drive Günlük Yedekleme Motoru (Sprint 14)
- [x] `backend/pkg/backup/exporter.go`: Veritabanından tüm aktif/kilitli dönemler, kasa işlemleri, tedarikçiler ve cari hareketleri atomik snapshot halinde çıkaran ve `.sql.gz` sıkıştıran motor. (TAMAMLANDI)
- [x] `backend/pkg/gdrive/uploader.go`: Native Google Drive v3 REST API ile Google Drive'a güvenli akış (streaming upload) yükleyicisi. (TAMAMLANDI)
- [x] `backend/pkg/gdrive/uploader.go` (retention): Belirlenen gün sayısından (30 gün) eski yedekleri tespit edip Drive'dan silen temizlik politikası. (TAMAMLANDI)
- [x] `backend/pkg/scheduler/scheduler.go`: Her gece 03:00'te otomatik çalışacak zamanlayıcı. (TAMAMLANDI)
- [x] `backend/pkg/telegram/document.go`: Yedekleme sonucunu ve `.sql.gz` dosyasını otomatik Telegram'a ileten raporlama entegrasyonu. (TAMAMLANDI)
- [x] `backend/cmd/backup/main.go`: İstenildiği an terminalden manuel yedekleme başlatan CLI aracı. (TAMAMLANDI)

---

## 4. Çevre Değişkenleri (.env) Matrisi

```env
# ==============================================================================
# TELEGRAM NOTIFICATION & ALARM SETTINGS
# ==============================================================================
TELEGRAM_ALERTS_ENABLED=true
TELEGRAM_BOT_TOKEN=
TELEGRAM_CHAT_ID=

# ==============================================================================
# GOOGLE DRIVE AUTOMATED BACKUP SETTINGS
# ==============================================================================
GDRIVE_BACKUP_ENABLED=false
GDRIVE_CREDENTIALS_FILE=credentials.json
GDRIVE_BACKUP_FOLDER_ID=
GDRIVE_BACKUP_CRON="0 3 * * *"
GDRIVE_RETENTION_DAYS=30
```

---

## 5. Test ve Doğrulama Adımları

1. **Telegram Test:**
   ```bash
   cd backend
   go run cmd/test_alert/main.go
   ```
   *Beklenen Sonuç:* Telegram'a formatlı `🚨 Öncü Otogaz Test Alarmı` mesajının 1 saniye içinde düşmesi.
2. **Panic & 500 Alarm Test:**
   * Backend'de `/api/v1/test/panic` simülasyonu çağrılır; sunucu çökmez, istemciye generic 500 döner, Telegram'a tam stack trace özeti gelir.
3. **Google Drive Yedekleme Test:**
   ```bash
   cd backend
   go run cmd/backup/main.go
   ```
   *Beklenen Sonuç:* Veritabanı yedeğinin `.sql.gz` olarak sıkıştırılması, Drive klasörüne yüklenmesi ve Telegram'a "✅ Günlük Yedekleme Başarılı" özetinin iletilmesi.
