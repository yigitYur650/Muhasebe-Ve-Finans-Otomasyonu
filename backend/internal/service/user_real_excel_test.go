package service_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/repository"
	"deftersystem/backend/internal/service"
)

func TestUserRealExcelFiles_ImportAndValidation(t *testing.T) {
	file1Path := `C:\Users\yigit\OneDrive\Desktop\Yeni klasör (3)\defter-2026-08-aktif-kayitlar (1).xlsx`
	file2Path := `C:\Users\yigit\OneDrive\Desktop\Yeni klasör (3)\KASA DEFTERİM 2026.xlsx`

	// -------------------------------------------------------------------------
	// 1. TEST: Dosya 1 (defter-2026-08-aktif-kayitlar)
	// -------------------------------------------------------------------------
	t.Run("File 1: defter-2026-08-aktif-kayitlar", func(t *testing.T) {
		info, err := os.Stat(file1Path)
		require.NoError(t, err, "Dosya 1 mevcut olmalıdır")
		t.Logf("📁 Dosya 1 Boyutu: %d KB", info.Size()/1024)

		f, err := excelize.OpenFile(file1Path)
		require.NoError(t, err, "Excel dosyası başarıyla açılabilmelidir")
		defer f.Close()

		sheetList := f.GetSheetList()
		t.Logf("📄 Dosya 1 Sayfa Listesi: %v", sheetList)
		require.NotEmpty(t, sheetList)

		firstSheet := sheetList[0]
		rows, err := f.GetRows(firstSheet)
		require.NoError(t, err)
		t.Logf("📊 [%s] sayfasında toplam satır sayısı: %d", firstSheet, len(rows))
		require.Greater(t, len(rows), 0)

		// Başlık ve ilk satır örnekleri
		for i := 0; i < len(rows) && i < 5; i++ {
			t.Logf("   Satır %d: %v", i+1, rows[i])
		}
	})

	// -------------------------------------------------------------------------
	// 2. TEST: Dosya 2 (KASA DEFTERİM 2026) -> Tedarikçi ve Cari İçe Aktarma Motoru
	// -------------------------------------------------------------------------
	t.Run("File 2: KASA DEFTERİM 2026 Tedarikçi & Cari Import", func(t *testing.T) {
		info, err := os.Stat(file2Path)
		require.NoError(t, err, "Dosya 2 mevcut olmalıdır")
		t.Logf("📁 Dosya 2 Boyutu: %d KB", info.Size()/1024)

		f, err := excelize.OpenFile(file2Path)
		require.NoError(t, err)
		defer f.Close()

		sheetList := f.GetSheetList()
		t.Logf("📄 Toplam Sayfa Sayısı: %d", len(sheetList))

		// Supplier Service ile gerçek import testi
		tenantID := uuid.New()
		periodID := uuid.New()
		userID := uuid.New()

		periodRepo := repository.NewMockPeriodRepo()
		supplierRepo := repository.NewMockSupplierRepository()
		supplierSvc := service.NewSupplierService(supplierRepo, periodRepo)

		periodRepo.Create(context.Background(), &domain.Period{
			ID:              periodID,
			TenantID:        tenantID,
			Label:           "2026-08",
			StartingBalance: decimal.Zero,
			Status:          domain.PeriodStatusOpen,
			OpenedAt:        time.Now(),
		})

		// AĞUSTOS26 sayfasını içe aktar
		targetSheet := "AĞUSTOS26"
		fileReader, err := os.Open(file2Path)
		require.NoError(t, err)
		defer fileReader.Close()

		result, err := supplierSvc.ImportExcel(context.Background(), tenantID, periodID, fileReader, targetSheet, &userID)
		require.NoError(t, err, "AĞUSTOS26 sayfası hatasız içe aktarılmalıdır")

		t.Logf("✅ Import Sonucu:")
		t.Logf("   Toplam Satır: %d", result.TotalRows)
		t.Logf("   İçe Aktarılan İşlem: %d", result.ImportedCount)
		t.Logf("   Oluşturulan Tedarikçiler: %v", result.SuppliersCreated)
		t.Logf("   Toplam Alınan Mal: %s ₺", result.TotalPurchase.StringFixed(2))
		t.Logf("   Toplam Yapılan Ödeme: %s ₺", result.TotalPayment.StringFixed(2))

		assert.Greater(t, result.ImportedCount, 0, "En az bir cari hareket aktarılmış olmalıdır")
		assert.NotEmpty(t, result.SuppliersCreated, "Tedarikçi kartları oluşmalıdır")

		// Tedarikçi bakiyelerinin hesaplandığı doğrulanır
		suppliers, err := supplierSvc.ListSuppliers(context.Background(), tenantID, &periodID)
		require.NoError(t, err)
		t.Logf("🚚 Sistemde Oluşan Tedarikçi Sayısı: %d", len(suppliers))
		for _, s := range suppliers {
			t.Logf("   - %-20s | Alınan: %12s ₺ | Ödenen: %12s ₺ | Bakiye: %12s ₺",
				s.Name, s.TotalPurchase.StringFixed(2), s.TotalPayment.StringFixed(2), s.Balance.StringFixed(2))
		}

		// Özet KPI doğrulanır
		summary, err := supplierSvc.GetSummary(context.Background(), tenantID, &periodID)
		require.NoError(t, err)
		t.Logf("📊 Genel Tedarikçi Özeti: Toplam Alınan: %s ₺ | Toplam Ödenen: %s ₺ | Net Borç: %s ₺",
			summary.TotalPurchases.StringFixed(2), summary.TotalPayments.StringFixed(2), summary.NetBalance.StringFixed(2))

		assert.True(t, summary.NetBalance.Equal(summary.TotalPurchases.Sub(summary.TotalPayments)))
	})

	// -------------------------------------------------------------------------
	// 3. TEST: Dosya 2 (KASA DEFTERİM 2026) -> Farklı Sayfaların (OCAK26, ŞUBAT26, MART26) Uyumluluğu
	// -------------------------------------------------------------------------
	t.Run("File 2: Çoklu Sayfa Uyumluluk Taraması (OCAK26, ŞUBAT26, MART26)", func(t *testing.T) {
		tenantID := uuid.New()
		periodID := uuid.New()
		userID := uuid.New()

		periodRepo := repository.NewMockPeriodRepo()
		supplierRepo := repository.NewMockSupplierRepository()
		supplierSvc := service.NewSupplierService(supplierRepo, periodRepo)

		periodRepo.Create(context.Background(), &domain.Period{
			ID:              periodID,
			TenantID:        tenantID,
			Label:           "2026-01",
			StartingBalance: decimal.Zero,
			Status:          domain.PeriodStatusOpen,
			OpenedAt:        time.Now(),
		})

		for _, sheetName := range []string{"OCAK26", "ŞUBAT26", "MART26", "MAYIS26"} {
			fReader, err := os.Open(file2Path)
			require.NoError(t, err)
			res, err := supplierSvc.ImportExcel(context.Background(), tenantID, periodID, fReader, sheetName, &userID)
			fReader.Close()
			if err == nil {
				t.Logf("✅ [%s] Sayfası Başarılı: %d satır aktarıldı (Tedarikçiler: %v)", sheetName, res.ImportedCount, res.SuppliersCreated)
			} else {
				t.Logf("ℹ️ [%s] Sayfası: %v", sheetName, err)
			}
		}
	})
}
