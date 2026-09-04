package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"deftersystem/backend/internal/domain"
)

type PostgresIdempotencyRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresIdempotencyRepository initializes idempotency repository using PostgreSQL.
func NewPostgresIdempotencyRepository(pool *pgxpool.Pool) domain.IdempotencyRepository {
	return &PostgresIdempotencyRepository{pool: pool}
}

func (r *PostgresIdempotencyRepository) Get(ctx context.Context, key string, tenantID uuid.UUID) (*domain.IdempotencyKey, error) {
	query := `SELECT key, tenant_id, response_body, response_status, created_at FROM public.idempotency_keys WHERE key = $1 AND tenant_id = $2`
	var ik domain.IdempotencyKey
	err := r.pool.QueryRow(ctx, query, key, tenantID).Scan(
		&ik.Key, &ik.TenantID, &ik.ResponseBody, &ik.ResponseStatus, &ik.CreatedAt,
	)
	if err != nil {
		return nil, MapSQLError(err)
	}
	return &ik, nil
}

func (r *PostgresIdempotencyRepository) Save(ctx context.Context, idem *domain.IdempotencyKey) error {
	query := `INSERT INTO public.idempotency_keys (key, tenant_id, response_body, response_status, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.pool.Exec(ctx, query, idem.Key, idem.TenantID, idem.ResponseBody, idem.ResponseStatus, idem.CreatedAt)
	return MapSQLError(err)
}
