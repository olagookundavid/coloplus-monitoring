package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"coloplus-backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

// IssuedExportRepository implements ports.IssuedExportRepository.
type IssuedExportRepository struct{ db *DB }

func NewIssuedExportRepository(db *DB) *IssuedExportRepository {
	return &IssuedExportRepository{db: db}
}

const issuedColumns = `id::text, kind, filename, params, row_count, size_bytes, sha256, storage_key, issued_by::text, created_at`

func scanIssued(row pgx.Row) (domain.IssuedExport, error) {
	var e domain.IssuedExport
	var params []byte
	err := row.Scan(&e.ID, &e.Kind, &e.Filename, &params, &e.RowCount, &e.SizeBytes, &e.SHA256, &e.StorageKey, &e.IssuedBy, &e.CreatedAt)
	if err == nil && len(params) > 0 {
		_ = json.Unmarshal(params, &e.Params)
	}
	return e, err
}

func (r *IssuedExportRepository) Create(ctx context.Context, e domain.IssuedExport) (domain.IssuedExport, error) {
	params, err := json.Marshal(e.Params)
	if err != nil {
		return domain.IssuedExport{}, fmt.Errorf("encoding export parameters: %w", err)
	}
	row := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO issued_exports (kind, filename, params, row_count, size_bytes, sha256, storage_key, issued_by, created_at)
		 VALUES ($1, $2, $3::jsonb, $4, $5, $6, $7, $8, $9) RETURNING `+issuedColumns,
		e.Kind, e.Filename, string(params), e.RowCount, e.SizeBytes, e.SHA256, e.StorageKey, e.IssuedBy, e.CreatedAt.UTC())
	created, err := scanIssued(row)
	return created, mapErr(err, "")
}

func (r *IssuedExportRepository) GetByID(ctx context.Context, id string) (domain.IssuedExport, error) {
	e, err := scanIssued(r.db.q(ctx).QueryRow(ctx, `SELECT `+issuedColumns+` FROM issued_exports WHERE id = $1`, id))
	return e, mapErr(err, "")
}

func (r *IssuedExportRepository) List(ctx context.Context, issuedBy string, limit, offset int) ([]domain.IssuedExport, int, error) {
	q := r.db.q(ctx)
	where, args := "", []any{}
	if issuedBy != "" {
		where, args = " WHERE issued_by = $1::uuid", []any{issuedBy}
	}
	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM issued_exports`+where, args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err, "")
	}
	args = append(args, limit, offset)
	rows, err := q.Query(ctx, `SELECT `+issuedColumns+` FROM issued_exports`+where+
		fmt.Sprintf(` ORDER BY seq DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, mapErr(err, "")
	}
	defer rows.Close()
	out := []domain.IssuedExport{}
	for rows.Next() {
		e, err := scanIssued(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning issued export: %w", err)
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}
