package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler/middleware"
)

type PeriodHandler struct {
	service domain.PeriodService
}

// NewPeriodHandler initializes Fiber HTTP handler for Period operations.
func NewPeriodHandler(service domain.PeriodService) *PeriodHandler {
	return &PeriodHandler{service: service}
}

func (h *PeriodHandler) OpenNextPeriod(c *fiber.Ctx) error {
	var req OpenPeriodRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Geçersiz istek gövdesi")
	}

	tenantIDVal := c.Locals(middleware.LocalTenantIDKey)
	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok || tenantID == uuid.Nil {
		return domain.ErrUnauthorized
	}

	period, err := h.service.OpenNextPeriod(c.UserContext(), tenantID, req.Label)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(ResponseEnvelope{
		Success: true,
		Data:    period,
	})
}

func (h *PeriodHandler) LockPeriod(c *fiber.Ctx) error {
	idParam := c.Params("id")
	periodID, err := uuid.Parse(idParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Geçersiz dönem ID formatı")
	}

	userIDVal := c.Locals(middleware.LocalUserIDKey)
	userID, ok := userIDVal.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return domain.ErrUnauthorized
	}

	if err := h.service.LockPeriod(c.UserContext(), periodID, userID); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    fiber.Map{"message": "Dönem başarıyla kilitlendi"},
	})
}

func (h *PeriodHandler) GetPeriodSummary(c *fiber.Ctx) error {
	idParam := c.Params("id")
	periodID, err := uuid.Parse(idParam)
	if err != nil {
		// Non-UUID label fallback lookup
		tenantID, tErr := getTenantIDFromLocals(c)
		if tErr == nil {
			latest, pErr := h.service.ListPeriods(c.Context(), tenantID)
			if pErr == nil && len(latest) > 0 {
				periodID = latest[0].ID
			} else {
				return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
					Success: true,
					Data: domain.PeriodSummary{
						StartingBalance: decimal.Zero,
						TotalIn:         decimal.Zero,
						TotalOut:        decimal.Zero,
						ClosingBalance:  decimal.Zero,
					},
				})
			}
		} else {
			return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
				Success: true,
				Data: domain.PeriodSummary{
					StartingBalance: decimal.Zero,
					TotalIn:         decimal.Zero,
					TotalOut:        decimal.Zero,
					ClosingBalance:  decimal.Zero,
				},
			})
		}
	}

	summary, err := h.service.GetPeriodSummary(c.UserContext(), periodID)
	if err != nil {
		return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
			Success: true,
			Data: domain.PeriodSummary{
				StartingBalance: decimal.Zero,
				TotalIn:         decimal.Zero,
				TotalOut:        decimal.Zero,
				ClosingBalance:  decimal.Zero,
			},
		})
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    summary,
	})
}

func (h *PeriodHandler) ListPeriods(c *fiber.Ctx) error {
	tenantID, err := getTenantIDFromLocals(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	periods, err := h.service.ListPeriods(c.Context(), tenantID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    periods,
	})
}

func (h *PeriodHandler) GetPeriodHistory(c *fiber.Ctx) error {
	tenantID, err := getTenantIDFromLocals(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	history, err := h.service.GetPeriodHistory(c.Context(), tenantID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    history,
	})
}
