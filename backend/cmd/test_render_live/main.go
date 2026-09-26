package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	supaURL := "https://lvsngrrdzjhbawhcuzqz.supabase.co"
	anonKey := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Imx2c25ncnJkempoYmF3aGN1enF6Iiwicm9sZSI6ImFub24iLCJpYXQiOjE3ODg1MzAxMzEsImV4cCI6MjEwNDEwNjEzMX0.yilfkY7aUMr4j5Q0oj2oH8kGH4fIelCNZKDjo_VOfls"

	fmt.Println("1. Render Backend Health Check:")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://muhasebe-ve-finans-otomasyonu-2.onrender.com/health")
	if err != nil {
		fmt.Printf("   Render error: %v\n", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("   Render Status: %d | Body: %s\n", resp.StatusCode, string(body))
	}

	fmt.Println("\n2. Supabase JWKS Endpoint Check:")
	jwksResp, err := client.Get(supaURL + "/auth/v1/.well-known/jwks.json")
	if err != nil {
		fmt.Printf("   JWKS error: %v\n", err)
	} else {
		body, _ := io.ReadAll(jwksResp.Body)
		jwksResp.Body.Close()
		fmt.Printf("   JWKS Status: %d | Body: %s\n", jwksResp.StatusCode, string(body))
	}

	fmt.Println("\n3. Testing Supabase Auth Logins:")
	testCreds := []struct {
		email    string
		password string
	}{
		{"admin@oncuotogaz.com", "oncu123456"},
		{"admin@oncuotogaz.com", "Password123!"},
		{"yigityur65@gmail.com", "oncu123456"},
		{"yigityur65@gmail.com", "Password123!"},
	}

	var accessToken string
	for _, tc := range testCreds {
		loginPayload := map[string]string{
			"email":    tc.email,
			"password": tc.password,
		}
		payloadBytes, _ := json.Marshal(loginPayload)
		req, _ := http.NewRequest("POST", supaURL+"/auth/v1/token?grant_type=password", bytes.NewBuffer(payloadBytes))
		req.Header.Set("apikey", anonKey)
		req.Header.Set("Content-Type", "application/json")

		authResp, err := client.Do(req)
		if err != nil {
			fmt.Printf("   [%s] error: %v\n", tc.email, err)
			continue
		}
		body, _ := io.ReadAll(authResp.Body)
		authResp.Body.Close()
		fmt.Printf("   [%s | %s] Status: %d\n", tc.email, tc.password, authResp.StatusCode)
		if authResp.StatusCode == 200 {
			var authData struct {
				AccessToken string `json:"access_token"`
			}
			json.Unmarshal(body, &authData)
			accessToken = authData.AccessToken
			fmt.Printf("   => SUCCESS! Access Token acquired for %s (len=%d)\n", tc.email, len(accessToken))
			break
		} else {
			fmt.Printf("   Response: %s\n", string(body))
		}
	}

	if accessToken != "" {
		// Parse and print token claims
		token, _, err := new(jwt.Parser).ParseUnverified(accessToken, jwt.MapClaims{})
		if err == nil {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				fmt.Println("\n4. Token Claims Inspection:")
				fmt.Printf("   aud: %v (type: %T)\n", claims["aud"], claims["aud"])
				fmt.Printf("   iss: %v\n", claims["iss"])
				fmt.Printf("   sub: %v\n", claims["sub"])
				fmt.Printf("   exp: %v (now: %v)\n", claims["exp"], time.Now().Unix())
				fmt.Printf("   alg header: %v\n", token.Header["alg"])
				fmt.Printf("   kid header: %v\n", token.Header["kid"])
			}
		}

		// Test Local API with this token
		fmt.Println("\n5. Testing Local Backend API with Token:")
		localReq, _ := http.NewRequest("GET", "http://localhost:8080/api/v1/periods/", nil)
		localReq.Header.Set("Authorization", "Bearer "+accessToken)
		localResp, err := client.Do(localReq)
		if err != nil {
			fmt.Printf("   Local API error: %v\n", err)
		} else {
			body, _ := io.ReadAll(localResp.Body)
			localResp.Body.Close()
			fmt.Printf("   Local API Status: %d | Body: %s\n", localResp.StatusCode, string(body))
		}

		// Test Render API with this token
		fmt.Println("\n6. Testing Live Render API with Token:")
		apiReq, _ := http.NewRequest("GET", "https://muhasebe-ve-finans-otomasyonu-2.onrender.com/api/v1/periods/", nil)
		apiReq.Header.Set("Authorization", "Bearer "+accessToken)
		apiResp, err := client.Do(apiReq)
		if err != nil {
			fmt.Printf("   Render API error: %v\n", err)
		} else {
			body, _ := io.ReadAll(apiResp.Body)
			apiResp.Body.Close()
			fmt.Printf("   Render API Status: %d | Body: %s\n", apiResp.StatusCode, string(body))
		}
	}
}

