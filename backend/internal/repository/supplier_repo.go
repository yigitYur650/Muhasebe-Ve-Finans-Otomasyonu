package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"deftersystem/backend/internal/domain"
)

type PostgresSupplierRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresSupplierRepository initializes supplier repository using PostgreSQL.
func NewPostgresSupplierRepository(pool *pgxpool.Pool) domain.SupplierRepository {
	return &PostgresSupplierRepository{pool: pool}
}

func (r *PostgresSupplierRepository) GetSuppliersWithBalances(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) ([]domain.Supplier, error) {
	var query strings.Builder
	query.WriteString(`
		SELECT 
			s.id, 
			s.tenant_id, 
			s.name, 
			s.is_active, 
			s.created_at,
			COALESCE(SUM(CASE WHEN st.direction = 'purchase' THEN st.amount ELSE 0 END), 0) AS total_purchase,
			COALESCE(SUM(CASE WHEN st.direction = 'payment' THEN st.amount ELSE 0 END), 0) AS total_payment,
			COALESCE(SUM(CASE WHEN st.direction = 'purchase' THEN st.amount ELSE -st.amount END), 0) AS balance,
			COUNT(st.id) AS transaction_count
		FROM public.suppliers s
		LEFT JOIN public.supplier_transactions st ON s.id = st.supplier_id AND s.tenant_id = st.tenant_id
	`)

	args := []interface{}{tenantID}
	query.WriteString(" WHERE s.tenant_id = $1")

	if periodID != nil && *periodID != uuid.Nil {
		args = append(args, *periodID)
		query.WriteString(fmt.Sprintf(" AND (st.period_id = $%d OR st.period_id IS NULL)", len(args)))
	}

	query.WriteString(" GROUP BY s.id, s.tenant_id, s.name, s.is_active, s.created_at ORDER BY s.name ASC")

	rows, err := r.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, MapSQLError(err)
	}
	defer rows.Close()

	suppliers := make([]domain.Supplier, 0)
	for rows.Next() {
		var s domain.Supplier
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.IsActive, &s.CreatedAt,
			&s.TotalPurchase, &s.TotalPayment, &s.Balance, &s.TransactionCount,
		); err != nil {
			return nil, MapSQLError(err)
		}
		suppliers = append(suppliers, s)
	}

	return suppliers, nil
}

func (r *PostgresSupplierRepository) GetSupplierByID(ctx context.Context, tenantID, supplierID uuid.UUID) (*domain.Supplier, error) {
	query := `
		SELECT 
			s.id, 
			s.tenant_id, 
			s.name, 
			s.is_active, 
			s.created_at,
			COALESCE(SUM(CASE WHEN st.direction = 'purchase' THEN st.amount ELSE 0 END), 0) AS total_purchase,
			COALESCE(SUM(CASE WHEN st.direction = 'payment' THEN st.amount ELSE 0 END), 0) AS total_payment,
			COALESCE(SUM(CASE WHEN st.direction = 'purchase' THEN st.amount ELSE -st.amount END), 0) AS balance,
			COUNT(st.id) AS transaction_count
		FROM public.suppliers s
		LEFT JOIN public.supplier_transactions st ON s.id = st.supplier_id AND s.tenant_id = st.tenant_id
		WHERE s.tenant_id = $1 AND s.id = $2
		GROUP BY s.id, s.tenant_id, s.name, s.is_active, s.created_at
	`
	var s domain.Supplier
	err := r.pool.QueryRow(ctx, query, tenantID, supplierID).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.IsActive, &s.CreatedAt,
		&s.TotalPurchase, &s.TotalPayment, &s.Balance, &s.TransactionCount,
	)
	if err != nil {
		return nil, MapSQLError(err)
	}
	return &s, nil
}

func (r *PostgresSupplierRepository) FindOrCreateSupplier(ctx context.Context, tenantID uuid.UUID, name string) (*domain.Supplier, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, domain.ErrSupplierNotFound
	}

	query := `
		INSERT INTO public.suppliers (tenant_id, name, is_active, created_at)
		VALUES ($1, $2, true, now())
		ON CONFLICT (tenant_id, name) DO UPDATE 
		SET is_active = true
		RETURNING id, tenant_id, name, is_active, created_at
	`
	var s domain.Supplier
	err := r.pool.QueryRow(ctx, query, tenantID, trimmedName).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.IsActive, &s.CreatedAt,
	)
	if err != nil {
		return nil, MapSQLError(err)
	}
	return &s, nil
}

