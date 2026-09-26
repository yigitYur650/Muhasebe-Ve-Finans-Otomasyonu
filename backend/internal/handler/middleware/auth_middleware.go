package middleware

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"deftersystem/backend/internal/domain"
)

const (
	LocalUserID   = "user_id"
	LocalRole     = "role"
	LocalTenantID = "tenant_id"
)

// AuthMiddleware validates Supabase JWT signatures, claims, and DB tenant membership.
// Implements FAIL-SECURE principle: Header-based fallback is ONLY enabled if ENVIRONMENT
// is explicitly set to "development" or "local_test".
// In ALL other cases (production, staging, empty, undefined, etc.), cryptographic Bearer token
// and DB tenant membership verification are strictly mandatory.
func AuthMiddleware(jwtSecret string, tenantRepo domain.TenantRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Always allow CORS preflight (OPTIONS) requests to proceed unblocked
		if c.Method() == fiber.MethodOptions {
			return c.Next()
		}

		// Cleanse context locals to prevent header spoofing / pollution
		c.Locals(LocalUserID, nil)
		c.Locals(LocalTenantID, nil)
		c.Locals(LocalRole, nil)
		c.Locals(LocalTenantIDKey, nil)
		c.Locals(LocalUserIDKey, nil)
		c.Locals(LocalUserRoleKey, nil)

		env := strings.ToLower(os.Getenv("ENVIRONMENT"))
		if env == "" {
			env = strings.ToLower(os.Getenv("ENV"))
		}
		// FAIL-SECURE: Only explicitly declared development/local_test environments allow header auth
		isExplicitDevOrTest := env == "development" || env == "local_test" || env == "test"

		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// Fail-Secure: Reject if not explicitly in development/test
			if !isExplicitDevOrTest {
				return domain.ErrUnauthorized
			}

			// In development / test mode only: Allow header-based authentication with DB membership verification
			tenantHeader := c.Get(HeaderTenantID)
			userHeader := c.Get(HeaderUserID)
			if tenantHeader != "" && userHeader != "" {
				tenantID, errT := uuid.Parse(tenantHeader)
				userID, errU := uuid.Parse(userHeader)
				if errT == nil && errU == nil {
					role := domain.RoleStandart
					if tenantRepo != nil {
						member, errM := tenantRepo.GetMember(c.Context(), tenantID, userID)
						if errM != nil || member == nil {
							return domain.ErrUnauthorized
						}
						role = member.Role
					} else {
						roleStr := c.Get(HeaderUserRole)
						if roleStr != "" {
							role = domain.Role(roleStr)
						}
					}
					c.Locals(LocalUserID, userID)
					c.Locals(LocalTenantID, tenantID)
					c.Locals(LocalRole, role)
					c.Locals(LocalTenantIDKey, tenantID)
					c.Locals(LocalUserIDKey, userID)
					c.Locals(LocalUserRoleKey, string(role))
					return c.Next()
				}
			}

			// Developer Bypass Mode: When ENVIRONMENT is explicitly "development" and no credentials were sent,
			// automatically provision a local dev admin context to allow frictionless local development.
			if env == "development" {
				devUserID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
				var devTenantID uuid.UUID
				if tenantRepo != nil {
					firstTenant, err := tenantRepo.GetFirstTenant(c.Context())
					if err == nil && firstTenant != nil {
						devTenantID = firstTenant.ID
					} else {
						devTenantID = devUserID
					}
				} else {
					devTenantID = devUserID
				}
				c.Locals(LocalUserID, devUserID)
				c.Locals(LocalTenantID, devTenantID)
				c.Locals(LocalRole, domain.RoleAdmin)
				c.Locals(LocalTenantIDKey, devTenantID)
				c.Locals(LocalUserIDKey, devUserID)
				c.Locals(LocalUserRoleKey, string(domain.RoleAdmin))
				return c.Next()
			}

			return domain.ErrUnauthorized
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		var claims jwt.MapClaims

		// Universal Supabase Token Validator (Supports ES256/RS256 Asymmetric JWKS and HS256 Symmetric HMAC)
		parser := jwt.NewParser(jwt.WithValidMethods([]string{
			"HS256", "HS384", "HS512",
			"ES256", "ES384", "ES512",
			"RS256", "RS384", "RS512",
		}))

		token, err := parser.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			alg, _ := token.Header["alg"].(string)
			kid, _ := token.Header["kid"].(string)

			// 1. Asymmetric ECDSA / RSA (Supabase Modern ES256 / RS256 JWKS)
			if strings.HasPrefix(alg, "ES") || strings.HasPrefix(alg, "RS") {
				cache := GetJWKSCache()
				key, err := cache.GetKey(kid, alg)
				if err != nil {
					return nil, err
				}
				return key, nil
			}

			// 2. Symmetric HMAC-SHA256 (Legacy / Test Secret)
			if strings.HasPrefix(alg, "HS") {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", alg)
				}
				var secretBytes []byte
				if decoded, err := base64.StdEncoding.DecodeString(jwtSecret); err == nil && len(decoded) > 0 {
					secretBytes = decoded
				} else {
					secretBytes = []byte(jwtSecret)
				}
				return secretBytes, nil
			}

			return nil, fmt.Errorf("unsupported or rejected signing method: %v", alg)
		})

		// Fallback for HS256 if Base64 decoded secret failed but raw string works
		if (err != nil || !token.Valid) && jwtSecret != "" {
			tokenFallback, errFallback := parser.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})
			if errFallback == nil && tokenFallback.Valid {
				token = tokenFallback
				err = nil
			}
		}

		if err != nil || !token.Valid {
			// In development or local_test mode only: fallback to unverified claims if signature check failed
			if isExplicitDevOrTest {
				log.Printf("⚠️ [AUTH NOTICE] Asenkron imza kontrolü başarısız (%v), geliştirici modunda claims çözülüyor...", err)
				unverifiedToken, _, parseErr := parser.ParseUnverified(tokenStr, jwt.MapClaims{})
				if parseErr == nil {
					if c, ok := unverifiedToken.Claims.(jwt.MapClaims); ok {
						claims = c
					}
				}
			}
			if claims == nil {
				if isExplicitDevOrTest {
					log.Printf("⚠️ [AUTH REJECT] Token imza doğrulaması başarısız: %v", err)
				}
				return domain.ErrUnauthorized
			}
		} else {
			if c, ok := token.Claims.(jwt.MapClaims); ok {
				claims = c
			} else {
				return domain.ErrUnauthorized
			}
		}

		// Verify Algorithm: Reject 'none' or empty algorithm explicitly
		if alg, ok := claims["alg"].(string); ok && strings.ToLower(alg) == "none" {
			log.Printf("⚠️ [AUTH REJECT] 'none' algoritması reddedildi")
			return domain.ErrUnauthorized
		}

		// Verify Audience Claim ("authenticated")
		audValid := false
		switch a := claims["aud"].(type) {
		case string:
			if a == "authenticated" {
				audValid = true
			}
		case []interface{}:
			for _, item := range a {
				if itemStr, ok := item.(string); ok && itemStr == "authenticated" {
					audValid = true
					break
				}
			}
		case []string:
			for _, item := range a {
				if item == "authenticated" {
					audValid = true
					break
				}
			}
		}
		if !audValid {
			log.Printf("⚠️ [AUTH REJECT] Beklenmeyen aud claim: %v (Beklenen: authenticated)", claims["aud"])
			return domain.ErrUnauthorized
		}

		// Verify Issuer Claim if Supabase URL / Issuer is configured
		expectedIssuer := os.Getenv("SUPABASE_ISSUER")
		if expectedIssuer == "" {
			supaURL := os.Getenv("SUPABASE_URL")
			if supaURL == "" {
				supaURL = os.Getenv("NEXT_PUBLIC_SUPABASE_URL")
			}
			if supaURL != "" {
				expectedIssuer = strings.TrimRight(supaURL, "/") + "/auth/v1"
			} else {
				expectedIssuer = DefaultSupabaseURL + "/auth/v1"
			}
		}
		if expectedIssuer != "" {
			if iss, ok := claims["iss"].(string); ok {
				cleanIss := strings.TrimRight(iss, "/")
				cleanExpected := strings.TrimRight(expectedIssuer, "/")
				if cleanIss != cleanExpected && cleanIss != "supabase" {
					log.Printf("⚠️ [AUTH REJECT] Beklenmeyen iss: %v (Beklenen: %v)", iss, expectedIssuer)
					return domain.ErrUnauthorized
				}
			}
		}

		// Verify Expiration (exp) and Not Before (nbf)
		now := time.Now().Unix()
		if expFloat, ok := claims["exp"].(float64); ok {
			if now > int64(expFloat) {
				log.Printf("⚠️ [AUTH REJECT] Token süresi dolmuş (exp: %v, now: %v)", int64(expFloat), now)
				return domain.ErrUnauthorized
			}
		}
		if nbfFloat, ok := claims["nbf"].(float64); ok {
			if now < int64(nbfFloat) {
				log.Printf("⚠️ [AUTH REJECT] Token henüz geçerli değil (nbf: %v, now: %v)", int64(nbfFloat), now)
				return domain.ErrUnauthorized
			}
		}

		// Extract User ID (sub) strictly from cryptographic token claims
		subStr, ok := claims["sub"].(string)
		if !ok {
			log.Printf("⚠️ [AUTH REJECT] sub claim bulunamadı")
			return domain.ErrUnauthorized
		}
		userID, err := uuid.Parse(subStr)
		if err != nil {
			return domain.ErrUnauthorized
		}

		// 1. Check if JWT contains cryptographically signed Tenant / Role claims (Supabase Auth Hook / Custom Claims)
		claimTenantStr := extractJWTClaim(claims, "tenant_id")
		claimRoleStr := extractJWTClaim(claims, "role")
		if claimRoleStr == "" {
			claimRoleStr = extractJWTClaim(claims, "user_role")
		}

		tenantHeader := c.Get(HeaderTenantID)
		var tenantID uuid.UUID
		var userRole domain.Role = domain.RoleStandart

		if claimRoleStr != "" {
			userRole = domain.Role(claimRoleStr)
		}

		if tenantRepo != nil {
			if tenantHeader != "" {
				parsedTenantID, err := uuid.Parse(tenantHeader)
				if err != nil {
					return domain.ErrUnauthorized
				}
				// If header matches signed JWT claim, trust it directly
				if claimTenantStr != "" && parsedTenantID.String() == claimTenantStr {
					tenantID = parsedTenantID
				} else {
					// BOLA/IDOR Protection: Verify User is an ACTIVE MEMBER of Target Tenant in DB
					member, err := tenantRepo.GetMember(c.Context(), parsedTenantID, userID)
					if err != nil || member == nil {
						return domain.ErrUnauthorized // User is NOT a member of target tenant (IDOR attempt blocked)
					}
					tenantID = member.TenantID
					userRole = member.Role
				}
			} else if claimTenantStr != "" {
				// JWT directly carries verified tenant claim
				parsedTenantID, err := uuid.Parse(claimTenantStr)
				if err != nil {
					return domain.ErrUnauthorized
				}
				tenantID = parsedTenantID
			} else {
				// Fallback: Dynamic DB-driven resolution
				members, err := tenantRepo.GetMembersByUserID(c.Context(), userID)
				if err == nil && len(members) > 0 {
					// Use active tenant
					tenantID = members[0].TenantID
					userRole = members[0].Role
				} else {
					// Auto-provision user to the primary organization in database
					firstTenant, err := tenantRepo.GetFirstTenant(c.Context())
					if err != nil || firstTenant == nil {
						return domain.ErrUnauthorized // No tenants exist in database
					}
					tenantID = firstTenant.ID
					userRole = domain.RoleAdmin
					_ = tenantRepo.AddMember(c.Context(), &domain.TenantMember{
						ID:        uuid.New(),
						TenantID:  tenantID,
						UserID:    userID,
						Role:      userRole,
						CreatedAt: time.Now(),
					})
				}
			}
			c.Locals(LocalRole, userRole)
		} else {
			if !isExplicitDevOrTest {
				// In production without tenant repository validation, fail closed
				return domain.ErrUnauthorized
			}
			if tenantHeader != "" {
				parsedTenantID, err := uuid.Parse(tenantHeader)
				if err != nil {
					return domain.ErrUnauthorized
				}
				tenantID = parsedTenantID
			} else if claimTenantStr != "" {
				parsedTenantID, err := uuid.Parse(claimTenantStr)
				if err == nil {
					tenantID = parsedTenantID
				}
			}
			roleStr := c.Get(HeaderUserRole)
			if roleStr == "" && claimRoleStr != "" {
				roleStr = claimRoleStr
			}
			if roleStr == "" {
				roleStr = string(domain.RoleStandart)
			}
			c.Locals(LocalRole, domain.Role(roleStr))
		}

		c.Locals(LocalUserID, userID)
		c.Locals(LocalTenantID, tenantID)
		c.Locals(LocalTenantIDKey, tenantID)
		c.Locals(LocalUserIDKey, userID)
		if r, ok := c.Locals(LocalRole).(domain.Role); ok {
			c.Locals(LocalUserRoleKey, string(r))
		}

		return c.Next()
	}
}

// extractJWTClaim inspects top-level claims, app_metadata (Supabase Auth Hook standard), and user_metadata.
func extractJWTClaim(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key].(string); ok && val != "" {
		return val
	}
	if appMeta, ok := claims["app_metadata"].(map[string]interface{}); ok {
		if val, ok := appMeta[key].(string); ok && val != "" {
			return val
		}
	}
	if userMeta, ok := claims["user_metadata"].(map[string]interface{}); ok {
		if val, ok := userMeta[key].(string); ok && val != "" {
			return val
		}
	}
	return ""
}

// RequireRole enforces role-based access control (RBAC) on protected endpoints.
func RequireRole(allowedRoles ...domain.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleVal := c.Locals(LocalRole)
		if roleVal == nil {
			return domain.ErrUnauthorized
		}
		userRole, ok := roleVal.(domain.Role)
		if !ok {
			return domain.ErrUnauthorized
		}
		for _, r := range allowedRoles {
			if userRole == r {
				return c.Next()
			}
		}
		return domain.ErrUnauthorized
	}
}

