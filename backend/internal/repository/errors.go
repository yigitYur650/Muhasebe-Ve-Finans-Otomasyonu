package repository

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"deftersystem/backend/internal/domain"
)

// MapSQLError translates PostgreSQL database errors into standardized domain errors.
func MapSQLError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			if pgErr.ConstraintName == "periods_tenant_id_label_key" || strings.Contains(pgErr.Message, "periods_tenant_id_label_key") {
				return domain.ErrPeriodAlreadyExists
			}
			return domain.ErrDuplicateIdempotencyKey
		case "P0001": // raise_exception (trigger preventing modifications on locked periods or append-only rule)
			msg := strings.ToLower(pgErr.Message)
			if strings.Contains(msg, "locked") || strings.Contains(msg, "kilitli") {
				return domain.ErrPeriodLocked
			}
			return err
		}
	}

	return err
}
