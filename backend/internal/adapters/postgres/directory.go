package postgres

import (
	"context"
	"fmt"
	"strings"
)

// DirectoryRepository implements ports.Directory: it turns ids into the
// words a person recognises, one query per kind however many ids there are.
type DirectoryRepository struct {
	db *DB
}

func NewDirectoryRepository(db *DB) *DirectoryRepository { return &DirectoryRepository{db: db} }

func (r *DirectoryRepository) UserNames(ctx context.Context, ids []string) (map[string]string, error) {
	return r.labels(ctx, `SELECT id::text, full_name FROM users WHERE id = ANY($1::uuid[])`, ids)
}

func (r *DirectoryRepository) SiteNames(ctx context.Context, ids []string) (map[string]string, error) {
	return r.labels(ctx, `SELECT id::text, name FROM sites WHERE id = ANY($1::uuid[])`, ids)
}

// labelQueries read the label of one kind of record from its id. Each is a
// single statement over a batch of uuids. The label says which record it is
// (the site, the day, the reference number), never the id.
var labelQueries = map[string]string{
	"user": `SELECT id::text, full_name FROM users WHERE id = ANY($1::uuid[])`,
	"site": `SELECT id::text, name FROM sites WHERE id = ANY($1::uuid[])`,
	"daily_report": `SELECT d.id::text, s.name || ' · ' || to_char(d.report_date, 'DD Mon YYYY')
		FROM daily_reports d JOIN sites s ON s.id = d.site_id WHERE d.id = ANY($1::uuid[])`,
	"incident": `SELECT id::text, 'INC-' || lpad(number::text, 6, '0') || ' · ' || title
		FROM incidents WHERE id = ANY($1::uuid[])`,
	"incident_correction": `SELECT c.id::text, 'COR-' || lpad(c.number::text, 6, '0') || ' · ' || s.name
		FROM incident_corrections c JOIN sites s ON s.id = c.site_id WHERE c.id = ANY($1::uuid[])`,
	"dispute": `SELECT id::text, 'DSP-' || lpad(number::text, 6, '0') || ' · ' || title
		FROM disputes WHERE id = ANY($1::uuid[])`,
	"ticket": `SELECT id::text, 'TKT-' || lpad(number::text, 6, '0') || ' · ' || subject
		FROM tickets WHERE id = ANY($1::uuid[])`,
	"acceptance": `SELECT a.id::text, s.name FROM acceptances a JOIN sites s ON s.id = a.site_id
		WHERE a.id = ANY($1::uuid[])`,
	"fuel_entry": `SELECT f.id::text, s.name || ' · ' || f.kind || ' ' || trim(trailing '.' FROM trim(trailing '0' FROM f.litres::text)) || ' L'
		FROM fuel_entries f JOIN sites s ON s.id = f.site_id WHERE f.id = ANY($1::uuid[])`,
	"maintenance_window": `SELECT m.id::text, s.name || ' · ' || m.reason
		FROM maintenance_windows m JOIN sites s ON s.id = m.site_id WHERE m.id = ANY($1::uuid[])`,
	"attachment": `SELECT id::text, filename FROM attachments WHERE id = ANY($1::uuid[])`,
	"calculation_run": `SELECT id::text, CASE WHEN label <> '' THEN label ELSE kind || ' ' || period_from || ' to ' || period_to END
		FROM calculation_runs WHERE id = ANY($1::uuid[])`,
	"notification": `SELECT id::text, kind FROM notifications WHERE id = ANY($1::uuid[])`,
}

// EntityLabels names records of one audit entity type. Types whose ids are
// not database ids (a month, an export) are named by the service, and types
// not listed here simply come back empty.
func (r *DirectoryRepository) EntityLabels(ctx context.Context, entityType string, ids []string) (map[string]string, error) {
	q, ok := labelQueries[entityType]
	if !ok {
		return map[string]string{}, nil
	}
	return r.labels(ctx, q, ids)
}

func (r *DirectoryRepository) labels(ctx context.Context, query string, ids []string) (map[string]string, error) {
	out := map[string]string{}
	ids = uuidsOnly(ids)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.db.q(ctx).Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("looking up names: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, label string
		if err := rows.Scan(&id, &label); err != nil {
			return nil, fmt.Errorf("scanning name: %w", err)
		}
		out[id] = strings.TrimSpace(label)
	}
	return out, rows.Err()
}

// uuidsOnly drops anything that is not shaped like a uuid, so one odd value
// in an audit row (a month, an export key) cannot fail the whole lookup.
func uuidsOnly(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if len(id) == 36 && strings.Count(id, "-") == 4 {
			out = append(out, id)
		}
	}
	return out
}
