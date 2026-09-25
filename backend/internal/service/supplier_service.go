package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"

	"deftersystem/backend/internal/domain"
)

type supplierService struct {
	supplierRepo domain.SupplierRepository
	periodRepo   domain.PeriodRepository
}

// NewSupplierService creates a new domain.SupplierService implementation.
func NewSupplierService(
	supplierRepo domain.SupplierRepository,
	periodRepo domain.PeriodRepository,
) domain.SupplierService {
	return &supplierService{
		supplierRepo: supplierRepo,
		periodRepo:   periodRepo,
	}
}

func (s *supplierService) ListSuppliers(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) ([]domain.Supplier, error) {
	return s.supplierRepo.GetSuppliersWithBalances(ctx, tenantID, periodID)
}

func (s *supplierService) GetSupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (*domain.Supplier, error) {
	return s.supplierRepo.GetSupplierByID(ctx, tenantID, supplierID)
}

func (s *supplierService) CreateSupplier(ctx context.Context, tenantID uuid.UUID, name string) (*domain.Supplier, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, domain.ErrSupplierNotFound
	}
	return s.supplierRepo.FindOrCreateSupplier(ctx, tenantID, trimmed)
}

func (s *supplierService) CreateTransaction(
	ctx context.Context,
	tenantID uuid.UUID,
	tx *domain.SupplierTransaction,
) (*domain.SupplierTransaction, error) {
	if tx.PeriodID == uuid.Nil {
		return nil, domain.ErrPeriodNotFound
	}

	// Check if period exists and is not locked
	period, err := s.periodRepo.GetByID(ctx, tx.PeriodID)
	if err != nil {
		return nil, err
	}
	if tenantID != uuid.Nil && period.TenantID != tenantID {
		return nil, domain.ErrUnauthorized
	}
	if period.IsLocked() {
		return nil, domain.ErrPeriodLocked
	}

	tx.TenantID = tenantID
	if err := s.supplierRepo.CreateTransaction(ctx, tx); err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *supplierService) ReverseTransaction(
	ctx context.Context,
	tenantID, origID uuid.UUID,
	reason string,
	createdBy *uuid.UUID,
) (*domain.SupplierTransaction, error) {
	orig, err := s.supplierRepo.GetTransactionByID(ctx, tenantID, origID)
	if err != nil {
		return nil, err
	}

	period, err := s.periodRepo.GetByID(ctx, orig.PeriodID)
	if err != nil {
		return nil, err
	}

	if period.IsLocked() {
		return nil, domain.ErrPeriodLocked
	}

	// Inverse direction rule: 'purchase' -> 'payment', 'payment' -> 'purchase'
	oppositeDirection := domain.SupplierDirectionPayment
	if orig.Direction == domain.SupplierDirectionPayment {
		oppositeDirection = domain.SupplierDirectionPurchase
	}

	desc := fmt.Sprintf("[İPTAL/TERS KAYIT] %s", strings.TrimSpace(reason))
	if strings.TrimSpace(reason) == "" {
		desc = "[İPTAL/TERS KAYIT] Hatalı işlem düzeltmesi"
	}

	revTx := &domain.SupplierTransaction{
		ID:             uuid.New(),
		TenantID:       tenantID,
		SupplierID:     orig.SupplierID,
		SupplierName:   orig.SupplierName,
		PeriodID:       orig.PeriodID,
		InvoiceNo:      orig.InvoiceNo,
		CustomerName:   orig.CustomerName,
		DocumentStatus: orig.DocumentStatus,
		TxDate:         time.Now(),
		Direction:      oppositeDirection,
		Amount:         orig.Amount, // Exact decimal amount preserved
		Description:    desc,
		CreatedBy:      createdBy,
		CreatedAt:      time.Now(),
		ReversedBy:     &origID,
	}

	if err := s.supplierRepo.ReverseTransaction(ctx, tenantID, origID, revTx); err != nil {
		return nil, err
	}

	return revTx, nil
}

func (s *supplierService) ListSupplierTransactions(
	ctx context.Context,
	tenantID, supplierID uuid.UUID,
	filter domain.SupplierTransactionFilter,
) ([]domain.SupplierTransaction, int, error) {
	return s.supplierRepo.GetTransactionsBySupplier(ctx, tenantID, supplierID, filter)
}

