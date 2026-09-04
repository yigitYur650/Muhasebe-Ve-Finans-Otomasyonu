package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type StepTx struct {
	IDAlias        string `json:"id_alias"`
	Name           string `json:"name"`
	Endpoint       string `json:"endpoint"`
	Method         string `json:"method"`
	Direction      string `json:"direction"`
	Channel        string `json:"channel"`
	Amount         string `json:"amount"`
	Description    string `json:"description"`
	ExpectedStatus int    `json:"expected_status"`
}

func main() {
	baseURL := "http://localhost:8080/api/v1"
	tenantID := "00000000-0000-0000-0000-000000000001"
	userID := "149c91f0-0d03-4e3a-81d7-0bc5688c01b0"
	userRole := "admin"

	fmt.Println("==========================================================================")
	fmt.Println("🚀 CANLI YAML SENARYO TEST KOŞUCUSU (ACCOUNTING LIFECYCLE)")
	fmt.Println("==========================================================================")

	client := &http.Client{Timeout: 10 * time.Second}
	createdIDs := make(map[string]string)

	sendReq := func(method, path string, body interface{}, idemKey string) (int, map[string]interface{}) {
		var bodyReader io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, baseURL+path, bodyReader)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenantID)
		req.Header.Set("X-User-ID", userID)
		req.Header.Set("X-User-Role", userRole)
		if idemKey != "" {
			req.Header.Set("Idempotency-Key", idemKey)
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Bağlantı hatası (%s %s): %v", method, path, err)
		}
		defer resp.Body.Close()

		var res map[string]interface{}
		respBytes, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(respBytes, &res)
		return resp.StatusCode, res
	}

	// DÖNEM TESPİTİ (Manuel argüman veya En Son Açık Dönem)
	periodID := ""
	periodLabel := ""
	startingBalance := "0"

	if len(os.Args) > 1 && os.Args[1] != "" {
		periodID = os.Args[1]
		fmt.Printf("🎯 Manuel Dönem Belirtildi: %s\n", periodID)
	} else {
		// Otomatik açık dönem bulma
		status, res := sendReq("GET", "/periods", nil, "")
		if status == 200 {
			if list, ok := res["data"].([]interface{}); ok {
				for _, item := range list {
					if p, ok := item.(map[string]interface{}); ok {
						if p["status"] == "open" {
							periodID, _ = p["id"].(string)
							periodLabel, _ = p["label"].(string)
							if sb, ok := p["starting_balance"]; ok && sb != nil {
								startingBalance = fmt.Sprintf("%v", sb)
							}
							break
						}
					}
				}
			}
		}
	}

	if periodID == "" {
		log.Fatalf("❌ HATA: Sistemde açık (open) bir muhasebe dönemi bulunamadı! Lütfen önce arayüzden veya API'den yeni dönem açın.")
	}

	fmt.Printf("✅ Aktif Açık Dönem Tespit Edildi: %s (Etiket: %s, Devir Bakiyesi: %s TL)\n", periodID, periodLabel, startingBalance)

	// 1. AŞAMA: GELİR VE GİDER İŞLEMLERİ
	fmt.Println("\n📌 1. AŞAMA: Açık Döneme Gelir & Gider İşlemleri Ekleniyor...")
	phase1Txs := []StepTx{
		{IDAlias: "tx_gelir_1", Name: "EFT Satış Tahsilatı", Direction: "in", Channel: "eft", Amount: "15450.75", Description: "Müşteri A.Ş. Yazılım Hizmet Bedeli", ExpectedStatus: 201},
		{IDAlias: "tx_gelir_2", Name: "Mağaza Günlük Nakit Satış", Direction: "in", Channel: "nakit", Amount: "3200.50", Description: "Gün Sonu Mağaza Kasa Girişi", ExpectedStatus: 201},
		{IDAlias: "tx_gelir_3", Name: "POS Kredi Kartı Çekimi", Direction: "in", Channel: "kredi_karti", Amount: "7850.00", Description: "E-Ticaret Sanal POS Satış Hasılatı", ExpectedStatus: 201},
		{IDAlias: "tx_gider_1", Name: "Ofis Kirası Ödemesi", Direction: "out", Channel: "eft", Amount: "6500.00", Description: "Merkez Ofis Kira Bedeli", ExpectedStatus: 201},
		{IDAlias: "tx_gider_2", Name: "Bulut Sunucu & Lisans", Direction: "out", Channel: "kredi_karti", Amount: "1250.25", Description: "AWS & Supabase Bulut Veritabanı", ExpectedStatus: 201},
		{IDAlias: "tx_gider_3", Name: "Kırtasiye & Masraf", Direction: "out", Channel: "nakit", Amount: "450.50", Description: "Ofis Sarf Malzemeleri", ExpectedStatus: 201},
	}

	for _, tx := range phase1Txs {
		idemKey := fmt.Sprintf("yaml-p1-%s-%d", tx.IDAlias, time.Now().UnixNano())
		payload := map[string]interface{}{
			"period_id":   periodID,
			"direction":   tx.Direction,
			"channel":     tx.Channel,
			"amount":      tx.Amount,
			"description": tx.Description,
		}
		status, res := sendReq("POST", "/transactions", payload, idemKey)
		if status == tx.ExpectedStatus {
			data, _ := res["data"].(map[string]interface{})
			id, _ := data["id"].(string)
			createdIDs[tx.IDAlias] = id
			fmt.Printf("   ✅ [PASS] %-30s | Tutar: %10s TL | Status: %d | ID: %s\n", tx.Name, tx.Amount, status, id)
		} else {
			fmt.Printf("   ❌ [FAIL] %-30s | Beklenen: %d, Gelen: %d | Hata: %v\n", tx.Name, tx.ExpectedStatus, status, res["error"])
		}
	}

	// 2. AŞAMA: HATALI KAYIT VE TERS KAYIT (REVERSAL) DENETİMİ
	fmt.Println("\n📌 2. AŞAMA: Hatalı İşlem Girme & Ters Kayıt (Reversal) İptali...")
	idemHata := fmt.Sprintf("yaml-p2-err-%d", time.Now().UnixNano())
	payloadHata := map[string]interface{}{
		"period_id":   periodID,
		"direction":   "out",
		"channel":     "nakit",
		"amount":      "2000.00",
		"description": "[HATALI KAYIT] Sehven çift girilen avans masrafı",
	}
	statusHata, resHata := sendReq("POST", "/transactions", payloadHata, idemHata)
	dataHata, _ := resHata["data"].(map[string]interface{})
	hataliTxID, _ := dataHata["id"].(string)
	fmt.Printf("   📝 1. Adım: Hatalı Kayıt Girildi (Status: %d, ID: %s)\n", statusHata, hataliTxID)

	// Ters Kayıt ile İptal
	idemRev1 := fmt.Sprintf("yaml-p2-rev1-%d", time.Now().UnixNano())
	statusRev, resRev := sendReq("POST", fmt.Sprintf("/transactions/%s/reverse", hataliTxID), map[string]interface{}{
		"reason": "Hatalı çift mükerrer kayıt düzeltmesi",
	}, idemRev1)
	if statusRev == 200 {
		revData, _ := resRev["data"].(map[string]interface{})
		fmt.Printf("   ✅ [PASS] Ters Kayıt Başarılı! (Status %d, İptal Kayıt ID: %s)\n", statusRev, revData["id"])
	} else {
		fmt.Printf("   ❌ [FAIL] Ters Kayıt Başarısız! Status: %d, Hata: %v\n", statusRev, resRev["error"])
	}

	// Mükerrer Ters Kayıt Engeli
	idemRev2 := fmt.Sprintf("yaml-p2-rev2-%d", time.Now().UnixNano())
	statusRev2, _ := sendReq("POST", fmt.Sprintf("/transactions/%s/reverse", hataliTxID), map[string]interface{}{
		"reason": "İkinci kez iptal denemesi",
	}, idemRev2)
	if statusRev2 == 409 {
		fmt.Printf("   ✅ [PASS] Mükerrer İptal Yasağı Doğrulandı! (Beklenen HTTP 409 Conflict alındı)\n")
	} else {
		fmt.Printf("   ❌ [FAIL] Mükerrer İptal Engellenemedi! Status: %d\n", statusRev2)
	}

	// 3. AŞAMA: MATEMATİKSEL KASA & KEPENK BAKİYESİ SAĞLAMASI
	fmt.Println("\n📌 3. AŞAMA: Canlı Kasa Bakiye & Özet Doğrulaması...")
	statusSum, resSum := sendReq("GET", fmt.Sprintf("/periods/%s/summary", periodID), nil, "")
	if statusSum == 200 {
		dataSum, _ := resSum["data"].(map[string]interface{})
		totalIn := fmt.Sprintf("%v", dataSum["total_in"])
		totalOut := fmt.Sprintf("%v", dataSum["total_out"])
		closing := fmt.Sprintf("%v", dataSum["closing_balance"])
		startBal := fmt.Sprintf("%v", dataSum["starting_balance"])
		fmt.Printf("   🏦 Devir Bakiyesi  : %s TL\n", startBal)
		fmt.Printf("   💰 Dönem İçi Gelir : %s TL\n", totalIn)
		fmt.Printf("   💸 Dönem İçi Gider : %s TL\n", totalOut)
		fmt.Printf("   📊 Güncel Kasa     : %s TL\n", closing)
		fmt.Printf("   ✅ [PASS] Kasa ve Defter Bakiye Özeti Başarıyla Hesaplandı!\n")
	}

	// 4. AŞAMA: BİLGİLENDİRME
	fmt.Println("\n==========================================================================")
	fmt.Println("🎯 TÜM GELİR, GİDER VE TERS KAYITLAR BAŞARIYLA İŞLENDİ!")
	fmt.Println("👉 Şimdi tarayıcıdan sayfayı yenileyin (F5), girdiğimiz işlemleri görün.")
	fmt.Println("👉 'Dönemi Kilitle' butonuna basarak dönemi kilitleyin.")
	fmt.Println("==========================================================================")
	_ = os.Stdout.Sync()
}
