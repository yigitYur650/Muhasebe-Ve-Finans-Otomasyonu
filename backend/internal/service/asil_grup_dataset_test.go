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

type asilGrupItem struct {
	Date      string
	InvoiceNo string
	Customer  string
	Status    string
	Direction string
	Amount    string
	Desc      string
}

// TestAsilGrupReversalSimulation_PureMock, veritabanına bağlanmadan
// tamamen bellek içi (in-memory mock) üzerinde kullanıcının 60 satırlık ASİL GRUP
// verilerini ve ₺50.467,56 tutarındaki 4 faturanın ters kayıt simülasyonunu doğrular.
func TestAsilGrupReversalSimulation_PureMock(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	periodID := uuid.New()

	mockPeriodRepo := new(MockPeriodRepo)
	mockPeriodRepo.On("GetByID", mock.Anything, periodID).Return(&domain.Period{
		ID:       periodID,
		TenantID: tenantID,
		Label:    "2026-10",
		Status:   domain.PeriodStatusOpen,
		OpenedAt: time.Now(),
	}, nil)

	mockSupplierRepo := NewMockSupplierRepo()
	svc := service.NewSupplierService(mockSupplierRepo, mockPeriodRepo)

	// 1. Tedarikçiyi oluştur
	supplier, err := svc.CreateSupplier(ctx, tenantID, "ASİL GRUP")
	require.NoError(t, err)

	// 2. Kullanıcının verdiği net 60 satırlık ASİL GRUP hareketleri
	baseRows := []asilGrupItem{
		{"08.10.2026", "-", "BİLAL ÜN AL", "Diğer", "payment", "3500.00", "YEDEK PARÇA"},
		{"08.10.2026", "-", "-", "Diğer", "payment", "914.02", "NAKİT"},
		{"06.10.2026", "-", "-", "Faturalı", "purchase", "21979.28", "-"},
		{"06.10.2026", "OOA2026000000051", "MEHMET AYDOĞAN", "Faturalı", "payment", "8000.00", "EVRAK VERİLDİ AYŞE AYDOĞAN ADINA"},
		{"06.10.2026", "OOA2026000000050", "İBRAHİM EFE", "Faturalı", "payment", "5500.00", "EVRAK VERİLDİ"},
		{"06.10.2026", "ASL2026000011180", "-", "Faturalı", "purchase", "18188.16", "-"},
		{"05.10.2026", "-", "-", "Faturalı", "purchase", "11179.47", "-"},
		{"05.10.2026", "-", "-", "Faturalı", "purchase", "20763.47", "-"},
		{"03.10.2026", "ASL2026000011053", "-", "Faturalı", "purchase", "18723.23", "-"},
		{"30.09.2026", "-", "DENİZ ÜNAL", "Fiş / Makbuz", "payment", "30500.00", "EVRAK DAHA SONRA"},
		{"30.09.2026", "-", "EMİN OYMAK", "Fiş / Makbuz", "payment", "5500.00", "FATURA KESİLMEDİ"},
		{"30.09.2026", "OOA2026000000044", "ŞÜKRÜ BURUŞ", "Faturalı", "payment", "7000.00", "EVRAK VERİLDİ"},
		{"30.09.2026", "OOA2026000000043", "ABDULLAH ERGİN", "Faturalı", "payment", "7000.00", "EVRAK VERİLDİ"},
		{"29.09.2026", "-", "-", "Faturalı", "purchase", "17546.71", "-"},
		{"29.09.2026", "OOF2026000000004", "ERKUT DEMİR", "Faturalı", "payment", "5500.00", "-"},
		{"29.09.2026", "-", "İBRAHİM TATAR", "Fiş / Makbuz", "payment", "2000.00", "-"},
		{"28.09.2026", "ASL2026000010817", "-", "Faturalı", "purchase", "14555.34", "-"},
		{"28.09.2026", "ASL2026000010829", "-", "Faturalı", "purchase", "9480.74", "MALZEME"},
		{"28.09.2026", "-", "ALİ RIZA TEKE", "Fiş / Makbuz", "payment", "5000.00", "-"},
		{"26.08.2026", "OOA2026000000041", "ATAULLAH SÖNMEZ", "Faturalı", "payment", "8000.00", "EVRAK VERİLDİ"},
		{"26.08.2026", "ASL2026000010747", "-", "Faturalı", "purchase", "17686.36", "-"},
		{"26.08.2026", "-", "SUAT ALTINOK", "Faturalı", "payment", "14000.00", "-"},
		{"26.08.2026", "-", "HALİL ARSOY", "Faturalı", "payment", "3500.00", "-"},
		{"26.08.2026", "ASL2026000010590", "-", "Faturalı", "purchase", "18234.63", "-"},
		{"26.08.2026", "-", "MURAT GÜNEŞ", "Faturalı", "payment", "6000.00", "-"},
		{"26.08.2026", "-", "-", "Faturalı", "purchase", "18554.90", "-"},
		{"26.08.2026", "ASL2026000010430", "-", "Faturalı", "purchase", "10745.34", "-"},
		{"26.08.2026", "ASL2026000010518", "-", "Faturalı", "purchase", "23382.59", "-"},
		{"26.08.2026", "-", "FERDİ DÜZENLİ", "Faturalı", "payment", "6000.00", "-"},
		{"26.08.2026", "-", "HASAN ÇIRAK", "Faturalı", "payment", "10000.00", "-"},
		{"26.08.2026", "ASL2026000010353", "-", "Faturalı", "purchase", "15817.92", "-"},
		{"26.08.2026", "-", "CEVAT KAÇAR", "Faturalı", "payment", "6000.00", "-"},
		{"26.08.2026", "-", "VELİ ÖZ", "Faturalı", "payment", "10000.00", "-"},
		{"26.08.2026", "ASL2026000010145", "-", "Faturalı", "purchase", "11629.74", "-"},
		{"26.08.2026", "-", "UFUK KESKEN", "Faturalı", "payment", "3500.00", "-"},
		{"26.08.2026", "-", "ÜLKÜ ÖZEL", "Faturalı", "payment", "17500.00", "-"},
		{"26.08.2026", "-", "MUSTAFA ÇİÇEK", "Faturalı", "payment", "16000.00", "-"},
		{"26.08.2026", "ASL2026000009907", "-", "Faturalı", "purchase", "11283.70", "-"},
		{"26.08.2026", "ASL2026000009907", "-", "Faturalı", "purchase", "2692.43", "-"},
		{"26.08.2026", "-", "MEHMET ÜREK", "Diğer", "payment", "6500.00", "-"},
		{"26.08.2026", "ASL2026000009896", "-", "Diğer", "purchase", "7440.54", "-"},
		{"26.08.2026", "ASL2026000009724", "-", "Diğer", "purchase", "18570.38", "-"},
		{"26.08.2026", "-", "MERVE DEMİRASLAN", "Diğer", "payment", "7000.00", "-"},
		{"26.08.2026", "ASL2026000009533", "-", "Diğer", "purchase", "4157.87", "-"},
		{"26.08.2026", "ASL2026000009531", "-", "Diğer", "purchase", "21619.51", "-"},
		{"26.08.2026", "-", "ÜMRAN GÖRÜNMEZ", "Diğer", "payment", "12000.00", "-"},
		{"26.08.2026", "ASL2026000009425", "-", "Fiş / Makbuz", "purchase", "38564.43", "-"},
		{"26.08.2026", "-", "ALİ YILDIRIM", "Fiş / Makbuz", "payment", "21000.00", "-"},
		{"26.08.2026", "-", "AHMET ÇELİK", "Faturalı", "payment", "20000.00", "-"},
		{"26.08.2026", "ASL2026000009297", "-", "Faturalı", "purchase", "10528.03", "-"},
		{"26.08.2026", "-", "MUAMMER BENLİ", "Faturalı", "payment", "5500.00", "-"},
		{"26.08.2026", "-", "KADRİYE ARMAN", "Faturalı", "payment", "1200.00", "-"},
		{"26.08.2026", "-", "EROL AYDINLIK", "Faturalı", "payment", "28000.00", "-"},
		{"26.08.2026", "-", "-", "Faturalı", "payment", "27000.00", "-"},
		{"26.08.2026", "ASL2026000009276", "-", "Faturalı", "purchase", "17451.26", "-"},
		{"26.08.2026", "ASL2026000010706", "-", "Faturalı", "purchase", "18866.48", "-"},
		{"26.08.2026", "-", "-", "Faturalı", "purchase", "25690.68", "-"},
		{"26.08.2026", "-", "-", "Faturalı", "purchase", "31138.74", "-"},
		{"26.08.2026", "-", "-", "Faturalı", "purchase", "12341.68", "-"},
		{"26.08.2026", "-", "CERAİL AÇIKGÖZ", "Faturalı", "payment", "27000.00", "-"},
	}

	for _, r := range baseRows {
		amt, err := decimal.NewFromString(r.Amount)
		require.NoError(t, err)
		parsedDate, _ := time.Parse("02.01.2006", r.Date)

		_, err = svc.CreateTransaction(ctx, tenantID, &domain.SupplierTransaction{
			ID:             uuid.New(),
			SupplierID:     supplier.ID,
			PeriodID:       periodID,
			InvoiceNo:      r.InvoiceNo,
			CustomerName:   r.Customer,
			DocumentStatus: r.Status,
			TxDate:         parsedDate,
			Direction:      r.Direction,
			Amount:         amt,
			Description:    r.Desc,
		})
		require.NoError(t, err)
	}

	// 3. İptal Edilen 4 Adet Faturanın Sisteme Girişi (Toplam ₺50.467,56)
	cancelledInvoices := []struct {
		InvoiceNo string
		Amount    string
	}{
		{"ASL-IPTAL-001", "15467.56"},
		{"ASL-IPTAL-002", "15000.00"},
		{"ASL-IPTAL-003", "10000.00"},
		{"ASL-IPTAL-004", "10000.00"},
	}

	var cancelledTxIDs []uuid.UUID
	for _, inv := range cancelledInvoices {
		amt, _ := decimal.NewFromString(inv.Amount)
		created, err := svc.CreateTransaction(ctx, tenantID, &domain.SupplierTransaction{
			ID:         uuid.New(),
			SupplierID: supplier.ID,
			PeriodID:   periodID,
			InvoiceNo:  inv.InvoiceNo,
			Direction:  domain.SupplierDirectionPurchase,
			Amount:     amt,
		})
		require.NoError(t, err)
		cancelledTxIDs = append(cancelledTxIDs, created.ID)
	}

	// İptal öncesi kontrol: Alış 468.813,61 + 50.467,56 = 519.281,17 TL
	expectedInflatedPurchase, _ := decimal.NewFromString("519281.17")
	summaryBeforeReversal, err := svc.GetSummary(ctx, tenantID, &periodID)
	require.NoError(t, err)
	assert.True(t, summaryBeforeReversal.TotalPurchases.Equal(expectedInflatedPurchase))

	// 4. Bu 4 Faturanın Ters Kayıt ile İptal Edilmesi
	for _, txID := range cancelledTxIDs {
		revTx, err := svc.ReverseTransaction(ctx, tenantID, txID, "Fatura İptali", nil)
		require.NoError(t, err)
		assert.NotNil(t, revTx)
		assert.Equal(t, domain.SupplierDirectionPayment, revTx.Direction)
	}

	// 5. İPTAL SONRASI NİHAİ KONTROL:
	// Task 1 düzeltmesi sayesinde:
	// - Alış Toplamı: 519.281,17 TL'den 50.467,56 TL düşüp NET 468.813,61 TL kalmalı
	// - Ödeme Toplamı: ASLA 386.581,58 TL'ye ŞİŞMEMELİ, NET 336.114,02 TL kalmalı
	// - Kalan Borç Bakiyesi: 132.699,59 TL olarak kuruşu kuruşuna korunmalı!
	summaryAfterReversal, err := svc.GetSummary(ctx, tenantID, &periodID)
	require.NoError(t, err)

	expectedNetPurchase, _ := decimal.NewFromString("468813.61")
	expectedNetPayment, _ := decimal.NewFromString("336114.02")
	expectedNetBalance, _ := decimal.NewFromString("132699.59")

	t.Logf("✅ Simülasyon Sonucu:")
	t.Logf("   Net Alış: %s (Beklenen: %s)", summaryAfterReversal.TotalPurchases, expectedNetPurchase)
	t.Logf("   Net Ödeme: %s (Beklenen: %s - Şişme Yok!)", summaryAfterReversal.TotalPayments, expectedNetPayment)
	t.Logf("   Kalan Borç Bakiyesi: %s (Beklenen: %s)", summaryAfterReversal.NetBalance, expectedNetBalance)

	assert.True(t, summaryAfterReversal.TotalPurchases.Equal(expectedNetPurchase), "Net Alış 468.813,61 TL olmalı")
	assert.True(t, summaryAfterReversal.TotalPayments.Equal(expectedNetPayment), "Net Ödeme şişmemeli, 336.114,02 TL kalmalı")
	assert.True(t, summaryAfterReversal.NetBalance.Equal(expectedNetBalance), "Kalan Borç 132.699,59 TL olmalı")
}
