package service_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/service"
)

// MockSupplierRepo implements domain.SupplierRepository for testing.
type MockSupplierRepo struct {
	suppliers    map[uuid.UUID]*domain.Supplier
	transactions []domain.SupplierTransaction
}

func NewMockSupplierRepo() *MockSupplierRepo {
	return &MockSupplierRepo{
		suppliers:    make(map[uuid.UUID]*domain.Supplier),
		transactions: make([]domain.SupplierTransaction, 0),
	}
}

func (m *MockSupplierRepo) GetSuppliersWithBalances(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) ([]domain.Supplier, error) {
	reversedMap := make(map[uuid.UUID]bool)
	for _, tx := range m.transactions {
		if tx.ReversedBy != nil {
			reversedMap[*tx.ReversedBy] = true
		}
	}

	list := make([]domain.Supplier, 0)
	for _, s := range m.suppliers {
		if s.TenantID == tenantID {
			supplierCopy := *s
			supplierCopy.TotalPurchase = decimal.Zero
			supplierCopy.TotalPayment = decimal.Zero
			supplierCopy.TransactionCount = 0

			for _, tx := range m.transactions {
				if tx.TenantID == tenantID && tx.SupplierID == s.ID {
					if tx.ReversedBy != nil || reversedMap[tx.ID] {
						continue
					}
					if periodID != nil && *periodID != uuid.Nil && tx.PeriodID != *periodID {
						continue
					}
					supplierCopy.TransactionCount++
					if tx.Direction == domain.SupplierDirectionPurchase {
						supplierCopy.TotalPurchase = supplierCopy.TotalPurchase.Add(tx.Amount)
					} else if tx.Direction == domain.SupplierDirectionPayment {
						supplierCopy.TotalPayment = supplierCopy.TotalPayment.Add(tx.Amount)
					}
				}
			}
			supplierCopy.Balance = supplierCopy.TotalPurchase.Sub(supplierCopy.TotalPayment)
			list = append(list, supplierCopy)
		}
	}
	return list, nil
}

func (m *MockSupplierRepo) GetSupplierByID(ctx context.Context, tenantID, supplierID uuid.UUID) (*domain.Supplier, error) {
	s, ok := m.suppliers[supplierID]
	if !ok || s.TenantID != tenantID {
		return nil, domain.ErrSupplierNotFound
	}
	reversedMap := make(map[uuid.UUID]bool)
	for _, tx := range m.transactions {
		if tx.ReversedBy != nil {
			reversedMap[*tx.ReversedBy] = true
		}
	}
	supplierCopy := *s
	supplierCopy.TotalPurchase = decimal.Zero
	supplierCopy.TotalPayment = decimal.Zero
	supplierCopy.TransactionCount = 0

	for _, tx := range m.transactions {
		if tx.TenantID == tenantID && tx.SupplierID == s.ID {
			if tx.ReversedBy != nil || reversedMap[tx.ID] {
				continue
			}
			supplierCopy.TransactionCount++
			if tx.Direction == domain.SupplierDirectionPurchase {
				supplierCopy.TotalPurchase = supplierCopy.TotalPurchase.Add(tx.Amount)
			} else if tx.Direction == domain.SupplierDirectionPayment {
				supplierCopy.TotalPayment = supplierCopy.TotalPayment.Add(tx.Amount)
			}
		}
	}
	supplierCopy.Balance = supplierCopy.TotalPurchase.Sub(supplierCopy.TotalPayment)
	return &supplierCopy, nil
}

func (m *MockSupplierRepo) FindOrCreateSupplier(ctx context.Context, tenantID uuid.UUID, name string) (*domain.Supplier, error) {
	for _, s := range m.suppliers {
		if s.TenantID == tenantID && s.Name == name {
			return s, nil
		}
	}
	newSupplier := &domain.Supplier{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	m.suppliers[newSupplier.ID] = newSupplier
	return newSupplier, nil
}

func (m *MockSupplierRepo) CreateSupplier(ctx context.Context, s *domain.Supplier) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	m.suppliers[s.ID] = s
	return nil
}

