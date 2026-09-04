package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"deftersystem/backend/internal/domain"
)

type importService struct {
	txRepo     domain.TransactionRepository
	periodRepo domain.PeriodRepository
}

// NewImportService creates a new somut ImportService implementation.
func NewImportService(txRepo domain.TransactionRepository, periodRepo domain.PeriodRepository) domain.ImportService {
	return &importService{
		txRepo:     txRepo,
		periodRepo: periodRepo,
	}
}

func (s *importService) ImportTransactionsFromCSV(
	ctx context.Context,
	tenantID, periodID uuid.UUID,
	r io.Reader,
	userID uuid.UUID,
) (*domain.ImportResult, error) {
	// 1. Check if period is locked
	period, err := s.periodRepo.GetByID(ctx, periodID)
	if err != nil {
		return nil, err
	}
	if period.IsLocked() {
		return nil, domain.ErrPeriodLocked
	}

	// Read content into memory to check BOM and parse CSV
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	// Strip UTF-8 BOM if present (\xEF\xBB\xBF)
	buf = bytes.TrimPrefix(buf, []byte("\xef\xbb\xbf"))

	// Auto-detect delimiter (tab, ';' or ',')
	delimiter := ','
	bufStr := string(buf)
	if strings.Contains(bufStr, "\t") {
		delimiter = '\t'
	} else if strings.Contains(bufStr, ";") && !strings.Contains(bufStr, ",") {
		delimiter = ';'
	}

	csvReader := csv.NewReader(bytes.NewReader(buf))
	csvReader.Comma = delimiter
	csvReader.TrimLeadingSpace = true
	csvReader.FieldsPerRecord = -1 // Flexible field count for title/summary rows

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv parse error: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("csv file empty or missing header/data rows")
	}

	// Find the header row by searching for key column indicators
	headerIdx := -1
	dirIdx, chanIdx, amtIdx, descIdx := -1, -1, -1, -1
	gelenIdx, gidenIdx := -1, -1

	for idx, row := range records {
		dIdx, cIdx, aIdx, dsIdx := -1, -1, -1, -1
		gIdx, gdIdx := -1, -1

		for i, col := range row {
			cleanCol := strings.ToLower(strings.TrimSpace(col))
			cleanCol = strings.ReplaceAll(cleanCol, "_", " ")

			if strings.Contains(cleanCol, "gelen") || strings.Contains(cleanCol, "giren") || strings.Contains(cleanCol, "alınan") {
				gIdx = i
			} else if strings.Contains(cleanCol, "giden") || strings.Contains(cleanCol, "çıkan") || strings.Contains(cleanCol, "ödenen") {
				gdIdx = i
			}

			switch cleanCol {
			case "direction", "yön", "yon", "tip", "tür":
				dIdx = i
			case "channel", "kanal":
				cIdx = i
			case "amount", "tutar", "miktar":
				aIdx = i
			case "description", "açıklama", "aciklama", "detay":
				dsIdx = i
			}
		}

		// A valid header row must contain either (gelen AND giden) OR (amount/tutar) OR (direction AND amount) OR (description AND amount/gelen/giden)
		if (gIdx != -1 && gdIdx != -1) || (aIdx != -1) || (dIdx != -1 && aIdx != -1) || (dsIdx != -1 && (gIdx != -1 || gdIdx != -1 || aIdx != -1)) {
			headerIdx = idx
			dirIdx, chanIdx, amtIdx, descIdx = dIdx, cIdx, aIdx, dsIdx
			gelenIdx, gidenIdx = gIdx, gdIdx
			break
		}
	}

	// Fallback: If no header found, assume row 0 is header
	if headerIdx == -1 {
		headerIdx = 0
		dirIdx, chanIdx, amtIdx, descIdx = 0, 1, 2, 3
	}

	var parsedTxs []*domain.Transaction
	totalAmount := decimal.Zero

	// Parse data rows starting after headerIdx
	for i := headerIdx + 1; i < len(records); i++ {
		row := records[i]
		lineNum := i + 1 // 1-indexed

		// Skip completely empty rows
		isEmpty := true
		for _, field := range row {
			if strings.TrimSpace(field) != "" {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			continue
		}

		var descStr *string
		if descIdx != -1 && descIdx < len(row) {
			d := strings.TrimSpace(row[descIdx])
			if d != "" {
				descStr = &d
			}
		}

		// Process rows
		// Scenario A: Dual column (GELEN TUTAR and GİDEN TUTAR)
		if gelenIdx != -1 && gidenIdx != -1 {
			var rawGelen, rawGiden string
			if gelenIdx < len(row) {
				rawGelen = strings.TrimSpace(row[gelenIdx])
			}
			if gidenIdx < len(row) {
				rawGiden = strings.TrimSpace(row[gidenIdx])
			}

			if rawGelen == "" && rawGiden == "" {
				continue
			}

			var dir domain.Direction
			var amt decimal.Decimal

			if rawGelen != "" {
				gAmt, err := cleanTurkishDecimalAmount(rawGelen)
				if err != nil {
					return nil, fmt.Errorf("line %d: %w (%s)", lineNum, domain.ErrInvalidAmount, rawGelen)
				}
				if gAmt.GreaterThan(decimal.Zero) {
					dir = domain.DirectionIn
					amt = gAmt
				}
			}

			if amt.IsZero() && rawGiden != "" {
				gdAmt, err := cleanTurkishDecimalAmount(rawGiden)
				if err != nil {
					return nil, fmt.Errorf("line %d: %w (%s)", lineNum, domain.ErrInvalidAmount, rawGiden)
				}
				if gdAmt.GreaterThan(decimal.Zero) {
					dir = domain.DirectionOut
					amt = gdAmt
				}
			}

			if amt.LessThanOrEqual(decimal.Zero) {
				continue // Skip header/summary/empty rows
			}

			// Infer channel if channel column is missing or empty
			var chanVal domain.Channel
			if chanIdx != -1 && chanIdx < len(row) {
				rawChan := strings.ToLower(strings.TrimSpace(row[chanIdx]))
				rawChan = strings.ReplaceAll(rawChan, " ", "_")
				c := domain.Channel(rawChan)
				if c.IsValid() {
					chanVal = c
				}
			}
			if !chanVal.IsValid() {
				if descStr != nil {
					chanVal = inferChannelFromDescription(*descStr)
				} else {
					chanVal = domain.ChannelNakit
				}
			}

			tx := &domain.Transaction{
				ID:          uuid.New(),
				TenantID:    tenantID,
				PeriodID:    periodID,
				Direction:   dir,
				Channel:     chanVal,
				Amount:      amt,
				Description: descStr,
				CreatedBy:   userID,
			}
			if err := tx.Validate(); err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			parsedTxs = append(parsedTxs, tx)
			totalAmount = totalAmount.Add(amt)

		} else { // Scenario B: Single Tutar / Direction column
			if amtIdx == -1 {
				amtIdx = 2
			}
			if dirIdx == -1 {
				dirIdx = 0
			}
			if chanIdx == -1 {
				chanIdx = 1
			}

			if len(row) <= amtIdx {
				continue
			}

			rawAmt := strings.TrimSpace(row[amtIdx])
			if rawAmt == "" {
				continue
			}

			amt, err := cleanTurkishDecimalAmount(rawAmt)
			if err != nil || amt.LessThanOrEqual(decimal.Zero) {
				return nil, fmt.Errorf("line %d: %w (%s)", lineNum, domain.ErrInvalidAmount, rawAmt)
			}

			// Direction
			var dir domain.Direction
			if dirIdx < len(row) {
				rawDir := strings.ToLower(strings.TrimSpace(row[dirIdx]))
				switch rawDir {
				case "in", "gelir", "g", "+", "giris", "giriş":
					dir = domain.DirectionIn
				case "out", "gider", "c", "-", "cikis", "çıkış":
					dir = domain.DirectionOut
				default:
					return nil, fmt.Errorf("line %d: %w (%s)", lineNum, domain.ErrInvalidDirection, rawDir)
				}
			}
			if dir == "" {
				dir = domain.DirectionIn
			}

			// Channel
			var chanVal domain.Channel
			if chanIdx < len(row) {
				rawChan := strings.ToLower(strings.TrimSpace(row[chanIdx]))
				rawChan = strings.ReplaceAll(rawChan, " ", "_")
				c := domain.Channel(rawChan)
				if c.IsValid() {
					chanVal = c
				} else {
					return nil, fmt.Errorf("line %d: %w (%s)", lineNum, domain.ErrInvalidChannel, rawChan)
				}
			}
			if !chanVal.IsValid() {
				if descStr != nil {
					chanVal = inferChannelFromDescription(*descStr)
				} else {
					chanVal = domain.ChannelNakit
				}
			}

			tx := &domain.Transaction{
				ID:          uuid.New(),
				TenantID:    tenantID,
				PeriodID:    periodID,
				Direction:   dir,
				Channel:     chanVal,
				Amount:      amt,
				Description: descStr,
				CreatedBy:   userID,
			}

			if err := tx.Validate(); err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}

			parsedTxs = append(parsedTxs, tx)
			totalAmount = totalAmount.Add(amt)
		}
	}

	if len(parsedTxs) == 0 {
		return nil, fmt.Errorf("geçerli hiçbir işlem verisi bulunamadı veya dosya boş")
	}

	// 3. Persist valid transactions
	for _, tx := range parsedTxs {
		if err := s.txRepo.Create(ctx, tx); err != nil {
			return nil, fmt.Errorf("failed to save imported transaction: %w", err)
		}
	}

	return &domain.ImportResult{
		ImportedCount: len(parsedTxs),
		TotalAmount:   totalAmount,
	}, nil
}

