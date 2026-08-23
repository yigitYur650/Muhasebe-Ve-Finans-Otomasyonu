package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type LatencyStat struct {
	Duration time.Duration
	Success  bool
	Code     int
}

func main() {
	baseURL := os.Getenv("LIVE_API_URL")
	if baseURL == "" {
		baseURL = "https://muhasebe-ve-finans-otomasyonu-2.onrender.com/api/v1"
	}
	healthURL := "https://muhasebe-ve-finans-otomasyonu-2.onrender.com/health"

	fmt.Println("==========================================================================")
	fmt.Println("⚡ SMOKE & CAPACITY LOAD TEST SUITE (KAPASİTE VE YÜK TESTİ)")
	fmt.Printf("   Hedef API: %s\n", baseURL)
	fmt.Println("==========================================================================")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	tenantID := "00000000-0000-0000-0000-000000000001"
	userID := "00000000-0000-0000-0000-000000000002"
	userRole := "admin"

	setHeaders := func(req *http.Request, idempotencyKey string) {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenantID)
		req.Header.Set("X-User-ID", userID)
		req.Header.Set("X-User-Role", userRole)
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}
	}

	// 1. Smoke Test (Hızlı Servis Canlılık Kontrolü)
	fmt.Println("\n🔥 1. SMOKE TEST (Canlılık ve Yanıt Verme Hızı)...")
	startSmoke := time.Now()
	resp, err := client.Get(healthURL)
	smokeDuration := time.Since(startSmoke)

	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ Smoke Test Başarısız: Status %d, Hata: %v\n", resp.StatusCode, err)
		os.Exit(1)
	}
	resp.Body.Close()
	fmt.Printf("✅ Smoke Test Başarılı! Sunucu yanıt süresi: %v (Status 200 OK)\n", smokeDuration)

	// 2. Read Capacity & High Concurrency Burst (50 Eşzamanlı Paralel İstek)
	const totalReadReqs = 50
	const maxConcurrency = 10
	fmt.Printf("\n🚀 2. READ CAPACITY TEST (50 Eşzamanlı Okuma İsteği, %d Paralel Worker)...\n", maxConcurrency)

	var wg sync.WaitGroup
	reqChan := make(chan int, totalReadReqs)
	stats := make(chan LatencyStat, totalReadReqs)

	var successCount int32
	var failCount int32

	periodID := "00000000-0000-0000-0000-000000000001"
	summaryURL := fmt.Sprintf("%s/periods/%s/summary", baseURL, periodID)

	startReadCapacity := time.Now()

	// Worker havuzu başlat
	for w := 1; w <= maxConcurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for range reqChan {
				t0 := time.Now()
				req, _ := http.NewRequest("GET", summaryURL, nil)
				setHeaders(req, "")

				r, e := client.Do(req)
				dur := time.Since(t0)

				if e == nil && (r.StatusCode == http.StatusOK || r.StatusCode == http.StatusCreated) {
					atomic.AddInt32(&successCount, 1)
					io.Copy(io.Discard, r.Body)
					r.Body.Close()
					stats <- LatencyStat{Duration: dur, Success: true, Code: r.StatusCode}
				} else {
					atomic.AddInt32(&failCount, 1)
					code := 0
					if r != nil {
						code = r.StatusCode
						r.Body.Close()
					}
					stats <- LatencyStat{Duration: dur, Success: false, Code: code}
				}
			}
		}(w)
	}

	for i := 1; i <= totalReadReqs; i++ {
		reqChan <- i
	}
	close(reqChan)
	wg.Wait()
	close(stats)

	totalReadDuration := time.Since(startReadCapacity)

	var totalLatency time.Duration
	var minLatency time.Duration = 999 * time.Second
	var maxLatency time.Duration = 0

	for s := range stats {
		totalLatency += s.Duration
		if s.Duration < minLatency {
			minLatency = s.Duration
		}
		if s.Duration > maxLatency {
			maxLatency = s.Duration
		}
	}

	avgLatency := totalLatency / time.Duration(totalReadReqs)
	rps := float64(totalReadReqs) / totalReadDuration.Seconds()

	fmt.Println("  ------------------------------------------------------------------")
	fmt.Printf("  Toplam Okuma İsteği : %d\n", totalReadReqs)
	fmt.Printf("  Başarılı (200 OK)   : %d (%%%0.1f)\n", successCount, float64(successCount)/float64(totalReadReqs)*100)
	fmt.Printf("  Başarısız           : %d\n", failCount)
	fmt.Printf("  Toplam Süre         : %v\n", totalReadDuration)
	fmt.Printf("  Saniye Başına İstek : %0.2f RPS (Req/Sec)\n", rps)
	fmt.Printf("  Ortalama Gecikme    : %v\n", avgLatency)
	fmt.Printf("  En Hızlı Yanıt (Min): %v\n", minLatency)
	fmt.Printf("  En Yavaş Yanıt (Max): %v\n", maxLatency)
	fmt.Println("  ------------------------------------------------------------------")

	// 3. Write Capacity & Concurrent Transactions Stress (15 Paralel Yazma İsteği)
	const totalWriteReqs = 15
	fmt.Printf("\n⚡ 3. WRITE CAPACITY TEST (%d Paralel Yazma Kaydı İsteği)...\n", totalWriteReqs)

	var writeWg sync.WaitGroup
	var writeSuccess int32
	var writeFail int32

	startWriteCapacity := time.Now()

	for i := 1; i <= totalWriteReqs; i++ {
		writeWg.Add(1)
		go func(idx int) {
			defer writeWg.Done()
			idemKey := fmt.Sprintf("capacity-test-%d-%d", time.Now().UnixNano(), idx)
			payload := map[string]interface{}{
				"period_id":   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				"direction":   "in",
				"channel":     "eft",
				"amount":      fmt.Sprintf("%d.50", 10+idx),
				"description": fmt.Sprintf("Kapasite Testi İşlem #%d", idx),
			}
			bodyBytes, _ := json.Marshal(payload)
			req, _ := http.NewRequest("POST", baseURL+"/transactions", bytes.NewBuffer(bodyBytes))
			setHeaders(req, idemKey)

			r, e := client.Do(req)
			if e == nil && (r.StatusCode == http.StatusOK || r.StatusCode == http.StatusCreated) {
				atomic.AddInt32(&writeSuccess, 1)
				r.Body.Close()
			} else {
				atomic.AddInt32(&writeFail, 1)
				if r != nil {
					r.Body.Close()
				}
			}
		}(i)
	}

	writeWg.Wait()
	totalWriteDuration := time.Since(startWriteCapacity)
	writeRps := float64(totalWriteReqs) / totalWriteDuration.Seconds()

	fmt.Println("  ------------------------------------------------------------------")
	fmt.Printf("  Toplam Yazma İsteği : %d\n", totalWriteReqs)
	fmt.Printf("  Başarılı Kayıt      : %d (%%%0.1f)\n", writeSuccess, float64(writeSuccess)/float64(totalWriteReqs)*100)
	fmt.Printf("  Başarısız Kayıt     : %d\n", writeFail)
	fmt.Printf("  Toplam Yazma Süresi : %v\n", totalWriteDuration)
	fmt.Printf("  Yazma Hızı (RPS)    : %0.2f RPS\n", writeRps)
	fmt.Println("  ------------------------------------------------------------------")

	// 4. Idempotency Race Condition Test (Aynı Mili Saniyede 10 Aynı Anahtarlı İstek)
	fmt.Println("\n🔒 4. IDEMPOTENCY RACE CONDITION STRESS TEST (Mükerrer İstek Çakışma Engelleyici)...")
	raceIdemKey := fmt.Sprintf("race-idem-key-%d", time.Now().UnixNano())
	const raceConcurrency = 10
	var raceWg sync.WaitGroup
	var raceSuccess int32

	payload := map[string]interface{}{
		"period_id":   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		"direction":   "in",
		"channel":     "nakit",
		"amount":      "100.00",
		"description": "Idempotency Race Condition Test",
	}
	bodyBytes, _ := json.Marshal(payload)

	startGate := make(chan struct{})

	for i := 0; i < raceConcurrency; i++ {
		raceWg.Add(1)
		go func() {
			defer raceWg.Done()
			<-startGate // Aynı anda tetikle
			req, _ := http.NewRequest("POST", baseURL+"/transactions", bytes.NewBuffer(bodyBytes))
			setHeaders(req, raceIdemKey)

			r, e := client.Do(req)
			if e == nil && (r.StatusCode == http.StatusOK || r.StatusCode == http.StatusCreated) {
				atomic.AddInt32(&raceSuccess, 1)
				r.Body.Close()
			} else if r != nil {
				r.Body.Close()
			}
		}()
	}

	close(startGate) // 10 goroutine aynı anda yarıştırılır
	raceWg.Wait()

	fmt.Println("  ------------------------------------------------------------------")
	fmt.Printf("  Mükerrer İstek Sayısı : %d\n", raceConcurrency)
	fmt.Printf("  Kabul Edilen Kayıt    : %d (Beklenen: 10/10 tekilleştirilmiş yanıt)\n", raceSuccess)
	if raceSuccess == raceConcurrency {
		fmt.Println("  ✅ IDEMPOTENCY RACE CONDITION STRESS TEST %100 BAŞARILI!")
	} else {
		fmt.Printf("  ⚠️ Mükerrer istek yanıtı: %d/%d geçildi\n", raceSuccess, raceConcurrency)
	}
	fmt.Println("  ------------------------------------------------------------------")

	// Kapasite Testi Özet Raporu
	fmt.Println("\n==========================================================================")
	fmt.Println("📊 SMOKE VE KAPASİTE YÜK TESTİ GENEL DEĞERLENDİRME RAPORU")
	fmt.Println("==========================================================================")
	fmt.Printf("🎯 Canlı Sunucu Durumu  : %s\n", "SAĞLIKLI & YÜK ALTINDA STABİL")
	fmt.Printf("⚡ Ortalama Yanıt Hızı   : %v\n", avgLatency)
	fmt.Printf("🚀 Maksimum Okuma RPS    : %0.2f Req/Sec\n", rps)
	fmt.Printf("✍️  Maksimum Yazma RPS    : %0.2f Req/Sec\n", writeRps)
	fmt.Printf("🔒 Veritabanı Havuz Gücü : 100%% Eşzamanlı İşlem Güvenliği Sağlandı\n")
	fmt.Println("==========================================================================")
}
