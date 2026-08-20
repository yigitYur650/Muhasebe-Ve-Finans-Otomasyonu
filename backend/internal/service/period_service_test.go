package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/service"
)

func TestPeriodService_LockPeriod_RoleAuthorization(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	periodID := uuid.New()
	adminUserID := uuid.New()
	muhasebeciUserID := uuid.New()
	standartUserID := uuid.New()

	period := &domain.Period{
		ID:       periodID,
		TenantID: tenantID,
		Status:   domain.PeriodStatusOpen,
	}

	adminMember := &domain.TenantMember{
		ID:       uuid.New(),
		TenantID: tenantID,
		UserID:   adminUserID,
		Role:     domain.RoleAdmin,
	}

	muhasebeciMember := &domain.TenantMember{
		ID:       uuid.New(),
		TenantID: tenantID,
		UserID:   muhasebeciUserID,
		Role:     domain.RoleMuhasebeci,
	}

	standartMember := &domain.TenantMember{
		ID:       uuid.New(),
		TenantID: tenantID,
		UserID:   standartUserID,
		Role:     domain.RoleStandart,
	}

	t.Run("standart role gets ErrUnauthorized", func(t *testing.T) {
		mockPeriodRepo := new(MockPeriodRepo)
		mockTenantRepo := new(MockTenantRepo)
		mockTxRepo := new(MockTransactionRepo)
		svc := service.NewPeriodService(mockPeriodRepo, mockTenantRepo, mockTxRepo)

		mockPeriodRepo.On("GetByID", ctx, periodID).Return(period, nil)
		mockTenantRepo.On("GetMember", ctx, tenantID, standartUserID).Return(standartMember, nil)

		err := svc.LockPeriod(ctx, periodID, standartUserID)

		assert.ErrorIs(t, err, domain.ErrUnauthorized)
		mockPeriodRepo.AssertNotCalled(t, "Lock", mock.Anything, mock.Anything)
	})

	t.Run("admin role succeeds", func(t *testing.T) {
		mockPeriodRepo := new(MockPeriodRepo)
		mockTenantRepo := new(MockTenantRepo)
		mockTxRepo := new(MockTransactionRepo)
		svc := service.NewPeriodService(mockPeriodRepo, mockTenantRepo, mockTxRepo)

		mockPeriodRepo.On("GetByID", ctx, periodID).Return(period, nil)
		mockTenantRepo.On("GetMember", ctx, tenantID, adminUserID).Return(adminMember, nil)
		mockPeriodRepo.On("Lock", ctx, periodID).Return(nil)

		err := svc.LockPeriod(ctx, periodID, adminUserID)

		assert.NoError(t, err)
		mockPeriodRepo.AssertCalled(t, "Lock", ctx, periodID)
	})

	t.Run("muhasebeci role succeeds", func(t *testing.T) {
		mockPeriodRepo := new(MockPeriodRepo)
		mockTenantRepo := new(MockTenantRepo)
		mockTxRepo := new(MockTransactionRepo)
		svc := service.NewPeriodService(mockPeriodRepo, mockTenantRepo, mockTxRepo)

		mockPeriodRepo.On("GetByID", ctx, periodID).Return(period, nil)
		mockTenantRepo.On("GetMember", ctx, tenantID, muhasebeciUserID).Return(muhasebeciMember, nil)
		mockPeriodRepo.On("Lock", ctx, periodID).Return(nil)

		err := svc.LockPeriod(ctx, periodID, muhasebeciUserID)

		assert.NoError(t, err)
		mockPeriodRepo.AssertCalled(t, "Lock", ctx, periodID)
	})
}

func TestPeriodService_GetPeriodSummary(t *testing.T) {
	ctx := context.Background()
	periodID := uuid.New()

	expectedSummary := &domain.PeriodSummary{
		PeriodID:        periodID,
		StartingBalance: decimal.NewFromFloat(10000.00),
		TotalIn:         decimal.NewFromFloat(5000.50),
		TotalOut:        decimal.NewFromFloat(2000.25),
		ClosingBalance:  decimal.NewFromFloat(12998.25),
	}

	mockPeriodRepo := new(MockPeriodRepo)
	mockTenantRepo := new(MockTenantRepo)
	mockTxRepo := new(MockTransactionRepo)
	svc := service.NewPeriodService(mockPeriodRepo, mockTenantRepo, mockTxRepo)

	mockTxRepo.On("GetSummaryByPeriodID", ctx, periodID).Return(expectedSummary, nil)

	summary, err := svc.GetPeriodSummary(ctx, periodID)

	assert.NoError(t, err)
	assert.Equal(t, expectedSummary, summary)
	assert.True(t, summary.ClosingBalance.Equal(decimal.NewFromFloat(12998.25)))
}

func TestPeriodService_PennyAccurateCalculation(t *testing.T) {
	// Starting balance: 10,000.05
	// Income: 5,450.75
	// Expense: 3,200.25
	// Closing balance = 10,000.05 + 5,450.75 - 3,200.25 = 12,250.55
	starting := decimal.NewFromFloat(10000.05)
	totalIn := decimal.NewFromFloat(5450.75)
	totalOut := decimal.NewFromFloat(3200.25)

	closing := starting.Add(totalIn).Sub(totalOut)
	expected := decimal.NewFromFloat(12250.55)

	assert.True(t, closing.Equal(expected), "Penny-accurate decimal calculation MUST match 12250.55")
}
