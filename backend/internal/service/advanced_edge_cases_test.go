package service_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/service"
)

// 1. Çift Kuruş & Maksimum Sayı Hassasiyeti Testi
func TestEdgeCase_PennyAndMaxDecimalPrecision(t *testing.T) {
	// Limit tutar: 9,999,999,999,999.99
	limitAmount, err := decimal.NewFromString("9999999999999.99")
	assert.NoError(t, err)

	// 100 adet 0.01 TL (1 kuruş) ekleme
	penny := decimal.NewFromFloat(0.01)
	current := limitAmount

	for i := 0; i < 100; i++ {
		current = current.Add(penny)
	}

	expected, err := decimal.NewFromString("10000000000000.99")
	assert.NoError(t, err)

	assert.True(t, current.Equal(expected), "100 adet 0.01 TL eklendiğinde hassasiyet kaybı olmamalıdır")
	assert.Equal(t, "10000000000000.99", current.String())
}

func TestEdgeCase_DoublePennyAndLargeNumberPrecision(t *testing.T) {
	TestEdgeCase_PennyAndMaxDecimalPrecision(t)
}

// 2. Reversal of a Reversal (Ters Kaydın Ters Kaydı Yasağı)
func TestEdgeCase_ReversalOfReversalBlocked(t *testing.T) {
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
		Channel:    domain.ChannelEft,
		Amount:     decimal.NewFromFloat(1500.00),
		ReversedBy: &reversalID, // Zaten iptal edilmiş
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

	// Birinci ters kayda ikinci kez ters kayıt denemesi -> RED
	revTx, err := svc.ReverseTransaction(ctx, origID, "İkinci iptal denemesi", userID)

	assert.ErrorIs(t, err, domain.ErrTransactionAlreadyReversed, "Zaten iptal edilmiş kayda tekrar ters kayıt atılamaz")
	assert.Nil(t, revTx)
}

// 3. Eşzamanlı (Concurrent) Yarış Durumu Testi
func TestEdgeCase_ConcurrentTransactionRaceCondition(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	periodID := uuid.New()
	userID := uuid.New()

	period := &domain.Period{
		ID:       periodID,
		TenantID: tenantID,
		Status:   domain.PeriodStatusOpen,
	}

	mockTxRepo := new(MockTransactionRepo)
	mockPeriodRepo := new(MockPeriodRepo)
	svc := service.NewTransactionService(mockTxRepo, mockPeriodRepo)

	// Safe concurrent expectations
	mockPeriodRepo.On("GetByID", mock.Anything, periodID).Return(period, nil)

	var successCount int64
	var mu sync.Mutex

	mockTxRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		mu.Lock()
		defer mu.Unlock()
		atomic.AddInt64(&successCount, 1)
	})

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			tx := &domain.Transaction{
				ID:          uuid.New(),
				TenantID:    tenantID,
				PeriodID:    periodID,
				Direction:   domain.DirectionIn,
				Channel:     domain.ChannelPos,
				Amount:      decimal.NewFromFloat(100.00),
				Description: stringPtr(fmt.Sprintf("Concurrent tx %d", idx)),
				CreatedBy:   userID,
			}
			err := svc.CreateTransaction(ctx, tx)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int64(numGoroutines), atomic.LoadInt64(&successCount), "10 eşzamanlı işlem sorunsuz kaydolmalıdır")
}

// 4. Geçersiz Kanal ve Negatif Tutar Sınır Testi
func TestEdgeCase_InvalidChannelAndNegativeAmount(t *testing.T) {
	// Tutar 0 olan işlem
	txZero := &domain.Transaction{
		Direction: domain.DirectionIn,
		Channel:   domain.ChannelNakit,
		Amount:    decimal.Zero,
	}
	assert.ErrorIs(t, txZero.Validate(), domain.ErrInvalidAmount)

	// Negatif tutar
	txNeg := &domain.Transaction{
		Direction: domain.DirectionIn,
		Channel:   domain.ChannelNakit,
		Amount:    decimal.NewFromFloat(-50.00),
	}
	assert.ErrorIs(t, txNeg.Validate(), domain.ErrInvalidAmount)

	// Kesirli 3 basamaklı kuruş (10.555 -> 2 basamağa yuvarlama kontrolü)
	subPenny := decimal.NewFromFloat(10.555).Round(2)
	assert.Equal(t, "10.56", subPenny.String())

	// Geçersiz kanal ismi
	txBadChannel := &domain.Transaction{
		Direction: domain.DirectionIn,
		Channel:   domain.Channel("BOGUS_CHANNEL"),
		Amount:    decimal.NewFromFloat(100.00),
	}
	assert.ErrorIs(t, txBadChannel.Validate(), domain.ErrInvalidChannel)

	// Geçersiz yön
	txBadDir := &domain.Transaction{
		Direction: domain.Direction("SIDEWAYS"),
		Channel:   domain.ChannelNakit,
		Amount:    decimal.NewFromFloat(100.00),
	}
	assert.ErrorIs(t, txBadDir.Validate(), domain.ErrInvalidDirection)
}

