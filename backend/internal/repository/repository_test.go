package repository_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/repository"
)

func TestMapSQLError_ErrNoRows(t *testing.T) {
	err := repository.MapSQLError(pgx.ErrNoRows)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestMapSQLError_UniqueViolation(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:    "23505",
		Message: "duplicate key value violates unique constraint",
	}
	err := repository.MapSQLError(pgErr)
	assert.ErrorIs(t, err, domain.ErrDuplicateIdempotencyKey)
}

func TestMapSQLError_PeriodLockedTriggerException(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:    "P0001",
		Message: "cannot insert transaction into a locked period",
	}
	err := repository.MapSQLError(pgErr)
	assert.ErrorIs(t, err, domain.ErrPeriodLocked)
}

func TestMapSQLError_UnhandledStandardError(t *testing.T) {
	stdErr := errors.New("connection reset by peer")
	err := repository.MapSQLError(stdErr)
	assert.Equal(t, stdErr, err)
}

func TestMapSQLError_NilError(t *testing.T) {
	assert.Nil(t, repository.MapSQLError(nil))
}