func (m *MockSupplierRepo) CreateTransaction(ctx context.Context, tx *domain.SupplierTransaction) error {
	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}
	m.transactions = append(m.transactions, *tx)
	return nil
}

func (m *MockSupplierRepo) GetTransactionsBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter domain.SupplierTransactionFilter) ([]domain.SupplierTransaction, int, error) {
	filter.SupplierID = &supplierID
	return m.GetAllTransactions(ctx, tenantID, filter)
}

func (m *MockSupplierRepo) GetAllTransactions(ctx context.Context, tenantID uuid.UUID, filter domain.SupplierTransactionFilter) ([]domain.SupplierTransaction, int, error) {
	result := make([]domain.SupplierTransaction, 0)
	for _, tx := range m.transactions {
		if tx.TenantID != tenantID {
			continue
		}
		if filter.SupplierID != nil && tx.SupplierID != *filter.SupplierID {
			continue
		}
		result = append(result, tx)
	}
	return result, len(result), nil
}

func (m *MockSupplierRepo) GetTransactionByID(ctx context.Context, tenantID, txID uuid.UUID) (*domain.SupplierTransaction, error) {
	for _, tx := range m.transactions {
		if tx.TenantID == tenantID && tx.ID == txID {
			return &tx, nil
		}
	}
	return nil, domain.ErrTransactionNotFound
}

func (m *MockSupplierRepo) ReverseTransaction(ctx context.Context, tenantID, origID uuid.UUID, revTx *domain.SupplierTransaction) error {
	found := false
	for i, tx := range m.transactions {
		if tx.TenantID == tenantID && tx.ID == origID {
			found = true
			m.transactions[i].ReversedBy = &revTx.ID
			break
		}
	}
	if !found {
		return domain.ErrTransactionNotFound
	}
	m.transactions = append(m.transactions, *revTx)
	return nil
}

func (m *MockSupplierRepo) BatchInsertTransactions(ctx context.Context, tenantID uuid.UUID, transactions []domain.SupplierTransaction) (int, error) {
	for _, tx := range transactions {
		tx.TenantID = tenantID
		m.transactions = append(m.transactions, tx)
	}
	return len(transactions), nil
}

func (m *MockSupplierRepo) GetSummary(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) (*domain.SupplierSummary, error) {
	summary := &domain.SupplierSummary{
		TotalPurchases:      decimal.Zero,
		TotalPayments:       decimal.Zero,
		NetBalance:          decimal.Zero,
		ActiveSupplierCount: len(m.suppliers),
	}
	reversedMap := make(map[uuid.UUID]bool)
	for _, tx := range m.transactions {
		if tx.ReversedBy != nil {
			reversedMap[*tx.ReversedBy] = true
		}
	}

	for _, tx := range m.transactions {
		if tx.TenantID != tenantID {
			continue
		}
		if tx.ReversedBy != nil || reversedMap[tx.ID] {
			continue
		}
		if periodID != nil && *periodID != uuid.Nil && tx.PeriodID != *periodID {
			continue
		}
		if tx.Direction == domain.SupplierDirectionPurchase {
			summary.TotalPurchases = summary.TotalPurchases.Add(tx.Amount)
		} else if tx.Direction == domain.SupplierDirectionPayment {
			summary.TotalPayments = summary.TotalPayments.Add(tx.Amount)
		}
	}
	summary.NetBalance = summary.TotalPurchases.Sub(summary.TotalPayments)
	summary.TotalTransactionRows = len(m.transactions)
	return summary, nil
}

// Tests

