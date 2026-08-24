package repository

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"deftersystem/backend/internal/domain"
)

// MockIdemRepo provides thread-safe in-memory idempotency storage
type MockIdemRepo struct {
	mu    sync.RWMutex
	store map[string]*domain.IdempotencyKey
}

func NewMockIdemRepo() *MockIdemRepo {
	return &MockIdemRepo{store: make(map[string]*domain.IdempotencyKey)}
}

func (m *MockIdemRepo) Get(ctx context.Context, key string, tenantID uuid.UUID) (*domain.IdempotencyKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	compositeKey := key + ":" + tenantID.String()
	if val, ok := m.store[compositeKey]; ok {
		return val, nil
	}
	return nil, domain.ErrNotFound
}

func (m *MockIdemRepo) Save(ctx context.Context, idem *domain.IdempotencyKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	compositeKey := idem.Key + ":" + idem.TenantID.String()
	m.store[compositeKey] = idem
	return nil
}

// MockPeriodRepo provides thread-safe in-memory period storage
type MockPeriodRepo struct {
	mu      sync.RWMutex
	periods map[uuid.UUID]*domain.Period
}

func NewMockPeriodRepo() *MockPeriodRepo {
	repo := &MockPeriodRepo{periods: make(map[uuid.UUID]*domain.Period)}
	// Pre-seed an initial default period
	defaultID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	defaultTenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	now := time.Now()
	repo.periods[defaultID] = &domain.Period{
		ID:              defaultID,
		TenantID:        defaultTenantID,
		Label:           "2026-08",
		StartingBalance: decimal.Zero,
		Status:          domain.PeriodStatusOpen,
		OpenedAt:        now,
	}
	return repo
}

func (m *MockPeriodRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Period, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.periods[id]; ok {
		return p, nil
	}
	return nil, domain.ErrPeriodNotFound
}

func (m *MockPeriodRepo) GetByLabel(ctx context.Context, tenantID uuid.UUID, label string) (*domain.Period, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.periods {
		if p.TenantID == tenantID && p.Label == label {
			return p, nil
		}
	}
	return nil, domain.ErrPeriodNotFound
}

func (m *MockPeriodRepo) GetLatestByTenant(ctx context.Context, tenantID uuid.UUID) (*domain.Period, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var latest *domain.Period
	for _, p := range m.periods {
		if p.TenantID == tenantID || tenantID == uuid.Nil {
			if latest == nil || p.Label > latest.Label {
				latest = p
			}
		}
	}
	if latest != nil {
		return latest, nil
	}
	return nil, domain.ErrPeriodNotFound
}

func (m *MockPeriodRepo) OpenNextPeriod(ctx context.Context, tenantID uuid.UUID, label string) (*domain.Period, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	newP := &domain.Period{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Label:           label,
		StartingBalance: decimal.Zero,
		Status:          domain.PeriodStatusOpen,
		OpenedAt:        now,
	}
	m.periods[newP.ID] = newP
	return newP, nil
}

func (m *MockPeriodRepo) Create(ctx context.Context, period *domain.Period) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.periods[period.ID] = period
	return nil
}

func (m *MockPeriodRepo) Lock(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.periods[id]; ok {
		p.Status = domain.PeriodStatusLocked
		now := time.Now()
		p.LockedAt = &now
		return nil
	}
	return domain.ErrPeriodNotFound
}

func (m *MockPeriodRepo) Unlock(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.periods[id]; ok {
		p.Status = domain.PeriodStatusOpen
		p.LockedAt = nil
		return nil
	}
	return domain.ErrPeriodNotFound
}

func (m *MockPeriodRepo) GetPeriodHistory(ctx context.Context, tenantID uuid.UUID) ([]domain.PeriodHistoryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []domain.PeriodHistoryItem
	for _, p := range m.periods {
		if p.TenantID == tenantID || tenantID == uuid.Nil {
			list = append(list, domain.PeriodHistoryItem{
				PeriodID:        p.ID,
				Label:           p.Label,
				Status:          p.Status,
				StartingBalance: p.StartingBalance,
				TotalIn:         decimal.Zero,
				TotalOut:        decimal.Zero,
				ClosingBalance:  p.StartingBalance,
				OpenedAt:        p.OpenedAt,
				LockedAt:        p.LockedAt,
			})
		}
	}
	return list, nil
}

