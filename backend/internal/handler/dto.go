package handler

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"deftersystem/backend/internal/domain"
)

// ResponseEnvelope is the standard JSON response format.
type ResponseEnvelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorData  `json:"error,omitempty"`
}

type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// OpenPeriodRequest represents payload to open a new period.
type OpenPeriodRequest struct {
	Label string `json:"label"` // e.g. "2026-09"
}

// CreateTransactionRequest represents payload to insert a new transaction.
type CreateTransactionRequest struct {
	PeriodID    uuid.UUID       `json:"period_id"`
	Direction   domain.Direction`json:"direction"`
	Channel     domain.Channel  `json:"channel"`
	Amount      decimal.Decimal `json:"amount"`
	Description *string         `json:"description,omitempty"`
}

// ReverseTransactionRequest represents payload to request a transaction reversal.
type ReverseTransactionRequest struct {
	Reason string `json:"reason"`
}
