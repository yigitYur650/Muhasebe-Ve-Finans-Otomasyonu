package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"deftersystem/backend/internal/domain"
)

type PostgresTransactionRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresTransactionRepository initializes transaction repository using PostgreSQL.
func NewPostgresTransactionRepository(pool *pgxpool.Pool) domain.TransactionRepository {
	return &PostgresTransactionRepository{pool: pool}
}

func (r *PostgresTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	query := `
		SELECT id, tenant_id, period_id, direction, channel, amount, description, created_by, created_at, reversed_by
		FROM public.transactions
		WHERE id = $1
	`
	var tx domain.Transaction
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&tx.ID, &tx.TenantID, &tx.PeriodID, &tx.Direction, &tx.Channel,
		&tx.Amount, &tx.Description, &tx.CreatedBy, &tx.CreatedAt, &tx.ReversedBy,
	)
	if err != nil {
		return nil, MapSQLError(err)
	}
	return &tx, nil
}

func (r *PostgresTransactionRepository) Create(ctx context.Context, tx *domain.Transaction) error {
	if err := tx.Validate(); err != nil {
		return err
	}

	query := `
		INSERT INTO public.transactions (id, tenant_id, period_id, direction, channel, amount, description, created_by, created_at, reversed_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		tx.ID, tx.TenantID, tx.PeriodID, tx.Direction, tx.Channel,
		tx.Amount, tx.Description, tx.CreatedBy, tx.CreatedAt, tx.ReversedBy,
	)
	return MapSQLError(err)
}

func (r *PostgresTransactionRepository) GetByPeriodID(ctx context.Context, periodID uuid.UUID) ([]domain.Transaction, error) {
	query := `
		SELECT id, tenant_id, period_id, direction, channel, amount, description, created_by, created_at, reversed_by
		FROM public.transactions
		WHERE period_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, periodID)
	if err != nil {
		return nil, MapSQLError(err)
	}
	defer rows.Close()

	var transactions []domain.Transaction
	for rows.Next() {
		var tx domain.Transaction
		if err := rows.Scan(
			&tx.ID, &tx.TenantID, &tx.PeriodID, &tx.Direction, &tx.Channel,
			&tx.Amount, &tx.Description, &tx.CreatedBy, &tx.CreatedAt, &tx.ReversedBy,
		); err != nil {
			return nil, MapSQLError(err)
		}
		transactions = append(transactions, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, MapSQLError(err)
	}
	return transactions, nil
}

func (r *PostgresTransactionRepository) GetSummaryByPeriodID(ctx context.Context, periodID uuid.UUID) (*domain.PeriodSummary, error) {
	query := `
		SELECT 
			p.id AS period_id,
			p.starting_balance,
			COALESCE(SUM(CASE WHEN t.direction = 'in' AND t.reversed_by IS NULL AND NOT EXISTS (SELECT 1 FROM public.transactions rev WHERE rev.reversed_by = t.id) THEN t.amount ELSE 0 END), 0) AS total_in,
			COALESCE(SUM(CASE WHEN t.direction = 'out' AND t.reversed_by IS NULL AND NOT EXISTS (SELECT 1 FROM public.transactions rev WHERE rev.reversed_by = t.id) THEN t.amount ELSE 0 END), 0) AS total_out
		FROM public.periods p
		LEFT JOIN public.transactions t ON p.id = t.period_id
		WHERE p.id = $1
		GROUP BY p.id, p.starting_balance
	`
	var summary domain.PeriodSummary
	var totalIn, totalOut decimal.Decimal
	err := r.pool.QueryRow(ctx, query, periodID).Scan(
		&summary.PeriodID, &summary.StartingBalance, &totalIn, &totalOut,
	)
	if err != nil {
		return nil, MapSQLError(err)
	}

	summary.TotalIn = totalIn
	summary.TotalOut = totalOut
	summary.ClosingBalance = summary.StartingBalance.Add(totalIn).Sub(totalOut)

	return &summary, nil
}

// ReverseTransaction performs an atomic reversal by inserting a new reversal entry referencing origID. Zero UPDATE statements are executed.
func (r *PostgresTransactionRepository) ReverseTransaction(ctx context.Context, origID uuid.UUID, revTx *domain.Transaction) error {
	if err := revTx.Validate(); err != nil {
		return err
	}

	dbTx, err := r.pool.Begin(ctx)
	if err != nil {
		return MapSQLError(err)
	}
	defer dbTx.Rollback(ctx)

	// Step 1: Ensure original transaction exists
	var exists bool
	checkOrigQuery := `SELECT EXISTS(SELECT 1 FROM public.transactions WHERE id = $1)`
	err = dbTx.QueryRow(ctx, checkOrigQuery, origID).Scan(&exists)
	if err != nil {
		return MapSQLError(err)
	}
	if !exists {
		return domain.ErrTransactionNotFound
	}

	// Step 2: Ensure original transaction has not already been reversed by another entry
	var alreadyReversed bool
	checkReversedQuery := `SELECT EXISTS(SELECT 1 FROM public.transactions WHERE reversed_by = $1)`
	err = dbTx.QueryRow(ctx, checkReversedQuery, origID).Scan(&alreadyReversed)
	if err != nil {
		return MapSQLError(err)
	}
	if alreadyReversed {
		return domain.ErrTransactionAlreadyReversed
	}

	// Step 3: Insert the reversal transaction entry (reversed_by = origID)
	revTx.ReversedBy = &origID
	insertQuery := `
		INSERT INTO public.transactions (id, tenant_id, period_id, direction, channel, amount, description, created_by, created_at, reversed_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = dbTx.Exec(ctx, insertQuery,
		revTx.ID, revTx.TenantID, revTx.PeriodID, revTx.Direction, revTx.Channel,
		revTx.Amount, revTx.Description, revTx.CreatedBy, revTx.CreatedAt, revTx.ReversedBy,
	)
	if err != nil {
		return MapSQLError(err)
	}

	if err := dbTx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit reversal transaction: %w", err)
	}

	return nil
}

func (r *PostgresTransactionRepository) MarkReversed(ctx context.Context, targetID, reversalID uuid.UUID) error {
	return nil
}