func (s *supplierService) ListAllTransactions(
	ctx context.Context,
	tenantID uuid.UUID,
	filter domain.SupplierTransactionFilter,
) ([]domain.SupplierTransaction, int, error) {
	return s.supplierRepo.GetAllTransactions(ctx, tenantID, filter)
}

func (s *supplierService) GetSummary(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) (*domain.SupplierSummary, error) {
	return s.supplierRepo.GetSummary(ctx, tenantID, periodID)
}

// SupplierColumnBlock defines the detected column positions for a specific supplier in the Excel sheet.
type SupplierColumnBlock struct {
	SupplierName   string
	SupplierID     uuid.UUID
	StartCol       int
	InvoiceNoCol   int
	CustomerCol    int
	StatusCol      int
	DateCol        int
	PaymentCol     int // Geçilen Tutar (Yapılan Ödeme)
	PurchaseCol    int // Alınan Tutar (Alınan Mal)
	BalanceCol     int // Bakiye
	DescriptionCol int
}

func (s *supplierService) ImportExcel(
	ctx context.Context,
	tenantID, periodID uuid.UUID,
	r io.Reader,
	sheetName string,
	userID *uuid.UUID,
) (*domain.SupplierImportResult, error) {
	// 1. Verify period status
	period, err := s.periodRepo.GetByID(ctx, periodID)
	if err != nil {
		return nil, err
	}
	if tenantID != uuid.Nil && period.TenantID != tenantID {
		return nil, domain.ErrUnauthorized
	}
	if period.IsLocked() {
		return nil, domain.ErrPeriodLocked
	}

	// 2. Read with 10MB limit to prevent OOM
	limitedReader := io.LimitReader(r, 10<<20)
	buf, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("dosya okunamadı: %w", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("excel dosyası açılamadı: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return nil, fmt.Errorf("excel dosyasında hiç sayfa bulunamadı")
	}

	// 3. Match Sheet for the active period (Dönem - Sayfa Eşleşmesi)
	targetSheet, err := matchSheetForPeriod(sheetList, sheetName, period.Label)
	if err != nil {
		return nil, err
	}

	rows, err := f.GetRows(targetSheet)
	if err != nil {
		return nil, fmt.Errorf("sayfa satırları okunamadı (%s): %w", targetSheet, err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("'%s' sayfasında yeterli veri satırı bulunamadı", targetSheet)
	}

	// 4. Scan header rows (rows 0, 1, 2) dynamically using strict normalized string matching
	blocks, err := s.detectSupplierBlocks(ctx, tenantID, rows)
	if err != nil {
		return nil, err
	}

	if len(blocks) == 0 {
		return nil, fmt.Errorf("'%s' sayfasında geçerli tedarikçi ve tutar sütunları ('ALINAN TUTAR' / 'GEÇİLEN TUTAR') tespit edilemedi", targetSheet)
	}

	result := &domain.SupplierImportResult{
		SuppliersCreated: make([]string, 0),
		Errors:           make([]string, 0),
		TotalPurchase:    decimal.Zero,
		TotalPayment:     decimal.Zero,
	}

	var allTransactions []domain.SupplierTransaction
	createdSupplierMap := make(map[string]bool)

	// 5. Parse row by row (starting after header rows)
	startRowIdx := 2
	if len(rows) > 3 && isHeaderRow(rows[2]) {
		startRowIdx = 3
	}

	for rIdx := startRowIdx; rIdx < len(rows); rIdx++ {
		row := rows[rIdx]
		if len(row) == 0 {
			continue
		}

		result.TotalRows++

		for _, block := range blocks {
			txs, skipped, parseErrors := s.parseBlockRow(row, block, periodID, tenantID, userID, period.OpenedAt)
			for _, errStr := range parseErrors {
				result.Errors = append(result.Errors, fmt.Sprintf("Satır %d [%s]: %s", rIdx+1, block.SupplierName, errStr))
			}

			if skipped {
				result.SkippedCount++
			}

			for _, tx := range txs {
				allTransactions = append(allTransactions, tx)
				result.ImportedCount++
				if tx.Direction == domain.SupplierDirectionPurchase {
					result.TotalPurchase = result.TotalPurchase.Add(tx.Amount)
				} else {
					result.TotalPayment = result.TotalPayment.Add(tx.Amount)
				}
				if !createdSupplierMap[block.SupplierName] {
					createdSupplierMap[block.SupplierName] = true
					result.SuppliersCreated = append(result.SuppliersCreated, block.SupplierName)
				}
			}
		}
	}

	// 6. Batch insert transactions into DB within a single transaction
	if len(allTransactions) > 0 {
		_, err := s.supplierRepo.BatchInsertTransactions(ctx, tenantID, allTransactions)
		if err != nil {
			return nil, fmt.Errorf("veritabanına toplu kayıt sırasında hata oluştu: %w", err)
		}
	}

	return result, nil
}

// matchSheetForPeriod matches the period label (e.g. 2026-08 or OCAK 2024) to a sheet in the Excel file.
func matchSheetForPeriod(sheetList []string, userSheetName, periodLabel string) (string, error) {
	if len(sheetList) == 0 {
		return "", fmt.Errorf("excel dosyasında hiç sayfa bulunamadı")
	}

	// 1. If explicit user sheet name provided
	if userSheetName != "" {
		for _, name := range sheetList {
			if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(userSheetName)) ||
				normalizeHeader(name) == normalizeHeader(userSheetName) {
				return name, nil
			}
		}
		return "", fmt.Errorf("belirtilen '%s' sayfası Excel dosyasında bulunamadı. Mevcut sayfalar: [%s]",
			userSheetName, strings.Join(sheetList, ", "))
	}

	// 2. If workbook contains only 1 sheet, automatically use it
	if len(sheetList) == 1 {
		return sheetList[0], nil
	}

	// 3. Period label matching (e.g. 2026-08 -> year 2026 / 26, month 08 -> AĞUSTOS / AGUSTOS)
	cleanPeriodLabel := strings.ToUpper(strings.ReplaceAll(periodLabel, " ", ""))
	cleanPeriodLabelNoDash := strings.ReplaceAll(cleanPeriodLabel, "-", "")

	// Direct label match
	for _, name := range sheetList {
		normName := normalizeHeader(name)
		if normName == cleanPeriodLabel || normName == cleanPeriodLabelNoDash {
			return name, nil
		}
	}

	// Turkish month mapping
	monthAliases := map[string][]string{
		"01": {"OCAK", "JAN"},
		"02": {"SUBAT", "FEB"},
		"03": {"MART", "MAR"},
		"04": {"NISAN", "APR"},
		"05": {"MAYIS", "MAY"},
		"06": {"HAZIRAN", "JUN"},
		"07": {"TEMMUZ", "JUL"},
		"08": {"AGUSTOS", "AUG"},
		"09": {"EYLUL", "SEP"},
		"10": {"EKIM", "OCT"},
		"11": {"KASIM", "NOV"},
		"12": {"ARALIK", "DEC"},
	}

	parts := strings.Split(periodLabel, "-")
	if len(parts) == 2 {
		year := parts[0]
		month := parts[1]
		shortYear := ""
		if len(year) == 4 {
			shortYear = year[2:]
		}

		aliases, ok := monthAliases[month]
		if ok {
			for _, sheet := range sheetList {
				normSheet := normalizeHeader(sheet)
				for _, alias := range aliases {
					if strings.Contains(normSheet, alias) {
						if shortYear != "" && strings.Contains(normSheet, shortYear) {
							return sheet, nil
						}
						if strings.Contains(normSheet, year) {
							return sheet, nil
						}
					}
				}
			}

			// If year was not matched, try matching month name alone
			for _, sheet := range sheetList {
				normSheet := normalizeHeader(sheet)
				for _, alias := range aliases {
					if strings.Contains(normSheet, alias) {
						return sheet, nil
					}
				}
			}
		}
	}

	// If multiple sheets exist and none matched the period, return descriptive error
	return "", fmt.Errorf("seçili dönem (%s) için uygun Excel sayfası bulunamadı. Lütfen sayfa adını girin veya dosyanızı kontrol edin. Dosyadaki sayfalar: [%s]",
		periodLabel, strings.Join(sheetList, ", "))
}

func normalizeHeader(s string) string {
	s = strings.TrimSpace(s)
	replacer := strings.NewReplacer(
		"ç", "c", "Ç", "C",
		"ğ", "g", "Ğ", "G",
		"ı", "i", "I", "I", "İ", "I", "i", "i",
		"ö", "o", "Ö", "O",
		"ş", "s", "Ş", "S",
		"ü", "u", "Ü", "U",
	)
	s = replacer.Replace(s)
	return strings.ToUpper(s)
}

func (s *supplierService) detectSupplierBlocks(
	ctx context.Context,
	tenantID uuid.UUID,
	rows [][]string,
) ([]SupplierColumnBlock, error) {
	var blocks []SupplierColumnBlock

	row0 := rows[0]
	var row1 []string
	if len(rows) > 1 {
		row1 = rows[1]
	}

	maxCols := len(row0)
	if len(row1) > maxCols {
		maxCols = len(row1)
	}

	for colIdx := 0; colIdx < maxCols; colIdx++ {
		cellVal0 := ""
		if colIdx < len(row0) {
			cellVal0 = strings.TrimSpace(row0[colIdx])
		}

		// Look for supplier company names in Row 0
		if cellVal0 != "" && !isGeneralKeyword(cellVal0) {
			supplierName := cleanSupplierName(cellVal0)
			if supplierName != "" {
				block := SupplierColumnBlock{
					SupplierName:   supplierName,
					StartCol:       colIdx,
					InvoiceNoCol:   -1,
					CustomerCol:    -1,
					StatusCol:      -1,
					DateCol:        -1,
					PaymentCol:     -1,
					PurchaseCol:    -1,
					BalanceCol:     -1,
					DescriptionCol: -1,
				}

				// Look at sub-columns in Row 1 and Row 0 dynamically
				endCol := colIdx + 12
				if endCol > maxCols {
					endCol = maxCols
				}

				for subCol := colIdx; subCol < endCol; subCol++ {
					// Stop if we encounter another company header in Row 0 (after startCol)
					if subCol > colIdx && subCol < len(row0) {
						otherCell := strings.TrimSpace(row0[subCol])
						if otherCell != "" && !isGeneralKeyword(otherCell) {
							break
						}
					}

					subHeader := ""
					if subCol < len(row1) {
						subHeader = normalizeHeader(row1[subCol])
					}
					if subHeader == "" && subCol < len(row0) {
						subHeader = normalizeHeader(row0[subCol])
					}
					if len(rows) > 2 && subHeader == "" && subCol < len(rows[2]) {
						subHeader = normalizeHeader(rows[2][subCol])
					}

					// Dynamic String Header Search
					if strings.Contains(subHeader, "FATURA") {
						block.InvoiceNoCol = subCol
					} else if strings.Contains(subHeader, "AD") || strings.Contains(subHeader, "SOYAD") || strings.Contains(subHeader, "MUSTERI") || strings.Contains(subHeader, "ISIM") {
						block.CustomerCol = subCol
					} else if strings.Contains(subHeader, "EVRAK") || strings.Contains(subHeader, "DURUM") {
						block.StatusCol = subCol
					} else if strings.Contains(subHeader, "TARIH") || strings.Contains(subHeader, "GUN") {
						block.DateCol = subCol
					} else if strings.Contains(subHeader, "GECILEN") || strings.Contains(subHeader, "ODENEN") || strings.Contains(subHeader, "ODEME") {
						block.PaymentCol = subCol
					} else if strings.Contains(subHeader, "ALINAN") || strings.Contains(subHeader, "GIRIS") || strings.Contains(subHeader, "BORC") {
						block.PurchaseCol = subCol
					} else if strings.Contains(subHeader, "BAKIYE") || strings.Contains(subHeader, "DEVIR") {
						block.BalanceCol = subCol
					} else if strings.Contains(subHeader, "ACIKLAMA") {
						block.DescriptionCol = subCol
					}
				}

				// STRICT RULE: Only register supplier block if explicit amount column ('ALINAN' or 'GEÇİLEN') was found
				// NO STATIC COLUMN OFFSETS ALLOWED!
				if block.PaymentCol != -1 || block.PurchaseCol != -1 {
					supplier, err := s.supplierRepo.FindOrCreateSupplier(ctx, tenantID, supplierName)
					if err != nil {
						return nil, fmt.Errorf("tedarikçi oluşturulamadı (%s): %w", supplierName, err)
					}
					block.SupplierID = supplier.ID
					blocks = append(blocks, block)

					// Advance colIdx to skip sub-columns of this registered block
					if block.PurchaseCol > colIdx {
						colIdx = block.PurchaseCol
					}
					if block.PaymentCol > colIdx {
						colIdx = block.PaymentCol
					}
					if block.BalanceCol > colIdx {
						colIdx = block.BalanceCol
					}
				}
			}
		}
	}

	return blocks, nil
}

func (s *supplierService) parseBlockRow(
	row []string,
	block SupplierColumnBlock,
	periodID, tenantID uuid.UUID,
	userID *uuid.UUID,
	defaultDate time.Time,
) ([]domain.SupplierTransaction, bool, []string) {
	var txs []domain.SupplierTransaction
	var errors []string

	getColVal := func(cIdx int) string {
		if cIdx >= 0 && cIdx < len(row) {
			return strings.TrimSpace(row[cIdx])
		}
		return ""
	}

	invoiceNo := getColVal(block.InvoiceNoCol)
	customerName := getColVal(block.CustomerCol)
	docStatus := getColVal(block.StatusCol)
	dateStr := getColVal(block.DateCol)
	paymentStr := getColVal(block.PaymentCol)
	purchaseStr := getColVal(block.PurchaseCol)
	descStr := getColVal(block.DescriptionCol)

	// Filter out devir, summary or header rows in body
	if isSummaryOrEmptyRow(invoiceNo, customerName, docStatus, paymentStr, purchaseStr) {
		return nil, true, nil
	}

	txDate := defaultDate
	if dateStr != "" {
		if parsedDate, err := ParseFlexibleDate(dateStr); err == nil {
			txDate = parsedDate
		}
	}

	// 1. Process Purchase (Alınan Tutar - Alınan Mal, Borç Artışı)
	if block.PurchaseCol != -1 && purchaseStr != "" && purchaseStr != "-" && purchaseStr != "0" {
		amount, err := ParseTurkishDecimal(purchaseStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Geçersiz alınan tutar formatı (%s): %v", purchaseStr, err))
		} else if amount.GreaterThan(decimal.Zero) {
			desc := descStr
			if desc == "" && invoiceNo != "" {
				desc = fmt.Sprintf("Fatura No: %s", invoiceNo)
			}
			txs = append(txs, domain.SupplierTransaction{
				ID:             uuid.New(),
				TenantID:       tenantID,
				SupplierID:     block.SupplierID,
				SupplierName:   block.SupplierName,
				PeriodID:       periodID,
				InvoiceNo:      invoiceNo,
				CustomerName:   customerName,
				DocumentStatus: docStatus,
				TxDate:         txDate,
				Direction:      domain.SupplierDirectionPurchase,
				Amount:         amount,
				Description:    desc,
				CreatedBy:      userID,
				CreatedAt:      time.Now(),
			})
		}
	}

	// 2. Process Payment (Geçilen Tutar - Yapılan Ödeme, Borç Azalışı)
	if block.PaymentCol != -1 && paymentStr != "" && paymentStr != "-" && paymentStr != "0" {
		amount, err := ParseTurkishDecimal(paymentStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Geçersiz geçilen tutar formatı (%s): %v", paymentStr, err))
		} else if amount.GreaterThan(decimal.Zero) {
			desc := descStr
			if desc == "" && customerName != "" {
				desc = fmt.Sprintf("Ödeme / Cari: %s", customerName)
			}
			txs = append(txs, domain.SupplierTransaction{
				ID:             uuid.New(),
				TenantID:       tenantID,
				SupplierID:     block.SupplierID,
				SupplierName:   block.SupplierName,
				PeriodID:       periodID,
				InvoiceNo:      invoiceNo,
				CustomerName:   customerName,
				DocumentStatus: docStatus,
				TxDate:         txDate,
				Direction:      domain.SupplierDirectionPayment,
				Amount:         amount,
				Description:    desc,
				CreatedBy:      userID,
				CreatedAt:      time.Now(),
			})
		}
	}

	return txs, len(txs) == 0 && len(errors) == 0, errors
}

// Helpers

func cleanSupplierName(raw string) string {
	name := strings.TrimSpace(raw)
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
	upper := normalizeHeader(name)
	if strings.Contains(upper, "DEVIR") || strings.Contains(upper, "TOPLAM") || strings.Contains(upper, "BAKIYE") || strings.Contains(upper, "KASA") {
		return ""
	}
	return name
}

func isGeneralKeyword(s string) bool {
	norm := normalizeHeader(s)
	keywords := []string{
		"TARIH", "GUN", "FATURA", "FATURA NO", "AD SOYAD", "MUSTERI", "ISIM",
		"EVRAK", "EVRAK DURUMU", "DURUM", "ALINAN", "ALINAN TUTAR", "GECILEN",
		"GECILEN TUTAR", "BAKIYE", "DEVIR", "DEVIR BAKIYESI", "TOPLAM",
		"GENEL TOPLAM", "SIRA NO", "NO", "ACIKLAMA",
	}
	for _, kw := range keywords {
		if norm == kw {
			return true
		}
	}
	return false
}

func isHeaderRow(row []string) bool {
	for _, cell := range row {
		norm := normalizeHeader(cell)
		if strings.Contains(norm, "FATURA") || strings.Contains(norm, "ALINAN") || strings.Contains(norm, "GECILEN") {
			return true
		}
	}
	return false
}

func isSummaryOrEmptyRow(invoiceNo, customer, status, payment, purchase string) bool {
	combined := normalizeHeader(invoiceNo + " " + customer + " " + status)
	if strings.Contains(combined, "TOPLAM") || strings.Contains(combined, "DEVIR") || strings.Contains(combined, "GENEL TOPLAM") || strings.Contains(combined, "BAKIYE") {
		return true
	}
	return payment == "" && purchase == "" && invoiceNo == "" && customer == ""
}

// ParseTurkishDecimal safely parses Turkish currency/decimal strings without floating-point errors.
func ParseTurkishDecimal(input string) (decimal.Decimal, error) {
	val := strings.TrimSpace(input)
	if val == "" || val == "-" {
		return decimal.Zero, nil
	}

	// Remove currency symbols, spaces and non-numeric chars except '.' and ',' and '-'
	val = strings.ReplaceAll(val, "TL", "")
	val = strings.ReplaceAll(val, "₺", "")
	val = strings.ReplaceAll(val, "TRY", "")
	val = strings.ReplaceAll(val, " ", "")
	val = strings.ReplaceAll(val, "\u00a0", "") // non-breaking space

	if val == "" || val == "-" {
		return decimal.Zero, nil
	}

	// Handle Turkish format: 1.234,56 or 1234,56
	// If both dot and comma exist:
	if strings.Contains(val, ".") && strings.Contains(val, ",") {
		// e.g. 1.250,50 -> remove dots, replace comma with dot
		val = strings.ReplaceAll(val, ".", "")
		val = strings.ReplaceAll(val, ",", ".")
	} else if strings.Contains(val, ",") {
		// e.g. 1250,50 -> replace comma with dot
		val = strings.ReplaceAll(val, ",", ".")
	}

	return decimal.NewFromString(val)
}

// ParseFlexibleDate parses multiple date formats including Excel serial numbers.
func ParseFlexibleDate(input string) (time.Time, error) {
	val := strings.TrimSpace(input)
	if val == "" {
		return time.Time{}, fmt.Errorf("tarih boş")
	}

	// Check if Excel serial integer (e.g. 45180 for a date in 2023)
	if serialDays, err := strconv.ParseInt(val, 10, 64); err == nil && serialDays > 30000 && serialDays < 70000 {
		// Excel base date is 1899-12-30 (due to Lotus 1-2-3 leap year bug)
		baseDate := time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)
		return baseDate.AddDate(0, 0, int(serialDays)), nil
	}

	formats := []string{
		"02.01.2006",
		"2.1.2006",
		"02/01/2006",
		"2/1/2006",
		"2006-01-02",
		"02-01-2006",
		"2006/01/02",
		"02.01.06",
		"02/01/06",
		time.RFC3339,
	}

	for _, fmtStr := range formats {
		if t, err := time.Parse(fmtStr, val); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("bilinmeyen tarih formatı: %s", val)
}

// Ensure unused math import does not trigger compiler error
var _ = math.MaxInt
