package postgres

import (
	"context"
	"fmt"
	"time"

	"coloplus-backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

// CalculationRunRepository implements ports.CalculationRunRepository.
type CalculationRunRepository struct{ db *DB }

func NewCalculationRunRepository(db *DB) *CalculationRunRepository {
	return &CalculationRunRepository{db: db}
}

const runColumns = `id::text, kind, period_from, period_to, region, label, formula, formula_version, site_count, created_by::text, created_at`

func scanRun(row pgx.Row) (domain.CalculationRun, error) {
	var r domain.CalculationRun
	err := row.Scan(&r.ID, &r.Kind, &r.PeriodFrom, &r.PeriodTo, &r.Region, &r.Label, &r.Formula, &r.FormulaVersion,
		&r.SiteCount, &r.CreatedBy, &r.CreatedAt)
	// DATE columns arrive as midnight in some zone; keep the calendar day as UTC.
	r.PeriodFrom = time.Date(r.PeriodFrom.Year(), r.PeriodFrom.Month(), r.PeriodFrom.Day(), 0, 0, 0, 0, time.UTC)
	r.PeriodTo = time.Date(r.PeriodTo.Year(), r.PeriodTo.Month(), r.PeriodTo.Day(), 0, 0, 0, 0, time.UTC)
	return r, err
}

func (r *CalculationRunRepository) Create(ctx context.Context, run domain.CalculationRun) (domain.CalculationRun, error) {
	q := r.db.q(ctx)
	created, err := scanRun(q.QueryRow(ctx,
		`INSERT INTO calculation_runs (kind, period_from, period_to, region, label, formula, formula_version, site_count, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING `+runColumns,
		run.Kind, run.PeriodFrom, run.PeriodTo, run.Region, run.Label, run.Formula, run.FormulaVersion,
		len(run.Rows), run.CreatedBy, run.CreatedAt.UTC()))
	if err != nil {
		return domain.CalculationRun{}, mapErr(err, "")
	}
	rows := make([][]any, len(run.Rows))
	for i, x := range run.Rows {
		rows[i] = []any{created.ID, x.SiteID, x.AirtelCode, x.SiteName, x.EligibleMinutes, x.ServedMinutes, x.OutageMinutes,
			x.UnknownMinutes, x.AssumedMinutes, x.DaysInPeriod, x.DaysReported, x.ReportedAvailability, x.ConfirmedServedShare,
			x.Coverage, x.GeneratorHours, x.IncidentsStarted, x.Fingerprint}
	}
	if _, err := q.CopyFrom(ctx, pgx.Identifier{"calculation_run_rows"}, runRowColumns, pgx.CopyFromRows(rows)); err != nil {
		return domain.CalculationRun{}, mapErr(err, "")
	}
	created.Rows = run.Rows
	return created, nil
}

var runRowColumns = []string{"run_id", "site_id", "airtel_code", "site_name", "eligible_minutes", "served_minutes", "outage_minutes",
	"unknown_minutes", "assumed_minutes", "days_in_period", "days_reported", "reported_availability", "confirmed_served_share",
	"coverage", "generator_hours", "incidents_started", "fingerprint"}

func (r *CalculationRunRepository) Get(ctx context.Context, id string) (domain.CalculationRun, error) {
	q := r.db.q(ctx)
	run, err := scanRun(q.QueryRow(ctx, `SELECT `+runColumns+` FROM calculation_runs WHERE id = $1`, id))
	if err != nil {
		return domain.CalculationRun{}, mapErr(err, "")
	}
	rows, err := q.Query(ctx,
		`SELECT site_id::text, airtel_code, site_name, eligible_minutes, served_minutes, outage_minutes, unknown_minutes,
		        assumed_minutes, days_in_period, days_reported, reported_availability, confirmed_served_share, coverage,
		        generator_hours, incidents_started, fingerprint
		 FROM calculation_run_rows WHERE run_id = $1 ORDER BY airtel_code`, id)
	if err != nil {
		return domain.CalculationRun{}, mapErr(err, "")
	}
	defer rows.Close()
	for rows.Next() {
		var x domain.RunRow
		if err := rows.Scan(&x.SiteID, &x.AirtelCode, &x.SiteName, &x.EligibleMinutes, &x.ServedMinutes, &x.OutageMinutes,
			&x.UnknownMinutes, &x.AssumedMinutes, &x.DaysInPeriod, &x.DaysReported, &x.ReportedAvailability,
			&x.ConfirmedServedShare, &x.Coverage, &x.GeneratorHours, &x.IncidentsStarted, &x.Fingerprint); err != nil {
			return domain.CalculationRun{}, fmt.Errorf("scanning run row: %w", err)
		}
		run.Rows = append(run.Rows, x)
	}
	return run, rows.Err()
}

func (r *CalculationRunRepository) List(ctx context.Context, limit, offset int) ([]domain.CalculationRun, int, error) {
	q := r.db.q(ctx)
	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM calculation_runs`).Scan(&total); err != nil {
		return nil, 0, mapErr(err, "")
	}
	rows, err := q.Query(ctx, `SELECT `+runColumns+` FROM calculation_runs ORDER BY seq DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, mapErr(err, "")
	}
	defer rows.Close()
	out := []domain.CalculationRun{}
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning run: %w", err)
		}
		out = append(out, run)
	}
	return out, total, rows.Err()
}
