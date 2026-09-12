package handler

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler/middleware"
)

type ExportHandler struct {
	txRepo     domain.TransactionRepository
	periodRepo domain.PeriodRepository
}

func NewExportHandler(txRepo domain.TransactionRepository, periodRepo domain.PeriodRepository) *ExportHandler {
	return &ExportHandler{
		txRepo:     txRepo,
		periodRepo: periodRepo,
	}
}

func (h *ExportHandler) ExportTransactionsCSV(c *fiber.Ctx) error {
	periodIDParam := c.Params("id")
	periodID, err := uuid.Parse(periodIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
			Success: false,
			Error: &ErrorData{
				Code:    "INVALID_ID",
				Message: "Geçersiz dönem kimliği",
			},
		})
	}

	period, err := h.periodRepo.GetByID(c.Context(), periodID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ResponseEnvelope{
			Success: false,
			Error: &ErrorData{
				Code:    "PERIOD_NOT_FOUND",
				Message: "İstenen dönem bulunamadı",
			},
		})
	}

	// Verify Tenant Ownership (IDOR Protection)
	tenantIDVal := c.Locals(middleware.LocalTenantIDKey)
	if tenantID, ok := tenantIDVal.(uuid.UUID); ok && tenantID != uuid.Nil {
		if period.TenantID != tenantID {
			return c.Status(fiber.StatusForbidden).JSON(ResponseEnvelope{
				Success: false,
				Error: &ErrorData{
					Code:    "FORBIDDEN",
					Message: "Bu döneme ait verilere erişim yetkiniz yok",
				},
			})
		}
	}

	txs, err := h.txRepo.GetByPeriodID(c.Context(), periodID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ResponseEnvelope{
			Success: false,
			Error: &ErrorData{
				Code:    "INTERNAL_ERROR",
				Message: "İşlemler listelenirken sunucu hatası oluştu",
			},
		})
	}

	statusFilter := c.Query("status", "all")

	// Pre-scan all reversal target IDs to accurately identify cancelled transactions
	reversedOriginalIDs := make(map[uuid.UUID]bool)
	for _, tx := range txs {
		if tx.ReversedBy != nil {
			reversedOriginalIDs[*tx.ReversedBy] = true
		}
	}

	buf := new(bytes.Buffer)

	// Write UTF-8 BOM for Excel Turkish character compatibility
	buf.WriteString("\xEF\xBB\xBF")

	writer := csv.NewWriter(buf)
	writer.Comma = ';' // Native Excel separator in Turkish Windows

	// Write CSV Header
	_ = writer.Write([]string{"Tarih", "Yön", "Kanal", "Tutar (TL)", "Açıklama", "Durum"})

	for _, tx := range txs {
		desc := ""
		if tx.Description != nil {
			desc = strings.TrimSpace(*tx.Description)
		}

		isReversedOriginal := reversedOriginalIDs[tx.ID]
		isReversalCounter := tx.ReversedBy != nil || strings.Contains(desc, "[İPTAL/TERS KAYIT]")

		// If user requested only active transactions, skip cancelled and reversal records
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

		if len(desc) > 0 && (desc[0] == '=' || desc[0] == '+' || desc[0] == '-' || desc[0] == '@' || desc[0] == '\t' || desc[0] == '\r') {
			desc = "'" + desc
		}

		// Format amount with comma as decimal mark for Turkish Excel number recognition
		amountStr := strings.ReplaceAll(tx.Amount.StringFixed(2), ".", ",")

		row := []string{
			tx.CreatedAt.Format("2006-01-02 15:04"),
			dirStr,
			formatChannel(tx.Channel),
			amountStr,
			desc,
			statusStr,
		}
		_ = writer.Write(row)
	}

	writer.Flush()

	suffix := "tum-kayitlar"
	if statusFilter == "active" {
		suffix = "aktif-kayitlar"
	}
	filename := fmt.Sprintf("defter-islem-defteri-%s-%s.csv", period.Label, suffix)

	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", filename))

	return c.Status(fiber.StatusOK).Send(buf.Bytes())
}