func (r *PostgresSupplierRepository) CreateSupplier(ctx context.Context, s *domain.Supplier) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	query := `
		INSERT INTO public.suppliers (id, tenant_id, name, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query, s.ID, s.TenantID, strings.TrimSpace(s.Name), s.IsActive, s.CreatedAt)
	return MapSQLError(err)
}

func (r *PostgresSupplierRepository) CreateTransaction(ctx context.Context, tx *domain.SupplierTransaction) error {
	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}
	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = time.Now()
	}
	if tx.TxDate.IsZero() {
		tx.TxDate = time.Now()
	}
	if tx.Amount.LessThanOrEqual(decimal.Zero) {
		return domain.ErrInvalidAmount
	}
	if tx.Direction != domain.SupplierDirectionPurchase && tx.Direction != domain.SupplierDirectionPayment {
		return domain.ErrInvalidSupplierDirection
	}

	query := `
		INSERT INTO public.supplier_transactions (
			id, tenant_id, supplier_id, period_id, invoice_no, customer_name,
			document_status, tx_date, direction, amount, description, created_by, created_at, reversed_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.pool.Exec(ctx, query,
		tx.ID, tx.TenantID, tx.SupplierID, tx.PeriodID, tx.InvoiceNo, tx.CustomerName,
		tx.DocumentStatus, tx.TxDate, tx.Direction, tx.Amount, tx.Description, tx.CreatedBy, tx.CreatedAt, tx.ReversedBy,
	)
	return MapSQLError(err)
}

func (r *PostgresSupplierRepository) GetTransactionByID(ctx context.Context, tenantID, txID uuid.UUID) (*domain.SupplierTransaction, error) {
	query := `
		SELECT 
			st.id, 
			st.tenant_id, 
			st.supplier_id, 
			s.name AS supplier_name,
			st.period_id, 
			p.label AS period_label,
			COALESCE(st.invoice_no, ''), 
			COALESCE(st.customer_name, ''), 
			COALESCE(st.document_status, ''), 
			st.tx_date, 
			st.direction, 
			st.amount, 
			COALESCE(st.description, ''), 
			st.created_by, 
			st.created_at,
			st.reversed_by
		FROM public.supplier_transactions st
		JOIN public.suppliers s ON st.supplier_id = s.id
		JOIN public.periods p ON st.period_id = p.id
		WHERE st.tenant_id = $1 AND st.id = $2
	`
	var tx domain.SupplierTransaction
	err := r.pool.QueryRow(ctx, query, tenantID, txID).Scan(
		&tx.ID, &tx.TenantID, &tx.SupplierID, &tx.SupplierName,
		&tx.PeriodID, &tx.PeriodLabel, &tx.InvoiceNo, &tx.CustomerName,
		&tx.DocumentStatus, &tx.TxDate, &tx.Direction, &tx.Amount,
		&tx.Description, &tx.CreatedBy, &tx.CreatedAt, &tx.ReversedBy,
	)
	if err != nil {
		return nil, MapSQLError(err)
	}
	return &tx, nil
}

