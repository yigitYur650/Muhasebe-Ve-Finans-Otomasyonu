package handler

import (
	"bytes"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler/middleware"
)

type ImportHandler struct {
	importSvc domain.ImportService
}

func NewImportHandler(importSvc domain.ImportService) *ImportHandler {
	return &ImportHandler{
		importSvc: importSvc,
	}
}

func (h *ImportHandler) ImportTransactionsCSV(c *fiber.Ctx) error {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	periodIDParam := c.Params("id")
	periodID, err := uuid.Parse(periodIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
			Success: false,
			Error: &ErrorData{
				Code:    "INVALID_ID",
				Message: "Geçersiz dönem kimliği",
			},
		})
	}

	// 1. Try reading multipart form file "file"
	file, fileErr := c.FormFile("file")
	var csvData []byte

	if fileErr == nil && file != nil {
		src, err := file.Open()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
				Success: false,
				Error: &ErrorData{
					Code:    "FILE_READ_ERROR",
					Message: "Dosya okunamadı",
				},
			})
		}
		defer src.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(src); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
				Success: false,
				Error: &ErrorData{
					Code:    "FILE_READ_ERROR",
					Message: "Yüklenen dosya okunamadı",
				},
			})
		}
		csvData = buf.Bytes()
	} else {
		// Fallback: Read raw request body
		csvData = c.Body()
	}

	if len(csvData) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
			Success: false,
			Error: &ErrorData{
				Code:    "EMPTY_FILE",
				Message: "Yüklenen CSV dosyası veya içerik boş",
			},
		})
	}

	// 2. Call ImportService
	result, err := h.importSvc.ImportTransactionsFromCSV(c.Context(), tenantID, periodID, bytes.NewReader(csvData), userID)
	if err != nil {
		if err == domain.ErrPeriodLocked {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(ResponseEnvelope{
				Success: false,
				Error: &ErrorData{
					Code:    "PERIOD_LOCKED",
					Message: "Kilitli döneme toplu işlem aktarılamaz",
				},
			})
		}

		return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
			Success: false,
			Error: &ErrorData{
				Code:    "IMPORT_VALIDATION_ERROR",
				Message: err.Error(),
			},
		})
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data: fiber.Map{
			"imported_count": result.ImportedCount,
			"total_amount":   result.TotalAmount.StringFixed(2),
		},
	})
}
