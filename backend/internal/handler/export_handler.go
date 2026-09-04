package handler

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"deftersystem/backend/internal/domain"
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

	buf := new(bytes.Buffer)

	// Write UTF-8 BOM for Excel Turkish character compatibility
	buf.WriteString("\xEF\xBB\xBF")

	writer := csv.NewWriter(buf)

	// Write CSV Header
	_ = writer.Write([]string{"Tarih", "Yön", "Kanal", "Tutar", "Açıklama", "Durum"})

	for _, tx := range txs {
		dirStr := "Gelir"
		if tx.Direction == domain.DirectionOut {
			dirStr = "Gider"
		}

		statusStr := "Aktif"
		if tx.ReversedBy != nil {
			statusStr = "İptal Edildi"
		}

		desc := ""
		if tx.Description != nil {
			desc = *tx.Description
		}

		row := []string{
			tx.CreatedAt.Format("2006-01-02 15:04"),
			dirStr,
			string(tx.Channel),
			tx.Amount.StringFixed(2),
			desc,
			statusStr,
		}
		_ = writer.Write(row)
	}

	writer.Flush()

	filename := fmt.Sprintf("defter-islem-defteri-%s.csv", period.Label)

	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", filename))

	return c.Status(fiber.StatusOK).Send(buf.Bytes())
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