// ReverseTransaction performs an atomic reversal by inserting a new reversal entry referencing origID. Zero UPDATE statements are executed.
func (r *PostgresSupplierRepository) ReverseTransaction(ctx context.Context, tenantID, origID uuid.UUID, revTx *domain.SupplierTransaction) error {
	if revTx.Amount.LessThanOrEqual(decimal.Zero) {
		return domain.ErrInvalidAmount
	}
	if revTx.Direction != domain.SupplierDirectionPurchase && revTx.Direction != domain.SupplierDirectionPayment {
		return domain.ErrInvalidSupplierDirection
	}

	dbTx, err := r.pool.Begin(ctx)
	if err != nil {
		return MapSQLError(err)
	}
	defer func() { _ = dbTx.Rollback(ctx) }()

	// Step 1: Ensure original transaction exists for this tenant
	var exists bool
	checkOrigQuery := `SELECT EXISTS(SELECT 1 FROM public.supplier_transactions WHERE id = $1 AND tenant_id = $2)`
	err = dbTx.QueryRow(ctx, checkOrigQuery, origID, tenantID).Scan(&exists)
	if err != nil {
		return MapSQLError(err)
	}
	if !exists {
		return domain.ErrTransactionNotFound
	}

	// Step 2: Ensure original transaction has not already been reversed
	var alreadyReversed bool
	checkReversedQuery := `
		SELECT EXISTS(
			SELECT 1 FROM public.supplier_transactions WHERE reversed_by = $1 AND tenant_id = $2
			UNION ALL
			SELECT 1 FROM public.supplier_transactions WHERE id = $1 AND tenant_id = $2 AND reversed_by IS NOT NULL
		)
	`
	err = dbTx.QueryRow(ctx, checkReversedQuery, origID, tenantID).Scan(&alreadyReversed)
	if err != nil {
		return MapSQLError(err)
	}
	if alreadyReversed {
		return domain.ErrTransactionAlreadyReversed
	}

	// Step 3: Insert reversal entry referencing origID
	if revTx.ID == uuid.Nil {
		revTx.ID = uuid.New()
	}
	if revTx.CreatedAt.IsZero() {
		revTx.CreatedAt = time.Now()
	}
	if revTx.TxDate.IsZero() {
		revTx.TxDate = time.Now()
	}
	revTx.ReversedBy = &origID
	revTx.TenantID = tenantID

	insertQuery := `
		INSERT INTO public.supplier_transactions (
			id, tenant_id, supplier_id, period_id, invoice_no, customer_name,
			document_status, tx_date, direction, amount, description, created_by, created_at, reversed_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err = dbTx.Exec(ctx, insertQuery,
		revTx.ID, revTx.TenantID, revTx.SupplierID, revTx.PeriodID, revTx.InvoiceNo, revTx.CustomerName,
		revTx.DocumentStatus, revTx.TxDate, revTx.Direction, revTx.Amount, revTx.Description, revTx.CreatedBy, revTx.CreatedAt, revTx.ReversedBy,
	)
	if err != nil {
		return MapSQLError(err)
	}

	if err := dbTx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit supplier reversal transaction: %w", err)
	}

	return nil
}

func (r *PostgresSupplierRepository) GetTransactionsBySupplier(
	ctx context.Context,
	tenantID, supplierID uuid.UUID,
	filter domain.SupplierTransactionFilter,
) ([]domain.SupplierTransaction, int, error) {
	filter.SupplierID = &supplierID
	return r.GetAllTransactions(ctx, tenantID, filter)
}

func (r *PostgresSupplierRepository) GetAllTransactions(
	ctx context.Context,
	tenantID uuid.UUID,
	filter domain.SupplierTransactionFilter,
) ([]domain.SupplierTransaction, int, error) {
	var whereClauses []string
	var args []interface{}

	args = append(args, tenantID)
	whereClauses = append(whereClauses, fmt.Sprintf("st.tenant_id = $%d", len(args)))

	if filter.SupplierID != nil && *filter.SupplierID != uuid.Nil {
		args = append(args, *filter.SupplierID)
		whereClauses = append(whereClauses, fmt.Sprintf("st.supplier_id = $%d", len(args)))
	}

	if filter.PeriodID != nil && *filter.PeriodID != uuid.Nil {
		args = append(args, *filter.PeriodID)
		whereClauses = append(whereClauses, fmt.Sprintf("st.period_id = $%d", len(args)))
	}

	if filter.Direction != "" {
		args = append(args, filter.Direction)
		whereClauses = append(whereClauses, fmt.Sprintf("st.direction = $%d", len(args)))
	}

	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		whereClauses = append(whereClauses, fmt.Sprintf("(st.invoice_no ILIKE $%d OR st.customer_name ILIKE $%d OR st.description ILIKE $%d OR s.name ILIKE $%d)", len(args), len(args), len(args), len(args)))
	}

	if filter.DateFrom != nil && !filter.DateFrom.IsZero() {
		args = append(args, *filter.DateFrom)
		whereClauses = append(whereClauses, fmt.Sprintf("st.tx_date >= $%d", len(args)))
	}

	if filter.DateTo != nil && !filter.DateTo.IsZero() {
		args = append(args, *filter.DateTo)
		whereClauses = append(whereClauses, fmt.Sprintf("st.tx_date <= $%d", len(args)))
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Total count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(st.id)
		FROM public.supplier_transactions st
		JOIN public.suppliers s ON st.supplier_id = s.id
		WHERE %s
	`, whereSQL)

	var totalCount int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, MapSQLError(err)
	}

	// Data query with pagination
	dataQuery := fmt.Sprintf(`
		SELECT 
			st.id, 
			st.tenant_id, 
			st.supplier_id, 
			s.name AS supplier_name,
			st.period_id, 
			p.label AS period_label,
			COALESCE(st.invoice_no, ''), 
			COALESCE(st.customer_name, ''), 
			COALESCE(st.document_status, ''), 
			st.tx_date, 
			st.direction, 
			st.amount, 
			COALESCE(st.description, ''), 
			st.created_by, 
			st.created_at,
			st.reversed_by
		FROM public.supplier_transactions st
		JOIN public.suppliers s ON st.supplier_id = s.id
		JOIN public.periods p ON st.period_id = p.id
		WHERE %s
		ORDER BY st.tx_date DESC, st.created_at DESC
	`, whereSQL)

	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		dataQuery += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if filter.Offset > 0 {
		args = append(args, filter.Offset)
		dataQuery += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, MapSQLError(err)
	}
	defer rows.Close()

	transactions := make([]domain.SupplierTransaction, 0)
	for rows.Next() {
		var tx domain.SupplierTransaction
		if err := rows.Scan(
			&tx.ID, &tx.TenantID, &tx.SupplierID, &tx.SupplierName,
			&tx.PeriodID, &tx.PeriodLabel, &tx.InvoiceNo, &tx.CustomerName,
			&tx.DocumentStatus, &tx.TxDate, &tx.Direction, &tx.Amount,
			&tx.Description, &tx.CreatedBy, &tx.CreatedAt, &tx.ReversedBy,
		); err != nil {
			return nil, 0, MapSQLError(err)
		}
		transactions = append(transactions, tx)
	}

	return transactions, totalCount, nil
}

