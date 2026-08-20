package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler/middleware"
)

type TransactionHandler struct {
	service domain.TransactionService
	txRepo  domain.TransactionRepository
}

// NewTransactionHandler initializes Fiber HTTP handler for Transaction operations.
func NewTransactionHandler(service domain.TransactionService, txRepo domain.TransactionRepository) *TransactionHandler {
	return &TransactionHandler{
		service: service,
		txRepo:  txRepo,
	}
}

func (h *TransactionHandler) CreateTransaction(c *fiber.Ctx) error {
	var req CreateTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Geçersiz istek gövdesi")
	}

	tenantIDVal := c.Locals(middleware.LocalTenantIDKey)
	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok || tenantID == uuid.Nil {
		return domain.ErrUnauthorized
	}

	userIDVal := c.Locals(middleware.LocalUserIDKey)
	userID, ok := userIDVal.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return domain.ErrUnauthorized
	}

	tx := &domain.Transaction{
		ID:          uuid.New(),
		TenantID:    tenantID,
		PeriodID:    req.PeriodID,
		Direction:   req.Direction,
		Channel:     req.Channel,
		Amount:      req.Amount,
		Description: req.Description,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
	}

	if err := h.service.CreateTransaction(c.UserContext(), tx); err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(ResponseEnvelope{
		Success: true,
		Data:    tx,
	})
}

func (h *TransactionHandler) ReverseTransaction(c *fiber.Ctx) error {
	idParam := c.Params("id")
	origID, err := uuid.Parse(idParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Geçersiz işlem ID formatı")
	}

	var req ReverseTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Geçersiz istek gövdesi")
	}

	userIDVal := c.Locals(middleware.LocalUserIDKey)
	userID, ok := userIDVal.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return domain.ErrUnauthorized
	}

	revTx, err := h.service.ReverseTransaction(c.UserContext(), origID, req.Reason, userID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    revTx,
	})
}

func (h *TransactionHandler) ListTransactions(c *fiber.Ctx) error {
	idParam := c.Params("id")
	periodID, err := uuid.Parse(idParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Geçersiz dönem ID formatı")
	}

	transactions, err := h.txRepo.GetByPeriodID(c.UserContext(), periodID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    transactions,
	})
}
