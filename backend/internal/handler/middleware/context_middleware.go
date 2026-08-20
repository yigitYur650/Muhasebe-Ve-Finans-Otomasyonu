package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	LocalTenantIDKey = "tenant_id"
	LocalUserIDKey   = "user_id"
	LocalUserRoleKey = "user_role"

	HeaderTenantID = "X-Tenant-ID"
	HeaderUserID   = "X-User-ID"
	HeaderUserRole = "X-User-Role"
)

// ContextMiddleware parses tenant, user, and role headers into Fiber context locals safely.
func ContextMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantHeader := c.Get(HeaderTenantID)
		userHeader := c.Get(HeaderUserID)
		roleHeader := c.Get(HeaderUserRole)

		if tenantHeader != "" {
			if tenantID, err := uuid.Parse(tenantHeader); err == nil {
				c.Locals(LocalTenantIDKey, tenantID)
			}
		}

		if userHeader != "" {
			if userID, err := uuid.Parse(userHeader); err == nil {
				c.Locals(LocalUserIDKey, userID)
			}
		}

		if roleHeader != "" {
			c.Locals(LocalUserRoleKey, roleHeader)
		}

		return c.Next()
	}
}
