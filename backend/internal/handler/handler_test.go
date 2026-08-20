package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler"
	"deftersystem/backend/internal/handler/middleware"
)

// Mock Idempotency Repository in-memory
type MockIdemRepo struct {
	store map[string]*domain.IdempotencyKey
}

func NewMockIdemRepo() *MockIdemRepo {
	return &MockIdemRepo{store: make(map[string]*domain.IdempotencyKey)}
}

func (m *MockIdemRepo) Get(ctx context.Context, key string) (*domain.IdempotencyKey, error) {
	if val, ok := m.store[key]; ok {
		return val, nil
	}
	return nil, domain.ErrNotFound
}

func (m *MockIdemRepo) Save(ctx context.Context, idem *domain.IdempotencyKey) error {
	m.store[idem.Key] = idem
	return nil
}

// Mock Service structs
type MockPeriodService struct {
	mock.Mock
}

func (m *MockPeriodService) OpenNextPeriod(ctx context.Context, tenantID uuid.UUID, label string) (*domain.Period, error) {
	args := m.Called(ctx, tenantID, label)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Period), args.Error(1)
}

func (m *MockPeriodService) LockPeriod(ctx context.Context, periodID, requestingUserID uuid.UUID) error {
	args := m.Called(ctx, periodID, requestingUserID)
	return args.Error(0)
}

func (m *MockPeriodService) GetPeriodSummary(ctx context.Context, periodID uuid.UUID) (*domain.PeriodSummary, error) {
	args := m.Called(ctx, periodID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PeriodSummary), args.Error(1)
}

func (m *MockPeriodService) ListPeriods(ctx context.Context, tenantID uuid.UUID) ([]domain.Period, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Period), args.Error(1)
}

type MockTransactionService struct {
	mock.Mock
}

func (m *MockTransactionService) CreateTransaction(ctx context.Context, tx *domain.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionService) ReverseTransaction(ctx context.Context, origID uuid.UUID, reason string, createdBy uuid.UUID) (*domain.Transaction, error) {
	args := m.Called(ctx, origID, reason, createdBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Transaction), args.Error(1)
}

type MockTransactionRepo struct {
	mock.Mock
}

func (m *MockTransactionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Transaction), args.Error(1)
}

func (m *MockTransactionRepo) Create(ctx context.Context, tx *domain.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepo) GetByPeriodID(ctx context.Context, periodID uuid.UUID) ([]domain.Transaction, error) {
	args := m.Called(ctx, periodID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Transaction), args.Error(1)
}

func (m *MockTransactionRepo) GetSummaryByPeriodID(ctx context.Context, periodID uuid.UUID) (*domain.PeriodSummary, error) {
	args := m.Called(ctx, periodID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PeriodSummary), args.Error(1)
}

func (m *MockTransactionRepo) ReverseTransaction(ctx context.Context, origID uuid.UUID, revTx *domain.Transaction) error {
	args := m.Called(ctx, origID, revTx)
	return args.Error(0)
}

func (m *MockTransactionRepo) MarkReversed(ctx context.Context, targetID, reversalID uuid.UUID) error {
	args := m.Called(ctx, targetID, reversalID)
	return args.Error(0)
}

func setupTestApp(
	periodSvc domain.PeriodService,
	txSvc domain.TransactionService,
	txRepo domain.TransactionRepository,
	idemRepo domain.IdempotencyRepository,
) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: handler.CustomErrorHandler,
	})
	handler.SetupRouter(app, periodSvc, txSvc, txRepo, idemRepo)
	return app
}

