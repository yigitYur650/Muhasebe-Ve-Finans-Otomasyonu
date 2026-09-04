package service

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"

	"deftersystem/backend/internal/domain"
)

var labelRegex = regexp.MustCompile(`^\d{4}-\d{2}$`)

type DefaultPeriodService struct {
	periodRepo domain.PeriodRepository
	tenantRepo domain.TenantRepository
	txRepo     domain.TransactionRepository
}

// NewPeriodService initializes period business service.
func NewPeriodService(
	periodRepo domain.PeriodRepository,
	tenantRepo domain.TenantRepository,
	txRepo domain.TransactionRepository,
) domain.PeriodService {
	return &DefaultPeriodService{
		periodRepo: periodRepo,
		tenantRepo: tenantRepo,
		txRepo:     txRepo,
	}
}

func (s *DefaultPeriodService) OpenNextPeriod(ctx context.Context, tenantID uuid.UUID, label string) (*domain.Period, error) {
	if !labelRegex.MatchString(label) {
		return nil, fmt.Errorf("geçersiz dönem etiketi formatı (beklenen: YYYY-MM): %s", label)
	}

	period, err := s.periodRepo.OpenNextPeriod(ctx, tenantID, label)
	if err != nil {
		return nil, err
	}
	return period, nil
}

func (s *DefaultPeriodService) LockPeriod(ctx context.Context, periodID uuid.UUID, requestingUserID uuid.UUID) error {
	period, err := s.periodRepo.GetByID(ctx, periodID)
	if err != nil {
		return err
	}

	member, err := s.tenantRepo.GetMember(ctx, period.TenantID, requestingUserID)
	if err != nil {
		return err
	}

	// Only admin or muhasebeci can lock a financial period
	if member.Role != domain.RoleAdmin && member.Role != domain.RoleMuhasebeci {
		return domain.ErrUnauthorized
	}

	return s.periodRepo.Lock(ctx, periodID)
}

func (s *DefaultPeriodService) UnlockPeriod(ctx context.Context, periodID uuid.UUID, requestingUserID uuid.UUID) error {
	period, err := s.periodRepo.GetByID(ctx, periodID)
	if err != nil {
		return err
	}

	member, err := s.tenantRepo.GetMember(ctx, period.TenantID, requestingUserID)
	if err != nil {
		return err
	}

	if member.Role != domain.RoleAdmin && member.Role != domain.RoleMuhasebeci {
		return domain.ErrUnauthorized
	}

	return s.periodRepo.Unlock(ctx, periodID)
}

func (s *DefaultPeriodService) GetPeriodSummary(ctx context.Context, periodID uuid.UUID) (*domain.PeriodSummary, error) {
	return s.txRepo.GetSummaryByPeriodID(ctx, periodID)
}

func (s *DefaultPeriodService) ListPeriods(ctx context.Context, tenantID uuid.UUID) ([]domain.Period, error) {
	latest, err := s.periodRepo.GetLatestByTenant(ctx, tenantID)
	if err != nil && err != domain.ErrNotFound {
		return nil, err
	}
	if latest != nil {
		return []domain.Period{*latest}, nil
	}
	return []domain.Period{}, nil
}

func (s *DefaultPeriodService) GetPeriodHistory(ctx context.Context, tenantID uuid.UUID) ([]domain.PeriodHistoryItem, error) {
	return s.periodRepo.GetPeriodHistory(ctx, tenantID)
}
