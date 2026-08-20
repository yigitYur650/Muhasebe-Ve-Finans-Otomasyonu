package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"deftersystem/backend/internal/domain"
)

type PostgresPeriodRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresPeriodRepository initializes period repository using PostgreSQL.
func NewPostgresPeriodRepository(pool *pgxpool.Pool) domain.PeriodRepository {
	return &PostgresPeriodRepository{pool: pool}
}

func (r *PostgresPeriodRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Period, error) {
	query := `SELECT id, tenant_id, label, starting_balance, status, opened_at, locked_at FROM public.periods WHERE id = $1`
	var p domain.Period
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.TenantID, &p.Label, &p.StartingBalance, &p.Status, &p.OpenedAt, &p.LockedAt,
	)
	if err != nil {
		return nil, MapSQLError(err)
	}
	return &p, nil
}

func (r *PostgresPeriodRepository) GetByLabel(ctx context.Context, tenantID uuid.UUID, label string) (*domain.Period, error) {
	query := `SELECT id, tenant_id, label, starting_balance, status, opened_at, locked_at FROM public.periods WHERE tenant_id = $1 AND label = $2`
	var p domain.Period
	err := r.pool.QueryRow(ctx, query, tenantID, label).Scan(
		&p.ID, &p.TenantID, &p.Label, &p.StartingBalance, &p.Status, &p.OpenedAt, &p.LockedAt,
	)
	if err != nil {
		return nil, MapSQLError(err)
	}
	return &p, nil
}

func (r *PostgresPeriodRepository) GetLatestByTenant(ctx context.Context, tenantID uuid.UUID) (*domain.Period, error) {
	query := `SELECT id, tenant_id, label, starting_balance, status, opened_at, locked_at FROM public.periods WHERE tenant_id = $1 ORDER BY label DESC LIMIT 1`
	var p domain.Period
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(
		&p.ID, &p.TenantID, &p.Label, &p.StartingBalance, &p.Status, &p.OpenedAt, &p.LockedAt,
	)
	if err != nil {
		return nil, MapSQLError(err)
	}
	return &p, nil
}

func (r *PostgresPeriodRepository) OpenNextPeriod(ctx context.Context, tenantID uuid.UUID, label string) (*domain.Period, error) {
	query := `SELECT id, tenant_id, label, starting_balance, status, opened_at, locked_at FROM public.open_next_period($1, $2)`
	var p domain.Period
	err := r.pool.QueryRow(ctx, query, tenantID, label).Scan(
		&p.ID, &p.TenantID, &p.Label, &p.StartingBalance, &p.Status, &p.OpenedAt, &p.LockedAt,
	)
	if err != nil {
		return nil, MapSQLError(err)
	}
	return &p, nil
}

func (r *PostgresPeriodRepository) Create(ctx context.Context, period *domain.Period) error {
	query := `INSERT INTO public.periods (id, tenant_id, label, starting_balance, status, opened_at, locked_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, query, period.ID, period.TenantID, period.Label, period.StartingBalance, period.Status, period.OpenedAt, period.LockedAt)
	return MapSQLError(err)
}

func (r *PostgresPeriodRepository) Lock(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE public.periods SET status = 'locked', locked_at = NOW() WHERE id = $1 AND status = 'open'`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return MapSQLError(err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrPeriodNotFound
	}
	return nil
}

func (r *PostgresPeriodRepository) GetPeriodHistory(ctx context.Context, tenantID uuid.UUID) ([]domain.PeriodHistoryItem, error) {
	query := `
		SELECT 
			p.id AS period_id,
			p.label,
			p.status,
			p.starting_balance,
			COALESCE(SUM(CASE WHEN t.direction = 'in' THEN t.amount ELSE 0 END), 0) AS total_in,
			COALESCE(SUM(CASE WHEN t.direction = 'out' THEN t.amount ELSE 0 END), 0) AS total_out,
			p.opened_at,
			p.locked_at
		FROM public.periods p
		LEFT JOIN public.transactions t ON p.id = t.period_id AND t.reversed_by IS NULL
		WHERE p.tenant_id = $1
		GROUP BY p.id, p.label, p.status, p.starting_balance, p.opened_at, p.locked_at
		ORDER BY p.opened_at DESC
	`
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, MapSQLError(err)
	}
	defer rows.Close()

	var history []domain.PeriodHistoryItem
	for rows.Next() {
		var item domain.PeriodHistoryItem
		var totalIn, totalOut decimal.Decimal
		if err := rows.Scan(
			&item.PeriodID, &item.Label, &item.Status, &item.StartingBalance,
			&totalIn, &totalOut, &item.OpenedAt, &item.LockedAt,
		); err != nil {
			return nil, MapSQLError(err)
		}
		item.TotalIn = totalIn
		item.TotalOut = totalOut
		item.ClosingBalance = item.StartingBalance.Add(totalIn).Sub(totalOut)
		history = append(history, item)
	}
	return history, nil
}
