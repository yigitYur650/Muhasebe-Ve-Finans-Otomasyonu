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
	// 1. Period kilitli mi kontrol et
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

	csvReader := csv.NewReader(bytes.NewReader(buf))
	csvReader.TrimLeadingSpace = true

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv parse error: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("csv file empty or missing header/data rows")
	}

	// Parse Header to find column indices
	header := records[0]
	dirIdx, chanIdx, amtIdx, descIdx := -1, -1, -1, -1

	for i, col := range header {
		cleanCol := strings.ToLower(strings.TrimSpace(col))
		switch cleanCol {
		case "direction", "yön", "yon", "tip", "tür":
			dirIdx = i
		case "channel", "kanal":
			chanIdx = i
		case "amount", "tutar", "miktar":
			amtIdx = i
		case "description", "açıklama", "aciklama", "detay":
			descIdx = i
		}
	}

	// Fallback to default indices if header not matched exactly
	if dirIdx == -1 {
		dirIdx = 0
	}
	if chanIdx == -1 {
		chanIdx = 1
	}
	if amtIdx == -1 {
		amtIdx = 2
	}
	if descIdx == -1 && len(header) > 3 {
		descIdx = 3
	}

	var parsedTxs []*domain.Transaction
	totalAmount := decimal.Zero

	// 2. Validate all rows line-by-line
	for i := 1; i < len(records); i++ {
		row := records[i]
		lineNum := i + 1 // 1-indexed for user readability

		if len(row) <= amtIdx || len(row) <= dirIdx || len(row) <= chanIdx {
			return nil, fmt.Errorf("line %d: insufficient columns", lineNum)
		}

		// Direction parsing
		rawDir := strings.ToLower(strings.TrimSpace(row[dirIdx]))
		var dir domain.Direction
		switch rawDir {
		case "in", "gelir", "g", "+", "giris", "giriş":
			dir = domain.DirectionIn
		case "out", "gider", "c", "-", "cikis", "çıkış":
			dir = domain.DirectionOut
		default:
			return nil, fmt.Errorf("line %d: %w (%s)", lineNum, domain.ErrInvalidDirection, rawDir)
		}

		// Channel parsing
		rawChan := strings.ToLower(strings.TrimSpace(row[chanIdx]))
		rawChan = strings.ReplaceAll(rawChan, " ", "_")
		chanVal := domain.Channel(rawChan)
		if !chanVal.IsValid() {
			return nil, fmt.Errorf("line %d: %w (%s)", lineNum, domain.ErrInvalidChannel, rawChan)
		}

		// Amount parsing
		rawAmt := strings.TrimSpace(row[amtIdx])
		rawAmt = strings.ReplaceAll(rawAmt, " TL", "")
		rawAmt = strings.ReplaceAll(rawAmt, "₺", "")
		rawAmt = strings.ReplaceAll(rawAmt, ",", ".")

		amt, err := decimal.NewFromString(rawAmt)
		if err != nil || amt.LessThanOrEqual(decimal.Zero) {
			return nil, fmt.Errorf("line %d: %w (%s)", lineNum, domain.ErrInvalidAmount, rawAmt)
		}

		// Description
		var desc *string
		if descIdx != -1 && descIdx < len(row) {
			d := strings.TrimSpace(row[descIdx])
			if d != "" {
				desc = &d
			}
		}

		tx := &domain.Transaction{
			ID:          uuid.New(),
			TenantID:    tenantID,
			PeriodID:    periodID,
			Direction:   dir,
			Channel:     chanVal,
			Amount:      amt,
			Description: desc,
			CreatedBy:   userID,
		}

		if err := tx.Validate(); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		parsedTxs = append(parsedTxs, tx)
		totalAmount = totalAmount.Add(amt)
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
