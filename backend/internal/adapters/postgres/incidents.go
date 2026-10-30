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

// IncidentRepository implements ports.IncidentRepository.
type IncidentRepository struct{ db *DB }

func NewIncidentRepository(db *DB) *IncidentRepository { return &IncidentRepository{db: db} }

const incidentColumns = `id::text, number, site_id::text, category, severity, impact, title, description, status,
	started_at, detected_at, reporter_id::text, COALESCE(owner_id::text, ''), next_action, next_update_due,
	restoration_target, restored_at, COALESCE(restored_by::text, ''), verified_at, COALESCE(verified_by::text, ''),
	closed_at, root_cause, published_at, created_at, updated_at`

func scanIncident(row pgx.Row) (domain.Incident, error) {
	var i domain.Incident
	var category, severity, impact, status string
	err := row.Scan(&i.ID, &i.Number, &i.SiteID, &category, &severity, &impact, &i.Title, &i.Description, &status,
		&i.StartedAt, &i.DetectedAt, &i.ReporterID, &i.OwnerID, &i.NextAction, &i.NextUpdateDue,
		&i.RestorationTarget, &i.RestoredAt, &i.RestoredBy, &i.VerifiedAt, &i.VerifiedBy,
		&i.ClosedAt, &i.RootCause, &i.PublishedAt, &i.CreatedAt, &i.UpdatedAt)
	i.Category, i.Severity, i.Impact = domain.IncidentCategory(category), domain.Severity(severity), domain.Impact(impact)
	i.Status = domain.IncidentStatus(status)
	return i, err
}

func (r *IncidentRepository) Create(ctx context.Context, i domain.Incident) (domain.Incident, error) {
	row := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO incidents (site_id, category, severity, impact, title, description, started_at, detected_at, reporter_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING `+incidentColumns,
		i.SiteID, string(i.Category), string(i.Severity), string(i.Impact), i.Title, i.Description,
		i.StartedAt.UTC(), i.DetectedAt.UTC(), i.ReporterID)
	created, err := scanIncident(row)
	return created, mapErr(err, "")
}

func (r *IncidentRepository) GetByID(ctx context.Context, id string, forUpdate bool) (domain.Incident, error) {
	q := `SELECT ` + incidentColumns + ` FROM incidents WHERE id = $1`
	if forUpdate {
		q += ` FOR UPDATE`
	}
	i, err := scanIncident(r.db.q(ctx).QueryRow(ctx, q, id))
	return i, mapErr(err, "")
}

func (r *IncidentRepository) Save(ctx context.Context, i domain.Incident) error {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE incidents SET status = $2, owner_id = NULLIF($3, '')::uuid, next_action = $4, next_update_due = $5,
		        restoration_target = $6, restored_at = $7, restored_by = NULLIF($8, '')::uuid,
		        verified_at = $9, verified_by = NULLIF($10, '')::uuid, closed_at = $11, root_cause = $12,
		        published_at = $13, updated_at = $14
		 WHERE id = $1`,
		i.ID, string(i.Status), i.OwnerID, i.NextAction, i.NextUpdateDue, i.RestorationTarget,
		i.RestoredAt, i.RestoredBy, i.VerifiedAt, i.VerifiedBy, i.ClosedAt, i.RootCause, i.PublishedAt, i.UpdatedAt)
	if err != nil {
		return mapErr(err, "")
	}
	if tag.RowsAffected() == 0 {
		return mapErr(pgx.ErrNoRows, "")
	}
	return nil
}

func (r *IncidentRepository) AddUpdate(ctx context.Context, u domain.IncidentUpdate) (domain.IncidentUpdate, error) {
	err := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO incident_updates (incident_id, author_id, kind, from_status, to_status, body, customer_visible, observed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id::text, created_at`,
		u.IncidentID, u.AuthorID, string(u.Kind), string(u.FromStatus), string(u.ToStatus), u.Body,
		u.CustomerVisible, u.ObservedAt).Scan(&u.ID, &u.CreatedAt)
	return u, mapErr(err, "")
}

func (r *IncidentRepository) Updates(ctx context.Context, incidentID string, customerOnly bool) ([]domain.IncidentUpdate, error) {
	q := `SELECT id::text, incident_id::text, author_id::text, kind, from_status, to_status, body,
	             customer_visible, observed_at, created_at
	      FROM incident_updates WHERE incident_id = $1`
	if customerOnly {
		q += ` AND customer_visible`
	}
	rows, err := r.db.q(ctx).Query(ctx, q+` ORDER BY created_at, id`, incidentID)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()

	out := []domain.IncidentUpdate{}
	for rows.Next() {
		var u domain.IncidentUpdate
		var kind, from, to string
		if err := rows.Scan(&u.ID, &u.IncidentID, &u.AuthorID, &kind, &from, &to, &u.Body,
			&u.CustomerVisible, &u.ObservedAt, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning incident update: %w", err)
		}
		u.Kind, u.FromStatus, u.ToStatus = domain.UpdateKind(kind), domain.IncidentStatus(from), domain.IncidentStatus(to)
		out = append(out, u)
	}
	return out, rows.Err()
}

// incidentWhere builds the shared filter clause for List and Counts.
func incidentWhere(f ports.IncidentFilter) (string, []any) {
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
	if f.OwnerID != "" {
		add("owner_id = ?::uuid", f.OwnerID)
	}
	if len(f.Statuses) > 0 {
		statuses := make([]string, len(f.Statuses))
		for i, s := range f.Statuses {
			statuses[i] = string(s)
		}
		add("status = ANY(?::text[])", statuses)
	}
	if f.Severity != "" {
		add("severity = ?", string(f.Severity))
	}
	if f.OpenOnly {
		where = append(where, "status <> 'closed'")
	}
	if f.PublishedOnly {
		where = append(where, "published_at IS NOT NULL")
	}
	if f.OverdueAt != nil {
		add("(status <> 'closed' AND next_update_due < ?)", f.OverdueAt.UTC())
	}
	if len(where) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(where, " AND "), args
}

func (r *IncidentRepository) List(ctx context.Context, f ports.IncidentFilter) ([]domain.Incident, int, error) {
	clause, args := incidentWhere(f)
	q := r.db.q(ctx)

	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM incidents`+clause, args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err, "")
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := q.Query(ctx,
		`SELECT `+incidentColumns+` FROM incidents`+clause+
			fmt.Sprintf(` ORDER BY started_at DESC, number DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, mapErr(err, "")
	}
	defer rows.Close()

	out := []domain.Incident{}
	for rows.Next() {
		i, err := scanIncident(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning incident: %w", err)
		}
		out = append(out, i)
	}
	return out, total, rows.Err()
}

func (r *IncidentRepository) Counts(ctx context.Context, f ports.IncidentFilter, now time.Time) (ports.IncidentCounts, error) {
	clause, args := incidentWhere(f)
	args = append(args, now.UTC())
	var c ports.IncidentCounts
	err := r.db.q(ctx).QueryRow(ctx, fmt.Sprintf(
		`SELECT count(*) FILTER (WHERE status <> 'closed'),
		        count(*) FILTER (WHERE status IN ('reported', 'acknowledged', 'assigned', 'in_progress')),
		        count(*) FILTER (WHERE status <> 'closed' AND severity = 'critical'),
		        count(*) FILTER (WHERE status <> 'closed' AND next_update_due < $%d)
		 FROM incidents`, len(args))+clause, args...).Scan(&c.Open, &c.Active, &c.Critical, &c.Overdue)
	return c, mapErr(err, "")
}
