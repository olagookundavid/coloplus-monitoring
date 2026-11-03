package postgres

import (
	"context"
	"fmt"
	"time"

	"coloplus-backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

// PeriodClosureRepository implements ports.PeriodClosureRepository.
type PeriodClosureRepository struct{ db *DB }

func NewPeriodClosureRepository(db *DB) *PeriodClosureRepository {
	return &PeriodClosureRepository{db: db}
}

const closureColumns = `id::text, month, closed_by::text, closed_at, run_id::text, coalesce(reopened_by::text, ''), reopened_at, reopen_reason`

func scanClosure(row pgx.Row) (domain.PeriodClosure, error) {
	var c domain.PeriodClosure
	err := row.Scan(&c.ID, &c.Month, &c.ClosedBy, &c.ClosedAt, &c.RunID, &c.ReopenedBy, &c.ReopenedAt, &c.ReopenReason)
	c.Month = time.Date(c.Month.Year(), c.Month.Month(), 1, 0, 0, 0, 0, time.UTC)
	return c, err
}

func (r *PeriodClosureRepository) Close(ctx context.Context, c domain.PeriodClosure) (domain.PeriodClosure, error) {
	out, err := scanClosure(r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO period_closures (month, closed_by, closed_at, run_id) VALUES ($1, $2, $3, $4) RETURNING `+closureColumns,
		c.Month, c.ClosedBy, c.ClosedAt.UTC(), c.RunID))
	return out, mapErr(err, "that month is already closed")
}

func (r *PeriodClosureRepository) Reopen(ctx context.Context, month time.Time, by, reason string, at time.Time) (domain.PeriodClosure, error) {
	out, err := scanClosure(r.db.q(ctx).QueryRow(ctx,
		`UPDATE period_closures SET reopened_by = $2, reopened_at = $3, reopen_reason = $4
		 WHERE month = $1 AND reopened_at IS NULL RETURNING `+closureColumns, month, by, at.UTC(), reason))
	if err != nil {
		return domain.PeriodClosure{}, mapErr(err, "")
	}
	return out, nil
}

func (r *PeriodClosureRepository) IsClosed(ctx context.Context, month time.Time) (bool, error) {
	var closed bool
	err := r.db.q(ctx).QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM period_closures WHERE month = $1 AND reopened_at IS NULL)`, domain.MonthOf(month)).Scan(&closed)
	return closed, mapErr(err, "")
}

func (r *PeriodClosureRepository) List(ctx context.Context, limit int) ([]domain.PeriodClosure, error) {
	rows, err := r.db.q(ctx).Query(ctx, `SELECT `+closureColumns+` FROM period_closures ORDER BY seq DESC LIMIT $1`, limit)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()
	out := []domain.PeriodClosure{}
	for rows.Next() {
		c, err := scanClosure(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning closure: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
