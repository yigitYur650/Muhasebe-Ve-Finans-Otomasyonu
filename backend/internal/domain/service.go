package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PeriodSummary represents the financial balance summary of a period.
type PeriodSummary struct {
	PeriodID        uuid.UUID       `json:"period_id"`
	StartingBalance decimal.Decimal `json:"starting_balance"`
	TotalIn         decimal.Decimal `json:"total_in"`
	TotalOut        decimal.Decimal `json:"total_out"`
	ClosingBalance  decimal.Decimal `json:"closing_balance"`
}

// PeriodService defines business logic operations for financial periods.
type PeriodService interface {
	OpenNextPeriod(ctx context.Context, tenantID uuid.UUID, label string) (*Period, error)
	LockPeriod(ctx context.Context, periodID uuid.UUID, requestingUserID uuid.UUID) error
	GetPeriodSummary(ctx context.Context, periodID uuid.UUID) (*PeriodSummary, error)
	ListPeriods(ctx context.Context, tenantID uuid.UUID) ([]Period, error)
}

// TransactionService defines business logic operations for ledger transactions.
type TransactionService interface {
	CreateTransaction(ctx context.Context, tx *Transaction) error
	ReverseTransaction(ctx context.Context, origID uuid.UUID, reason string, createdBy uuid.UUID) (*Transaction, error)
}

// TenantService defines business logic operations for tenant members and roles.
type TenantService interface {
	ListMembers(ctx context.Context, tenantID uuid.UUID) ([]TenantMember, error)
	AddMember(ctx context.Context, tenantID, requestingUserID, targetUserID uuid.UUID, role Role) error
	UpdateMemberRole(ctx context.Context, tenantID, requestingUserID, targetUserID uuid.UUID, newRole Role) error
	RemoveMember(ctx context.Context, tenantID, requestingUserID, targetUserID uuid.UUID) error
}
