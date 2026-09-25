package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"deftersystem/backend/internal/repository"
)

const dbURL = "postgres://postgres.lvsngrrdzjhbawhcuzqz:nptn0P5vEbyLm6iM@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require"

type PeriodInfo struct {
	ID         string
	TenantID   string
	TenantName string
	Label      string
	StartBal   float64
	Status     string
	OpenedAt   time.Time
	LockedAt   *time.Time
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("==========================================================================")
	fmt.Println("🚀 SUPABASE VERİTABANI TAM SAHA DENETİM VE İNCELEME")
	fmt.Println("==========================================================================")

	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil {
		log.Fatalf("❌ Supabase bağlantı hatası: %v", err)
	}
	defer pool.Close()

	fmt.Println("✅ Supabase veritabanına başarıyla bağlanıldı!")

	// 1. Tablo Satır Sayıları
	tables := []string{
		"public.tenants",
		"public.tenant_members",
		"public.periods",
		"public.transactions",
		"public.idempotency_keys",
		"public.user_security",
		"auth.users",
	}

	fmt.Println("📊 1. TABLO KAYIT SAYILARI:")
	fmt.Println("--------------------------------------------------------------------------")
	for _, t := range tables {
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", t)
		err := pool.QueryRow(ctx, query).Scan(&count)
		if err != nil {
			fmt.Printf("   ⚠️ %-25s : Hata (%v)\n", t, err)
		} else {
			fmt.Printf("   📌 %-25s : %d kayıt\n", t, count)
		}
	}

	// 2. Tenants Listesi
	fmt.Println("\n🏢 2. İŞLETMELER (TENANTS):")
	fmt.Println("--------------------------------------------------------------------------")
	tenantRows, err := pool.Query(ctx, "SELECT id::text, name, created_at FROM public.tenants ORDER BY created_at ASC")
	if err == nil {
		type TItem struct {
			id, name string
			cAt      time.Time
		}
		var tlist []TItem
		for tenantRows.Next() {
			var ti TItem
			_ = tenantRows.Scan(&ti.id, &ti.name, &ti.cAt)
			tlist = append(tlist, ti)
		}
		tenantRows.Close()
		for _, ti := range tlist {
			fmt.Printf("   • ID: %s | Ad: %-25s | Oluşturma: %s\n", ti.id, ti.name, ti.cAt.Format("2006-01-02 15:04:05"))
		}
	}

	// 3. Kullanıcılar & Üyelikler
	fmt.Println("\n👥 3. KULLANICI ÜYELİKLERİ & ROLLER (TENANT MEMBERS):")
	fmt.Println("--------------------------------------------------------------------------")
	memberRows, err := pool.Query(ctx, `
		SELECT tm.tenant_id::text, t.name, tm.user_id::text, tm.role, tm.created_at, COALESCE(u.email, 'Bilinmiyor')
		FROM public.tenant_members tm
		LEFT JOIN public.tenants t ON tm.tenant_id = t.id
		LEFT JOIN auth.users u ON tm.user_id = u.id
		ORDER BY tm.created_at ASC
	`)
	if err == nil {
		type MItem struct {
			tID, tName, uID, role, email string
			cAt                          time.Time
		}
		var mlist []MItem
		for memberRows.Next() {
			var mi MItem
			_ = memberRows.Scan(&mi.tID, &mi.tName, &mi.uID, &mi.role, &mi.cAt, &mi.email)
			mlist = append(mlist, mi)
		}
		memberRows.Close()
		for _, mi := range mlist {
			fmt.Printf("   • Kullanıcı: %-30s | Rol: %-10s | İşletme: %-25s\n     └─ User ID: %s\n", mi.email, mi.role, mi.tName, mi.uID)
		}
	}

	// 4. Dönemler (Periods) ve Bakiye Kontrolü
	fmt.Println("\n📅 4. DÖNEMLER & BAKİYE / İŞLEM ÖZETLERİ (PERIODS):")
	fmt.Println("--------------------------------------------------------------------------")
	periodRows, err := pool.Query(ctx, `
		SELECT p.id::text, p.tenant_id::text, t.name, p.label, p.starting_balance, p.status, p.opened_at, p.locked_at
		FROM public.periods p
		LEFT JOIN public.tenants t ON p.tenant_id = t.id
		ORDER BY p.opened_at ASC
	`)
	var plist []PeriodInfo
	if err == nil {
		for periodRows.Next() {
			var p PeriodInfo
			_ = periodRows.Scan(&p.ID, &p.TenantID, &p.TenantName, &p.Label, &p.StartBal, &p.Status, &p.OpenedAt, &p.LockedAt)
			plist = append(plist, p)
		}
		periodRows.Close()

		for _, p := range plist {
			var inSum, outSum float64
			var txCount int64
			_ = pool.QueryRow(ctx, `
				SELECT 
					COALESCE(SUM(CASE WHEN direction = 'in' THEN amount ELSE 0 END), 0),
					COALESCE(SUM(CASE WHEN direction = 'out' THEN amount ELSE 0 END), 0),
					COUNT(*)
				FROM public.transactions
				WHERE period_id = $1
			`, p.ID).Scan(&inSum, &outSum, &txCount)

			closingBal := p.StartBal + inSum - outSum
			lockInfo := "Açık (İşlem Yapılabilir)"
			if p.Status == "locked" && p.LockedAt != nil {
				lockInfo = fmt.Sprintf("Kilitli (%s)", p.LockedAt.Format("2006-01-02 15:04"))
			}

			fmt.Printf("   📁 Dönem: %s [%s] | Durum: %s\n", p.Label, p.TenantName, lockInfo)
			fmt.Printf("      - Period ID     : %s\n", p.ID)
			fmt.Printf("      - Devir/Başlangıç: %.2f TL\n", p.StartBal)
			fmt.Printf("      - Toplam Giriş  : +%.2f TL\n", inSum)
			fmt.Printf("      - Toplam Çıkış  : -%.2f TL\n", outSum)
			fmt.Printf("      - Kapanış/Mevcut: %.2f TL (Toplam %d İşlem)\n\n", closingBal, txCount)
		}
	}

	// 5. İşlemler Detaylı Analizi (Transactions)
	fmt.Println("💳 5. TÜM İŞLEMLERDEN ÖZET VE KANAL DAĞILIMI:")
	fmt.Println("--------------------------------------------------------------------------")
	var totalIn, totalOut float64
	var totalTxCount int64
	_ = pool.QueryRow(ctx, `
		SELECT 
			COALESCE(SUM(CASE WHEN direction = 'in' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN direction = 'out' THEN amount ELSE 0 END), 0),
			COUNT(*)
		FROM public.transactions
	`).Scan(&totalIn, &totalOut, &totalTxCount)

	fmt.Printf("   🌟 Genel Toplam İşlem Sayısı : %d\n", totalTxCount)
	fmt.Printf("   🟢 Toplam Gelir Girişi       : +%.2f TL\n", totalIn)
	fmt.Printf("   🔴 Toplam Gider Çıkışı       : -%.2f TL\n", totalOut)
	fmt.Printf("   💰 Net Kasa/Banka Bakiyesi   : %.2f TL\n\n", totalIn-totalOut)

	// Kanal bazlı dağılım
	fmt.Println("   📊 Kanal Bazlı Dağılım:")
	channelRows, err := pool.Query(ctx, `
		SELECT channel, direction, COUNT(*), SUM(amount)
		FROM public.transactions
		GROUP BY channel, direction
		ORDER BY direction DESC, count DESC
	`)
	if err == nil {
		type CItem struct {
			channel, direction string
			count              int64
			sum                float64
		}
		var clist []CItem
		for channelRows.Next() {
			var c CItem
			_ = channelRows.Scan(&c.channel, &c.direction, &c.count, &c.sum)
			clist = append(clist, c)
		}
		channelRows.Close()

		for _, c := range clist {
			fmt.Printf("      • %-12s [%-3s]: %3d adet | Toplam: %12.2f TL\n", c.channel, c.direction, c.count, c.sum)
		}
	}

	// 6. Veri Kaybı & Bütünlük Kontrolleri (Data Loss / Anomaly Checks)
	fmt.Println("\n🔍 6. VERİ KAYBI VE BÜTÜNLÜK KONTROLÜ (INTEGRITY / LOSS AUDIT):")
	fmt.Println("--------------------------------------------------------------------------")

	// Kontrol A: Sahipsiz İşlemler
	var orphanTxCount int64
	_ = pool.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM public.transactions t
		LEFT JOIN public.periods p ON t.period_id = p.id
		WHERE p.id IS NULL
	`).Scan(&orphanTxCount)
	if orphanTxCount > 0 {
		fmt.Printf("   ❌ KRİTİK: %d adet dönemsiz/sahipsiz (orphan) işlem bulundu!\n", orphanTxCount)
	} else {
		fmt.Println("   ✅ Sahipsiz İşlem Yok: Tüm işlemler geçerli bir döneme ve işletmeye bağlı.")
	}

	// Kontrol B: Tenant Uyumsuzluğu
	var mismatchTenantCount int64
	_ = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM public.transactions t
		JOIN public.periods p ON t.period_id = p.id
		WHERE t.tenant_id != p.tenant_id
	`).Scan(&mismatchTenantCount)
	if mismatchTenantCount > 0 {
		fmt.Printf("   ❌ KRİTİK: %d adet tenant uyumsuzluğu olan işlem var!\n", mismatchTenantCount)
	} else {
		fmt.Println("   ✅ Tenant Tutarlılığı: Tüm işlemler ait olduğu dönemin tenant'ı ile %100 eşleşiyor.")
	}

	// Kontrol C: Ters Kayıt (Reversed) Bütünlüğü
	var invalidReversals int64
	_ = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM public.transactions t
		WHERE t.reversed_by IS NOT NULL 
		  AND NOT EXISTS (SELECT 1 FROM public.transactions r WHERE r.id = t.reversed_by)
	`).Scan(&invalidReversals)
	if invalidReversals > 0 {
		fmt.Printf("   ❌ KRİTİK: %d adet geçersiz reversed_by referansı var!\n", invalidReversals)
	} else {
		fmt.Println("   ✅ Ters Kayıt Bütünlüğü: Tüm iptal/düzeltme referansları tutarlı.")
	}

	// Kontrol D: Eksi veya Sıfır Tutar Kontrolü
	var invalidAmounts int64
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM public.transactions WHERE amount <= 0`).Scan(&invalidAmounts)
	if invalidAmounts > 0 {
		fmt.Printf("   ❌ KRİTİK: %d adet sıfır veya negatif tutarlı işlem var!\n", invalidAmounts)
	} else {
		fmt.Println("   ✅ Tutar Tutarlılığı: Sıfır veya eksi tutarlı hatalı kayıt yok.")
	}

	// Kontrol E: Kilitli Döneme Sonradan İşlem Eklenmiş mi?
	var lockedViolations int64
	_ = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM public.transactions t
		JOIN public.periods p ON t.period_id = p.id
		WHERE p.status = 'locked' AND p.locked_at IS NOT NULL AND t.created_at > p.locked_at
	`).Scan(&lockedViolations)
	if lockedViolations > 0 {
		fmt.Printf("   ❌ UYARI: Kilitlendikten sonra eklenmiş %d işlem bulundu!\n", lockedViolations)
	} else {
		fmt.Println("   ✅ Dönem Kilidi Bütünlüğü: Kilitli dönemlere sonradan işlem sızması yok.")
	}

	// Kontrol F: Silinmiş / Kayıp Veri İpuçları (Idempotency Key ile Eşleşmeyen İşlem veya Ters Kayıt Oranı)
	var reversedCount int64
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM public.transactions WHERE reversed_by IS NOT NULL`).Scan(&reversedCount)
	fmt.Printf("   ℹ️ İptal/Ters Kayıt Edilmiş İşlem Sayısı: %d adet\n", reversedCount)

	// 7. En Son Eklenen 10 İşlem
	fmt.Println("\n🕒 7. EN SON EKLENEN 10 İŞLEM:")
	fmt.Println("--------------------------------------------------------------------------")
	recentRows, err := pool.Query(ctx, `
		SELECT id::text, direction, channel, amount, description, created_at
		FROM public.transactions
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err == nil {
		type RItem struct {
			id, dir, ch, desc string
			amount            float64
			cAt               time.Time
		}
		var rlist []RItem
		for recentRows.Next() {
			var r RItem
			_ = recentRows.Scan(&r.id, &r.dir, &r.ch, &r.amount, &r.desc, &r.cAt)
			rlist = append(rlist, r)
		}
		recentRows.Close()

		for idx, r := range rlist {
			sign := "+"
			if r.dir == "out" {
				sign = "-"
			}
			fmt.Printf("   %2d. [%s] %s%10.2f TL | %-12s | %-35s (%s)\n",
				idx+1, r.cAt.Format("2006-01-02 15:04:05"), sign, r.amount, r.ch, r.desc, r.id[:8]+"...")
		}
	}

	fmt.Println("\n==========================================================================")
	fmt.Println("🏁 DENETİM BAŞARIYLA TAMAMLANDI")
	fmt.Println("==========================================================================")
}
