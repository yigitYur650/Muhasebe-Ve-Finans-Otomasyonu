# PROJECT_BRIEF.md — Kasa ve Defter-i Kebir Yönetim Platformu

> **Durum:** Taslak — henüz sprint başlamadı.
> **Bu dosya SSOT'tur.** Ajan her session başında bunu okur, kararları burada değişmeden uygulamaz.

---

## 1. Problem ve Hedef

Müşteri, işletme nakit döngüsünü (banka, POS, elden nakit, giderler) aylık Excel sayfalarında manuel takip ediyor. İki somut acı noktası var:

1. **Devir kırılganlığı:** Ay başında bir önceki ayın net bakiyesi elle yeni sayfaya taşınıyor; geçmiş bir ay düzeltildiğinde sonraki tüm ayların bakiyesi bozuluyor.
2. **Merkezi görünürlük yok:** Gelen/giden kalemler (EFT, POS, elden nakit, kredi / kira, maaş, kredi kartı, kartuş, yemek, yakıt) tek bir yerde anlık izlenemiyor.

Hedef: Excel'in hızını (klavye ile seri satır girişi) koruyan, ama devir hesaplamasını otomatik ve hatasız yapan, verisi güvenli bir veritabanında duran web tabanlı bir sistem.

**Hedef kullanıcı:** Tek işletme (veya birkaç şube), düşük teknik yetkinlikli günlük kullanıcı + admin/muhasebeci rolü.

---

## 2. Tech Stack (KİLİTLİ — burada olmayan kütüphane önerilmez/kurulmaz)

| Katman | Teknoloji |
|---|---|
| Frontend | **Next.js 15** (App Router) |
| UI | **shadcn/ui** + Tailwind, **TanStack Table** (klavye odaklı hızlı veri girişi) |
| Backend | **Go** (Fiber v2) — Clean Architecture: Domain → Repository → Service → Handler |
| Para/Ondalık | **shopspring/decimal** (Go tarafı) — float YASAK |
| Veritabanı | **PostgreSQL** (Supabase) — `NUMERIC(15,2)` bakiye alanları, RLS |
| Idempotency | `Idempotency-Key` header + backend'de tekilleştirme tablosu |
| i18n | JSON dil dosyaları — hardcoded metin yasak |
| Export/Import | Excel (xlsx) ve PDF içe/dışa aktarım modülü |

---

## 3. AÇIK KARAR — Event Sourcing mi, Append-Only Ledger mi?

Orijinal spec hem "Event Sourcing" hem "double-entry" hem de "append-only işlem tablosu" ifadelerini birlikte kullanıyor. Bunlar üç ayrı mimari karar ve karmaşıklıkları çok farklı:

- **Tam event sourcing:** State hiç saklanmaz, her an event log'dan yeniden türetilir (replay, snapshot, event versioning gerekir). Bu proje ölçeğine göre fazla ağır.
- **Double-entry:** Her işlem debit/credit çifti olarak kaydedilir (klasik muhasebe defteri). Denetlenebilirlik için güçlü ama kullanıcı arayüzünde "tek satır gelir/gider girişi" beklentisiyle ekstra çeviri katmanı gerektirir.
- **Append-only ledger + period lock (varsayılan öneri):** İşlemler asla UPDATE/DELETE edilmez, sadece eklenir veya "ters kayıt" (reversal) ile düzeltilir; her dönem kapanışta bakiye snapshot'ı alınır ve sonraki dönem bu snapshot'tan başlar. Devir kırılganlığı sorununu doğrudan çözer, double-entry'nin denetim gücünü büyük ölçüde korur, ama tam muhasebe defteri karmaşıklığına girmez.

**Bu brief, üçüncü seçeneği (append-only + period snapshot) varsayılan olarak alır.** Onaylanmadıysa Sprint 1 başlamadan netleştirilmeli — şema tasarımı buna göre dallanıyor.

---

## 4. Dönem (Period) Motoru — Çekirdek Kural

- Her ay bağımsız bir `periods` kaydı: `id, tenant_id, label (ör. "2025-05"), starting_balance, status (open/locked), opened_at, locked_at`.
- Yeni dönem açıldığında `starting_balance`, bir önceki dönemin **kapanış anındaki hesaplanmış bakiyesi** olarak otomatik kopyalanır — elle girilmez.
- Geçmişte düzeltme yapılırsa: kilitli (locked) dönemde doğrudan UPDATE yasak; düzeltme yeni bir "reversal + correction" işlem çifti olarak açık dönemde veya belirlenmiş bir düzeltme akışıyla girilir. Kapanmış dönemin `starting_balance`'ı asla geriye dönük otomatik olarak yeniden hesaplanmaz — bu, "bir düzeltme tüm sonraki ayları bozuyor" sorununun kök nedeniydi ve bilinçli olarak burada kesiliyor.
- `period_lock` sonrası o döneme yazma denemesi backend'de 403 döner (RLS + service layer çift kontrol).

