package postgres

import (
	"context"
	"fmt"
	"time"

	"coloplus-backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

// FuelRepository implements ports.FuelRepository.
type FuelRepository struct{ db *DB }

func NewFuelRepository(db *DB) *FuelRepository { return &FuelRepository{db: db} }

const fuelColumns = `id::text, site_id::text, kind, litres::float8, occurred_at, note, recorded_by::text, client_key,
	created_at, voided_at, COALESCE(voided_by::text, ''), void_reason`

func scanFuel(row pgx.Row) (domain.FuelEntry, error) {
	var e domain.FuelEntry
	var kind string
	err := row.Scan(&e.ID, &e.SiteID, &kind, &e.Litres, &e.OccurredAt, &e.Note, &e.RecordedBy, &e.ClientKey,
		&e.CreatedAt, &e.VoidedAt, &e.VoidedBy, &e.VoidReason)
	e.Kind = domain.FuelKind(kind)
	return e, err
}

func (r *FuelRepository) Create(ctx context.Context, e domain.FuelEntry) (domain.FuelEntry, error) {
	row := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO fuel_entries (site_id, kind, litres, occurred_at, note, recorded_by, client_key, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING `+fuelColumns,
		e.SiteID, string(e.Kind), e.Litres, e.OccurredAt.UTC(), e.Note, e.RecordedBy, e.ClientKey, e.CreatedAt.UTC())
	created, err := scanFuel(row)
	return created, mapErr(err, "this entry was already sent")
}

func (r *FuelRepository) ByClientKey(ctx context.Context, recordedBy, key string) (domain.FuelEntry, error) {
	e, err := scanFuel(r.db.q(ctx).QueryRow(ctx,
		`SELECT `+fuelColumns+` FROM fuel_entries WHERE recorded_by = $1 AND client_key = $2`, recordedBy, key))
	return e, mapErr(err, "")
}

func (r *FuelRepository) GetByID(ctx context.Context, id string, forUpdate bool) (domain.FuelEntry, error) {
	q := `SELECT ` + fuelColumns + ` FROM fuel_entries WHERE id = $1`
	if forUpdate {
		q += ` FOR UPDATE`
	}
	e, err := scanFuel(r.db.q(ctx).QueryRow(ctx, q, id))
	return e, mapErr(err, "")
}

func (r *FuelRepository) Void(ctx context.Context, id, by, reason string, at time.Time) (bool, error) {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE fuel_entries SET voided_at = $4, voided_by = $2::uuid, void_reason = $3 WHERE id = $1 AND voided_at IS NULL`,
		id, by, reason, at.UTC())
	if err != nil {
		return false, mapErr(err, "")
	}
	return tag.RowsAffected() == 1, nil
}

func (r *FuelRepository) List(ctx context.Context, siteID string, includeVoided bool, limit int) ([]domain.FuelEntry, error) {
	q := `SELECT ` + fuelColumns + ` FROM fuel_entries WHERE site_id = $1`
	if !includeVoided {
		q += ` AND voided_at IS NULL`
	}
	rows, err := r.db.q(ctx).Query(ctx, q+` ORDER BY occurred_at DESC, seq DESC LIMIT $2`, siteID, limit)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()
	out := []domain.FuelEntry{}
	for rows.Next() {
		e, err := scanFuel(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning fuel entry: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
