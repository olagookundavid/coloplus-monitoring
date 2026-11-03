package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/ports"

	"github.com/jackc/pgx/v5"
)

// MaintenanceRepository implements ports.MaintenanceRepository.
type MaintenanceRepository struct{ db *DB }

func NewMaintenanceRepository(db *DB) *MaintenanceRepository { return &MaintenanceRepository{db: db} }

const maintenanceColumns = `id::text, site_id::text, starts_at, ends_at, reason, status, requested_by::text, requested_at,
	coalesce(decided_by::text, ''), decided_at, decision_note`

func scanWindow(row pgx.Row) (domain.MaintenanceWindow, error) {
	var w domain.MaintenanceWindow
	var status string
	err := row.Scan(&w.ID, &w.SiteID, &w.StartsAt, &w.EndsAt, &w.Reason, &status, &w.RequestedBy, &w.RequestedAt,
		&w.DecidedBy, &w.DecidedAt, &w.DecisionNote)
	w.Status = domain.MaintenanceStatus(status)
	return w, err
}

func (r *MaintenanceRepository) Create(ctx context.Context, w domain.MaintenanceWindow) (domain.MaintenanceWindow, error) {
	out, err := scanWindow(r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO maintenance_windows (site_id, starts_at, ends_at, reason, requested_by, requested_at)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+maintenanceColumns,
		w.SiteID, w.StartsAt.UTC(), w.EndsAt.UTC(), w.Reason, w.RequestedBy, w.RequestedAt.UTC()))
	return out, mapErr(err, "")
}

func (r *MaintenanceRepository) GetByID(ctx context.Context, id string, forUpdate bool) (domain.MaintenanceWindow, error) {
	q := `SELECT ` + maintenanceColumns + ` FROM maintenance_windows WHERE id = $1`
	if forUpdate {
		q += ` FOR UPDATE`
	}
	w, err := scanWindow(r.db.q(ctx).QueryRow(ctx, q, id))
	return w, mapErr(err, "")
}

func (r *MaintenanceRepository) Decide(ctx context.Context, id string, from []domain.MaintenanceStatus, to domain.MaintenanceStatus, by, note string, at time.Time) (bool, error) {
	states := make([]string, len(from))
	for i, f := range from {
		states[i] = string(f)
	}
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE maintenance_windows SET status = $2, decided_by = $3, decided_at = $4, decision_note = $5
		 WHERE id = $1 AND status = ANY($6::text[])`, id, string(to), by, at.UTC(), note, states)
	if err != nil {
		return false, mapErr(err, "")
	}
	return tag.RowsAffected() == 1, nil
}

func (r *MaintenanceRepository) List(ctx context.Context, f ports.MaintenanceFilter) ([]domain.MaintenanceWindow, int, error) {
	var where []string
	var args []any
	add := func(clause string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(clause, "?", fmt.Sprintf("$%d", len(args))))
	}
	if !f.AllSites {
		add("site_id = ANY(?::uuid[])", nonNil(f.SiteIDs))
	}
	if f.SiteID != "" {
		add("site_id = ?::uuid", f.SiteID)
	}
	if f.Status != "" {
		add("status = ?", string(f.Status))
	}
	if f.From != nil {
		add("starts_at >= ?", f.From.UTC())
	}
	if f.To != nil {
		add("starts_at < ?", f.To.UTC())
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.db.q(ctx).QueryRow(ctx, `SELECT count(*) FROM maintenance_windows`+clause, args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err, "")
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.q(ctx).Query(ctx, `SELECT `+maintenanceColumns+` FROM maintenance_windows`+clause+
		fmt.Sprintf(` ORDER BY starts_at DESC, seq DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, mapErr(err, "")
	}
	defer rows.Close()
	out := []domain.MaintenanceWindow{}
	for rows.Next() {
		w, err := scanWindow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning maintenance window: %w", err)
		}
		out = append(out, w)
	}
	return out, total, rows.Err()
}

func (r *MaintenanceRepository) Approved(ctx context.Context, siteIDs []string, from, to time.Time) ([]domain.MaintenanceWindow, error) {
	if len(siteIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.q(ctx).Query(ctx,
		`SELECT `+maintenanceColumns+` FROM maintenance_windows
		 WHERE status = 'approved' AND site_id = ANY($1::uuid[]) AND starts_at < $3 AND ends_at > $2`,
		siteIDs, from.UTC(), to.UTC())
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()
	var out []domain.MaintenanceWindow
	for rows.Next() {
		w, err := scanWindow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning maintenance window: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *MaintenanceRepository) Booked(ctx context.Context, siteID string, from, to time.Time) (time.Duration, error) {
	var secs float64
	err := r.db.q(ctx).QueryRow(ctx,
		`SELECT coalesce(sum(extract(epoch FROM ends_at - starts_at)), 0) FROM maintenance_windows
		 WHERE site_id = $1 AND status IN ('requested', 'approved') AND starts_at >= $2 AND starts_at < $3`,
		siteID, from.UTC(), to.UTC()).Scan(&secs)
	return time.Duration(secs * float64(time.Second)), mapErr(err, "")
}

func (r *MaintenanceRepository) Overlapping(ctx context.Context, siteID string, start, end time.Time) (int, error) {
	var n int
	err := r.db.q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM maintenance_windows
		 WHERE site_id = $1 AND status IN ('requested', 'approved') AND starts_at < $3 AND ends_at > $2`,
		siteID, start.UTC(), end.UTC()).Scan(&n)
	return n, mapErr(err, "")
}
