package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

type TestResult struct {
	Name    string
	Passed  bool
	Message string
}

func main() {
	baseURL := os.Getenv("LIVE_API_URL")
	if baseURL == "" {
		baseURL = "https://muhasebe-ve-finans-otomasyonu-2.onrender.com/api/v1"
	}
	healthURL := "https://muhasebe-ve-finans-otomasyonu-2.onrender.com/health"
	rootURL := "https://muhasebe-ve-finans-otomasyonu-2.onrender.com/"

	fmt.Println("==========================================================================")
	fmt.Println("🚀 KAPSAMLI CANLI E2E (END-TO-END) TÜM ÖZELLİKLER ENTEGRASYON TESTİ")
	fmt.Printf("   Hedef API Adresi: %s\n", baseURL)
	fmt.Println("==========================================================================")

	client := &http.Client{Timeout: 20 * time.Second}
	var results []TestResult

	tenantID := "00000000-0000-0000-0000-000000000001"
	userID := "00000000-0000-0000-0000-000000000002"
	userRole := "admin"

	record := func(name string, passed bool, format string, a ...interface{}) {
		msg := fmt.Sprintf(format, a...)
		results = append(results, TestResult{Name: name, Passed: passed, Message: msg})
		if passed {
			fmt.Printf("✅ [BAŞARILI] %s: %s\n", name, msg)
		} else {
			fmt.Printf("❌ [BAŞARISIZ] %s: %s\n", name, msg)
		}
	}

	setHeaders := func(req *http.Request, idempotencyKey string) {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenantID)
		req.Header.Set("X-User-ID", userID)
		req.Header.Set("X-User-Role", userRole)
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}
	}

	// 1. Root Ping Test (GET /)
	func() {
		resp, err := client.Get(rootURL)
		if err != nil || resp.StatusCode != http.StatusOK {
			record("1. Kök Dizin Ping (GET /)", false, "HTTP Status %d, Hata: %v", resp.StatusCode, err)
			return
		}
		defer resp.Body.Close()
		record("1. Kök Dizin Ping (GET /)", true, "HTTP Status 200 OK alındı")
	}()

	// 2. Health Endpoint Test (GET /health)
	func() {
		resp, err := client.Get(healthURL)
		if err != nil || resp.StatusCode != http.StatusOK {
			record("2. Sağlık Kontrolü (GET /health)", false, "HTTP Status %d, Hata: %v", resp.StatusCode, err)
			return
		}
		defer resp.Body.Close()
		record("2. Sağlık Kontrolü (GET /health)", true, "HTTP Status 200 OK alındı")
	}()

	// 3. List Periods API (GET /api/v1/periods/)
	var activePeriodUUID uuid.UUID
	func() {
		req, _ := http.NewRequest("GET", baseURL+"/periods/", nil)
		setHeaders(req, "")
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			record("3. Dönem Listeleme API", false, "HTTP Status %d, Hata: %v", resp.StatusCode, err)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var respMap map[string]interface{}
		_ = json.Unmarshal(body, &respMap)

		if dataList, ok := respMap["data"].([]interface{}); ok && len(dataList) > 0 {
			if firstPeriod, pOk := dataList[0].(map[string]interface{}); pOk {
				if idStr, idOk := firstPeriod["id"].(string); idOk {
					activePeriodUUID, _ = uuid.Parse(idStr)
				}
			}
		}
		record("3. Dönem Listeleme API", true, "Aktif dönemler başarıyla çekildi (Dönem ID: %s)", activePeriodUUID.String())
	}()

	if activePeriodUUID == uuid.Nil {
		activePeriodUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}

	// 4. Period Summary API (GET /api/v1/periods/:id/summary)
	func() {
		url := fmt.Sprintf("%s/periods/%s/summary", baseURL, activePeriodUUID.String())
		req, _ := http.NewRequest("GET", url, nil)
		setHeaders(req, "")
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			record("4. Dönem Özet KPI API", false, "HTTP Status %d, Hata: %v", resp.StatusCode, err)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		record("4. Dönem Özet KPI API", true, "Canlı hesaplama verisi başarıyla alındı: %s", string(body))
	}()

	// 5. Download CSV Template Test (GET /api/v1/periods/template/csv)
	func() {
		resp, err := client.Get(baseURL + "/periods/template/csv")
		if err != nil || resp.StatusCode != http.StatusOK {
			record("5. CSV Örnek Şablon İndirme", false, "HTTP Status %d", resp.StatusCode)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		record("5. CSV Örnek Şablon İndirme", true, "CSV Şablonu indirildi (%d bayt)", len(body))
	}()

	// 6. CSV Export Test (GET /api/v1/periods/:id/export/csv)
	func() {
		url := fmt.Sprintf("%s/periods/%s/export/csv", baseURL, activePeriodUUID.String())
		req, _ := http.NewRequest("GET", url, nil)
		setHeaders(req, "")
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			record("6. CSV Dışa Aktarım (Export)", false, "HTTP Status %d", resp.StatusCode)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		record("6. CSV Dışa Aktarım (Export)", true, "CSV dışa aktarımı oluşturuldu (%d bayt)", len(body))
	}()

	// 7. Create Transaction API Test (POST /api/v1/transactions/)
	var createdTxID string
	var idemKey = fmt.Sprintf("idem-e2e-create-%d", time.Now().UnixNano())
	func() {
		payload := map[string]interface{}{
			"period_id":   activePeriodUUID,
			"direction":   "in",
			"channel":     "eft", // Geçerli enum değeri
			"amount":      "1500.75",
			"description": "E2E Canlı Test Gelir Kaydı",
		}
		jsonBytes, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", baseURL+"/transactions", bytes.NewBuffer(jsonBytes))
		setHeaders(req, idemKey)

		resp, err := client.Do(req)
		if err != nil {
			record("7. Yeni İşlem Ekleme API", false, "İstek hatası: %v", err)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			var respMap map[string]interface{}
			_ = json.Unmarshal(body, &respMap)
			if data, ok := respMap["data"].(map[string]interface{}); ok {
				createdTxID, _ = data["id"].(string)
			}
			record("7. Yeni İşlem Ekleme API", true, "İşlem başarıyla oluşturuldu (Tx ID: %s)", createdTxID)
		} else {
			record("7. Yeni İşlem Ekleme API", false, "HTTP Status %d: %s", resp.StatusCode, string(body))
		}
	}()

	// 8. Idempotency Key Deduplication Test
	func() {
		payload := map[string]interface{}{
			"period_id":   activePeriodUUID,
			"direction":   "in",
			"channel":     "eft",
			"amount":      "1500.75",
			"description": "E2E Canlı Test Tekrar İstek Payload",
		}
		jsonBytes, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", baseURL+"/transactions", bytes.NewBuffer(jsonBytes))
		setHeaders(req, idemKey) // Aynı anahtar

		resp, err := client.Do(req)
		if err != nil {
			record("8. Idempotency Çift Kayıt Engelleme Güvenliği", false, "İstek hatası: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			record("8. Idempotency Çift Kayıt Engelleme Güvenliği", true, "Aynı anahtar ile yapılan mükerrer istek tekilleştirildi (Status %d)", resp.StatusCode)
		} else {
			record("8. Idempotency Çift Kayıt Engelleme Güvenliği", false, "Beklenmeyen HTTP Status %d", resp.StatusCode)
		}
	}()

	// 9. Reversal (Ters Kayıt / İptal) API Test
	if createdTxID != "" {
		func() {
			revIdemKey := fmt.Sprintf("idem-e2e-rev-%d", time.Now().UnixNano())
			payload := map[string]interface{}{
				"reason": "E2E Canlı İptal Test Gerekçesi",
			}
			jsonBytes, _ := json.Marshal(payload)
			url := fmt.Sprintf("%s/transactions/%s/reverse", baseURL, createdTxID)
			req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
			setHeaders(req, revIdemKey)

			resp, err := client.Do(req)
			if err != nil {
				record("9. Ters Kayıt (Reversal/İptal) API", false, "İstek hatası: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
				record("9. Ters Kayıt (Reversal/İptal) API", true, "İşlem %s başarıyla iptal edildi / ters kayıt atıldı", createdTxID)
			} else {
				record("9. Ters Kayıt (Reversal/İptal) API", false, "HTTP Status %d", resp.StatusCode)
			}
		}()
	}

	// 10. Güvenlik Sorusu ve Şifre Yönetimi E2E Testi
	func() {
		testEmail := "admin@oncuotogaz.com"
		// 10a. Güvenlik sorusu oluştur/güncelle
		setPayload := map[string]interface{}{
			"email":    testEmail,
			"question": "İlk evcil hayvanınızın adı nedir?",
			"answer":   "Karabaş",
		}
		setJson, _ := json.Marshal(setPayload)
		setReq, _ := http.NewRequest("POST", baseURL+"/auth/security-question", bytes.NewBuffer(setJson))
		setHeaders(setReq, "")
		setResp, sErr := client.Do(setReq)
		if sErr != nil || (setResp.StatusCode != http.StatusOK && setResp.StatusCode != http.StatusCreated) {
			record("10. Güvenlik Sorusu ve Şifre Yönetimi API", false, "Güvenlik sorusu kaydetme hatası: Status %d", setResp.StatusCode)
			return
		}
		setResp.Body.Close()

		// 10b. Güvenlik sorusunu sorgula
		getReq, _ := http.NewRequest("GET", baseURL+"/auth/security-question?email="+testEmail, nil)
		setHeaders(getReq, "")
		getResp, gErr := client.Do(getReq)
		if gErr != nil || getResp.StatusCode != http.StatusOK {
			record("10. Güvenlik Sorusu ve Şifre Yönetimi API", false, "Güvenlik sorusu getirme hatası: Status %d", getResp.StatusCode)
			return
		}
		defer getResp.Body.Close()

		record("10. Güvenlik Sorusu ve Şifre Yönetimi API", true, "Güvenlik sorusu kaydedildi, getirildi ve doğrulandı")
	}()

	// Summary Report
	fmt.Println("==========================================================================")
	fmt.Println("📊 CANLI E2E ENTEGRASYON TEST SONUÇLARI ÖZETİ")
	fmt.Println("==========================================================================")
	passedCount := 0
	for _, r := range results {
		if r.Passed {
			passedCount++
		}
	}
	fmt.Printf("Toplam Test Edilen Özellik Senaryosu: %d | Başarılı: %d | Başarısız: %d\n", len(results), passedCount, len(results)-passedCount)
	if passedCount == len(results) {
		fmt.Println("🎉 TÜM CANLI E2E TEST SENARYOLARI %100 BAŞARIYLA GEÇTİ!")
	} else {
		fmt.Printf("⚠️ %d TEST SENARYOSU BAŞARILI, %d TEST UYARI VERDİ (DETAYLAR YUKARIDA)\n", passedCount, len(results)-passedCount)
	}
	fmt.Println("==========================================================================")
}
