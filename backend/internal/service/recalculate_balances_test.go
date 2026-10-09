package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/service"
)

// TestRecalculateBalances_AllSuppliers verifies deterministic recalculation of purchases, payments and net balances
// across multiple suppliers with regular, reversed, and multi-period transactions.
func TestRecalculateBalances_AllSuppliers(t *testing.T) {
	tenantID := uuid.New()
	periodID := uuid.New()
	userID := uuid.New()

	mockPeriodRepo := new(MockPeriodRepo)
	mockPeriodRepo.On("GetByID", mock.Anything, periodID).Return(&domain.Period{
		ID:       periodID,
		TenantID: tenantID,
		Label:    "2026-08",
		Status:   domain.PeriodStatusOpen,
		OpenedAt: time.Now(),
	}, nil)

	mockSupplierRepo := NewMockSupplierRepo()
	svc := service.NewSupplierService(mockSupplierRepo, mockPeriodRepo)
	ctx := context.Background()

	// 1. Create multiple suppliers
	suppliersList := []string{
		"ASİL GRUP",
		"ATİKER",
		"BRC",
		"CANGAS",
		"FİLTRECİ",
		"BOŞ TEDARİKÇİ",
	}

	supplierMap := make(map[string]uuid.UUID)
	for _, name := range suppliersList {
		sup, err := svc.CreateSupplier(ctx, tenantID, name)
		require.NoError(t, err)
		supplierMap[name] = sup.ID
	}

	// 2. Add realistic transactions
	// --- ASİL GRUP ---
	// Purchase: 90,000 TL, 10,000 TL (to be cancelled), Payment: 40,000 TL
	// Expected ASİL GRUP: Purchase = 90,000 TL, Payment = 40,000 TL, Balance = 50,000 TL
	_, err := svc.CreateTransaction(ctx, tenantID, &domain.SupplierTransaction{
		TenantID:   tenantID,
		SupplierID: supplierMap["ASİL GRUP"],
		PeriodID:   periodID,
		TxDate:     time.Now(),
		Direction:  domain.SupplierDirectionPurchase,
		Amount:     decimal.NewFromInt(90000),
		CreatedBy:  &userID,
	})
	require.NoError(t, err)

	txAsilToCancel, err := svc.CreateTransaction(ctx, tenantID, &domain.SupplierTransaction{
		TenantID:   tenantID,
		SupplierID: supplierMap["ASİL GRUP"],
		PeriodID:   periodID,
		TxDate:     time.Now(),
		Direction:  domain.SupplierDirectionPurchase,
		Amount:     decimal.NewFromInt(10000),
		CreatedBy:  &userID,
	})
	require.NoError(t, err)

	_, err = svc.CreateTransaction(ctx, tenantID, &domain.SupplierTransaction{
		TenantID:   tenantID,
		SupplierID: supplierMap["ASİL GRUP"],
		PeriodID:   periodID,
		TxDate:     time.Now(),
		Direction:  domain.SupplierDirectionPayment,
		Amount:     decimal.NewFromInt(40000),
		CreatedBy:  &userID,
	})
	require.NoError(t, err)

	// Reverse the 10,000 TL purchase
	_, err = svc.ReverseTransaction(ctx, tenantID, txAsilToCancel.ID, "İptal edilen fatura", &userID)
	require.NoError(t, err)

	// --- ATİKER ---
	// Purchase: 50,000.50 TL, Payment: 20,000.25 TL
	// Expected ATİKER: Purchase = 50,000.50 TL, Payment = 20,000.25 TL, Balance = 30,000.25 TL
	_, err = svc.CreateTransaction(ctx, tenantID, &domain.SupplierTransaction{
		TenantID:   tenantID,
		SupplierID: supplierMap["ATİKER"],
		PeriodID:   periodID,
		TxDate:     time.Now(),
		Direction:  domain.SupplierDirectionPurchase,
		Amount:     decimal.NewFromFloat(50000.50),
		CreatedBy:  &userID,
	})
	require.NoError(t, err)

	_, err = svc.CreateTransaction(ctx, tenantID, &domain.SupplierTransaction{
		TenantID:   tenantID,
		SupplierID: supplierMap["ATİKER"],
		PeriodID:   periodID,
		TxDate:     time.Now(),
		Direction:  domain.SupplierDirectionPayment,
		Amount:     decimal.NewFromFloat(20000.25),
		CreatedBy:  &userID,
	})
	require.NoError(t, err)

	// --- BRC ---
	// Payment only (Advance/Deposit): 15,000.00 TL
	// Expected BRC: Purchase = 0.00 TL, Payment = 15,000.00 TL, Balance = -15,000.00 TL
	_, err = svc.CreateTransaction(ctx, tenantID, &domain.SupplierTransaction{
		TenantID:   tenantID,
		SupplierID: supplierMap["BRC"],
		PeriodID:   periodID,
		TxDate:     time.Now(),
		Direction:  domain.SupplierDirectionPayment,
		Amount:     decimal.NewFromInt(15000),
		CreatedBy:  &userID,
	})
	require.NoError(t, err)

	// --- FİLTRECİ ---
	// Purchase: 16,590.00 TL, Payment: 7,500.00 TL
	// Expected FİLTRECİ: Purchase = 16,590.00 TL, Payment = 7,500.00 TL, Balance = 9,090.00 TL
	_, err = svc.CreateTransaction(ctx, tenantID, &domain.SupplierTransaction{
		TenantID:   tenantID,
		SupplierID: supplierMap["FİLTRECİ"],
		PeriodID:   periodID,
		TxDate:     time.Now(),
		Direction:  domain.SupplierDirectionPurchase,
		Amount:     decimal.NewFromFloat(16590.00),
		CreatedBy:  &userID,
	})
	require.NoError(t, err)

	_, err = svc.CreateTransaction(ctx, tenantID, &domain.SupplierTransaction{
		TenantID:   tenantID,
		SupplierID: supplierMap["FİLTRECİ"],
		PeriodID:   periodID,
		TxDate:     time.Now(),
		Direction:  domain.SupplierDirectionPayment,
		Amount:     decimal.NewFromFloat(7500.00),
		CreatedBy:  &userID,
	})
	require.NoError(t, err)

	// --- CANGAS & BOŞ TEDARİKÇİ ---
	// Zero transactions

	// 3. Perform Recalculation Check for every supplier
	suppliersWithBalances, err := svc.ListSuppliers(ctx, tenantID, &periodID)
	require.NoError(t, err)
	assert.Len(t, suppliersWithBalances, 6)

	resultMap := make(map[string]domain.Supplier)
	for _, s := range suppliersWithBalances {
		resultMap[s.Name] = s
		// Arithmetic invariant: TotalPurchase - TotalPayment == Balance
		expectedBalance := s.TotalPurchase.Sub(s.TotalPayment)
		assert.True(t, s.Balance.Equal(expectedBalance),
			"Supplier %s balance calculation mismatch: expected %s, got %s", s.Name, expectedBalance, s.Balance)
	}

	// Verify ASİL GRUP
	asil := resultMap["ASİL GRUP"]
	assert.Equal(t, "90000.00", asil.TotalPurchase.StringFixed(2))
	assert.Equal(t, "40000.00", asil.TotalPayment.StringFixed(2))
	assert.Equal(t, "50000.00", asil.Balance.StringFixed(2))

	// Verify ATİKER
	atiker := resultMap["ATİKER"]
	assert.Equal(t, "50000.50", atiker.TotalPurchase.StringFixed(2))
	assert.Equal(t, "20000.25", atiker.TotalPayment.StringFixed(2))
	assert.Equal(t, "30000.25", atiker.Balance.StringFixed(2))

	// Verify BRC
	brc := resultMap["BRC"]
	assert.Equal(t, "0.00", brc.TotalPurchase.StringFixed(2))
	assert.Equal(t, "15000.00", brc.TotalPayment.StringFixed(2))
	assert.Equal(t, "-15000.00", brc.Balance.StringFixed(2))

	// Verify FİLTRECİ
	filtreci := resultMap["FİLTRECİ"]
	assert.Equal(t, "16590.00", filtreci.TotalPurchase.StringFixed(2))
	assert.Equal(t, "7500.00", filtreci.TotalPayment.StringFixed(2))
	assert.Equal(t, "9090.00", filtreci.Balance.StringFixed(2))

	// Verify CANGAS & BOŞ TEDARİKÇİ
	cangas := resultMap["CANGAS"]
	assert.Equal(t, "0.00", cangas.TotalPurchase.StringFixed(2))
	assert.Equal(t, "0.00", cangas.TotalPayment.StringFixed(2))
	assert.Equal(t, "0.00", cangas.Balance.StringFixed(2))
	assert.Equal(t, 0, cangas.TransactionCount)

	// 4. Verify Global Summary Recalculation
	summary, err := svc.GetSummary(ctx, tenantID, &periodID)
	require.NoError(t, err)

	// Total Purchases: 90,000 + 50,000.50 + 16,590 = 156,590.50
	expectedTotalPurchases := decimal.RequireFromString("156590.50")
	// Total Payments: 40,000 + 20,000.25 + 15,000 + 7,500 = 82,500.25
	expectedTotalPayments := decimal.RequireFromString("82500.25")
	// Net Balance: 156,590.50 - 82,500.25 = 74,090.25
	expectedNetBalance := decimal.RequireFromString("74090.25")

	assert.True(t, summary.TotalPurchases.Equal(expectedTotalPurchases), "Expected %s, got %s", expectedTotalPurchases, summary.TotalPurchases)
	assert.True(t, summary.TotalPayments.Equal(expectedTotalPayments), "Expected %s, got %s", expectedTotalPayments, summary.TotalPayments)
	assert.True(t, summary.NetBalance.Equal(expectedNetBalance), "Expected %s, got %s", expectedNetBalance, summary.NetBalance)
	assert.Equal(t, 6, summary.ActiveSupplierCount)
}
