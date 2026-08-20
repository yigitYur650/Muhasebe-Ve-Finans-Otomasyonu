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
// Security Hardening: Performs composite key lookup (key, tenant_id) and strictly prevents caching 5xx server errors.
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

		// Extract tenantID from context locals or header for composite lookup
		tenantIDVal := c.Locals(LocalTenantIDKey)
		tenantID := uuid.Nil
		if tid, ok := tenantIDVal.(uuid.UUID); ok {
			tenantID = tid
		}
		if tenantID == uuid.Nil {
			if tHeader := c.Get(HeaderTenantID); tHeader != "" {
				if parsed, err := uuid.Parse(tHeader); err == nil {
					tenantID = parsed
				}
			}
		}

		ctx := c.UserContext()

		// Check if key already exists for this tenant in repository (composite key lookup)
		cached, err := idemRepo.Get(ctx, key, tenantID)
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

		// Save response in repository ONLY IF status code is successful/controlled (< 500)
		responseStatus := c.Response().StatusCode()
		responseBody := c.Response().Body()

		if responseStatus >= 200 && responseStatus < 500 {
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