func TestParseTurkishDecimal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasErr   bool
	}{
		{"1.250,50 TL", "1250.5", false},
		{"₺ 45.000,00", "45000", false},
		{"1500,75", "1500.75", false},
		{"250.00", "250", false},
		{"0", "0", false},
		{"-", "0", false},
		{"", "0", false},
	}

	for _, tc := range tests {
		res, err := service.ParseTurkishDecimal(tc.input)
		if tc.hasErr {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			expectedDec, _ := decimal.NewFromString(tc.expected)
			assert.True(t, res.Equal(expectedDec), "Expected %s for input %s, got %s", tc.expected, tc.input, res.String())
		}
	}
}

func TestParseFlexibleDate(t *testing.T) {
	d, err := service.ParseFlexibleDate("15.08.2026")
	require.NoError(t, err)
	assert.Equal(t, 2026, d.Year())
	assert.Equal(t, time.Month(8), d.Month())
	assert.Equal(t, 15, d.Day())

	// Excel serial date 45321 (around 2024-01-30)
	d2, err := service.ParseFlexibleDate("45321")
	require.NoError(t, err)
	assert.Equal(t, 2024, d2.Year())
}

func TestSupplierService_ImportExcel(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	periodID := uuid.New()

	mockPeriodRepo := new(MockPeriodRepo)
	mockPeriodRepo.On("GetByID", mock.Anything, periodID).Return(&domain.Period{
		ID:        periodID,
		TenantID:  tenantID,
		Label:     "2026-08",
		Status:    domain.PeriodStatusOpen,
		OpenedAt:  time.Now(),
	}, nil)

	mockSupplierRepo := NewMockSupplierRepo()
	svc := service.NewSupplierService(mockSupplierRepo, mockPeriodRepo)

	// Create test Excel file using excelize
	f := excelize.NewFile()
	sheetName := "AĞUSTOS26"
	_, err := f.NewSheet(sheetName)
	require.NoError(t, err)

	// Row 1: Header with Supplier names
	_ = f.SetCellValue(sheetName, "A1", "ATİKER")
	_ = f.SetCellValue(sheetName, "I1", "PRİNS")

	// Row 2: Sub-headers (Dynamic headers)
	_ = f.SetCellValue(sheetName, "A2", "Devir Bakiyesi")
	_ = f.SetCellValue(sheetName, "B2", "Fatura No")
	_ = f.SetCellValue(sheetName, "C2", "Ad Soyad")
	_ = f.SetCellValue(sheetName, "D2", "Evrak Durumu")
	_ = f.SetCellValue(sheetName, "E2", "Tarih")
	_ = f.SetCellValue(sheetName, "F2", "Geçilen Tutar")
	_ = f.SetCellValue(sheetName, "G2", "Alınan Tutar")
	_ = f.SetCellValue(sheetName, "H2", "Bakiye")

	_ = f.SetCellValue(sheetName, "I2", "Devir Bakiyesi")
	_ = f.SetCellValue(sheetName, "J2", "Fatura No")
	_ = f.SetCellValue(sheetName, "K2", "Ad Soyad")
	_ = f.SetCellValue(sheetName, "L2", "Evrak Durumu")
	_ = f.SetCellValue(sheetName, "M2", "Tarih")
	_ = f.SetCellValue(sheetName, "N2", "Geçilen Tutar")
	_ = f.SetCellValue(sheetName, "O2", "Alınan Tutar")
	_ = f.SetCellValue(sheetName, "P2", "Bakiye")

	// Row 3: Devir Row (should be automatically skipped)
	_ = f.SetCellValue(sheetName, "A3", "100.000,00 TL")
	_ = f.SetCellValue(sheetName, "B3", "DEVİR")
	_ = f.SetCellValue(sheetName, "H3", "100.000,00 TL")

	// Row 4: Data row for Atiker and Prins
	_ = f.SetCellValue(sheetName, "B4", "FAT-001")
	_ = f.SetCellValue(sheetName, "C4", "Ahmet Yılmaz")
	_ = f.SetCellValue(sheetName, "D4", "Faturalı")
	_ = f.SetCellValue(sheetName, "E4", "05.08.2026")
	_ = f.SetCellValue(sheetName, "F4", "5.000,00 TL")  // Geçilen (payment)
	_ = f.SetCellValue(sheetName, "G4", "12.500,00 TL") // Alınan (purchase)

	_ = f.SetCellValue(sheetName, "J4", "FAT-002")
	_ = f.SetCellValue(sheetName, "K4", "Mehmet Kaya")
	_ = f.SetCellValue(sheetName, "L4", "İrsaliyeli")
	_ = f.SetCellValue(sheetName, "M4", "06.08.2026")
	_ = f.SetCellValue(sheetName, "N4", "10.000,00 TL") // Geçilen (payment)
	_ = f.SetCellValue(sheetName, "O4", "25.000,00 TL") // Alınan (purchase)

	var buf bytes.Buffer
	err = f.Write(&buf)
	require.NoError(t, err)

	// Execute Import with auto period matching (without specifying sheet name)
	res, err := svc.ImportExcel(ctx, tenantID, periodID, &buf, "", nil)
	require.NoError(t, err)
	assert.NotNil(t, res)

	// Verify imported count (2 purchase + 2 payment = 4 transactions, DEVİR row was skipped)
	assert.Equal(t, 4, res.ImportedCount)
	assert.Equal(t, 2, len(res.SuppliersCreated))
	assert.Contains(t, res.SuppliersCreated, "ATİKER")
	assert.Contains(t, res.SuppliersCreated, "PRİNS")

	// Check total amounts (12500 + 25000 = 37500 purchase, 5000 + 10000 = 15000 payment)
	expectedPurchase := decimal.NewFromInt(12500).Add(decimal.NewFromInt(25000))
	expectedPayment := decimal.NewFromInt(5000).Add(decimal.NewFromInt(10000))
	assert.True(t, res.TotalPurchase.Equal(expectedPurchase), "TotalPurchase mismatch: %s != %s", res.TotalPurchase, expectedPurchase)
	assert.True(t, res.TotalPayment.Equal(expectedPayment), "TotalPayment mismatch: %s != %s", res.TotalPayment, expectedPayment)

	// Verify direction in stored transactions
	for _, tx := range mockSupplierRepo.transactions {
		if tx.InvoiceNo == "FAT-001" && tx.Direction == domain.SupplierDirectionPurchase {
			assert.True(t, tx.Amount.Equal(decimal.NewFromInt(12500)))
		}
		if tx.InvoiceNo == "FAT-001" && tx.Direction == domain.SupplierDirectionPayment {
			assert.True(t, tx.Amount.Equal(decimal.NewFromInt(5000)))
		}
	}

	// Check summary
	summary, err := svc.GetSummary(ctx, tenantID, &periodID)
	require.NoError(t, err)
	assert.True(t, summary.NetBalance.Equal(expectedPurchase.Sub(expectedPayment)))
}

