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

type MegaStage struct {
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
	fmt.Println("🚀 50,000 MEGA STRESS TEST: EXTREME RENDER BREAKPOINT & CAPACITY LIMIT")
	fmt.Printf("   Hedef Canlı Render API: %s\n", baseURL)
	fmt.Println("==========================================================================")

	client := &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 1000,
			IdleConnTimeout:     120 * time.Second,
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

	stages := []MegaStage{
		{Name: "Stage 1 (2.5K Request)", TotalReqs: 2500, Concurrency: 50},
		{Name: "Stage 2 (5K Request)", TotalReqs: 5000, Concurrency: 100},
		{Name: "Stage 3 (10K Request)", TotalReqs: 10000, Concurrency: 200},
		{Name: "Stage 4 (25K Request)", TotalReqs: 25000, Concurrency: 350},
		{Name: "Stage 5 (50K EXTREME Request)", TotalReqs: 50000, Concurrency: 500},
	}

	var breakpointReport string

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
					var req *http.Request

					if time.Now().UnixNano()%4 == 0 {
						// Yazma İsteği
						payload := map[string]interface{}{
							"period_id":   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
							"direction":   "in",
							"channel":     "eft",
							"amount":      "100.00",
							"description": "50K Mega Stress Kaydı",
						}
						b, _ := json.Marshal(payload)
						req, _ = http.NewRequest("POST", baseURL+"/transactions", bytes.NewBuffer(b))
						setHeaders(req, fmt.Sprintf("mega-%d", time.Now().UnixNano()))
					} else {
						// Okuma İsteği
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

		// Progress Ticker
		tickerDone := make(chan struct{})
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-tickerDone:
					return
				case <-ticker.C:
					currSucc := atomic.LoadInt32(&successCount)
					currFail := atomic.LoadInt32(&failCount)
					fmt.Printf("   ... İlerleme: %d / %d İşlendi (Başarılı: %d, Hata: %d)\n",
						currSucc+currFail, stage.TotalReqs, currSucc, currFail)
				}
			}
		}()

		wg.Wait()
		close(tickerDone)
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

		var avgLat, p95Lat, p99Lat, maxLat time.Duration
		if len(durations) > 0 {
			var tot time.Duration
			for _, d := range durations {
				tot += d
			}
			avgLat = tot / time.Duration(len(durations))

			p95Idx := int(float64(len(durations)) * 0.95)
			if p95Idx >= len(durations) {
				p95Idx = len(durations) - 1
			}
			p95Lat = durations[p95Idx]

			p99Idx := int(float64(len(durations)) * 0.99)
			if p99Idx >= len(durations) {
				p99Idx = len(durations) - 1
			}
			p99Lat = durations[p99Idx]

			maxLat = durations[len(durations)-1]
		}

		failRate := float64(failCount) / float64(stage.TotalReqs) * 100

		fmt.Printf("   📊 Tamamlanma Süresi       : %v\n", stageDuration)
		fmt.Printf("   🚀 Saniye Başına Kapasite   : %0.2f RPS\n", rps)
		fmt.Printf("   ✅ Başarılı Yanıt          : %d (%%%0.1f)\n", successCount, 100-failRate)
		fmt.Printf("   ❌ Hata / Kesinti           : %d (%%%0.1f)\n", failCount, failRate)
		fmt.Printf("   ⏱️ Latency (Avg/P95/P99/Max): %v / %v / %v / %v\n", avgLat, p95Lat, p99Lat, maxLat)
		fmt.Printf("   🚦 HTTP Status Dağılımı     : %v\n", statusMap)

		if (failRate > 5.0 || avgLat > 3*time.Second) && breakpointReport == "" {
			breakpointReport = fmt.Sprintf("%s (%d İstek, %d Concurrency, %0.2f RPS, Hata Oranı: %%%0.1f, Avg Latency: %v)",
				stage.Name, stage.TotalReqs, stage.Concurrency, rps, failRate, avgLat)
			fmt.Printf("   ⚠️ ⚠️ KIRILMA NOKTASI TESPİT EDİLDİ! (Render/Sunucu Sınırına Ulaşıldı)\n")
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Println("\n==========================================================================")
	fmt.Println("📊 50,000 MEGA STRESS TEST DEĞERLENDİRME RAPORU")
	fmt.Println("==========================================================================")
	if breakpointReport != "" {
		fmt.Printf("🔴 CANLI RENDER SUNUCUSUNUN KESİN KIRILMA / KAPASİTE SINIRI:\n   👉 %s\n", breakpointReport)
	} else {
		fmt.Printf("🟢 İNANILMAZ REKOR! Sunucu 50,000 İstek ve 500 Eşzamanlı Paralel Yük Altında ÇÖKMEDİ!\n")
	}
	fmt.Println("==========================================================================")
}
