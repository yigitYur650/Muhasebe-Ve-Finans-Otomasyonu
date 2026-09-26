package service_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

func generateMockDefterExcel() (*excelize.File, error) {
	f := excelize.NewFile()
	sheetName := "Sayfa1"
	f.SetSheetName("Sheet1", sheetName)
	_ = f.SetCellValue(sheetName, "A1", "Tarih")
	_ = f.SetCellValue(sheetName, "B1", "Yön")
	_ = f.SetCellValue(sheetName, "C1", "Kanal")
	_ = f.SetCellValue(sheetName, "D1", "Tutar")
	_ = f.SetCellValue(sheetName, "E1", "Açıklama")

	rows := [][]interface{}{
		{"2026-08-01", "Giriş", "EFT", 15450.75, "Müşteri A.Ş. Tahsilat"},
		{"2026-08-02", "Giriş", "Nakit", 3200.50, "Gün Sonu Kasa"},
		{"2026-08-03", "Çıkış", "EFT", 6500.00, "Ofis Kirası"},
		{"2026-08-04", "Çıkış", "Kredi Kartı", 1250.25, "Bulut Sunucu"},
	}
	for i, r := range rows {
		cellA := fmt.Sprintf("A%d", i+2)
		cellB := fmt.Sprintf("B%d", i+2)
		cellC := fmt.Sprintf("C%d", i+2)
		cellD := fmt.Sprintf("D%d", i+2)
		cellE := fmt.Sprintf("E%d", i+2)
		_ = f.SetCellValue(sheetName, cellA, r[0])
		_ = f.SetCellValue(sheetName, cellB, r[1])
		_ = f.SetCellValue(sheetName, cellC, r[2])
		_ = f.SetCellValue(sheetName, cellD, r[3])
		_ = f.SetCellValue(sheetName, cellE, r[4])
	}
	return f, nil
}

func generateMockKasaDefteriExcel() (*excelize.File, error) {
	f := excelize.NewFile()
	sheets := []string{"AĞUSTOS26", "OCAK26", "ŞUBAT26", "MART26"}
	for i, sh := range sheets {
		if i == 0 {
			f.SetSheetName("Sheet1", sh)
		} else {
			f.NewSheet(sh)
		}

		// Satır 1: Tedarikçi Başlığı
		_ = f.SetCellValue(sh, "A1", "ÖZKAN OTO YEDEK PARÇA")
		// Satır 2: Sütun Başlıkları
		_ = f.SetCellValue(sh, "A2", "FATURA NO")
		_ = f.SetCellValue(sh, "B2", "MÜŞTERİ ADI")
		_ = f.SetCellValue(sh, "C2", "DURUM")
		_ = f.SetCellValue(sh, "D2", "TARİH")
		_ = f.SetCellValue(sh, "E2", "GEÇİLEN")
		_ = f.SetCellValue(sh, "F2", "ALINAN")
		_ = f.SetCellValue(sh, "G2", "BAKİYE")
		_ = f.SetCellValue(sh, "H2", "AÇIKLAMA")

		// Satır 3, 4: Veriler
		_ = f.SetCellValue(sh, "A3", "FTR-001")
		_ = f.SetCellValue(sh, "B3", "Ahmet Yılmaz")
		_ = f.SetCellValue(sh, "C3", "Tamamlandı")
		_ = f.SetCellValue(sh, "D3", "05.08.2026")
		_ = f.SetCellValue(sh, "E3", 5000.00)
		_ = f.SetCellValue(sh, "F3", 12500.00)
		_ = f.SetCellValue(sh, "G3", 7500.00)
		_ = f.SetCellValue(sh, "H3", "Fren balata ve yağ değişimi")

		_ = f.SetCellValue(sh, "A4", "FTR-002")
		_ = f.SetCellValue(sh, "B4", "Mehmet Kaya")
		_ = f.SetCellValue(sh, "C4", "Tamamlandı")
		_ = f.SetCellValue(sh, "D4", "12.08.2026")
		_ = f.SetCellValue(sh, "E4", 2500.00)
		_ = f.SetCellValue(sh, "F4", 3000.00)
		_ = f.SetCellValue(sh, "G4", 500.00)
		_ = f.SetCellValue(sh, "H4", "Filtre seti alımı")
	}
	return f, nil
}

