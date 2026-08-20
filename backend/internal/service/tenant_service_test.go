package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/service"
)

func TestTenantService_StandartUserUnauthorized(t *testing.T) {
	tenantID := uuid.New()
	standartUserID := uuid.New()
	targetUserID := uuid.New()

	mockTenantRepo := new(MockTenantRepo)
	tenantSvc := service.NewTenantService(mockTenantRepo)

	standartMember := &domain.TenantMember{
		TenantID: tenantID,
		UserID:   standartUserID,
		Role:     domain.RoleStandart,
	}

	mockTenantRepo.On("GetMember", mock.Anything, tenantID, standartUserID).Return(standartMember, nil)

	// Attempting to add member as standard user
	errAdd := tenantSvc.AddMember(context.Background(), tenantID, standartUserID, targetUserID, domain.RoleMuhasebeci)
	assert.Equal(t, domain.ErrUnauthorized, errAdd, "Standard user MUST NOT add members")

	// Attempting to update member role as standard user
	errUpdate := tenantSvc.UpdateMemberRole(context.Background(), tenantID, standartUserID, targetUserID, domain.RoleAdmin)
	assert.Equal(t, domain.ErrUnauthorized, errUpdate, "Standard user MUST NOT update member roles")

	// Attempting to remove member as standard user
	errRemove := tenantSvc.RemoveMember(context.Background(), tenantID, standartUserID, targetUserID)
	assert.Equal(t, domain.ErrUnauthorized, errRemove, "Standard user MUST NOT remove members")
}

func TestTenantService_RemoveLastAdminBlocked(t *testing.T) {
	tenantID := uuid.New()
	adminUserID := uuid.New()

	mockTenantRepo := new(MockTenantRepo)
	tenantSvc := service.NewTenantService(mockTenantRepo)

	adminMember := &domain.TenantMember{
		TenantID: tenantID,
		UserID:   adminUserID,
		Role:     domain.RoleAdmin,
	}

	// Single admin in tenant members list
	allMembers := []domain.TenantMember{*adminMember}

	mockTenantRepo.On("GetMember", mock.Anything, tenantID, adminUserID).Return(adminMember, nil)
	mockTenantRepo.On("GetMembersByTenantID", mock.Anything, tenantID).Return(allMembers, nil)

	// Attempting to remove sole admin MUST return ErrCannotRemoveLastAdmin
	errRemove := tenantSvc.RemoveMember(context.Background(), tenantID, adminUserID, adminUserID)
	assert.Equal(t, domain.ErrCannotRemoveLastAdmin, errRemove, "Removing the last admin in tenant MUST be blocked")

	// Attempting to demote sole admin to standard user MUST return ErrCannotRemoveLastAdmin
	errDemote := tenantSvc.UpdateMemberRole(context.Background(), tenantID, adminUserID, adminUserID, domain.RoleStandart)
	assert.Equal(t, domain.ErrCannotRemoveLastAdmin, errDemote, "Demoting the last admin in tenant MUST be blocked")
}
