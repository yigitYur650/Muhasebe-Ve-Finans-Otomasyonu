package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
			COALESCE(SUM(CASE WHEN t.direction = 'in' THEN t.amount ELSE 0 END), 0) AS total_in,
			COALESCE(SUM(CASE WHEN t.direction = 'out' THEN t.amount ELSE 0 END), 0) AS total_out
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

// ReverseTransaction performs an atomic reversal by marking the original transaction and inserting a reversal entry.
func (r *PostgresTransactionRepository) ReverseTransaction(ctx context.Context, origID uuid.UUID, revTx *domain.Transaction) error {
	if err := revTx.Validate(); err != nil {
		return err
	}

	dbTx, err := r.pool.Begin(ctx)
	if err != nil {
		return MapSQLError(err)
	}
	defer dbTx.Rollback(ctx)

	// Step 1: Ensure original transaction exists and is not already reversed
	var isReversed bool
	checkQuery := `SELECT reversed_by IS NOT NULL FROM public.transactions WHERE id = $1`
	err = dbTx.QueryRow(ctx, checkQuery, origID).Scan(&isReversed)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ErrTransactionNotFound
		}
		return MapSQLError(err)
	}
	if isReversed {
		return domain.ErrTransactionAlreadyReversed
	}

	// Step 2: Mark original transaction as reversed by revTx.ID
	updateQuery := `UPDATE public.transactions SET reversed_by = $1 WHERE id = $2 AND reversed_by IS NULL`
	cmd, err := dbTx.Exec(ctx, updateQuery, revTx.ID, origID)
	if err != nil {
		return MapSQLError(err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrTransactionAlreadyReversed
	}

	// Step 3: Insert the reversal transaction entry
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
	query := `UPDATE public.transactions SET reversed_by = $1 WHERE id = $2 AND reversed_by IS NULL`
	cmd, err := r.pool.Exec(ctx, query, reversalID, targetID)
	if err != nil {
		return MapSQLError(err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrTransactionAlreadyReversed
	}
	return nil
}
