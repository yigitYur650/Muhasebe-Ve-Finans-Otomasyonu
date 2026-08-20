package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"deftersystem/backend/internal/domain"
)

type DefaultTransactionService struct {
	txRepo     domain.TransactionRepository
	periodRepo domain.PeriodRepository
}

// NewTransactionService initializes transaction business service.
func NewTransactionService(txRepo domain.TransactionRepository, periodRepo domain.PeriodRepository) domain.TransactionService {
	return &DefaultTransactionService{
		txRepo:     txRepo,
		periodRepo: periodRepo,
	}
}

func (s *DefaultTransactionService) CreateTransaction(ctx context.Context, tx *domain.Transaction) error {
	if err := tx.Validate(); err != nil {
		return err
	}

	period, err := s.periodRepo.GetByID(ctx, tx.PeriodID)
	if err != nil {
		return err
	}

	if period.IsLocked() {
		return domain.ErrPeriodLocked
	}

	return s.txRepo.Create(ctx, tx)
}

func (s *DefaultTransactionService) ReverseTransaction(ctx context.Context, origID uuid.UUID, reason string, createdBy uuid.UUID) (*domain.Transaction, error) {
	orig, err := s.txRepo.GetByID(ctx, origID)
	if err != nil {
		return nil, err
	}

	if orig.ReversedBy != nil {
		return nil, domain.ErrTransactionAlreadyReversed
	}

	period, err := s.periodRepo.GetByID(ctx, orig.PeriodID)
	if err != nil {
		return nil, err
	}

	if period.IsLocked() {
		return nil, domain.ErrPeriodLocked
	}

	// Inverse direction rule: 'in' -> 'out', 'out' -> 'in'
	oppositeDirection := domain.DirectionOut
	if orig.Direction == domain.DirectionOut {
		oppositeDirection = domain.DirectionIn
	}

	desc := fmt.Sprintf("[İPTAL/TERS KAYIT] %s", reason)

	revTx := &domain.Transaction{
		ID:          uuid.New(),
		TenantID:    orig.TenantID,
		PeriodID:    orig.PeriodID,
		Direction:   oppositeDirection,
		Channel:     orig.Channel,
		Amount:      orig.Amount, // Exact decimal amount preserved
		Description: &desc,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		ReversedBy:  nil,
	}

	if err := s.txRepo.ReverseTransaction(ctx, origID, revTx); err != nil {
		return nil, err
	}

	return revTx, nil
}