// cleanTurkishDecimalAmount handles Turkish currency numbers (e.g. 1.135.651,04 ₺, 39.000,00 ₺, 150,00)
func cleanTurkishDecimalAmount(raw string) (decimal.Decimal, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, "₺", "")
	raw = strings.ReplaceAll(raw, "TL", "")
	raw = strings.ReplaceAll(raw, "tl", "")
	raw = strings.ReplaceAll(raw, "€", "")
	raw = strings.ReplaceAll(raw, "$", "")
	raw = strings.TrimSpace(raw)

	if raw == "" || raw == "-" {
		return decimal.Zero, fmt.Errorf("empty amount")
	}

	hasDot := strings.Contains(raw, ".")
	hasComma := strings.Contains(raw, ",")

	if hasDot && hasComma {
		// Turkish format: 1.135.651,04 (dot = thousands, comma = decimal)
		raw = strings.ReplaceAll(raw, ".", "")
		raw = strings.ReplaceAll(raw, ",", ".")
	} else if hasComma {
		// 150,00 -> 150.00
		raw = strings.ReplaceAll(raw, ",", ".")
	}

	return decimal.NewFromString(raw)
}

// inferChannelFromDescription auto-detects transaction channel from description keywords
func inferChannelFromDescription(desc string) domain.Channel {
	d := strings.ToLower(strings.TrimSpace(desc))
	d = strings.ReplaceAll(d, "ı", "i")
	d = strings.ReplaceAll(d, "ş", "s")
	d = strings.ReplaceAll(d, "ğ", "g")
	d = strings.ReplaceAll(d, "ü", "u")
	d = strings.ReplaceAll(d, "ö", "o")
	d = strings.ReplaceAll(d, "ç", "c")

	if strings.Contains(d, "eft") || strings.Contains(d, "banka") || strings.Contains(d, "havale") {
		if strings.Contains(d, "maas") {
			return domain.ChannelMaasBanka
		}
		return domain.ChannelEft
	}

	if strings.Contains(d, "pos") {
		return domain.ChannelPos
	}

	if strings.Contains(d, "nakit") {
		if strings.Contains(d, "maas") {
			return domain.ChannelMaasElden
		}
		return domain.ChannelNakit
	}

	if strings.Contains(d, "kredi karti") || strings.Contains(d, "kredi kart") {
		return domain.ChannelKrediKarti
	}

	if strings.Contains(d, "kredi") {
		return domain.ChannelKredi
	}

	if strings.Contains(d, "kira") {
		return domain.ChannelKira
	}

	if strings.Contains(d, "maas") {
		return domain.ChannelMaasBanka
	}

	if strings.Contains(d, "yemek") {
		return domain.ChannelYemek
	}

	if strings.Contains(d, "yakit") || strings.Contains(d, "gaz") || strings.Contains(d, "benzin") {
		return domain.ChannelYakit
	}

	if strings.Contains(d, "kartus") {
		return domain.ChannelKartus
	}

	return domain.ChannelDiger
}
