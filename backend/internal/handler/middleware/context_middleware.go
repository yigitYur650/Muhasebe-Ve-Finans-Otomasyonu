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

// GetTenantID retrieves the parsed and verified tenant UUID from context locals.
func GetTenantID(c *fiber.Ctx) uuid.UUID {
	if val := c.Locals(LocalTenantIDKey); val != nil {
		if id, ok := val.(uuid.UUID); ok {
			return id
		}
	}
	if val := c.Locals(LocalTenantID); val != nil {
		if id, ok := val.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

// GetUserID retrieves the parsed and verified user UUID from context locals.
func GetUserID(c *fiber.Ctx) uuid.UUID {
	if val := c.Locals(LocalUserIDKey); val != nil {
		if id, ok := val.(uuid.UUID); ok {
			return id
		}
	}
	if val := c.Locals(LocalUserID); val != nil {
		if id, ok := val.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

