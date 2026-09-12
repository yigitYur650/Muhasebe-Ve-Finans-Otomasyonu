package handler

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"deftersystem/backend/internal/domain"
)

type excelStyles struct {
	headerStyle          int
	amountStyle          int
	dateStyle            int
	textStyle            int
	centerStyle          int
	cancelledCenterStyle int
}

func initExcelStyles(f *excelize.File) (*excelStyles, error) {
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   11,
			Color:  "FFFFFF",
			Family: "Segoe UI",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"1E293B"}, // Slate-800
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Vertical:   "center",
			Horizontal: "center",
		},
		Border: []excelize.Border{
			{Type: "bottom", Color: "0F172A", Style: 2},
			{Type: "top", Color: "334155", Style: 1},
			{Type: "left", Color: "334155", Style: 1},
			{Type: "right", Color: "334155", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	numFmt := "#,##0.00"
	cellBorders := []excelize.Border{
		{Type: "bottom", Color: "E2E8F0", Style: 1},
		{Type: "top", Color: "E2E8F0", Style: 1},
		{Type: "left", Color: "E2E8F0", Style: 1},
		{Type: "right", Color: "E2E8F0", Style: 1},
	}

	amountStyle, err := f.NewStyle(&excelize.Style{
		CustomNumFmt: &numFmt,
		Alignment: &excelize.Alignment{
			Horizontal: "right",
			Vertical:   "center",
		},
		Font: &excelize.Font{
			Family: "Segoe UI",
			Size:   10,
		},
		Border: cellBorders,
	})
	if err != nil {
		return nil, err
	}

	dateStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Font: &excelize.Font{
			Family: "Segoe UI",
			Size:   10,
		},
		Border: cellBorders,
	})
	if err != nil {
		return nil, err
	}

	textStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
		Font: &excelize.Font{
			Family: "Segoe UI",
			Size:   10,
		},
		Border: cellBorders,
	})
	if err != nil {
		return nil, err
	}

	centerStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Font: &excelize.Font{
			Family: "Segoe UI",
			Size:   10,
		},
		Border: cellBorders,
	})
	if err != nil {
		return nil, err
	}

	cancelledCenterStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Font: &excelize.Font{
			Family: "Segoe UI",
			Size:   10,
			Color:  "94A3B8", // Slate-400
			Italic: true,
		},
		Border: cellBorders,
	})
	if err != nil {
		return nil, err
	}

	return &excelStyles{
		headerStyle:          headerStyle,
		amountStyle:          amountStyle,
		dateStyle:            dateStyle,
		textStyle:            textStyle,
		centerStyle:          centerStyle,
		cancelledCenterStyle: cancelledCenterStyle,
	}, nil
}

// GenerateTransactionsExcel builds a fully formatted XLSX file with auto column widths, styling and filters.
func GenerateTransactionsExcel(txs []domain.Transaction, statusFilter string) ([]byte, error) {
	// Pre-scan all reversal target IDs to accurately identify cancelled transactions
	reversedOriginalIDs := make(map[uuid.UUID]bool)
	for _, tx := range txs {
		if tx.ReversedBy != nil {
			reversedOriginalIDs[*tx.ReversedBy] = true
		}
	}

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "İşlem Defteri"
	f.SetSheetName("Sheet1", sheetName)

	styles, err := initExcelStyles(f)
	if err != nil {
		return nil, err
	}

	// Header row height
	_ = f.SetRowHeight(sheetName, 1, 28)

	headers := []string{"Tarih", "Yön", "Kanal", "Tutar (TL)", "Açıklama", "Durum"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheetName, cell, h)
		_ = f.SetCellStyle(sheetName, cell, cell, styles.headerStyle)
	}

	maxColC := 22.0
	maxColE := 40.0
	maxColF := 26.0

	rowIdx := 2
	for _, tx := range txs {
		desc := ""
		if tx.Description != nil {
			desc = strings.TrimSpace(*tx.Description)
		}

		isReversedOriginal := reversedOriginalIDs[tx.ID]
		isReversalCounter := tx.ReversedBy != nil || strings.Contains(desc, "[İPTAL/TERS KAYIT]")

		if statusFilter == "active" && (isReversedOriginal || isReversalCounter) {
			continue
		}

		dirStr := "Gelir"
		if tx.Direction == domain.DirectionOut {
			dirStr = "Gider"
		}

		statusStr := "Aktif"
		if isReversedOriginal {
			statusStr = "İptal Edildi (Geçersiz)"
		} else if isReversalCounter {
			statusStr = "Ters Kayıt (Denkleştirme)"
		}

		chStr := formatChannel(tx.Channel)

		// Dynamic column width adjustments
		if l := float64(len([]rune(chStr))) + 4; l > maxColC {
			maxColC = l
		}
		if l := float64(len([]rune(desc))) + 4; l > maxColE {
			if l > 75 {
				maxColE = 75
			} else {
				maxColE = l
			}
		}
		if l := float64(len([]rune(statusStr))) + 4; l > maxColF {
			maxColF = l
		}

		_ = f.SetRowHeight(sheetName, rowIdx, 22)

		rowStatusStyle := styles.centerStyle
		if isReversedOriginal {
			rowStatusStyle = styles.cancelledCenterStyle
		}

		dateCell, _ := excelize.CoordinatesToCellName(1, rowIdx)
		dirCell, _ := excelize.CoordinatesToCellName(2, rowIdx)
		chCell, _ := excelize.CoordinatesToCellName(3, rowIdx)
		amountCell, _ := excelize.CoordinatesToCellName(4, rowIdx)
		descCell, _ := excelize.CoordinatesToCellName(5, rowIdx)
		statusCell, _ := excelize.CoordinatesToCellName(6, rowIdx)

		_ = f.SetCellValue(sheetName, dateCell, tx.CreatedAt.Format("2006-01-02 15:04"))
		_ = f.SetCellStyle(sheetName, dateCell, dateCell, styles.dateStyle)

		_ = f.SetCellValue(sheetName, dirCell, dirStr)
		_ = f.SetCellStyle(sheetName, dirCell, dirCell, styles.centerStyle)

		_ = f.SetCellValue(sheetName, chCell, chStr)
		_ = f.SetCellStyle(sheetName, chCell, chCell, styles.textStyle)

		amountFloat, _ := tx.Amount.Float64()
		_ = f.SetCellValue(sheetName, amountCell, amountFloat)
		_ = f.SetCellStyle(sheetName, amountCell, amountCell, styles.amountStyle)

		_ = f.SetCellValue(sheetName, descCell, desc)
		_ = f.SetCellStyle(sheetName, descCell, descCell, styles.textStyle)

		_ = f.SetCellValue(sheetName, statusCell, statusStr)
		_ = f.SetCellStyle(sheetName, statusCell, statusCell, rowStatusStyle)

		rowIdx++
	}

	// Set dynamic column widths
	_ = f.SetColWidth(sheetName, "A", "A", 20)
	_ = f.SetColWidth(sheetName, "B", "B", 12)
	_ = f.SetColWidth(sheetName, "C", "C", maxColC)
	_ = f.SetColWidth(sheetName, "D", "D", 18)
	_ = f.SetColWidth(sheetName, "E", "E", maxColE)
	_ = f.SetColWidth(sheetName, "F", "F", maxColF)

	// Enable AutoFilter
	lastRow := rowIdx - 1
	if lastRow < 1 {
		lastRow = 1
	}
	_ = f.AutoFilter(sheetName, fmt.Sprintf("A1:F%d", lastRow), nil)

	// Freeze header pane so row 1 stays visible during vertical scrolling
	_ = f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