// MockTransactionRepo provides thread-safe in-memory transaction storage
type MockTransactionRepo struct {
	mu           sync.RWMutex
	transactions map[uuid.UUID]*domain.Transaction
}

func NewMockTransactionRepo() *MockTransactionRepo {
	return &MockTransactionRepo{transactions: make(map[uuid.UUID]*domain.Transaction)}
}

func (m *MockTransactionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if tx, ok := m.transactions[id]; ok {
		return tx, nil
	}
	return nil, domain.ErrNotFound
}

func (m *MockTransactionRepo) Create(ctx context.Context, tx *domain.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions[tx.ID] = tx
	return nil
}

func (m *MockTransactionRepo) GetByPeriodID(ctx context.Context, periodID uuid.UUID) ([]domain.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []domain.Transaction
	for _, tx := range m.transactions {
		if tx.PeriodID == periodID {
			list = append(list, *tx)
		}
	}
	return list, nil
}

func (m *MockTransactionRepo) GetSummaryByPeriodID(ctx context.Context, periodID uuid.UUID) (*domain.PeriodSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var totalIn, totalOut decimal.Decimal
	for _, tx := range m.transactions {
		if tx.PeriodID == periodID && tx.ReversedBy == nil {
			if tx.Direction == domain.DirectionIn {
				totalIn = totalIn.Add(tx.Amount)
			} else if tx.Direction == domain.DirectionOut {
				totalOut = totalOut.Add(tx.Amount)
			}
		}
	}
	return &domain.PeriodSummary{
		PeriodID:        periodID,
		StartingBalance: decimal.Zero,
		TotalIn:         totalIn,
		TotalOut:        totalOut,
		ClosingBalance:  totalIn.Sub(totalOut),
	}, nil
}

func (m *MockTransactionRepo) ReverseTransaction(ctx context.Context, origID uuid.UUID, revTx *domain.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions[revTx.ID] = revTx
	if orig, ok := m.transactions[origID]; ok {
		orig.ReversedBy = &revTx.ID
	}
	return nil
}

func (m *MockTransactionRepo) MarkReversed(ctx context.Context, targetID, reversalID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if orig, ok := m.transactions[targetID]; ok {
		orig.ReversedBy = &reversalID
	}
	return nil
}

// MockTenantRepo provides thread-safe in-memory tenant storage
type MockTenantRepo struct {
	mu      sync.RWMutex
	members map[string]*domain.TenantMember
}

func NewMockTenantRepo() *MockTenantRepo {
	return &MockTenantRepo{members: make(map[string]*domain.TenantMember)}
}

func (m *MockTenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	return &domain.Tenant{
		ID:        id,
		Name:      "Öncü Otogaz Ana Şube",
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockTenantRepo) Create(ctx context.Context, tenant *domain.Tenant) error {
	return nil
}

func (m *MockTenantRepo) GetMembersByTenantID(ctx context.Context, tenantID uuid.UUID) ([]domain.TenantMember, error) {
	return m.ListMembers(ctx, tenantID)
}

func (m *MockTenantRepo) GetMember(ctx context.Context, tenantID, userID uuid.UUID) (*domain.TenantMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := tenantID.String() + ":" + userID.String()
	if mem, ok := m.members[key]; ok {
		return mem, nil
	}
	// Fallback admin role in mock mode
	return &domain.TenantMember{
		TenantID: tenantID,
		UserID:   userID,
		Role:     domain.RoleAdmin,
	}, nil
}

func (m *MockTenantRepo) ListMembers(ctx context.Context, tenantID uuid.UUID) ([]domain.TenantMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []domain.TenantMember
	for _, mem := range m.members {
		if mem.TenantID == tenantID {
			list = append(list, *mem)
		}
	}
	return list, nil
}

func (m *MockTenantRepo) AddMember(ctx context.Context, member *domain.TenantMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := member.TenantID.String() + ":" + member.UserID.String()
	m.members[key] = member
	return nil
}

func (m *MockTenantRepo) UpdateMemberRole(ctx context.Context, tenantID, userID uuid.UUID, role domain.Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := tenantID.String() + ":" + userID.String()
	if mem, ok := m.members[key]; ok {
		mem.Role = role
	}
	return nil
}

func (m *MockTenantRepo) RemoveMember(ctx context.Context, tenantID, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := tenantID.String() + ":" + userID.String()
	delete(m.members, key)
	return nil
}


func (m *MockTenantRepo) CountAdmins(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return 1, nil
}