---

## 5. Veri Modeli İlkeleri

- Tüm parasal alanlar `NUMERIC(15,2)` — asla `FLOAT`/`REAL` değil.
- `transactions` tablosu append-only: `id, tenant_id, period_id, direction (in/out), channel (eft/pos/nakit/kredi/kira/maaş/...), amount, description, created_by, created_at, reversed_by (nullable, self-reference)`.
- Silme yok — sadece `reversed_by` ile ters kayıt zinciri (audit trail korunur).
- Her mutasyon `created_by` ile loglanır (kim, ne zaman).
- Çoklu kiracı (tenant) izolasyonu: her tabloda `tenant_id` + RLS policy — Kap-App denetim raporundaki SEC-010 hatası (herkese `USING(true)`) burada tekrarlanmayacak, politikalar `tenant_id = current_tenant()` ile daraltılacak.

---

## 6. Güvenlik Kuralları (Kap-App denetiminden çıkarılan dersler)

Kap-App güvenlik raporunda görülen ve bu projede baştan engellenecek hatalar:

- `.env` asla repoya girmez, ilk commit'ten önce `.gitignore` doğrulanır.
- Service-role / admin anahtarlar sadece backend env'inde, frontend'e asla sızmaz, console'a asla loglanmaz.
- CORS production'da wildcard (`*`) değil, açık origin listesi.
- Hiçbir kullanıcı/hesap için hardcoded auth istisnası yok (Kap-App'teki `halil@gmail.com` 2FA muafiyeti gibi bir pattern yasak).
- RLS politikaları `USING(true)` gibi genel geçirgen yazılmaz; her policy tenant/role bazlı daraltılır.
- Hata mesajlarında iç detay (stack trace, DB hatası) istemciye dönmez — generic mesaj + sunucu tarafı log.
- Kapalı (locked) döneme yazma denemesi hem RLS hem service layer'da engellenir (defense in depth).

---

## 7. Kodlama Kuralları (Hard Rules — asla ihlal edilmez)

- Monolitik dosya yasak: Go'da Domain → Repository → Service → Handler ayrımı; frontend'de ayrık bileşen ağacı.
- UI sadece shadcn/ui + Tailwind; serbest CSS/başka kütüphane yok.
- Hardcoded metin yasak — her kullanıcıya görünen string i18n JSON'dan gelir.
- `.env` dışına gizli veri yazılmaz; sunucu tarafı yetkilendirme zorunlu.
- Gelişigüzel `console.log` / `fmt.Println` yasak — merkezi yapılandırılmış loglama.
- Stack burada listelenmeyen bir kütüphane eklenmeden önce insana sorulur.

---

## 8. Klasör Yapısı (öngörülen — sprint ilerledikçe netleşir)

```
/backend
  /internal
    /domain
    /repository
    /service
    /handler
    /middleware
  /cmd/server
/frontend
  /app
  /components
  /lib
  /i18n
/supabase
  /migrations
/docs
  PROJECT_BRIEF.md
  TASK.md
  BUG_AND_FIX.md
  PROJECT_MAP_FOR_LLM.md
  SECURITY_AUDIT_REPORT.md
antigravityrules (veya .cursorrules)
RELEASE_NOTES.md
```

---

## 9. Test ve Doğrulama Politikası

- Her mikro-prompt tamamlandığında: SQL integrity/RLS snapshot testi + Go unit/edge-case testi (negatif tutar, geçersiz tarih, yetkisiz kullanıcı, kilitli döneme yazma denemesi) + ilgiliyse Playwright E2E.
- Testler geçmeden `TASK.md`'de `[x]` işaretlenmez, bir sonraki mikro-prompta geçilmez.
- Dönem devri ve tenant izolasyonu için özel E2E senaryoları zorunlu (bkz. TASK.md Sprint 6).

---

## 10. Session Özet Bloğu (Ajan her session sonunda bunu üretir)

```
### Session Özeti
- Tamamlanan görev: [TASK.md'deki madde]
- Değiştirilen dosyalar: [liste]
- Çalıştırılan testler ve sonuç: [liste]
- BUG_AND_FIX.md'ye eklenen kayıt (varsa): [başlık]
- Bir sonraki adım: [TASK.md'deki bir sonraki madde]
- Açık riskler / insana sorulması gerekenler: [varsa]
```
