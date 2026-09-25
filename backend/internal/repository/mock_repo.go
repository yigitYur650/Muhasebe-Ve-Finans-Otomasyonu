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
		if p.Status == domain.PeriodStatusLocked {
			return domain.ErrPeriodLocked
		}
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

func (m *MockTransactionRepo) GetByPeriodIDPaginated(ctx context.Context, periodID uuid.UUID, limit, offset int) ([]domain.Transaction, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var all []domain.Transaction
	for _, tx := range m.transactions {
		if tx.PeriodID == periodID {
			all = append(all, *tx)
		}
	}
	total := len(all)
	if offset >= total {
		return []domain.Transaction{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (m *MockTransactionRepo) GetSummaryByPeriodID(ctx context.Context, periodID uuid.UUID) (*domain.PeriodSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	reversedTargets := make(map[uuid.UUID]bool)
	for _, tx := range m.transactions {
		if tx.ReversedBy != nil {
			reversedTargets[*tx.ReversedBy] = true
		}
	}
	var totalIn, totalOut decimal.Decimal
	for _, tx := range m.transactions {
		if tx.PeriodID == periodID {
			if tx.ReversedBy != nil || reversedTargets[tx.ID] {
				continue
			}
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
	revTx.ReversedBy = &origID
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

func (m *MockTenantRepo) GetFirstTenant(ctx context.Context) (*domain.Tenant, error) {
	return &domain.Tenant{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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

func (m *MockTenantRepo) GetMembersByUserID(ctx context.Context, userID uuid.UUID) ([]domain.TenantMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []domain.TenantMember
	for _, mem := range m.members {
		if mem.UserID == userID {
			list = append(list, *mem)
		}
	}
	if len(list) == 0 {
		// Return default membership for mock
		list = append(list, domain.TenantMember{
			TenantID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			UserID:   userID,
			Role:     domain.RoleAdmin,
		})
	}
	return list, nil
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

// MockSupplierRepository provides in-memory supplier storage
type MockSupplierRepository struct {
	mu           sync.RWMutex
	suppliers    map[uuid.UUID]*domain.Supplier
	transactions map[uuid.UUID]*domain.SupplierTransaction
}

func NewMockSupplierRepository() *MockSupplierRepository {
	return &MockSupplierRepository{
		suppliers:    make(map[uuid.UUID]*domain.Supplier),
		transactions: make(map[uuid.UUID]*domain.SupplierTransaction),
	}
}

func (m *MockSupplierRepository) GetSuppliersWithBalances(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) ([]domain.Supplier, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []domain.Supplier
	for _, s := range m.suppliers {
		if s.TenantID == tenantID || tenantID == uuid.Nil {
			list = append(list, *s)
		}
	}
	return list, nil
}

func (m *MockSupplierRepository) GetSupplierByID(ctx context.Context, tenantID, supplierID uuid.UUID) (*domain.Supplier, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.suppliers[supplierID]; ok {
		return s, nil
	}
	return nil, domain.ErrNotFound
}

func (m *MockSupplierRepository) FindOrCreateSupplier(ctx context.Context, tenantID uuid.UUID, name string) (*domain.Supplier, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.suppliers {
		if s.TenantID == tenantID && s.Name == name {
			return s, nil
		}
	}
	s := &domain.Supplier{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	m.suppliers[s.ID] = s
	return s, nil
}

func (m *MockSupplierRepository) CreateSupplier(ctx context.Context, s *domain.Supplier) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.suppliers[s.ID] = s
	return nil
}

func (m *MockSupplierRepository) CreateTransaction(ctx context.Context, tx *domain.SupplierTransaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions[tx.ID] = tx
	return nil
}

func (m *MockSupplierRepository) GetTransactionByID(ctx context.Context, tenantID, txID uuid.UUID) (*domain.SupplierTransaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if tx, ok := m.transactions[txID]; ok {
		return tx, nil
	}
	return nil, domain.ErrNotFound
}

func (m *MockSupplierRepository) ReverseTransaction(ctx context.Context, tenantID, origID uuid.UUID, revTx *domain.SupplierTransaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions[revTx.ID] = revTx
	if orig, ok := m.transactions[origID]; ok {
		orig.ReversedBy = &revTx.ID
	}
	return nil
}

func (m *MockSupplierRepository) GetTransactionsBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter domain.SupplierTransactionFilter) ([]domain.SupplierTransaction, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []domain.SupplierTransaction
	for _, tx := range m.transactions {
		if tx.SupplierID == supplierID {
			list = append(list, *tx)
		}
	}
	return list, len(list), nil
}

func (m *MockSupplierRepository) GetAllTransactions(ctx context.Context, tenantID uuid.UUID, filter domain.SupplierTransactionFilter) ([]domain.SupplierTransaction, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []domain.SupplierTransaction
	for _, tx := range m.transactions {
		if tx.TenantID == tenantID || tenantID == uuid.Nil {
			list = append(list, *tx)
		}
	}
	return list, len(list), nil
}

func (m *MockSupplierRepository) BatchInsertTransactions(ctx context.Context, tenantID uuid.UUID, transactions []domain.SupplierTransaction) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range transactions {
		m.transactions[transactions[i].ID] = &transactions[i]
	}
	return len(transactions), nil
}

func (m *MockSupplierRepository) GetSummary(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) (*domain.SupplierSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var purchases, payments decimal.Decimal
	for _, tx := range m.transactions {
		if tx.TenantID == tenantID || tenantID == uuid.Nil {
			if tx.ReversedBy == nil {
				if tx.Direction == domain.SupplierDirectionPurchase {
					purchases = purchases.Add(tx.Amount)
				} else if tx.Direction == domain.SupplierDirectionPayment {
					payments = payments.Add(tx.Amount)
				}
			}
		}
	}
	return &domain.SupplierSummary{
		TotalPurchases:       purchases,
		TotalPayments:        payments,
		NetBalance:           purchases.Sub(payments),
		ActiveSupplierCount:  len(m.suppliers),
		TotalTransactionRows: len(m.transactions),
	}, nil
}