func TestSupplierService_ImportExcel_UnmatchedSheetReturnsError(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	periodID := uuid.New()

	mockPeriodRepo := new(MockPeriodRepo)
	mockPeriodRepo.On("GetByID", mock.Anything, periodID).Return(&domain.Period{
		ID:        periodID,
		TenantID:  tenantID,
		Label:     "2026-11", // KASIM
		Status:    domain.PeriodStatusOpen,
		OpenedAt:  time.Now(),
	}, nil)

	mockSupplierRepo := NewMockSupplierRepo()
	svc := service.NewSupplierService(mockSupplierRepo, mockPeriodRepo)

	f := excelize.NewFile()
	_, _ = f.NewSheet("OCAK 2024")
	var buf bytes.Buffer
	_ = f.Write(&buf)

	// Attempt import with period 2026-11 on a file only containing OCAK 2024
	_, err := svc.ImportExcel(ctx, tenantID, periodID, &buf, "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "seçili dönem (2026-11) için uygun Excel sayfası bulunamadı")
}

func TestSupplierService_ReversalDoesNotInflatePaymentSummary(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	periodID := uuid.New()

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

	// 1. Bir tedarikçi oluştur
	supplier, err := svc.CreateSupplier(ctx, tenantID, "ATİKER")
	require.NoError(t, err)

	// 2. 4 adet alış faturası gir (Toplam: 50.467,56 TL)
	amounts := []string{"10000.00", "15467.56", "20000.00", "5000.00"}
	var originalTxIDs []uuid.UUID

	for _, amtStr := range amounts {
		amt, err := decimal.NewFromString(amtStr)
		require.NoError(t, err)

		tx := &domain.SupplierTransaction{
			ID:         uuid.New(),
			SupplierID: supplier.ID,
			PeriodID:   periodID,
			Direction:  domain.SupplierDirectionPurchase,
			Amount:     amt,
			InvoiceNo:  "FAT-" + amtStr,
		}
		created, err := svc.CreateTransaction(ctx, tenantID, tx)
		require.NoError(t, err)
		originalTxIDs = append(originalTxIDs, created.ID)
	}

	// 3. İptal öncesi kontrol: TotalPurchase = 50467.56, TotalPayment = 0.00, Balance = 50467.56
	expectedTotal, _ := decimal.NewFromString("50467.56")
	summaryBefore, err := svc.GetSummary(ctx, tenantID, &periodID)
	require.NoError(t, err)
	assert.True(t, summaryBefore.TotalPurchases.Equal(expectedTotal), "İptal öncesi alış toplamı 50467.56 TL olmalı")
	assert.True(t, summaryBefore.TotalPayments.IsZero(), "İptal öncesi ödeme toplamı 0 TL olmalı")
	assert.True(t, summaryBefore.NetBalance.Equal(expectedTotal), "İptal öncesi net borç 50467.56 TL olmalı")

	// 4. 4 adet faturayı ters kayıtla iptal et
	for _, txID := range originalTxIDs {
		revTx, err := svc.ReverseTransaction(ctx, tenantID, txID, "Fatura iptali", nil)
		require.NoError(t, err)
		assert.NotNil(t, revTx)
		assert.Equal(t, domain.SupplierDirectionPayment, revTx.Direction)
	}

	// 5. İPTAL SONRASI KRİTİK KONTROL:
	// Ödeme tutarı ASLA 50.467,56 TL'ye şişmemeli!
	// TotalPurchases = 0.00, TotalPayments = 0.00, NetBalance = 0.00 olmalı.
	summaryAfter, err := svc.GetSummary(ctx, tenantID, &periodID)
	require.NoError(t, err)

	assert.True(t, summaryAfter.TotalPurchases.IsZero(), "İptal edilen faturalar alış toplamından düşmeli, 0.00 TL olmalı. Aldığı: %s", summaryAfter.TotalPurchases)
	assert.True(t, summaryAfter.TotalPayments.IsZero(), "İptal edilen alış faturası ödeme toplamını şişirmemeli (0.00 TL kalmalı). Aldığı: %s", summaryAfter.TotalPayments)
	assert.True(t, summaryAfter.NetBalance.IsZero(), "Net bakiye 0.00 TL olmalı. Aldığı: %s", summaryAfter.NetBalance)

	// 6. Tedarikçi bakiye listesinde de kontrol
	suppliers, err := svc.ListSuppliers(ctx, tenantID, &periodID)
	require.NoError(t, err)
	require.Len(t, suppliers, 1)
	assert.True(t, suppliers[0].TotalPurchase.IsZero())
	assert.True(t, suppliers[0].TotalPayment.IsZero())
	assert.True(t, suppliers[0].Balance.IsZero())
	assert.Equal(t, 0, suppliers[0].TransactionCount)
}
