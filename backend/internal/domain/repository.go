package domain

import (
	"context"

	"github.com/google/uuid"
)

// TenantRepository defines database operations for tenants and members.
type TenantRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	Create(ctx context.Context, tenant *Tenant) error
	GetMember(ctx context.Context, tenantID, userID uuid.UUID) (*TenantMember, error)
	GetMembersByTenantID(ctx context.Context, tenantID uuid.UUID) ([]TenantMember, error)
}

// PeriodRepository defines database operations for financial periods.
type PeriodRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Period, error)
	GetByLabel(ctx context.Context, tenantID uuid.UUID, label string) (*Period, error)
	GetLatestByTenant(ctx context.Context, tenantID uuid.UUID) (*Period, error)
	OpenNextPeriod(ctx context.Context, tenantID uuid.UUID, label string) (*Period, error)
	Create(ctx context.Context, period *Period) error
	Lock(ctx context.Context, id uuid.UUID) error
	GetPeriodHistory(ctx context.Context, tenantID uuid.UUID) ([]PeriodHistoryItem, error)
}

// TransactionRepository defines database operations for ledger transactions.
type TransactionRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	Create(ctx context.Context, tx *Transaction) error
	GetByPeriodID(ctx context.Context, periodID uuid.UUID) ([]Transaction, error)
	GetSummaryByPeriodID(ctx context.Context, periodID uuid.UUID) (*PeriodSummary, error)
	ReverseTransaction(ctx context.Context, origID uuid.UUID, revTx *Transaction) error
	MarkReversed(ctx context.Context, targetID, reversalID uuid.UUID) error
}

// IdempotencyRepository defines database operations for idempotency keys.
type IdempotencyRepository interface {
	Get(ctx context.Context, key string, tenantID uuid.UUID) (*IdempotencyKey, error)
	Save(ctx context.Context, key *IdempotencyKey) error
}
