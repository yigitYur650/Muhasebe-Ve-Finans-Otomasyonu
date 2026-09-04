package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler"
	"deftersystem/backend/internal/handler/middleware"
)

// Scenario 1: Uçtan Uca Defter Akışı (3 Gelir, 2 Gider, Kuruşu Kuruşuna Devir ve Bakiye)
func TestEndToEndLedgerFlow_PennyAccurateBalances(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	periodID := uuid.New()

	mockPeriodSvc := new(MockPeriodService)
	mockTxSvc := new(MockTransactionService)
	mockTxRepo := new(MockTransactionRepo)
	idemRepo := NewMockIdemRepo()

	app := setupTestApp(mockPeriodSvc, mockTxSvc, mockTxRepo, idemRepo)

	// 1. Step: Open Period
	openedPeriod := &domain.Period{
		ID:              periodID,
		TenantID:        tenantID,
		Label:           "2026-08",
		StartingBalance: decimal.NewFromFloat(0.00),
		Status:          domain.PeriodStatusOpen,
	}
	mockPeriodSvc.On("OpenNextPeriod", mock.Anything, tenantID, "2026-08").Return(openedPeriod, nil).Once()

	openReqBody, _ := json.Marshal(handler.OpenPeriodRequest{Label: "2026-08"})
	reqOpen := httptest.NewRequest(http.MethodPost, "/api/v1/periods/open", bytes.NewReader(openReqBody))
	reqOpen.Header.Set("Content-Type", "application/json")
	reqOpen.Header.Set(middleware.HeaderTenantID, tenantID.String())
	reqOpen.Header.Set(middleware.HeaderUserID, userID.String())

	resOpen, err := app.Test(reqOpen, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resOpen.StatusCode)

	// 2. Step: Create 3 Income Entries (1000.00, 2500.50, 500.00 -> Total In = 4000.50)
	incomes := []decimal.Decimal{
		decimal.NewFromFloat(1000.00),
		decimal.NewFromFloat(2500.50),
		decimal.NewFromFloat(500.00),
	}
	mockTxSvc.On("CreateTransaction", mock.Anything, mock.Anything).Return(nil).Times(3)

	for _, amt := range incomes {
		payload, _ := json.Marshal(handler.CreateTransactionRequest{
			PeriodID:  periodID,
			Direction: domain.DirectionIn,
			Channel:   domain.ChannelEft,
			Amount:    amt,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(middleware.HeaderTenantID, tenantID.String())
		req.Header.Set(middleware.HeaderUserID, userID.String())

		res, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, res.StatusCode)
	}

	// 3. Step: Create 2 Expense Entries (1200.00, 300.25 -> Total Out = 1500.25)
	expenses := []decimal.Decimal{
		decimal.NewFromFloat(1200.00),
		decimal.NewFromFloat(300.25),
	}
	mockTxSvc.On("CreateTransaction", mock.Anything, mock.Anything).Return(nil).Times(2)

	for _, amt := range expenses {
		payload, _ := json.Marshal(handler.CreateTransactionRequest{
			PeriodID:  periodID,
			Direction: domain.DirectionOut,
			Channel:   domain.ChannelKira,
			Amount:    amt,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(middleware.HeaderTenantID, tenantID.String())
		req.Header.Set(middleware.HeaderUserID, userID.String())

		res, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, res.StatusCode)
	}

	// 4. Step: Verify Period Summary Balance (Starting 0.00 + Total In 4000.50 - Total Out 1500.25 = Net 2500.25)
	expectedSummary := &domain.PeriodSummary{
		PeriodID:        periodID,
		StartingBalance: decimal.NewFromFloat(0.00),
		TotalIn:         decimal.NewFromFloat(4000.50),
		TotalOut:        decimal.NewFromFloat(1500.25),
		ClosingBalance:  decimal.NewFromFloat(2500.25),
	}
	mockPeriodSvc.On("GetPeriodSummary", mock.Anything, periodID).Return(expectedSummary, nil).Once()

	reqSummary := httptest.NewRequest(http.MethodGet, "/api/v1/periods/"+periodID.String()+"/summary", nil)
	reqSummary.Header.Set(middleware.HeaderTenantID, tenantID.String())
	reqSummary.Header.Set(middleware.HeaderUserID, userID.String())

	resSummary, err := app.Test(reqSummary, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resSummary.StatusCode)

	var env handler.ResponseEnvelope
	json.NewDecoder(resSummary.Body).Decode(&env)
	assert.True(t, env.Success)

	summaryData, _ := json.Marshal(env.Data)
	var summary domain.PeriodSummary
	json.Unmarshal(summaryData, &summary)

	assert.Equal(t, "4000.5", summary.TotalIn.String())
	assert.Equal(t, "1500.25", summary.TotalOut.String())
	assert.Equal(t, "2500.25", summary.ClosingBalance.String(), "Net balance must equal exactly 2500.25 TL")
}

// Scenario 2: Ters Kayıt (Audit Trail) Bütünlüğü ve Denkleştirme Kaydı
func TestReversalAuditIntegrity_OffsetEntry(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	origTxID := uuid.New()

	mockPeriodSvc := new(MockPeriodService)
	mockTxSvc := new(MockTransactionService)
	mockTxRepo := new(MockTransactionRepo)
	idemRepo := NewMockIdemRepo()

	app := setupTestApp(mockPeriodSvc, mockTxSvc, mockTxRepo, idemRepo)

	reason := "Müşteri iade talebi"
	reversalTxID := uuid.New()
	reversalTx := &domain.Transaction{
		ID:          reversalTxID,
		PeriodID:    uuid.New(),
		TenantID:    tenantID,
		Direction:   domain.DirectionOut, // Invert from IN to OUT
		Channel:     domain.ChannelEft,
		Amount:      decimal.NewFromFloat(2500.50),
		Description: &reason,
		CreatedBy:   userID,
	}

	mockTxSvc.On("ReverseTransaction", mock.Anything, origTxID, reason, userID).Return(reversalTx, nil).Once()

	payload, _ := json.Marshal(handler.ReverseTransactionRequest{
		Reason: reason,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/"+origTxID.String()+"/reverse", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())
	req.Header.Set(middleware.HeaderUserID, userID.String())

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var env handler.ResponseEnvelope
	json.NewDecoder(res.Body).Decode(&env)
	assert.True(t, env.Success)

	txData, _ := json.Marshal(env.Data)
	var createdReversal domain.Transaction
	json.Unmarshal(txData, &createdReversal)

	assert.Equal(t, domain.DirectionOut, createdReversal.Direction, "Reversal of income entry MUST be an expense entry")
	assert.Equal(t, "2500.5", createdReversal.Amount.String(), "Reversal amount MUST equal original transaction amount")
}

// Scenario 3: Idempotency Güvenliği ve Mükerrer İstek Engeli
func TestIdempotencySecurity_DuplicateInterception(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	periodID := uuid.New()
	idemKey := "idempotency-key-uuid-9999"

	mockPeriodSvc := new(MockPeriodService)
	mockTxSvc := new(MockTransactionService)
	mockTxRepo := new(MockTransactionRepo)
	idemRepo := NewMockIdemRepo()

	app := setupTestApp(mockPeriodSvc, mockTxSvc, mockTxRepo, idemRepo)

	mockTxSvc.On("CreateTransaction", mock.Anything, mock.Anything).Return(nil).Once()

	reqPayload := handler.CreateTransactionRequest{
		PeriodID:  periodID,
		Direction: domain.DirectionIn,
		Channel:   domain.ChannelPos,
		Amount:    decimal.NewFromFloat(750.00),
	}
	bodyBytes, _ := json.Marshal(reqPayload)

	// First HTTP POST
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set(middleware.HeaderTenantID, tenantID.String())
	req1.Header.Set(middleware.HeaderUserID, userID.String())
	req1.Header.Set(middleware.HeaderIdempotencyKey, idemKey)

	res1, err := app.Test(req1, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, res1.StatusCode)
	res1Body, _ := io.ReadAll(res1.Body)

	// Duplicate HTTP POST with identical Idempotency-Key
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(middleware.HeaderTenantID, tenantID.String())
	req2.Header.Set(middleware.HeaderUserID, userID.String())
	req2.Header.Set(middleware.HeaderIdempotencyKey, idemKey)

	res2, err := app.Test(req2, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, res2.StatusCode)
	assert.Equal(t, "HIT", res2.Header.Get("X-Cache-Lookup"), "Duplicate request MUST return X-Cache-Lookup: HIT header")

	res2Body, _ := io.ReadAll(res2.Body)
	assert.Equal(t, string(res1Body), string(res2Body), "Cached response body MUST equal original response")

	mockTxSvc.AssertNumberOfCalls(t, "CreateTransaction", 1)
}

// Scenario 4: Kilitli Dönem ve Append-Only Koruması
func TestLockedPeriodAndAppendOnlyProtection_HTTP422(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	lockedPeriodID := uuid.New()

	mockPeriodSvc := new(MockPeriodService)
	mockTxSvc := new(MockTransactionService)
	mockTxRepo := new(MockTransactionRepo)
	idemRepo := NewMockIdemRepo()

	app := setupTestApp(mockPeriodSvc, mockTxSvc, mockTxRepo, idemRepo)

	// Attempting to create transaction in locked period returns ErrPeriodLocked
	mockTxSvc.On("CreateTransaction", mock.Anything, mock.Anything).Return(domain.ErrPeriodLocked)

	reqPayload := handler.CreateTransactionRequest{
		PeriodID:  lockedPeriodID,
		Direction: domain.DirectionOut,
		Channel:   domain.ChannelYakit,
		Amount:    decimal.NewFromFloat(450.00),
	}
	bodyBytes, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())
	req.Header.Set(middleware.HeaderUserID, userID.String())

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode, "Locked period mutations MUST return HTTP 422 Unprocessable Entity")

	var env handler.ResponseEnvelope
	json.NewDecoder(res.Body).Decode(&env)
	assert.False(t, env.Success)
	assert.Equal(t, "PERIOD_LOCKED", env.Error.Code)
}

// Scenario 5: Multi-Tenant ve Rol İzolasyonu (RBAC Protection)
func TestMultiTenantAndRoleIsolation_HTTP403(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	periodID := uuid.New()

	mockPeriodSvc := new(MockPeriodService)
	mockTxSvc := new(MockTransactionService)
	mockTxRepo := new(MockTransactionRepo)
	idemRepo := NewMockIdemRepo()

	app := setupTestApp(mockPeriodSvc, mockTxSvc, mockTxRepo, idemRepo)

	// Standard user attempting to lock period
	mockPeriodSvc.On("LockPeriod", mock.Anything, periodID, userID).Return(domain.ErrUnauthorized)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/periods/"+periodID.String()+"/lock", nil)
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())
	req.Header.Set(middleware.HeaderUserID, userID.String())
	req.Header.Set(middleware.HeaderUserRole, "standart")

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, res.StatusCode, "Standard user locking period MUST return HTTP 403 Forbidden")

	var env handler.ResponseEnvelope
	json.NewDecoder(res.Body).Decode(&env)
	assert.False(t, env.Success)
	assert.Equal(t, "UNAUTHORIZED", env.Error.Code)
}
