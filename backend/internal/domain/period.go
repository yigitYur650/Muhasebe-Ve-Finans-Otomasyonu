package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PeriodStatus string

const (
	PeriodStatusOpen   PeriodStatus = "open"
	PeriodStatusLocked PeriodStatus = "locked"
)

func (s PeriodStatus) IsValid() bool {
	switch s {
	case PeriodStatusOpen, PeriodStatusLocked:
		return true
	default:
		return false
	}
}

// Period represents a monthly financial period for a tenant.
type Period struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	Label           string          `json:"label"` // e.g. "2025-05"
	StartingBalance decimal.Decimal `json:"starting_balance"`
	Status          PeriodStatus    `json:"status"`
	OpenedAt        time.Time       `json:"opened_at"`
	LockedAt        *time.Time      `json:"locked_at,omitempty"`
}

// PeriodHistoryItem represents historical period balance summary.
type PeriodHistoryItem struct {
	PeriodID        uuid.UUID       `json:"period_id"`
	Label           string          `json:"label"`
	Status          PeriodStatus    `json:"status"`
	StartingBalance decimal.Decimal `json:"starting_balance"`
	TotalIn         decimal.Decimal `json:"total_in"`
	TotalOut        decimal.Decimal `json:"total_out"`
	ClosingBalance  decimal.Decimal `json:"closing_balance"`
	OpenedAt        time.Time       `json:"opened_at"`
	LockedAt        *time.Time      `json:"locked_at,omitempty"`
}

// IsLocked checks whether the period is locked for new transaction insertions.
func (p *Period) IsLocked() bool {
	return p.Status == PeriodStatusLocked
}
