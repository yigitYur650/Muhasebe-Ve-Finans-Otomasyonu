package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type StageConfig struct {
	Name        string
	TotalReqs   int
	Concurrency int
}

type ReqResult struct {
	Duration time.Duration
	Status   int
	Err      error
}

func main() {
	baseURL := os.Getenv("LIVE_API_URL")
	if baseURL == "" {
		baseURL = "https://muhasebe-ve-finans-otomasyonu-2.onrender.com/api/v1"
	}

	fmt.Println("==========================================================================")
	fmt.Println("💥 BREAKPOINT & STRESS TEST: MAKSİMUM KAPASİTE VE ÇÖKME NOKTASI TESPİTİ")
	fmt.Printf("   Hedef Canlı API: %s\n", baseURL)
	fmt.Println("==========================================================================")

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        500,
			MaxIdleConnsPerHost: 500,
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

	stages := []StageConfig{
		{Name: "Kademe 1 (Hafif Yük)", TotalReqs: 50, Concurrency: 10},
		{Name: "Kademe 2 (Orta Yük)", TotalReqs: 150, Concurrency: 25},
		{Name: "Kademe 3 (Yüksek Yük)", TotalReqs: 300, Concurrency: 50},
		{Name: "Kademe 4 (Aşırı Yük - Stress)", TotalReqs: 600, Concurrency: 100},
		{Name: "Kademe 5 (Kırılma Noktası - Breakpoint)", TotalReqs: 1000, Concurrency: 150},
	}

	var breakpointDetected string

	for idx, stage := range stages {
		fmt.Printf("\n🔥 [%d/%d] RUNNING %s: %d İstek, %d Paralel Worker...\n",
			idx+1, len(stages), stage.Name, stage.TotalReqs, stage.Concurrency)

		var wg sync.WaitGroup
		reqChan := make(chan int, stage.TotalReqs)
		resChan := make(chan ReqResult, stage.TotalReqs)

		var successCount int32
		var failCount int32
		statusMap := make(map[int]int)
		var statusMu sync.Mutex

		startTime := time.Now()

		for w := 0; w < stage.Concurrency; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range reqChan {
					t0 := time.Now()
					// Karışık Okuma/Yazma Yükü (%80 Okuma, %20 Yazma)
					var req *http.Request
					if time.Now().UnixNano()%5 == 0 {
						// Yazma isteği
						payload := map[string]interface{}{
							"period_id":   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
							"direction":   "in",
							"channel":     "eft",
							"amount":      "50.00",
							"description": "Stress Test Kaydı",
						}
						b, _ := json.Marshal(payload)
						req, _ = http.NewRequest("POST", baseURL+"/transactions", bytes.NewBuffer(b))
						setHeaders(req, fmt.Sprintf("stress-%d", time.Now().UnixNano()))
					} else {
						// Okuma isteği
						req, _ = http.NewRequest("GET", baseURL+"/periods/00000000-0000-0000-0000-000000000001/summary", nil)
						setHeaders(req, "")
					}

					resp, err := client.Do(req)
					dur := time.Since(t0)

					code := 0
					if err == nil {
						code = resp.StatusCode
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
						if code >= 200 && code < 400 {
							atomic.AddInt32(&successCount, 1)
						} else {
							atomic.AddInt32(&failCount, 1)
						}
					} else {
						atomic.AddInt32(&failCount, 1)
					}

					statusMu.Lock()
					statusMap[code]++
					statusMu.Unlock()

					resChan <- ReqResult{Duration: dur, Status: code, Err: err}
				}
			}()
		}

		for i := 0; i < stage.TotalReqs; i++ {
			reqChan <- i
		}
		close(reqChan)
		wg.Wait()
		close(resChan)

		stageDuration := time.Since(startTime)
		rps := float64(stage.TotalReqs) / stageDuration.Seconds()

		var durations []time.Duration
		for r := range resChan {
			durations = append(durations, r.Duration)
		}

		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})

		var avgLat, p95Lat, maxLat time.Duration
		if len(durations) > 0 {
			var tot time.Duration
			for _, d := range durations {
				tot += d
			}
			avgLat = tot / time.Duration(len(durations))
			p95Index := int(float64(len(durations)) * 0.95)
			if p95Index >= len(durations) {
				p95Index = len(durations) - 1
			}
			p95Lat = durations[p95Index]
			maxLat = durations[len(durations)-1]
		}

		failRate := float64(failCount) / float64(stage.TotalReqs) * 100

		fmt.Printf("   📊 Tamamlanma Süresi  : %v\n", stageDuration)
		fmt.Printf("   🚀 Saniye Başına Yük   : %0.2f RPS\n", rps)
		fmt.Printf("   ✅ Başarılı İstek    : %d (%%%0.1f)\n", successCount, 100-failRate)
		fmt.Printf("   ❌ Hata / Kesinti     : %d (%%%0.1f)\n", failCount, failRate)
		fmt.Printf("   ⏱️ Latency (Avg/P95/Max): %v / %v / %v\n", avgLat, p95Lat, maxLat)
		fmt.Printf("   🚦 Status Dağılımı    : %v\n", statusMap)

		if (failRate > 10.0 || avgLat > 2*time.Second) && breakpointDetected == "" {
			breakpointDetected = fmt.Sprintf("%s (%d İstek, %d Paralel Worker, %0.2f RPS, Hata Oranı: %%%0.1f, Ortalama Latency: %v)",
				stage.Name, stage.TotalReqs, stage.Concurrency, rps, failRate, avgLat)
			fmt.Printf("   ⚠️ ⚠️ KIRILMA NOKTASI TESPİT EDİLDİ! (Hata oranı veya gecikme eşiği aşıldı)\n")
		}

		time.Sleep(1 * time.Second) // Kademeler arası soğuma süresi
	}

	fmt.Println("\n==========================================================================")
	fmt.Println("📊 MAKSİMUM KAPASİTE VE KIRILMA NOKTASI (BREAKPOINT) DEĞERLENDİRMESİ")
	fmt.Println("==========================================================================")
	if breakpointDetected != "" {
		fmt.Printf("🔴 CANLI SUNUCUNUN KIRILMA / YAVAŞLAMA NOKTASI: %s\n", breakpointDetected)
	} else {
		fmt.Printf("🟢 MÜKEMMEL! Sunucu 1000 Eşzamanlı İstek ve 150 Paralel Worker Yükü Altında HİÇ ÇÖKMEDİ! (Maksimum Kapasite > 1000 Req / 150 Concurrency)\n")
	}
	fmt.Println("==========================================================================")
}
