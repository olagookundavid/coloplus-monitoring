package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"

	"github.com/jackc/pgx/v5"
)

// ReportRepository implements ports.ReportRepository.
type ReportRepository struct{ db *DB }

func NewReportRepository(db *DB) *ReportRepository { return &ReportRepository{db: db} }

const reportColumns = `id::text, site_id::text, report_date, revision, status, author_id::text, notes,
	submitted_at, reviewed_at, COALESCE(reviewer_id::text, ''), review_comment,
	COALESCE(supersedes_id::text, ''), correction_reason, row_version, created_at, updated_at`

func scanReport(row pgx.Row) (domain.DailyReport, error) {
	var r domain.DailyReport
	var status string
	err := row.Scan(&r.ID, &r.SiteID, &r.Date, &r.Revision, &status, &r.AuthorID, &r.Notes,
		&r.SubmittedAt, &r.ReviewedAt, &r.ReviewerID, &r.ReviewComment,
		&r.SupersedesID, &r.CorrectionReason, &r.RowVersion, &r.CreatedAt, &r.UpdatedAt)
	r.Status = domain.ReportStatus(status)
	return r, err
}

const staleMsg = "someone else changed this report while you were editing; reload it and try again"

func (r *ReportRepository) Create(ctx context.Context, rep domain.DailyReport) (domain.DailyReport, error) {
	row := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO daily_reports (site_id, report_date, revision, status, author_id, notes, supersedes_id, correction_reason)
		 VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::uuid, $8) RETURNING `+reportColumns,
		rep.SiteID, rep.Date, rep.Revision, string(rep.Status), rep.AuthorID, rep.Notes, rep.SupersedesID, rep.CorrectionReason)
	created, err := scanReport(row)
	if err != nil {
		return domain.DailyReport{}, mapErr(err, "a report for this site and day is already in progress or approved")
	}
	if err := r.insertIntervals(ctx, created.ID, rep.Intervals); err != nil {
		return domain.DailyReport{}, err
	}
	return r.GetByID(ctx, created.ID, false)
}

func (r *ReportRepository) insertIntervals(ctx context.Context, reportID string, ivs []domain.PowerInterval) error {
	if len(ivs) == 0 {
		return nil
	}
	var sb strings.Builder
	args := make([]any, 0, len(ivs)*5+1)
	args = append(args, reportID)
	sb.WriteString(`INSERT INTO power_intervals (report_id, start_at, end_at, state, source, basis) VALUES `)
	for i, iv := range ivs {
		if i > 0 {
			sb.WriteString(", ")
		}
		n := len(args)
		fmt.Fprintf(&sb, "($1, $%d, $%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4, n+5)
		args = append(args, iv.Start.UTC(), iv.End.UTC(), string(iv.State), string(iv.Source), string(iv.Basis))
	}
	if _, err := r.db.q(ctx).Exec(ctx, sb.String(), args...); err != nil {
		return mapErr(err, "")
	}
	return nil
}

func (r *ReportRepository) GetByID(ctx context.Context, id string, forUpdate bool) (domain.DailyReport, error) {
	q := `SELECT ` + reportColumns + ` FROM daily_reports WHERE id = $1`
	if forUpdate {
		q += ` FOR UPDATE`
	}
	rep, err := scanReport(r.db.q(ctx).QueryRow(ctx, q, id))
	if err != nil {
		return domain.DailyReport{}, mapErr(err, "")
	}
	byReport, err := r.loadIntervals(ctx, []string{rep.ID})
	if err != nil {
		return domain.DailyReport{}, err
	}
	rep.Intervals = byReport[rep.ID]
	return rep, nil
}

func (r *ReportRepository) loadIntervals(ctx context.Context, reportIDs []string) (map[string][]domain.PowerInterval, error) {
	out := make(map[string][]domain.PowerInterval, len(reportIDs))
	if len(reportIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.q(ctx).Query(ctx,
		`SELECT id::text, report_id::text, start_at, end_at, state, source, basis
		 FROM power_intervals WHERE report_id = ANY($1::uuid[]) ORDER BY start_at, end_at`, reportIDs)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()

	for rows.Next() {
		var iv domain.PowerInterval
		var reportID, state, source, basis string
		if err := rows.Scan(&iv.ID, &reportID, &iv.Start, &iv.End, &state, &source, &basis); err != nil {
			return nil, fmt.Errorf("scanning interval: %w", err)
		}
		iv.State, iv.Source, iv.Basis = domain.IntervalState(state), domain.PowerSource(source), domain.Basis(basis)
		out[reportID] = append(out[reportID], iv)
	}
	return out, rows.Err()
}

func (r *ReportRepository) attachIntervals(ctx context.Context, reps []domain.DailyReport) error {
	ids := make([]string, len(reps))
	for i, rep := range reps {
		ids[i] = rep.ID
	}
	byReport, err := r.loadIntervals(ctx, ids)
	if err != nil {
		return err
	}
	for i := range reps {
		reps[i].Intervals = byReport[reps[i].ID]
	}
	return nil
}

func (r *ReportRepository) SaveDraft(ctx context.Context, rep domain.DailyReport, rowVersion int) (domain.DailyReport, error) {
	q := r.db.q(ctx)
	tag, err := q.Exec(ctx,
		`UPDATE daily_reports
		 SET notes = $2, status = 'draft', row_version = row_version + 1, updated_at = now()
		 WHERE id = $1 AND row_version = $3 AND status IN ('draft', 'rejected')`,
		rep.ID, rep.Notes, rowVersion)
	if err != nil {
		return domain.DailyReport{}, mapErr(err, "")
	}
	if tag.RowsAffected() == 0 {
		return domain.DailyReport{}, fmt.Errorf("%w: %s", apperrors.ErrConflict, staleMsg)
	}
	if _, err := q.Exec(ctx, `DELETE FROM power_intervals WHERE report_id = $1`, rep.ID); err != nil {
		return domain.DailyReport{}, mapErr(err, "")
	}
	if err := r.insertIntervals(ctx, rep.ID, rep.Intervals); err != nil {
		return domain.DailyReport{}, err
	}
	return r.GetByID(ctx, rep.ID, false)
}

func (r *ReportRepository) MarkSubmitted(ctx context.Context, id string, at time.Time) error {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE daily_reports SET status = 'submitted', submitted_at = $2, row_version = row_version + 1, updated_at = now()
		 WHERE id = $1 AND status IN ('draft', 'rejected')`, id, at)
	if err != nil {
		return mapErr(err, "")
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s", apperrors.ErrConflict, staleMsg)
	}
	return nil
}