func (h *ExportHandler) ExportTransactionsExcel(c *fiber.Ctx) error {
	periodIDParam := c.Params("id")
	periodID, err := uuid.Parse(periodIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ResponseEnvelope{
			Success: false,
			Error: &ErrorData{
				Code:    "INVALID_ID",
				Message: "Geçersiz dönem kimliği",
			},
		})
	}

	period, err := h.periodRepo.GetByID(c.Context(), periodID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ResponseEnvelope{
			Success: false,
			Error: &ErrorData{
				Code:    "PERIOD_NOT_FOUND",
				Message: "İstenen dönem bulunamadı",
			},
		})
	}

	// Verify Tenant Ownership (IDOR Protection)
	tenantIDVal := c.Locals(middleware.LocalTenantIDKey)
	if tenantID, ok := tenantIDVal.(uuid.UUID); ok && tenantID != uuid.Nil {
		if period.TenantID != tenantID {
			return c.Status(fiber.StatusForbidden).JSON(ResponseEnvelope{
				Success: false,
				Error: &ErrorData{
					Code:    "FORBIDDEN",
					Message: "Bu döneme ait verilere erişim yetkiniz yok",
				},
			})
		}
	}

	txs, err := h.txRepo.GetByPeriodID(c.Context(), periodID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ResponseEnvelope{
			Success: false,
			Error: &ErrorData{
				Code:    "INTERNAL_ERROR",
				Message: "İşlemler listelenirken sunucu hatası oluştu",
			},
		})
	}

	statusFilter := c.Query("status", "all")

	excelBytes, err := GenerateTransactionsExcel(txs, statusFilter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ResponseEnvelope{
			Success: false,
			Error: &ErrorData{
				Code:    "EXCEL_GEN_ERROR",
				Message: "Excel dosyası oluşturulurken hata oluştu",
			},
		})
	}

	suffix := "tum-kayitlar"
	if statusFilter == "active" {
		suffix = "aktif-kayitlar"
	}
	filename := fmt.Sprintf("defter-islem-defteri-%s-%s.xlsx", period.Label, suffix)

	c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", filename))

	return c.Status(fiber.StatusOK).Send(excelBytes)
}


func formatChannel(ch domain.Channel) string {
	switch ch {
	case domain.ChannelNakit:
		return "Elden Nakit"
	case domain.ChannelPos:
		return "POS Çekimi"
	case domain.ChannelEft:
		return "EFT / Havale"
	case domain.ChannelMaasElden:
		return "Maaş (Elden)"
	case domain.ChannelMaasBanka:
		return "Maaş (Banka)"
	case domain.ChannelKrediKarti:
		return "Kredi Kartı"
	case domain.ChannelKredi:
		return "Kredi"
	case domain.ChannelKira:
		return "Kira"
	case domain.ChannelKartus:
		return "Kartuş / Sarf"
	case domain.ChannelYemek:
		return "Yemek Masrafı"
	case domain.ChannelYakit:
		return "Yakıt"
	case domain.ChannelDiger:
		return "Diğer"
	default:
		return string(ch)
	}
}

func (h *ExportHandler) DownloadSampleCSVTemplate(c *fiber.Ctx) error {
	buf := new(bytes.Buffer)
	buf.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(buf)

	_ = writer.Write([]string{"Yön", "Kanal", "Tutar", "Açıklama"})
	_ = writer.Write([]string{"Gelir", "eft", "1500.50", "Örnek Müşteri Ödemesi"})
	_ = writer.Write([]string{"Gider", "kira", "3200.00", "Örnek Ofis Kirası"})
	_ = writer.Write([]string{"Gelir", "pos", "450.75", "Örnek POS Çekimi"})
	writer.Flush()

	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, "attachment; filename=\"defter-import-sablonu.csv\"")

	return c.Status(fiber.StatusOK).Send(buf.Bytes())
}
