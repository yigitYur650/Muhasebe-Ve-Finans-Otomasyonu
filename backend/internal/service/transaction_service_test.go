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

func TestReverseTransaction_InverseDirectionAndDecimalAmount(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	periodID := uuid.New()
	origID := uuid.New()
	userID := uuid.New()

	amount := decimal.NewFromFloat(7500.50)
	origDesc := "Hatalı EFT girişi"

	origTx := &domain.Transaction{
		ID:          origID,
		TenantID:    tenantID,
		PeriodID:    periodID,
		Direction:   domain.DirectionIn, // Gelir (in)
		Channel:     domain.ChannelEft,
		Amount:      amount,
		Description: &origDesc,
		CreatedBy:   userID,
		ReversedBy:  nil,
	}

	period := &domain.Period{
		ID:       periodID,
		TenantID: tenantID,
		Status:   domain.PeriodStatusOpen,
	}

	mockTxRepo := new(MockTransactionRepo)
	mockPeriodRepo := new(MockPeriodRepo)
	svc := service.NewTransactionService(mockTxRepo, mockPeriodRepo)

	mockTxRepo.On("GetByID", ctx, origID).Return(origTx, nil)
	mockPeriodRepo.On("GetByID", ctx, periodID).Return(period, nil)
	mockTxRepo.On("ReverseTransaction", ctx, origID, mock.MatchedBy(func(revTx *domain.Transaction) bool {
		return revTx.Direction == domain.DirectionOut &&
			revTx.Amount.Equal(amount) &&
			revTx.Channel == domain.ChannelEft &&
			*revTx.Description == "[İPTAL/TERS KAYIT] Yanlış tutar yazıldı"
	})).Return(nil)

	revTx, err := svc.ReverseTransaction(ctx, origID, "Yanlış tutar yazıldı", userID)

	assert.NoError(t, err)
	assert.NotNil(t, revTx)
	assert.Equal(t, domain.DirectionOut, revTx.Direction, "In direction must be reversed to Out direction")
	assert.True(t, revTx.Amount.Equal(amount), "Decimal amount must match original exactly")
	assert.Equal(t, "[İPTAL/TERS KAYIT] Yanlış tutar yazıldı", *revTx.Description)
}

func TestReverseTransaction_DoubleReversalBlocked(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	periodID := uuid.New()
	origID := uuid.New()
	reversalID := uuid.New()
	userID := uuid.New()

	origTx := &domain.Transaction{
		ID:         origID,
		TenantID:   tenantID,
		PeriodID:   periodID,
		Direction:  domain.DirectionIn,
		Channel:    domain.ChannelPos,
		Amount:     decimal.NewFromFloat(500.00),
		ReversedBy: &reversalID, // Already reversed
	}

	period := &domain.Period{
		ID:       periodID,
		TenantID: tenantID,
		Status:   domain.PeriodStatusOpen,
	}

	mockTxRepo := new(MockTransactionRepo)
	mockPeriodRepo := new(MockPeriodRepo)
	svc := service.NewTransactionService(mockTxRepo, mockPeriodRepo)

	mockTxRepo.On("GetByID", ctx, origID).Return(origTx, nil)
	mockPeriodRepo.On("GetByID", ctx, periodID).Return(period, nil)
	mockTxRepo.On("ReverseTransaction", ctx, origID, mock.Anything).Return(domain.ErrTransactionAlreadyReversed)

	revTx, err := svc.ReverseTransaction(ctx, origID, "Tekrar iptal denemesi", userID)

	assert.ErrorIs(t, err, domain.ErrTransactionAlreadyReversed)
	assert.Nil(t, revTx)
}

func TestCreateTransaction_LockedPeriodBlocked(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	periodID := uuid.New()
	userID := uuid.New()

	lockedPeriod := &domain.Period{
		ID:       periodID,
		TenantID: tenantID,
		Status:   domain.PeriodStatusLocked,
	}

	tx := &domain.Transaction{
		ID:        uuid.New(),
		TenantID:  tenantID,
		PeriodID:  periodID,
		Direction: domain.DirectionOut,
		Channel:   domain.ChannelYakit,
		Amount:    decimal.NewFromFloat(450.00),
		CreatedBy: userID,
	}

	mockTxRepo := new(MockTransactionRepo)
	mockPeriodRepo := new(MockPeriodRepo)
	svc := service.NewTransactionService(mockTxRepo, mockPeriodRepo)

	mockPeriodRepo.On("GetByID", ctx, periodID).Return(lockedPeriod, nil)

	err := svc.CreateTransaction(ctx, tx)

	assert.ErrorIs(t, err, domain.ErrPeriodLocked)
	mockTxRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}
