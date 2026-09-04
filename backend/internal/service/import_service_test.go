package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/service"
)

func TestImportService_InvalidAmountReturnsErrorWithLineNumber(t *testing.T) {
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
	svc := service.NewImportService(mockTxRepo, mockPeriodRepo)

	mockPeriodRepo.On("GetByID", ctx, periodID).Return(period, nil)

	// Line 4 has invalid amount "abc"
	csvContent := `Yön,Kanal,Tutar,Açıklama
Gelir,eft,1500.50,Müşteri ödemesi
Gider,kira,3200.00,Ofis kirası
Gelir,pos,abc,Hatalı satır
`

	res, err := svc.ImportTransactionsFromCSV(ctx, tenantID, periodID, strings.NewReader(csvContent), userID)

	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "line 4")
	assert.ErrorIs(t, err, domain.ErrInvalidAmount)
	mockTxRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestImportService_LockedPeriodReturnsErrPeriodLocked(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	periodID := uuid.New()
	userID := uuid.New()

	lockedPeriod := &domain.Period{
		ID:       periodID,
		TenantID: tenantID,
		Status:   domain.PeriodStatusLocked,
	}

	mockTxRepo := new(MockTransactionRepo)
	mockPeriodRepo := new(MockPeriodRepo)
	svc := service.NewImportService(mockTxRepo, mockPeriodRepo)

	mockPeriodRepo.On("GetByID", ctx, periodID).Return(lockedPeriod, nil)

	csvContent := `Yön,Kanal,Tutar,Açıklama
Gelir,eft,1000.00,Kilitli döneme aktarım denemesi
`

	res, err := svc.ImportTransactionsFromCSV(ctx, tenantID, periodID, strings.NewReader(csvContent), userID)

	assert.ErrorIs(t, err, domain.ErrPeriodLocked)
	assert.Nil(t, res)
	mockTxRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestImportService_SuccessfulImportPennyAccurate(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	periodID := uuid.New()
	userID := uuid.New()

	openPeriod := &domain.Period{
		ID:       periodID,
		TenantID: tenantID,
		Status:   domain.PeriodStatusOpen,
	}

	mockTxRepo := new(MockTransactionRepo)
	mockPeriodRepo := new(MockPeriodRepo)
	svc := service.NewImportService(mockTxRepo, mockPeriodRepo)

	mockPeriodRepo.On("GetByID", ctx, periodID).Return(openPeriod, nil)
	mockTxRepo.On("Create", ctx, mock.Anything).Return(nil)

	// UTF-8 BOM + 3 valid rows (1500.50 + 3200.25 + 450.75 = 5151.50)
	csvContent := "\xef\xbb\xbfYön,Kanal,Tutar,Açıklama\nGelir,eft,1500.50,Satış EFT\nGider,kira,3200.25,Kira ödemesi\nGelir,pos,450.75,POS tahsilat\n"

	res, err := svc.ImportTransactionsFromCSV(ctx, tenantID, periodID, strings.NewReader(csvContent), userID)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 3, res.ImportedCount)
	assert.True(t, res.TotalAmount.Equal(decimal.NewFromFloat(5151.50)), "Total amount must be exactly 5151.50")
	mockTxRepo.AssertNumberOfCalls(t, "Create", 3)
}

func TestImportService_TurkishExcelFormat(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	periodID := uuid.New()
	userID := uuid.New()

	openPeriod := &domain.Period{
		ID:       periodID,
		TenantID: tenantID,
		Status:   domain.PeriodStatusOpen,
	}

	mockTxRepo := new(MockTransactionRepo)
	mockPeriodRepo := new(MockPeriodRepo)
	svc := service.NewImportService(mockTxRepo, mockPeriodRepo)

	mockPeriodRepo.On("GetByID", ctx, periodID).Return(openPeriod, nil)
	mockTxRepo.On("Create", ctx, mock.Anything).Return(nil)

	// Turkish Excel format matching user's MAYIS25 spreadsheet
	csvContent := `MAYIS AYI NAKİT,,₺221.646,05,,
Tarih,Açıklama,GELEN TUTAR,GİDEN TUTAR
1.05.2025,NİSAN AYINDAN MAYIS AYINA DEVİR,"265.698,04 ₺",
1.05.2025,DUKKAN KIRASI,,"39.000,00 ₺"
1.05.2025,HALİL YUR FİBABANK KREDİ GELİRİ,"220.618,00 ₺",
1.05.2025,POS,"11.550,00 ₺",
5.05.2025,PERSONEL MAAŞ ELDEN,,"40.854,00 ₺"
`

	res, err := svc.ImportTransactionsFromCSV(ctx, tenantID, periodID, strings.NewReader(csvContent), userID)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 5, res.ImportedCount)
	mockTxRepo.AssertNumberOfCalls(t, "Create", 5)
}
