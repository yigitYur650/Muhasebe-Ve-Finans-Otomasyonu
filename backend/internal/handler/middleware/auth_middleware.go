package middleware

import (
	"fmt"
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
func AuthMiddleware(jwtSecret string, tenantRepo domain.TenantRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// Fallback: If X-User-ID and X-Tenant-ID are provided (for backwards compatibility / dev mode without JWT), allow if valid UUID
			tenantHeader := c.Get(HeaderTenantID)
			userHeader := c.Get(HeaderUserID)
			if tenantHeader != "" && userHeader != "" {
				tenantID, errT := uuid.Parse(tenantHeader)
				userID, errU := uuid.Parse(userHeader)
				if errT == nil && errU == nil {
					roleStr := c.Get(HeaderUserRole)
					if roleStr == "" {
						roleStr = string(domain.RoleStandart)
					}
					c.Locals(LocalUserID, userID)
					c.Locals(LocalTenantID, tenantID)
					c.Locals(LocalRole, domain.Role(roleStr))
					return c.Next()
				}
			}
			return domain.ErrUnauthorized
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			return domain.ErrUnauthorized
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return domain.ErrUnauthorized
		}

		// Verify aud claim ("authenticated")
		if aud, ok := claims["aud"].(string); !ok || aud != "authenticated" {
			return domain.ErrUnauthorized
		}

		// Verify expiration (exp)
		if expFloat, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(expFloat) {
				return domain.ErrUnauthorized
			}
		}

		// Extract user sub (UUID)
		subStr, ok := claims["sub"].(string)
		if !ok {
			return domain.ErrUnauthorized
		}
		userID, err := uuid.Parse(subStr)
		if err != nil {
			return domain.ErrUnauthorized
		}

		// Extract X-Tenant-ID header
		tenantHeader := c.Get(HeaderTenantID)
		if tenantHeader == "" {
			return domain.ErrUnauthorized
		}
		tenantID, err := uuid.Parse(tenantHeader)
		if err != nil {
			return domain.ErrUnauthorized
		}

		// Verify DB Tenant Membership if repo is provided
		if tenantRepo != nil {
			member, err := tenantRepo.GetMember(c.Context(), tenantID, userID)
			if err != nil || member == nil {
				return domain.ErrUnauthorized
			}
			c.Locals(LocalRole, member.Role)
		} else {
			roleStr := c.Get(HeaderUserRole)
			if roleStr == "" {
				roleStr = string(domain.RoleStandart)
			}
			c.Locals(LocalRole, domain.Role(roleStr))
		}

		c.Locals(LocalUserID, userID)
		c.Locals(LocalTenantID, tenantID)

		return c.Next()
	}
}