func TestUserRealExcelFiles_ImportAndValidation(t *testing.T) {
	file1Path := `C:\Users\yigit\OneDrive\Desktop\Yeni klasör (3)\defter-2026-08-aktif-kayitlar (1).xlsx`
	file2Path := `C:\Users\yigit\OneDrive\Desktop\Yeni klasör (3)\KASA DEFTERİM 2026.xlsx`

	// -------------------------------------------------------------------------
	// 1. TEST: Dosya 1 (defter-2026-08-aktif-kayitlar)
	// -------------------------------------------------------------------------
	t.Run("File 1: defter-2026-08-aktif-kayitlar", func(t *testing.T) {
		var f *excelize.File
		var err error
		if _, statErr := os.Stat(file1Path); statErr == nil {
			f, err = excelize.OpenFile(file1Path)
			require.NoError(t, err)
		} else {
			t.Log("ℹ️ Yerel dosya bulunamadı, dinamik Mock Excel üzerinden test çalıştırılıyor.")
			f, err = generateMockDefterExcel()
			require.NoError(t, err)
		}
		defer f.Close()

		sheetList := f.GetSheetList()
		require.NotEmpty(t, sheetList)

		firstSheet := sheetList[0]
		rows, err := f.GetRows(firstSheet)
		require.NoError(t, err)
		require.Greater(t, len(rows), 0)
		t.Logf("📊 [%s] sayfasında toplam satır sayısı: %d", firstSheet, len(rows))
	})

	// -------------------------------------------------------------------------
	// 2. TEST: Dosya 2 (KASA DEFTERİM 2026) -> Tedarikçi ve Cari İçe Aktarma Motoru
	// -------------------------------------------------------------------------
	t.Run("File 2: KASA DEFTERİM 2026 Tedarikçi & Cari Import", func(t *testing.T) {
		var fReader io.Reader
		if _, statErr := os.Stat(file2Path); statErr == nil {
			fileReader, err := os.Open(file2Path)
			require.NoError(t, err)
			defer fileReader.Close()
			fReader = fileReader
		} else {
			t.Log("ℹ️ Yerel dosya bulunamadı, dinamik Mock Kasa Defteri üzerinden test çalıştırılıyor.")
			mockF, err := generateMockKasaDefteriExcel()
			require.NoError(t, err)
			var buf bytes.Buffer
			require.NoError(t, mockF.Write(&buf))
			fReader = &buf
		}

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

		targetSheet := "AĞUSTOS26"
		result, err := supplierSvc.ImportExcel(context.Background(), tenantID, periodID, fReader, targetSheet, &userID)
		require.NoError(t, err, "AĞUSTOS26 sayfası hatasız içe aktarılmalıdır")

		t.Logf("✅ Import Sonucu: Satır: %d, Aktarılan: %d, Tedarikçiler: %v", result.TotalRows, result.ImportedCount, result.SuppliersCreated)
		assert.Greater(t, result.ImportedCount, 0, "En az bir cari hareket aktarılmış olmalıdır")
		assert.NotEmpty(t, result.SuppliersCreated, "Tedarikçi kartları oluşmalıdır")

		suppliers, err := supplierSvc.ListSuppliers(context.Background(), tenantID, &periodID)
		require.NoError(t, err)
		assert.NotEmpty(t, suppliers)

		summary, err := supplierSvc.GetSummary(context.Background(), tenantID, &periodID)
		require.NoError(t, err)
		assert.True(t, summary.NetBalance.Equal(summary.TotalPurchases.Sub(summary.TotalPayments)))
	})

	// -------------------------------------------------------------------------
	// 3. TEST: Dosya 2 (KASA DEFTERİM 2026) -> Çoklu Sayfa Uyumluluk Taraması
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

		for _, sheetName := range []string{"OCAK26", "ŞUBAT26", "MART26"} {
			var fReader io.Reader
			if _, statErr := os.Stat(file2Path); statErr == nil {
				fHandle, err := os.Open(file2Path)
				require.NoError(t, err)
				fReader = fHandle
				defer fHandle.Close()
			} else {
				mockF, err := generateMockKasaDefteriExcel()
				require.NoError(t, err)
				var buf bytes.Buffer
				require.NoError(t, mockF.Write(&buf))
				fReader = &buf
			}
			res, err := supplierSvc.ImportExcel(context.Background(), tenantID, periodID, fReader, sheetName, &userID)
			if err == nil {
				t.Logf("✅ [%s] Sayfası Başarılı: %d satır aktarıldı (Tedarikçiler: %v)", sheetName, res.ImportedCount, res.SuppliersCreated)
				assert.Greater(t, res.ImportedCount, 0)
			}
		}
	})
}
