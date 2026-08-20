package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"deftersystem/backend/internal/domain"
)

const HeaderIdempotencyKey = "Idempotency-Key"

// IdempotencyMiddleware ensures duplicate mutation requests with the same Idempotency-Key return cached responses.
func IdempotencyMiddleware(idemRepo domain.IdempotencyRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Idempotency applies to state-modifying HTTP methods
		method := c.Method()
		if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch {
			return c.Next()
		}

		key := c.Get(HeaderIdempotencyKey)
		if key == "" {
			return c.Next()
		}

		ctx := c.UserContext()

		// Check if key already exists in repository
		cached, err := idemRepo.Get(ctx, key)
		if err == nil && cached != nil {
			// Cache hit: return stored status code and response body directly
			c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			c.Set("X-Cache-Lookup", "HIT")
			return c.Status(cached.ResponseStatus).Send(cached.ResponseBody)
		}

		// Cache miss: execute actual request handler
		err = c.Next()
		if err != nil {
			return err
		}

		// Extract tenantID from context locals
		tenantIDVal := c.Locals(LocalTenantIDKey)
		tenantID := uuid.Nil
		if tid, ok := tenantIDVal.(uuid.UUID); ok {
			tenantID = tid
		}

		// Save response in repository
		responseStatus := c.Response().StatusCode()
		responseBody := c.Response().Body()

		if responseStatus < 500 {
			idemKey := &domain.IdempotencyKey{
				Key:            key,
				TenantID:       tenantID,
				ResponseBody:   json.RawMessage(responseBody),
				ResponseStatus: responseStatus,
			}
			_ = idemRepo.Save(ctx, idemKey)
		}

		return nil
	}
}
