package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"deftersystem/backend/internal/domain"
)

type DefaultTenantService struct {
	tenantRepo domain.TenantRepository
}

// NewTenantService initializes tenant & member business service.
func NewTenantService(tenantRepo domain.TenantRepository) domain.TenantService {
	return &DefaultTenantService{
		tenantRepo: tenantRepo,
	}
}

func (s *DefaultTenantService) ListMembers(ctx context.Context, tenantID uuid.UUID) ([]domain.TenantMember, error) {
	return s.tenantRepo.GetMembersByTenantID(ctx, tenantID)
}

func (s *DefaultTenantService) AddMember(
	ctx context.Context,
	tenantID, requestingUserID, targetUserID uuid.UUID,
	role domain.Role,
) error {
	reqMember, err := s.tenantRepo.GetMember(ctx, tenantID, requestingUserID)
	if err != nil || reqMember == nil {
		return domain.ErrUnauthorized
	}
	if reqMember.Role != domain.RoleAdmin {
		return domain.ErrUnauthorized
	}

	if !role.IsValid() {
		return domain.ErrInvalidRole
	}

	return s.tenantRepo.AddMember(ctx, &domain.TenantMember{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    targetUserID,
		Role:      role,
		CreatedAt: time.Now(),
	})
}

func (s *DefaultTenantService) UpdateMemberRole(
	ctx context.Context,
	tenantID, requestingUserID, targetUserID uuid.UUID,
	newRole domain.Role,
) error {
	reqMember, err := s.tenantRepo.GetMember(ctx, tenantID, requestingUserID)
	if err != nil || reqMember == nil {
		return domain.ErrUnauthorized
	}
	if reqMember.Role != domain.RoleAdmin {
		return domain.ErrUnauthorized
	}

	if !newRole.IsValid() {
		return domain.ErrInvalidRole
	}

	targetMember, err := s.tenantRepo.GetMember(ctx, tenantID, targetUserID)
	if err != nil || targetMember == nil {
		return domain.ErrNotFound
	}

	// Check last admin protection if demoting an admin to non-admin
	if targetMember.Role == domain.RoleAdmin && newRole != domain.RoleAdmin {
		members, err := s.tenantRepo.GetMembersByTenantID(ctx, tenantID)
		if err == nil {
			adminCount := 0
			for _, m := range members {
				if m.Role == domain.RoleAdmin {
					adminCount++
				}
			}
			if adminCount <= 1 {
				return domain.ErrCannotRemoveLastAdmin
			}
		}
	}

	return s.tenantRepo.UpdateMemberRole(ctx, tenantID, targetUserID, newRole)
}

func (s *DefaultTenantService) RemoveMember(
	ctx context.Context,
	tenantID, requestingUserID, targetUserID uuid.UUID,
) error {
	reqMember, err := s.tenantRepo.GetMember(ctx, tenantID, requestingUserID)
	if err != nil || reqMember == nil {
		return domain.ErrUnauthorized
	}
	if reqMember.Role != domain.RoleAdmin {
		return domain.ErrUnauthorized
	}

	targetMember, err := s.tenantRepo.GetMember(ctx, tenantID, targetUserID)
	if err != nil || targetMember == nil {
		return domain.ErrNotFound
	}

	// Prevent removing last admin
	if targetMember.Role == domain.RoleAdmin {
		members, err := s.tenantRepo.GetMembersByTenantID(ctx, tenantID)
		if err == nil {
			adminCount := 0
			for _, m := range members {
				if m.Role == domain.RoleAdmin {
					adminCount++
				}
			}
			if adminCount <= 1 {
				return domain.ErrCannotRemoveLastAdmin
			}
		}
	}

	return s.tenantRepo.RemoveMember(ctx, tenantID, targetUserID)
}
