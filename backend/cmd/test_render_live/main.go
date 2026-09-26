package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	supaURL := "https://lvsngrrdzjhbawhcuzqz.supabase.co"

	fmt.Println("1. Test: Render Backend Health Check:")
	resp, err := http.Get("https://muhasebe-ve-finans-otomasyonu-2.onrender.com/health")
	if err != nil {
		fmt.Printf("   Render error: %v\n", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("   Render Status: %d | Body: %s\n", resp.StatusCode, string(body))
	}

	fmt.Println("\n2. Test: Supabase JWKS Endpoint Check:")
	jwksResp, err := http.Get(supaURL + "/auth/v1/.well-known/jwks.json")
	if err != nil {
		fmt.Printf("   JWKS error: %v\n", err)
	} else {
		body, _ := io.ReadAll(jwksResp.Body)
		jwksResp.Body.Close()
		fmt.Printf("   JWKS Status: %d | Body: %s\n", jwksResp.StatusCode, string(body))
	}
}
