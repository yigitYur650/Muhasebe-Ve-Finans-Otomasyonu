package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"deftersystem/backend/internal/domain"
)

type PostgresTenantRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresTenantRepository initializes tenant repository using PostgreSQL.
func NewPostgresTenantRepository(pool *pgxpool.Pool) domain.TenantRepository {
	return &PostgresTenantRepository{pool: pool}
}

func (r *PostgresTenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `SELECT id, name, created_at FROM public.tenants WHERE id = $1`
	var t domain.Tenant
	err := r.pool.QueryRow(ctx, query, id).Scan(&t.ID, &t.Name, &t.CreatedAt)
	if err != nil {
		return nil, MapSQLError(err)
	}
	return &t, nil
}

func (r *PostgresTenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	query := `INSERT INTO public.tenants (id, name, created_at) VALUES ($1, $2, $3)`
	_, err := r.pool.Exec(ctx, query, tenant.ID, tenant.Name, tenant.CreatedAt)
	return MapSQLError(err)
}

func (r *PostgresTenantRepository) GetMember(ctx context.Context, tenantID, userID uuid.UUID) (*domain.TenantMember, error) {
	query := `SELECT id, tenant_id, user_id, role, created_at FROM public.tenant_members WHERE tenant_id = $1 AND user_id = $2`
	var tm domain.TenantMember
	err := r.pool.QueryRow(ctx, query, tenantID, userID).Scan(&tm.ID, &tm.TenantID, &tm.UserID, &tm.Role, &tm.CreatedAt)
	if err != nil {
		return nil, MapSQLError(err)
	}
	return &tm, nil
}

func (r *PostgresTenantRepository) GetMembersByTenantID(ctx context.Context, tenantID uuid.UUID) ([]domain.TenantMember, error) {
	query := `SELECT id, tenant_id, user_id, role, created_at FROM public.tenant_members WHERE tenant_id = $1`
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, MapSQLError(err)
	}
	defer rows.Close()

	var members []domain.TenantMember
	for rows.Next() {
		var tm domain.TenantMember
		if err := rows.Scan(&tm.ID, &tm.TenantID, &tm.UserID, &tm.Role, &tm.CreatedAt); err != nil {
			return nil, MapSQLError(err)
		}
		members = append(members, tm)
	}
	if err := rows.Err(); err != nil {
		return nil, MapSQLError(err)
	}
	return members, nil
}
