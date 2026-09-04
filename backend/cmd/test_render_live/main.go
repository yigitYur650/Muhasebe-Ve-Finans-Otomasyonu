package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	req, _ := http.NewRequest("GET", "https://muhasebe-ve-finans-otomasyonu-2.onrender.com/api/v1/periods/00000000-0000-0000-0000-000000000001/transactions", nil)
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	req.Header.Set("X-User-ID", "00000000-0000-0000-0000-000000000002")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Err: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Periods list on Render: %s\n", string(body))
}