func (r *PostgresSupplierRepository) BatchInsertTransactions(
	ctx context.Context,
	tenantID uuid.UUID,
	transactions []domain.SupplierTransaction,
) (int, error) {
	if len(transactions) == 0 {
		return 0, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("transaction begin failed: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	insertQuery := `
		INSERT INTO public.supplier_transactions (
			id, tenant_id, supplier_id, period_id, invoice_no, customer_name,
			document_status, tx_date, direction, amount, description, created_by, created_at, reversed_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	inserted := 0
	for _, item := range transactions {
		id := item.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		createdAt := item.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		txDate := item.TxDate
		if txDate.IsZero() {
			txDate = time.Now()
		}

		_, err := tx.Exec(ctx, insertQuery,
			id, tenantID, item.SupplierID, item.PeriodID, item.InvoiceNo, item.CustomerName,
			item.DocumentStatus, txDate, item.Direction, item.Amount, item.Description, item.CreatedBy, createdAt, item.ReversedBy,
		)
		if err != nil {
			return 0, fmt.Errorf("batch insert row failed (supplier_id: %s, amount: %s): %w", item.SupplierID, item.Amount, err)
		}
		inserted++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("batch insert commit failed: %w", err)
	}

	return inserted, nil
}

func (r *PostgresSupplierRepository) GetSummary(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) (*domain.SupplierSummary, error) {
	var query strings.Builder
	query.WriteString(`
		SELECT 
			COALESCE(SUM(CASE WHEN direction = 'purchase' THEN amount ELSE 0 END), 0) AS total_purchases,
			COALESCE(SUM(CASE WHEN direction = 'payment' THEN amount ELSE 0 END), 0) AS total_payments,
			COALESCE(SUM(CASE WHEN direction = 'purchase' THEN amount ELSE -amount END), 0) AS net_balance,
			COUNT(id) AS total_rows
		FROM public.supplier_transactions
		WHERE tenant_id = $1
	`)

	args := []interface{}{tenantID}
	if periodID != nil && *periodID != uuid.Nil {
		args = append(args, *periodID)
		query.WriteString(fmt.Sprintf(" AND period_id = $%d", len(args)))
	}

	var summary domain.SupplierSummary
	err := r.pool.QueryRow(ctx, query.String(), args...).Scan(
		&summary.TotalPurchases,
		&summary.TotalPayments,
		&summary.NetBalance,
		&summary.TotalTransactionRows,
	)
	if err != nil {
		return nil, MapSQLError(err)
	}

	// Count active suppliers
	countQuery := `SELECT COUNT(id) FROM public.suppliers WHERE tenant_id = $1 AND is_active = true`
	_ = r.pool.QueryRow(ctx, countQuery, tenantID).Scan(&summary.ActiveSupplierCount)

	return &summary, nil
}
