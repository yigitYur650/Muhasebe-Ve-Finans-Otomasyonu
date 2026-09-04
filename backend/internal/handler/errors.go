package handler

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	"deftersystem/backend/internal/domain"
)

// CustomErrorHandler translates domain and framework errors into standardized JSON responses.
func CustomErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	errCode := "INTERNAL_SERVER_ERROR"
	errMsg := "Sunucu tarafında beklenmeyen bir hata oluştu"

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code = fiberErr.Code
		errMsg = fiberErr.Message
		errCode = "HTTP_ERROR"
	} else {
		switch {
		case errors.Is(err, domain.ErrNotFound), errors.Is(err, domain.ErrTransactionNotFound), errors.Is(err, domain.ErrPeriodNotFound), errors.Is(err, domain.ErrTenantNotFound):
			code = fiber.StatusNotFound
			errCode = "NOT_FOUND"
			errMsg = err.Error()

		case errors.Is(err, domain.ErrUnauthorized):
			code = fiber.StatusForbidden
			errCode = "UNAUTHORIZED"
			errMsg = err.Error()

		case errors.Is(err, domain.ErrPeriodLocked):
			code = fiber.StatusUnprocessableEntity
			errCode = "PERIOD_LOCKED"
			errMsg = err.Error()

		case errors.Is(err, domain.ErrTransactionAlreadyReversed):
			code = fiber.StatusConflict
			errCode = "TRANSACTION_ALREADY_REVERSED"
			errMsg = err.Error()

		case errors.Is(err, domain.ErrDuplicateIdempotencyKey):
			code = fiber.StatusConflict
			errCode = "DUPLICATE_IDEMPOTENCY_KEY"
			errMsg = err.Error()

		case errors.Is(err, domain.ErrInvalidAmount):
			code = fiber.StatusBadRequest
			errCode = "INVALID_AMOUNT"
			errMsg = err.Error()

		case errors.Is(err, domain.ErrInvalidDirection), errors.Is(err, domain.ErrInvalidChannel), errors.Is(err, domain.ErrInvalidRole):
			code = fiber.StatusBadRequest
			errCode = "INVALID_INPUT"
			errMsg = err.Error()

		default:
			log.Printf("[ERROR] Internal unhandled error: %v", err)
		}
	}

	return c.Status(code).JSON(ResponseEnvelope{
		Success: false,
		Error: &ErrorData{
			Code:    errCode,
			Message: errMsg,
		},
	})
}
