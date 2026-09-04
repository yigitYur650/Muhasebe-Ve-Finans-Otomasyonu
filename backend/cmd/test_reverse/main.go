package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func main() {
	txID := "4b0f4ed9-dd8d-40d2-9af2-a68c4c32e82e"
	url := fmt.Sprintf("https://muhasebe-ve-finans-otomasyonu-2.onrender.com/api/v1/transactions/%s/reverse", txID)

	jsonBody := []byte(`{"reason":"Müşteri İade Talebi"}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("Req err: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-brand-new-999")
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	req.Header.Set("X-User-ID", "00000000-0000-0000-0000-000000000002")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Do err: %v\n", err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\nResponse: %s\n", resp.StatusCode, string(bodyBytes))
}
