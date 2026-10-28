package postgres

import (
	"errors"
	"fmt"

	apperrors "coloplus-backend/internal/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	pgUniqueViolation  = "23505"
	pgInvalidText      = "22P02" // malformed uuid, date and similar
	pgCheckViolation   = "23514"
	pgRaiseException   = "P0001"
	pgForeignKeyFailed = "23503"
)

func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// mapErr turns driver errors into the application's sentinels. conflictMsg
// is the worded reason used when a unique constraint is hit.
func mapErr(err error, conflictMsg string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		return apperrors.ErrNotFound
	}
	switch pgCode(err) {
	case pgUniqueViolation:
		return fmt.Errorf("%w: %s", apperrors.ErrConflict, conflictMsg)
	case pgInvalidText:
		return apperrors.ErrNotFound
	case pgForeignKeyFailed:
		return fmt.Errorf("%w: a referenced record does not exist", apperrors.ErrInvalidInput)
	}
	return err
}