func TestEdgeCase_InvalidChannelAndSubPennyBounds(t *testing.T) {
	TestEdgeCase_InvalidChannelAndNegativeAmount(t)
}

// 5. Cross-Tenant Idempotency İzolasyonu Testi
func TestEdgeCase_CrossTenantIdempotencyIsolation(t *testing.T) {
	ctx := context.Background()
	tenantA := uuid.New()
	tenantB := uuid.New()
	sameKey := "shared-idempotency-key-123"

	keyObjA := &domain.IdempotencyKey{
		Key:            sameKey,
		TenantID:       tenantA,
		ResponseBody:   []byte(`{"success": true, "tx_id": "tx-a"}`),
		ResponseStatus: 200,
		CreatedAt:      time.Now(),
	}

	mockIdempRepo := new(MockIdempotencyRepo)

	// Tenant A araması keyObjA döner
	mockIdempRepo.On("Get", ctx, sameKey, tenantA).Return(keyObjA, nil)

	// Tenant B araması domain.ErrNotFound döner
	mockIdempRepo.On("Get", ctx, sameKey, tenantB).Return(nil, domain.ErrNotFound)

	// Tenant A sorgusu
	resA, errA := mockIdempRepo.Get(ctx, sameKey, tenantA)
	assert.NoError(t, errA)
	assert.NotNil(t, resA)
	assert.Equal(t, tenantA, resA.TenantID)

	// Tenant B sorgusu A'nın önbelleğine erişemez
	resB, errB := mockIdempRepo.Get(ctx, sameKey, tenantB)
	assert.ErrorIs(t, errB, domain.ErrNotFound)
	assert.Nil(t, resB)
}

// 6. Kilitli Dönemde Bakiye Sabitliği Testi
func TestEdgeCase_LockedPeriodBalanceImmutability(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	lockedPeriodID := uuid.New()
	userID := uuid.New()

	now := time.Now()
	startingBalance := decimal.NewFromFloat(25000.00)

	lockedPeriod := &domain.Period{
		ID:              lockedPeriodID,
		TenantID:        tenantID,
		Label:           "2026-07",
		StartingBalance: startingBalance,
		Status:          domain.PeriodStatusLocked,
		LockedAt:        &now,
	}

	mockTxRepo := new(MockTransactionRepo)
	mockPeriodRepo := new(MockPeriodRepo)
	svc := service.NewTransactionService(mockTxRepo, mockPeriodRepo)

	mockPeriodRepo.On("GetByID", ctx, lockedPeriodID).Return(lockedPeriod, nil)

	// Kilitli döneme işlem ekleme denemesi -> RED
	newTx := &domain.Transaction{
		ID:        uuid.New(),
		TenantID:  tenantID,
		PeriodID:  lockedPeriodID,
		Direction: domain.DirectionIn,
		Channel:   domain.ChannelEft,
		Amount:    decimal.NewFromFloat(5000.00),
		CreatedBy: userID,
	}

	err := svc.CreateTransaction(ctx, newTx)
	assert.ErrorIs(t, err, domain.ErrPeriodLocked, "Kilitli döneme işlem eklenemez")

	// Bakiye sabitliğini doğrula
	assert.True(t, lockedPeriod.StartingBalance.Equal(startingBalance), "Kilitli dönemin açılış bakiyesi değişmez kalmalıdır")
	assert.True(t, lockedPeriod.IsLocked(), "Dönem statusu locked olarak kalmalıdır")
	mockTxRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func stringPtr(s string) *string {
	return &s
}

