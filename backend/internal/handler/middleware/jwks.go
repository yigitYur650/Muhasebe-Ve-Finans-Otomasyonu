package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// JWK represents a cryptographic key in JSON Web Key format (RFC 7517)
type JWK struct {
	Alg string `json:"alg"`
	Crv string `json:"crv"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	X   string `json:"x"`
	Y   string `json:"y"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKSResponse represents the Supabase JWKS endpoint payload
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWKSCache holds parsed public keys in memory with thread-safe refresh capabilities
type JWKSCache struct {
	mu        sync.RWMutex
	keys      map[string]interface{} // kid -> *ecdsa.PublicKey or *rsa.PublicKey
	lastFetch time.Time
	jwksURL   string
	client    *http.Client
}

var (
	globalJWKSCache *JWKSCache
	jwksOnce        sync.Once
)

// GetJWKSCache returns the singleton JWKS cache initialized with Supabase endpoint
func GetJWKSCache() *JWKSCache {
	jwksOnce.Do(func() {
		jwksURL := os.Getenv("SUPABASE_JWKS_URL")
		if jwksURL == "" {
			supaURL := os.Getenv("SUPABASE_URL")
			if supaURL == "" {
				supaURL = os.Getenv("NEXT_PUBLIC_SUPABASE_URL")
			}
			if supaURL != "" {
				jwksURL = strings.TrimRight(supaURL, "/") + "/auth/v1/.well-known/jwks.json"
			}
		}
		globalJWKSCache = &JWKSCache{
			keys:    make(map[string]interface{}),
			jwksURL: jwksURL,
			client:  &http.Client{Timeout: 6 * time.Second},
		}
	})
	return globalJWKSCache
}

// GetKey retrieves a public key by key ID (kid) or returns the single matching key if only one exists
func (c *JWKSCache) GetKey(kid string, alg string) (interface{}, error) {
	c.mu.RLock()
	key, exists := c.keys[kid]
	recent := time.Since(c.lastFetch) < 1*time.Hour
	keyCount := len(c.keys)
	c.mu.RUnlock()

	if exists && recent {
		return key, nil
	}

	// Fetch or refresh keys
	if err := c.refresh(); err != nil {
		if exists {
			return key, nil // return cached key if refresh fails
		}
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	key, exists = c.keys[kid]
	if !exists {
		if kid == "" && len(c.keys) == 1 {
			for _, k := range c.keys {
				return k, nil
			}
		}
		if keyCount > 0 {
			// If kid is not found but there are keys available, return first matching key
			for _, k := range c.keys {
				return k, nil
			}
		}
		return nil, fmt.Errorf("public key with kid '%s' (alg: %s) not found in JWKS", kid, alg)
	}
	return key, nil
}

func (c *JWKSCache) refresh() error {
	if c.jwksURL == "" {
		// Try resolving dynamic URL on refresh
		supaURL := os.Getenv("SUPABASE_URL")
		if supaURL == "" {
			supaURL = os.Getenv("NEXT_PUBLIC_SUPABASE_URL")
		}
		if supaURL != "" {
			c.jwksURL = strings.TrimRight(supaURL, "/") + "/auth/v1/.well-known/jwks.json"
		} else {
			return fmt.Errorf("no SUPABASE_URL or SUPABASE_JWKS_URL configured")
		}
	}

	req, err := http.NewRequest(http.MethodGet, c.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create JWKS request: %w", err)
	}
	req.Header.Set("User-Agent", "DefterSystem-Backend/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS from %s: %w", c.jwksURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint %s returned status %d", c.jwksURL, resp.StatusCode)
	}

	var jwks JWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS JSON: %w", err)
	}

	newKeys := make(map[string]interface{})
	for _, k := range jwks.Keys {
		if k.Kty == "EC" && (k.Crv == "P-256" || k.Alg == "ES256") {
			xBytes, errX := base64.RawURLEncoding.DecodeString(k.X)
			yBytes, errY := base64.RawURLEncoding.DecodeString(k.Y)
			if errX == nil && errY == nil {
				pubKey := &ecdsa.PublicKey{
					Curve: elliptic.P256(),
					X:     new(big.Int).SetBytes(xBytes),
					Y:     new(big.Int).SetBytes(yBytes),
				}
				newKeys[k.Kid] = pubKey
			}
		} else if k.Kty == "RSA" || strings.HasPrefix(k.Alg, "RS") {
			nBytes, errN := base64.RawURLEncoding.DecodeString(k.N)
			eBytes, errE := base64.RawURLEncoding.DecodeString(k.E)
			if errN == nil && errE == nil {
				eInt := 0
				for _, b := range eBytes {
					eInt = (eInt << 8) | int(b)
				}
				pubKey := &rsa.PublicKey{
					N: new(big.Int).SetBytes(nBytes),
					E: eInt,
				}
				newKeys[k.Kid] = pubKey
			}
		}
	}

	if len(newKeys) == 0 {
		return fmt.Errorf("no valid EC or RSA public keys found in JWKS response")
	}

	c.mu.Lock()
	c.keys = newKeys
	c.lastFetch = time.Now()
	c.mu.Unlock()

	log.Printf("🔐 [AUTH JWKS] Successfully loaded %d public key(s) from Supabase JWKS", len(newKeys))
	return nil
}
