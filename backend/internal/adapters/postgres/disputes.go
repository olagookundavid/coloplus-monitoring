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

// DisputeRepository implements ports.DisputeRepository.
type DisputeRepository struct{ db *DB }

func NewDisputeRepository(db *DB) *DisputeRepository { return &DisputeRepository{db: db} }

const disputeColumns = `id::text, number, site_id::text, COALESCE(report_id::text, ''), COALESCE(interval_id::text, ''),
	metric, period_from, period_to, category, title, description, status, raised_by::text,
	COALESCE(owner_id::text, ''), outcome, resolution, COALESCE(resolution_report_id::text, ''),
	resolved_at, COALESCE(resolved_by::text, ''), acknowledged_at, COALESCE(acknowledged_by::text, ''),
	reopen_count, created_at, updated_at`

func scanDispute(row pgx.Row) (domain.Dispute, error) {
	var d domain.Dispute
	var metric, category, status, outcome string
	err := row.Scan(&d.ID, &d.Number, &d.SiteID, &d.ReportID, &d.IntervalID, &metric, &d.PeriodFrom, &d.PeriodTo,
		&category, &d.Title, &d.Description, &status, &d.RaisedBy, &d.OwnerID, &outcome, &d.Resolution,
		&d.ResolutionReportID, &d.ResolvedAt, &d.ResolvedBy, &d.AcknowledgedAt, &d.AcknowledgedBy,
		&d.ReopenCount, &d.CreatedAt, &d.UpdatedAt)
	d.Metric, d.Category = domain.DisputeMetric(metric), domain.DisputeCategory(category)
	d.Status, d.Outcome = domain.DisputeStatus(status), domain.DisputeOutcome(outcome)
	return d, err
}

func (r *DisputeRepository) Create(ctx context.Context, d domain.Dispute) (domain.Dispute, error) {
	row := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO disputes (site_id, report_id, interval_id, metric, period_from, period_to, category, title,
		                       description, raised_by, created_at, updated_at)
		 VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $11)
		 RETURNING `+disputeColumns,
		d.SiteID, d.ReportID, d.IntervalID, string(d.Metric), d.PeriodFrom, d.PeriodTo, string(d.Category),
		d.Title, d.Description, d.RaisedBy, d.CreatedAt.UTC())
	created, err := scanDispute(row)
	return created, mapErr(err, "this is already being disputed; add to that dispute instead")
}

func (r *DisputeRepository) GetByID(ctx context.Context, id string, forUpdate bool) (domain.Dispute, error) {
	q := `SELECT ` + disputeColumns + ` FROM disputes WHERE id = $1`
	if forUpdate {
		q += ` FOR UPDATE`
	}
	d, err := scanDispute(r.db.q(ctx).QueryRow(ctx, q, id))
	return d, mapErr(err, "")
}

func (r *DisputeRepository) Save(ctx context.Context, d domain.Dispute) error {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE disputes SET status = $2, owner_id = NULLIF($3, '')::uuid, outcome = $4, resolution = $5,
		        resolution_report_id = NULLIF($6, '')::uuid, resolved_at = $7, resolved_by = NULLIF($8, '')::uuid,
		        acknowledged_at = $9, acknowledged_by = NULLIF($10, '')::uuid, reopen_count = $11, updated_at = $12
		 WHERE id = $1`,
		d.ID, string(d.Status), d.OwnerID, string(d.Outcome), d.Resolution, d.ResolutionReportID,
		d.ResolvedAt, d.ResolvedBy, d.AcknowledgedAt, d.AcknowledgedBy, d.ReopenCount, d.UpdatedAt.UTC())
	if err != nil {
		return mapErr(err, "")
	}
	if tag.RowsAffected() == 0 {
		return mapErr(pgx.ErrNoRows, "")
	}
	return nil
}

