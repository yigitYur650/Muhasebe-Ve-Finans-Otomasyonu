package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler"
	"deftersystem/backend/internal/handler/middleware"
)

type MockSupplierService struct {
	mock.Mock
}

func (m *MockSupplierService) ListSuppliers(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) ([]domain.Supplier, error) {
	args := m.Called(ctx, tenantID, periodID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Supplier), args.Error(1)
}

func (m *MockSupplierService) GetSupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (*domain.Supplier, error) {
	args := m.Called(ctx, tenantID, supplierID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Supplier), args.Error(1)
}

func (m *MockSupplierService) CreateSupplier(ctx context.Context, tenantID uuid.UUID, name string) (*domain.Supplier, error) {
	args := m.Called(ctx, tenantID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Supplier), args.Error(1)
}

func (m *MockSupplierService) CreateTransaction(ctx context.Context, tenantID uuid.UUID, tx *domain.SupplierTransaction) (*domain.SupplierTransaction, error) {
	args := m.Called(ctx, tenantID, tx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SupplierTransaction), args.Error(1)
}

func (m *MockSupplierService) ListSupplierTransactions(ctx context.Context, tenantID, supplierID uuid.UUID, filter domain.SupplierTransactionFilter) ([]domain.SupplierTransaction, int, error) {
	args := m.Called(ctx, tenantID, supplierID, filter)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]domain.SupplierTransaction), args.Int(1), args.Error(2)
}

func (m *MockSupplierService) ListAllTransactions(ctx context.Context, tenantID uuid.UUID, filter domain.SupplierTransactionFilter) ([]domain.SupplierTransaction, int, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]domain.SupplierTransaction), args.Int(1), args.Error(2)
}

func (m *MockSupplierService) GetSummary(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) (*domain.SupplierSummary, error) {
	args := m.Called(ctx, tenantID, periodID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SupplierSummary), args.Error(1)
}

func (m *MockSupplierService) ReverseTransaction(ctx context.Context, tenantID, origID uuid.UUID, reason string, createdBy *uuid.UUID) (*domain.SupplierTransaction, error) {
	args := m.Called(ctx, tenantID, origID, reason, createdBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SupplierTransaction), args.Error(1)
}

func (m *MockSupplierService) ImportExcel(ctx context.Context, tenantID, periodID uuid.UUID, r io.Reader, sheetName string, userID *uuid.UUID) (*domain.SupplierImportResult, error) {
	args := m.Called(ctx, tenantID, periodID, r, sheetName, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SupplierImportResult), args.Error(1)
}

func setupSupplierTestApp(supplierSvc domain.SupplierService, tenantID, userID uuid.UUID) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: handler.CustomErrorHandler,
	})

	// Inject mock tenant and user IDs via middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.LocalTenantIDKey, tenantID)
		c.Locals(middleware.LocalUserIDKey, userID)
		return c.Next()
	})

	supH := handler.NewSupplierHandler(supplierSvc)
	g := app.Group("/api/v1/suppliers")
	g.Get("/", supH.ListSuppliers)
	g.Get("/summary", supH.GetSummary)
	g.Post("/", supH.CreateSupplier)
	g.Get("/transactions", supH.ListAllTransactions)
	g.Post("/transactions", supH.CreateTransaction)
	g.Get("/:id/transactions", supH.ListSupplierTransactions)

	return app
}

func TestSupplierHandler_ListSuppliers(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	mockSvc := new(MockSupplierService)

	suppliers := []domain.Supplier{
		{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          "ATİKER",
			IsActive:      true,
			TotalPurchase: decimal.NewFromInt(50000),
			TotalPayment:  decimal.NewFromInt(20000),
			Balance:       decimal.NewFromInt(30000),
		},
	}

	mockSvc.On("ListSuppliers", mock.Anything, tenantID, (*uuid.UUID)(nil)).Return(suppliers, nil)

	app := setupSupplierTestApp(mockSvc, tenantID, userID)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/suppliers", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var env handler.ResponseEnvelope
	_ = json.NewDecoder(resp.Body).Decode(&env)
	assert.True(t, env.Success)
}

func TestSupplierHandler_CreateSupplier(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	mockSvc := new(MockSupplierService)

	created := &domain.Supplier{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      "PRİNS",
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	mockSvc.On("CreateSupplier", mock.Anything, tenantID, "PRİNS").Return(created, nil)

	app := setupSupplierTestApp(mockSvc, tenantID, userID)
	body := []byte(`{"name":"PRİNS"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/suppliers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
