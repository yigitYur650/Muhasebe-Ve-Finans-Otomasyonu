package handler

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/pkg/telegram"
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
		case errors.Is(err, domain.ErrNotFound), errors.Is(err, domain.ErrTransactionNotFound), errors.Is(err, domain.ErrPeriodNotFound), errors.Is(err, domain.ErrTenantNotFound), errors.Is(err, domain.ErrSupplierNotFound):
			code = fiber.StatusNotFound
			errCode = "NOT_FOUND"
			errMsg = err.Error()

		case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrForbidden):
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

		case errors.Is(err, domain.ErrDuplicateIdempotencyKey), errors.Is(err, domain.ErrDuplicateSupplierName):
			code = fiber.StatusConflict
			errCode = "DUPLICATE_KEY"
			errMsg = err.Error()

		case errors.Is(err, domain.ErrPeriodAlreadyExists):
			code = fiber.StatusConflict
			errCode = "PERIOD_ALREADY_EXISTS"
			errMsg = "Bu dönem etiketi zaten mevcut. Lütfen farklı bir dönem adı belirleyin."

		case errors.Is(err, domain.ErrInvalidAmount):
			code = fiber.StatusBadRequest
			errCode = "INVALID_AMOUNT"
			errMsg = err.Error()

		case errors.Is(err, domain.ErrInvalidDirection), errors.Is(err, domain.ErrInvalidChannel), errors.Is(err, domain.ErrInvalidRole), errors.Is(err, domain.ErrInvalidSupplierDirection):
			code = fiber.StatusBadRequest
			errCode = "INVALID_INPUT"
			errMsg = err.Error()

		default:
			log.Printf("[ERROR] Internal unhandled error: %v", err)
			// Trigger Telegram alert for critical 500 internal server errors
			tenantID, _ := c.Locals("tenant_id").(string)
			telegram.Global().SendAlert("SUNUCU HATASI (500)", err.Error(), c.Method(), c.Path(), tenantID, c.IP(), "")
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