func (r *ReportRepository) MarkReviewed(ctx context.Context, id string, status domain.ReportStatus, reviewerID, comment string, at time.Time) error {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE daily_reports
		 SET status = $2, reviewer_id = $3, review_comment = $4, reviewed_at = $5, row_version = row_version + 1, updated_at = now()
		 WHERE id = $1 AND status = 'submitted'`, id, string(status), reviewerID, comment, at)
	if err != nil {
		return mapErr(err, "another report for this site and day is already approved")
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s", apperrors.ErrConflict, staleMsg)
	}
	return nil
}

func (r *ReportRepository) MarkSuperseded(ctx context.Context, id string) error {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE daily_reports SET status = 'superseded', updated_at = now() WHERE id = $1 AND status = 'approved'`, id)
	if err != nil {
		return mapErr(err, "")
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: the report being corrected is no longer the approved one", apperrors.ErrConflict)
	}
	return nil
}

func (r *ReportRepository) FindApproved(ctx context.Context, siteID string, date time.Time) (domain.DailyReport, error) {
	rep, err := scanReport(r.db.q(ctx).QueryRow(ctx,
		`SELECT `+reportColumns+` FROM daily_reports WHERE site_id = $1 AND report_date = $2 AND status = 'approved'`, siteID, date))
	return rep, mapErr(err, "")
}

func (r *ReportRepository) NextRevision(ctx context.Context, siteID string, date time.Time) (int, error) {
	var n int
	err := r.db.q(ctx).QueryRow(ctx,
		`SELECT COALESCE(MAX(revision), 0) + 1 FROM daily_reports WHERE site_id = $1 AND report_date = $2`, siteID, date).Scan(&n)
	return n, mapErr(err, "")
}

func (r *ReportRepository) List(ctx context.Context, f ports.ReportFilter) ([]domain.DailyReport, int, error) {
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
	if f.From != nil {
		add("report_date >= ?", *f.From)
	}
	if f.To != nil {
		add("report_date <= ?", *f.To)
	}
	if len(f.Statuses) > 0 {
		statuses := make([]string, len(f.Statuses))
		for i, s := range f.Statuses {
			statuses[i] = string(s)
		}
		add("status = ANY(?::text[])", statuses)
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	q := r.db.q(ctx)
	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM daily_reports`+clause, args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err, "")
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := q.Query(ctx,
		`SELECT `+reportColumns+` FROM daily_reports`+clause+
			fmt.Sprintf(` ORDER BY report_date DESC, revision DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, mapErr(err, "")
	}
	defer rows.Close()

	reports := []domain.DailyReport{}
	for rows.Next() {
		rep, err := scanReport(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning report: %w", err)
		}
		reports = append(reports, rep)
	}
	return reports, total, rows.Err()
}

func (r *ReportRepository) collect(ctx context.Context, rows pgx.Rows) ([]domain.DailyReport, error) {
	defer rows.Close()
	var reps []domain.DailyReport
	for rows.Next() {
		rep, err := scanReport(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning report: %w", err)
		}
		reps = append(reps, rep)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	return reps, r.attachIntervals(ctx, reps)
}

func (r *ReportRepository) ApprovedOnDate(ctx context.Context, siteIDs []string, date time.Time) ([]domain.DailyReport, error) {
	rows, err := r.db.q(ctx).Query(ctx,
		`SELECT `+reportColumns+` FROM daily_reports
		 WHERE site_id = ANY($1::uuid[]) AND report_date = $2 AND status = 'approved'`, nonNil(siteIDs), date)
	if err != nil {
		return nil, mapErr(err, "")
	}
	return r.collect(ctx, rows)
}

func (r *ReportRepository) LatestApproved(ctx context.Context, siteIDs []string) (map[string]domain.DailyReport, error) {
	rows, err := r.db.q(ctx).Query(ctx,
		`SELECT DISTINCT ON (site_id) `+reportColumns+` FROM daily_reports
		 WHERE site_id = ANY($1::uuid[]) AND status = 'approved'
		 ORDER BY site_id, report_date DESC`, nonNil(siteIDs))
	if err != nil {
		return nil, mapErr(err, "")
	}
	reps, err := r.collect(ctx, rows)
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.DailyReport, len(reps))
	for _, rep := range reps {
		out[rep.SiteID] = rep
	}
	return out, nil
}
