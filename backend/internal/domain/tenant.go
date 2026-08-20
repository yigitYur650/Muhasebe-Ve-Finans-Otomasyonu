package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin      Role = "admin"
	RoleMuhasebeci Role = "muhasebeci"
	RoleStandart   Role = "standart"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleMuhasebeci, RoleStandart:
		return true
	default:
		return false
	}
}

// Tenant represents a multi-tenant business entity.
type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// TenantMember maps a user (auth.users) to a tenant with a specific role.
type TenantMember struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}