func (r *DisputeRepository) AddEvent(ctx context.Context, e domain.DisputeEvent) (domain.DisputeEvent, error) {
	err := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO dispute_events (dispute_id, author_id, party, kind, from_status, to_status, body, internal, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id::text`,
		e.DisputeID, e.AuthorID, string(e.Party), string(e.Kind), string(e.FromStatus), string(e.ToStatus),
		e.Body, e.Internal, e.CreatedAt.UTC()).Scan(&e.ID)
	return e, mapErr(err, "")
}

func (r *DisputeRepository) Events(ctx context.Context, disputeID string, publicOnly bool) ([]domain.DisputeEvent, error) {
	q := `SELECT id::text, dispute_id::text, author_id::text, party, kind, from_status, to_status, body, internal, created_at
	      FROM dispute_events WHERE dispute_id = $1`
	if publicOnly {
		q += ` AND NOT internal`
	}
	rows, err := r.db.q(ctx).Query(ctx, q+` ORDER BY seq`, disputeID)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()

	out := []domain.DisputeEvent{}
	for rows.Next() {
		var e domain.DisputeEvent
		var party, kind, from, to string
		if err := rows.Scan(&e.ID, &e.DisputeID, &e.AuthorID, &party, &kind, &from, &to, &e.Body, &e.Internal, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning dispute event: %w", err)
		}
		e.Party, e.Kind = domain.DisputeParty(party), domain.DisputeEventKind(kind)
		e.FromStatus, e.ToStatus = domain.DisputeStatus(from), domain.DisputeStatus(to)
		out = append(out, e)
	}
	return out, rows.Err()
}

func disputeWhere(f ports.DisputeFilter) (string, []any) {
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
	if f.ReportID != "" {
		add("report_id = ?::uuid", f.ReportID)
	}
	if len(f.Statuses) > 0 {
		statuses := make([]string, len(f.Statuses))
		for i, s := range f.Statuses {
			statuses[i] = string(s)
		}
		add("status = ANY(?::text[])", statuses)
	}
	if f.OpenOnly {
		where = append(where, "status <> 'closed'")
	}
	if len(where) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(where, " AND "), args
}

func (r *DisputeRepository) List(ctx context.Context, f ports.DisputeFilter) ([]domain.Dispute, int, error) {
	clause, args := disputeWhere(f)
	q := r.db.q(ctx)

	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM disputes`+clause, args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err, "")
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := q.Query(ctx,
		`SELECT `+disputeColumns+` FROM disputes`+clause+
			fmt.Sprintf(` ORDER BY created_at DESC, number DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, mapErr(err, "")
	}
	defer rows.Close()

	out := []domain.Dispute{}
	for rows.Next() {
		d, err := scanDispute(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning dispute: %w", err)
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

func (r *DisputeRepository) Counts(ctx context.Context, f ports.DisputeFilter) (ports.DisputeCounts, error) {
	clause, args := disputeWhere(f)
	var c ports.DisputeCounts
	err := r.db.q(ctx).QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE status <> 'closed'),
		        count(*) FILTER (WHERE status IN ('open', 'under_review', 'agreed_correction')),
		        count(*) FILTER (WHERE status IN ('awaiting_evidence', 'resolved'))
		 FROM disputes`+clause, args...).Scan(&c.Open, &c.WithColoplus, &c.WithAirtel)
	return c, mapErr(err, "")
}

func (r *DisputeRepository) OpenByReport(ctx context.Context, reportIDs []string) (map[string]int, error) {
	out := map[string]int{}
	if len(reportIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.q(ctx).Query(ctx,
		`SELECT report_id::text, count(*) FROM disputes
		 WHERE report_id = ANY($1::uuid[]) AND status <> 'closed' GROUP BY report_id`, reportIDs)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, fmt.Errorf("scanning dispute counts: %w", err)
		}
		out[id] = n
	}
	return out, rows.Err()
}

func (r *DisputeRepository) ByDay(ctx context.Context, siteIDs []string, from, to time.Time) (map[string]ports.DisputeDay, error) {
	out := map[string]ports.DisputeDay{}
	if len(siteIDs) == 0 {
		return out, nil
	}
	add := func(siteID string, date time.Time, closed bool) {
		k := ports.DayKey(siteID, date)
		d := out[k]
		d.Total++
		if !closed {
			d.Open++
		}
		out[k] = d
	}

	// Against a report: any revision of that day counts, so a correction does
	// not hide the dispute that led to it.
	rows, err := r.db.q(ctx).Query(ctx,
		`SELECT r.site_id::text, r.report_date, d.status = 'closed'
		 FROM disputes d JOIN daily_reports r ON r.id = d.report_id
		 WHERE r.site_id = ANY($1::uuid[]) AND r.report_date BETWEEN $2 AND $3`, siteIDs, from, to)
	if err != nil {
		return nil, mapErr(err, "")
	}
	for rows.Next() {
		var siteID string
		var date time.Time
		var closed bool
		if err := rows.Scan(&siteID, &date, &closed); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning disputes by day: %w", err)
		}
		add(siteID, date, closed)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// By period alone: counted on every day the period covers.
	rows, err = r.db.q(ctx).Query(ctx,
		`SELECT site_id::text, period_from, period_to, status = 'closed'
		 FROM disputes
		 WHERE report_id IS NULL AND period_from IS NOT NULL AND site_id = ANY($1::uuid[])
		   AND period_from <= $3 AND period_to >= $2`, siteIDs, from, to)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()
	for rows.Next() {
		var siteID string
		var pf, pt time.Time
		var closed bool
		if err := rows.Scan(&siteID, &pf, &pt, &closed); err != nil {
			return nil, fmt.Errorf("scanning disputes by period: %w", err)
		}
		if pf.Before(from) {
			pf = from
		}
		if pt.After(to) {
			pt = to
		}
		for d := pf; !d.After(pt); d = d.AddDate(0, 0, 1) {
			add(siteID, d, closed)
		}
	}
	return out, rows.Err()
}
