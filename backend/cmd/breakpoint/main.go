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

type LevelConfig struct {
	Level       int
	Workers     int
	TotalReqs   int
}

func main() {
	baseURL := os.Getenv("LIVE_API_URL")
	if baseURL == "" {
		baseURL = "https://muhasebe-ve-finans-otomasyonu-2.onrender.com/api/v1"
	}

	fmt.Println("==========================================================================")
	fmt.Println("💥 %50 HATA ORANI KIRILMA NOKTASI TESPİT TESTİ (ULTIMATE BREAKPOINT FINDER)")
	fmt.Printf("   Hedef Canlı Render API: %s\n", baseURL)
	fmt.Println("   Kural: Hata / Yanıtsız kalma oranı %%50'yi aşana kadar yük artırılacak!")
	fmt.Println("==========================================================================")

	// Kısa zaman aşımı (4 saniye) ile sunucunun yanıt veremediği durumları hemen tespit et
	client := &http.Client{
		Timeout: 4 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        2000,
			MaxIdleConnsPerHost: 2000,
			IdleConnTimeout:     60 * time.Second,
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

	levels := []LevelConfig{
		{Level: 1, Workers: 100, TotalReqs: 1000},
		{Level: 2, Workers: 250, TotalReqs: 2500},
		{Level: 3, Workers: 500, TotalReqs: 5000},
		{Level: 4, Workers: 750, TotalReqs: 7500},
		{Level: 5, Workers: 1000, TotalReqs: 10000},
		{Level: 6, Workers: 1500, TotalReqs: 15000},
		{Level: 7, Workers: 2000, TotalReqs: 20000},
		{Level: 8, Workers: 3000, TotalReqs: 30000},
	}

	var ultimateBreakpoint string

	for _, lvl := range levels {
		fmt.Printf("\n🔥 Seviye %d: %d Eşzamanlı Paralel Worker, Toplam %d İstek Atılıyor...\n",
			lvl.Level, lvl.Workers, lvl.TotalReqs)

		var wg sync.WaitGroup
		reqChan := make(chan int, lvl.TotalReqs)

		var successCount int32
		var failCount int32

		startTime := time.Now()

		for w := 0; w < lvl.Workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range reqChan {
					var req *http.Request
					if time.Now().UnixNano()%3 == 0 {
						// Yazma isteği
						payload := map[string]interface{}{
							"period_id":   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
							"direction":   "in",
							"channel":     "eft",
							"amount":      "150.00",
							"description": "50% Breakpoint Test Kaydı",
						}
						b, _ := json.Marshal(payload)
						req, _ = http.NewRequest("POST", baseURL+"/transactions", bytes.NewBuffer(b))
						setHeaders(req, fmt.Sprintf("bp-%d", time.Now().UnixNano()))
					} else {
						// Okuma isteği
						req, _ = http.NewRequest("GET", baseURL+"/periods/00000000-0000-0000-0000-000000000001/summary", nil)
						setHeaders(req, "")
					}

					resp, err := client.Do(req)
					if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 400 {
						atomic.AddInt32(&successCount, 1)
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					} else {
						atomic.AddInt32(&failCount, 1)
						if resp != nil {
							resp.Body.Close()
						}
					}
				}
			}()
		}

		for i := 0; i < lvl.TotalReqs; i++ {
			reqChan <- i
		}
		close(reqChan)

		wg.Wait()
		duration := time.Since(startTime)
		rps := float64(lvl.TotalReqs) / duration.Seconds()
		failRate := float64(failCount) / float64(lvl.TotalReqs) * 100
		successRate := float64(successCount) / float64(lvl.TotalReqs) * 100

		fmt.Printf("   📊 Tamamlanma Süresi: %v\n", duration)
		fmt.Printf("   🚀 İşlem Hızı (RPS) : %0.2f RPS\n", rps)
		fmt.Printf("   ✅ Başarılı Yanıt   : %d (%%%0.1f)\n", successCount, successRate)
		fmt.Printf("   ❌ Yanıtsız / Hata  : %d (%%%0.1f)\n", failCount, failRate)

		if failRate >= 50.0 {
			ultimateBreakpoint = fmt.Sprintf("Seviye %d (%d Paralel Worker, %d İstek, %0.2f RPS, Hata Oranı: %%%0.1f)",
				lvl.Level, lvl.Workers, lvl.TotalReqs, rps, failRate)
			fmt.Printf("🎯 %s YANITSIZ/TIMEOUT BREAKPOINT NOKTASI TESPİT EDİLDİ! (%d Eşzamanlı İş Parçacığı)\n", "%50+", lvl.Workers)
			break
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Println("\n==========================================================================")
	fmt.Println("📊 %50 YANI TSIZ/HATA KIRILMA NOKTASI RAPORU")
	fmt.Println("==========================================================================")
	if ultimateBreakpoint != "" {
		fmt.Printf("🔴 CANLI RENDER SUNUCUSU BURADA ÇÖKTÜ / %s YANITSIZ KALDI:\n   👉 %s\n", "%50+", ultimateBreakpoint)
	} else {
		fmt.Println("🟢 SUNUCU İNANILMAZ! 3,000 Eşzamanlı Paralel Worker Yükü Altında Bile %50 Hata Oranına Ulaşmadı!")
	}
	fmt.Println("==========================================================================")
}
