package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler/middleware"
)

type SupplierHandler struct {
	service domain.SupplierService
}

// NewSupplierHandler initializes Fiber HTTP handler for Supplier operations.
func NewSupplierHandler(service domain.SupplierService) *SupplierHandler {
	return &SupplierHandler{service: service}
}

// CreateSupplierRequest DTO
type CreateSupplierRequest struct {
	Name string `json:"name"`
}

// CreateSupplierTxRequest DTO
type CreateSupplierTxRequest struct {
	SupplierID     uuid.UUID       `json:"supplier_id"`
	PeriodID       uuid.UUID       `json:"period_id"`
	InvoiceNo      string          `json:"invoice_no"`
	CustomerName   string          `json:"customer_name"`
	DocumentStatus string          `json:"document_status"`
	TxDate         string          `json:"tx_date"`
	Direction      string          `json:"direction"`
	Amount         decimal.Decimal `json:"amount"`
	Description    string          `json:"description"`
}

func (h *SupplierHandler) ListSuppliers(c *fiber.Ctx) error {
	tenantID, err := getTenantIDFromCtx(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	var periodID *uuid.UUID
	periodParam := c.Query("period_id")
	if periodParam != "" {
		if pID, err := uuid.Parse(periodParam); err == nil {
			periodID = &pID
		}
	}

	suppliers, err := h.service.ListSuppliers(c.UserContext(), tenantID, periodID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    suppliers,
	})
}

func (h *SupplierHandler) GetSummary(c *fiber.Ctx) error {
	tenantID, err := getTenantIDFromCtx(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	var periodID *uuid.UUID
	periodParam := c.Query("period_id")
	if periodParam != "" {
		if pID, err := uuid.Parse(periodParam); err == nil {
			periodID = &pID
		}
	}

	summary, err := h.service.GetSummary(c.UserContext(), tenantID, periodID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    summary,
	})
}

func (h *SupplierHandler) CreateSupplier(c *fiber.Ctx) error {
	tenantID, err := getTenantIDFromCtx(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	var req CreateSupplierRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Geçersiz istek gövdesi")
	}

	supplier, err := h.service.CreateSupplier(c.UserContext(), tenantID, req.Name)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(ResponseEnvelope{
		Success: true,
		Data:    supplier,
	})
}

func (h *SupplierHandler) ListSupplierTransactions(c *fiber.Ctx) error {
	tenantID, err := getTenantIDFromCtx(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	supplierIDParam := c.Params("id")
	supplierID, err := uuid.Parse(supplierIDParam)
	if err != nil {
		return domain.ErrNotFound
	}

	filter := parseSupplierTxFilter(c)
	filter.SupplierID = &supplierID

	txs, total, err := h.service.ListSupplierTransactions(c.UserContext(), tenantID, supplierID, filter)
	if err != nil {
		return err
	}

	c.Set("X-Total-Count", strconv.Itoa(total))
	page := 1
	if filter.Limit > 0 {
		page = (filter.Offset / filter.Limit) + 1
	}
	c.Set("X-Page", strconv.Itoa(page))
	c.Set("X-Limit", strconv.Itoa(filter.Limit))

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    txs,
		Total:   &total,
		Page:    &page,
		Limit:   &filter.Limit,
	})
}

func (h *SupplierHandler) ListAllTransactions(c *fiber.Ctx) error {
	tenantID, err := getTenantIDFromCtx(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	filter := parseSupplierTxFilter(c)

	txs, total, err := h.service.ListAllTransactions(c.UserContext(), tenantID, filter)
	if err != nil {
		return err
	}

	c.Set("X-Total-Count", strconv.Itoa(total))
	page := 1
	if filter.Limit > 0 {
		page = (filter.Offset / filter.Limit) + 1
	}
	c.Set("X-Page", strconv.Itoa(page))
	c.Set("X-Limit", strconv.Itoa(filter.Limit))

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    txs,
		Total:   &total,
		Page:    &page,
		Limit:   &filter.Limit,
	})
}

