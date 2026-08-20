package service_test

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"deftersystem/backend/internal/domain"
)

type MockTenantRepo struct {
	mock.Mock
}

func (m *MockTenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Tenant), args.Error(1)
}

func (m *MockTenantRepo) Create(ctx context.Context, tenant *domain.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepo) GetMember(ctx context.Context, tenantID, userID uuid.UUID) (*domain.TenantMember, error) {
	args := m.Called(ctx, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TenantMember), args.Error(1)
}

func (m *MockTenantRepo) GetMembersByTenantID(ctx context.Context, tenantID uuid.UUID) ([]domain.TenantMember, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.TenantMember), args.Error(1)
}

type MockPeriodRepo struct {
	mock.Mock
}

func (m *MockPeriodRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Period, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Period), args.Error(1)
}

func (m *MockPeriodRepo) GetByLabel(ctx context.Context, tenantID uuid.UUID, label string) (*domain.Period, error) {
	args := m.Called(ctx, tenantID, label)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Period), args.Error(1)
}

func (m *MockPeriodRepo) GetLatestByTenant(ctx context.Context, tenantID uuid.UUID) (*domain.Period, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Period), args.Error(1)
}

func (m *MockPeriodRepo) OpenNextPeriod(ctx context.Context, tenantID uuid.UUID, label string) (*domain.Period, error) {
	args := m.Called(ctx, tenantID, label)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Period), args.Error(1)
}

func (m *MockPeriodRepo) Create(ctx context.Context, period *domain.Period) error {
	args := m.Called(ctx, period)
	return args.Error(0)
}

func (m *MockPeriodRepo) Lock(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPeriodRepo) GetPeriodHistory(ctx context.Context, tenantID uuid.UUID) ([]domain.PeriodHistoryItem, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PeriodHistoryItem), args.Error(1)
}

type MockTransactionRepo struct {
	mock.Mock
}

func (m *MockTransactionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Transaction), args.Error(1)
}

func (m *MockTransactionRepo) Create(ctx context.Context, tx *domain.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepo) GetByPeriodID(ctx context.Context, periodID uuid.UUID) ([]domain.Transaction, error) {
	args := m.Called(ctx, periodID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Transaction), args.Error(1)
}

func (m *MockTransactionRepo) GetSummaryByPeriodID(ctx context.Context, periodID uuid.UUID) (*domain.PeriodSummary, error) {
	args := m.Called(ctx, periodID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PeriodSummary), args.Error(1)
}

func (m *MockTransactionRepo) ReverseTransaction(ctx context.Context, origID uuid.UUID, revTx *domain.Transaction) error {
	args := m.Called(ctx, origID, revTx)
	return args.Error(0)
}

func (m *MockTransactionRepo) MarkReversed(ctx context.Context, targetID, reversalID uuid.UUID) error {
	args := m.Called(ctx, targetID, reversalID)
	return args.Error(0)
}

type MockIdempotencyRepo struct {
	mock.Mock
}

func (m *MockIdempotencyRepo) Get(ctx context.Context, key string, tenantID uuid.UUID) (*domain.IdempotencyKey, error) {
	args := m.Called(ctx, key, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.IdempotencyKey), args.Error(1)
}

func (m *MockIdempotencyRepo) Save(ctx context.Context, key *domain.IdempotencyKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

