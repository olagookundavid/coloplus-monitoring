package postgres

import (
	"context"
	"fmt"
	"strings"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/ports"

	"github.com/jackc/pgx/v5"
)

// IncidentCorrectionRepository implements ports.IncidentCorrectionRepository.
type IncidentCorrectionRepository struct{ db *DB }

func NewIncidentCorrectionRepository(db *DB) *IncidentCorrectionRepository {
	return &IncidentCorrectionRepository{db: db}
}

const correctionColumns = `id::text, number, incident_id::text, site_id::text, field, original_value,
	corrected_value, reason, status, requested_by::text, COALESCE(reviewed_by::text, ''), reviewed_at,
	review_note, created_at, updated_at`

func scanCorrection(row pgx.Row) (domain.IncidentCorrection, error) {
	var c domain.IncidentCorrection
	var field, status string
	err := row.Scan(&c.ID, &c.Number, &c.IncidentID, &c.SiteID, &field, &c.OriginalValue, &c.CorrectedValue,
		&c.Reason, &status, &c.RequestedBy, &c.ReviewedBy, &c.ReviewedAt, &c.ReviewNote, &c.CreatedAt, &c.UpdatedAt)
	c.Field, c.Status = domain.CorrectionField(field), domain.CorrectionStatus(status)
	return c, err
}

func (r *IncidentCorrectionRepository) Create(ctx context.Context, c domain.IncidentCorrection) (domain.IncidentCorrection, error) {
	row := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO incident_corrections (incident_id, site_id, field, original_value, corrected_value, reason, requested_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+correctionColumns,
		c.IncidentID, c.SiteID, string(c.Field), c.OriginalValue.UTC(), c.CorrectedValue.UTC(), c.Reason, c.RequestedBy)
	created, err := scanCorrection(row)
	return created, mapErr(err, "a correction for this field is already awaiting review")
}

func (r *IncidentCorrectionRepository) GetByID(ctx context.Context, id string, forUpdate bool) (domain.IncidentCorrection, error) {
	q := `SELECT ` + correctionColumns + ` FROM incident_corrections WHERE id = $1`
	if forUpdate {
		q += ` FOR UPDATE`
	}
	c, err := scanCorrection(r.db.q(ctx).QueryRow(ctx, q, id))
	return c, mapErr(err, "")
}

func (r *IncidentCorrectionRepository) Save(ctx context.Context, c domain.IncidentCorrection) error {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE incident_corrections SET status = $2, reviewed_by = NULLIF($3, '')::uuid, reviewed_at = $4,
		        review_note = $5, updated_at = $6
		 WHERE id = $1`,
		c.ID, string(c.Status), c.ReviewedBy, c.ReviewedAt, c.ReviewNote, c.UpdatedAt.UTC())
	if err != nil {
		return mapErr(err, "")
	}
	if tag.RowsAffected() == 0 {
		return mapErr(pgx.ErrNoRows, "")
	}
	return nil
}

func correctionWhere(f ports.IncidentCorrectionFilter) (string, []any) {
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
	if f.IncidentID != "" {
		add("incident_id = ?::uuid", f.IncidentID)
	}
	if len(f.Statuses) > 0 {
		statuses := make([]string, len(f.Statuses))
		for i, s := range f.Statuses {
			statuses[i] = string(s)
		}
		add("status = ANY(?::text[])", statuses)
	}
	if len(where) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(where, " AND "), args
}

func (r *IncidentCorrectionRepository) List(ctx context.Context, f ports.IncidentCorrectionFilter) ([]domain.IncidentCorrection, int, error) {
	clause, args := correctionWhere(f)
	q := r.db.q(ctx)

	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM incident_corrections`+clause, args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err, "")
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := q.Query(ctx,
		`SELECT `+correctionColumns+` FROM incident_corrections`+clause+
			fmt.Sprintf(` ORDER BY created_at DESC, number DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, mapErr(err, "")
	}
	defer rows.Close()

	out := []domain.IncidentCorrection{}
	for rows.Next() {
		c, err := scanCorrection(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning incident correction: %w", err)
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}