func TestIdempotencyMiddleware_DuplicateRequestCached(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	periodID := uuid.New()
	idemKey := "key-unique-test-123"

	mockPeriodSvc := new(MockPeriodService)
	mockTxSvc := new(MockTransactionService)
	mockTxRepo := new(MockTransactionRepo)
	idemRepo := NewMockIdemRepo()

	app := setupTestApp(mockPeriodSvc, mockTxSvc, mockTxRepo, idemRepo)

	mockTxSvc.On("CreateTransaction", mock.Anything, mock.Anything).Return(nil).Once()

	reqPayload := handler.CreateTransactionRequest{
		PeriodID:  periodID,
		Direction: domain.DirectionIn,
		Channel:   domain.ChannelEft,
		Amount:    decimal.NewFromFloat(1500.00),
	}
	bodyBytes, _ := json.Marshal(reqPayload)

	// First Request
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set(middleware.HeaderTenantID, tenantID.String())
	req1.Header.Set(middleware.HeaderUserID, userID.String())
	req1.Header.Set(middleware.HeaderIdempotencyKey, idemKey)

	res1, err := app.Test(req1, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, res1.StatusCode)

	res1Body, _ := io.ReadAll(res1.Body)

	// Second Request with identical Idempotency-Key
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(middleware.HeaderTenantID, tenantID.String())
	req2.Header.Set(middleware.HeaderUserID, userID.String())
	req2.Header.Set(middleware.HeaderIdempotencyKey, idemKey)

	res2, err := app.Test(req2, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, res2.StatusCode)
	assert.Equal(t, "HIT", res2.Header.Get("X-Cache-Lookup"))

	res2Body, _ := io.ReadAll(res2.Body)
	assert.Equal(t, string(res1Body), string(res2Body), "Cached response body must equal first response body")

	// Verify CreateTransaction was only called ONCE
	mockTxSvc.AssertNumberOfCalls(t, "CreateTransaction", 1)
}

func TestCreateTransaction_NegativeAmountReturns400(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	periodID := uuid.New()

	mockPeriodSvc := new(MockPeriodService)
	mockTxSvc := new(MockTransactionService)
	mockTxRepo := new(MockTransactionRepo)
	idemRepo := NewMockIdemRepo()

	app := setupTestApp(mockPeriodSvc, mockTxSvc, mockTxRepo, idemRepo)

	mockTxSvc.On("CreateTransaction", mock.Anything, mock.Anything).Return(domain.ErrInvalidAmount)

	reqPayload := handler.CreateTransactionRequest{
		PeriodID:  periodID,
		Direction: domain.DirectionIn,
		Channel:   domain.ChannelNakit,
		Amount:    decimal.NewFromFloat(-500.00),
	}
	bodyBytes, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())
	req.Header.Set(middleware.HeaderUserID, userID.String())

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	var envelope handler.ResponseEnvelope
	json.NewDecoder(res.Body).Decode(&envelope)
	assert.False(t, envelope.Success)
	assert.Equal(t, "INVALID_AMOUNT", envelope.Error.Code)
}

func TestCreateTransaction_LockedPeriodReturns422(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	periodID := uuid.New()

	mockPeriodSvc := new(MockPeriodService)
	mockTxSvc := new(MockTransactionService)
	mockTxRepo := new(MockTransactionRepo)
	idemRepo := NewMockIdemRepo()

	app := setupTestApp(mockPeriodSvc, mockTxSvc, mockTxRepo, idemRepo)

	mockTxSvc.On("CreateTransaction", mock.Anything, mock.Anything).Return(domain.ErrPeriodLocked)

	reqPayload := handler.CreateTransactionRequest{
		PeriodID:  periodID,
		Direction: domain.DirectionOut,
		Channel:   domain.ChannelKira,
		Amount:    decimal.NewFromFloat(2000.00),
	}
	bodyBytes, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())
	req.Header.Set(middleware.HeaderUserID, userID.String())

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)

	var envelope handler.ResponseEnvelope
	json.NewDecoder(res.Body).Decode(&envelope)
	assert.False(t, envelope.Success)
	assert.Equal(t, "PERIOD_LOCKED", envelope.Error.Code)
}

func TestLockPeriod_StandartUserReturns403(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	periodID := uuid.New()

	mockPeriodSvc := new(MockPeriodService)
	mockTxSvc := new(MockTransactionService)
	mockTxRepo := new(MockTransactionRepo)
	idemRepo := NewMockIdemRepo()

	app := setupTestApp(mockPeriodSvc, mockTxSvc, mockTxRepo, idemRepo)

	mockPeriodSvc.On("LockPeriod", mock.Anything, periodID, userID).Return(domain.ErrUnauthorized)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/periods/"+periodID.String()+"/lock", nil)
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())
	req.Header.Set(middleware.HeaderUserID, userID.String())
	req.Header.Set(middleware.HeaderUserRole, "standart")

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)

	var envelope handler.ResponseEnvelope
	json.NewDecoder(res.Body).Decode(&envelope)
	assert.False(t, envelope.Success)
	assert.Equal(t, "UNAUTHORIZED", envelope.Error.Code)
}