func (h *SupplierHandler) CreateTransaction(c *fiber.Ctx) error {
	tenantID, err := getTenantIDFromCtx(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	var req CreateSupplierTxRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Geçersiz istek gövdesi")
	}

	var userID *uuid.UUID
	if uid, err := getUserIDFromCtx(c); err == nil {
		userID = &uid
	}

	txDate := time.Now()
	if req.TxDate != "" {
		if parsed, err := time.Parse("2006-01-02", req.TxDate); err == nil {
			txDate = parsed
		}
	}

	tx := &domain.SupplierTransaction{
		ID:             uuid.New(),
		TenantID:       tenantID,
		SupplierID:     req.SupplierID,
		PeriodID:       req.PeriodID,
		InvoiceNo:      req.InvoiceNo,
		CustomerName:   req.CustomerName,
		DocumentStatus: req.DocumentStatus,
		TxDate:         txDate,
		Direction:      req.Direction,
		Amount:         req.Amount,
		Description:    req.Description,
		CreatedBy:      userID,
		CreatedAt:      time.Now(),
	}

	createdTx, err := h.service.CreateTransaction(c.UserContext(), tenantID, tx)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(ResponseEnvelope{
		Success: true,
		Data:    createdTx,
	})
}

// ReverseSupplierTxRequest DTO
type ReverseSupplierTxRequest struct {
	Reason string `json:"reason"`
}

func (h *SupplierHandler) ReverseTransaction(c *fiber.Ctx) error {
	tenantID, err := getTenantIDFromCtx(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	txIDParam := c.Params("id")
	txID, err := uuid.Parse(txIDParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Geçersiz işlem kimliği")
	}

	var req ReverseSupplierTxRequest
	_ = c.BodyParser(&req)

	var userID *uuid.UUID
	if uid, err := getUserIDFromCtx(c); err == nil {
		userID = &uid
	}

	revTx, err := h.service.ReverseTransaction(c.UserContext(), tenantID, txID, req.Reason, userID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(ResponseEnvelope{
		Success: true,
		Data:    revTx,
	})
}

func (h *SupplierHandler) ImportExcel(c *fiber.Ctx) error {
	tenantID, err := getTenantIDFromCtx(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	periodIDParam := c.FormValue("period_id")
	if periodIDParam == "" {
		periodIDParam = c.Query("period_id")
	}
	periodID, err := uuid.Parse(periodIDParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Geçerli bir period_id belirtilmelidir")
	}

	sheetName := c.FormValue("sheet_name")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Lütfen 'file' alanında bir Excel (.xlsx) dosyası yükleyin")
	}

	// Validate extension
	fileName := strings.ToLower(fileHeader.Filename)
	if !strings.HasSuffix(fileName, ".xlsx") && !strings.HasSuffix(fileName, ".xlsm") {
		return fiber.NewError(fiber.StatusBadRequest, "Yalnızca Excel (.xlsx, .xlsm) dosyaları kabul edilir")
	}

	// 10MB max file size check
	if fileHeader.Size > 10<<20 {
		return fiber.NewError(fiber.StatusBadRequest, "Excel dosya boyutu 10MB sınırını aşamaz")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Yüklenen dosya açılamadı")
	}
	defer func() { _ = file.Close() }()

	var userID *uuid.UUID
	if uid, err := getUserIDFromCtx(c); err == nil {
		userID = &uid
	}

	result, err := h.service.ImportExcel(c.UserContext(), tenantID, periodID, file, sheetName, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    result,
	})
}

// Helpers

func getTenantIDFromCtx(c *fiber.Ctx) (uuid.UUID, error) {
	val := c.Locals(middleware.LocalTenantIDKey)
	tenantID, ok := val.(uuid.UUID)
	if !ok || tenantID == uuid.Nil {
		return uuid.Nil, domain.ErrUnauthorized
	}
	return tenantID, nil
}

func getUserIDFromCtx(c *fiber.Ctx) (uuid.UUID, error) {
	val := c.Locals(middleware.LocalUserIDKey)
	userID, ok := val.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, domain.ErrUnauthorized
	}
	return userID, nil
}

func parseSupplierTxFilter(c *fiber.Ctx) domain.SupplierTransactionFilter {
	filter := domain.SupplierTransactionFilter{
		Direction: c.Query("direction"),
		Search:    c.Query("search"),
		Limit:     50,
		Offset:    0,
	}

	if periodParam := c.Query("period_id"); periodParam != "" {
		if pID, err := uuid.Parse(periodParam); err == nil {
			filter.PeriodID = &pID
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > 200 {
				l = 200
			}
			filter.Limit = l
		}
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			filter.Offset = (p - 1) * filter.Limit
		}
	}

	return filter
}
