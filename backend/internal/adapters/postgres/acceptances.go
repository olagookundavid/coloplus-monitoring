package postgres

import (
	"context"
	"fmt"
	"strings"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/ports"

	"github.com/jackc/pgx/v5"
)

// AcceptanceRepository implements ports.AcceptanceRepository.
type AcceptanceRepository struct{ db *DB }

func NewAcceptanceRepository(db *DB) *AcceptanceRepository { return &AcceptanceRepository{db: db} }

const acceptanceColumns = `id::text, site_id::text, stage, held_from, responsible, next_action, due_at, dossier, reason,
	hold_reason, requested_at, built_at, ready_at, inspected_at, decided_at, COALESCE(decided_by::text, ''),
	created_by::text, created_at, updated_at`

func scanAcceptance(row pgx.Row) (domain.Acceptance, error) {
	var a domain.Acceptance
	var stage, held, resp string
	err := row.Scan(&a.ID, &a.SiteID, &stage, &held, &resp, &a.NextAction, &a.DueAt, &a.Dossier, &a.Reason,
		&a.HoldReason, &a.RequestedAt, &a.BuiltAt, &a.ReadyAt, &a.InspectedAt, &a.DecidedAt, &a.DecidedBy,
		&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	a.Stage, a.HeldFrom, a.Responsible = domain.AcceptanceStage(stage), domain.AcceptanceStage(held), domain.DisputeParty(resp)
	return a, err
}

func (r *AcceptanceRepository) Create(ctx context.Context, a domain.Acceptance) (domain.Acceptance, error) {
	row := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO acceptances (site_id, responsible, next_action, due_at, requested_at, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $5, $5) RETURNING `+acceptanceColumns,
		a.SiteID, string(a.Responsible), a.NextAction, a.DueAt, a.RequestedAt.UTC(), a.CreatedBy)
	created, err := scanAcceptance(row)
	return created, mapErr(err, "this site already has an acceptance process")
}

func (r *AcceptanceRepository) GetBySite(ctx context.Context, siteID string, forUpdate bool) (domain.Acceptance, error) {
	q := `SELECT ` + acceptanceColumns + ` FROM acceptances WHERE site_id = $1`
	if forUpdate {
		q += ` FOR UPDATE`
	}
	a, err := scanAcceptance(r.db.q(ctx).QueryRow(ctx, q, siteID))
	return a, mapErr(err, "")
}

func (r *AcceptanceRepository) Save(ctx context.Context, a domain.Acceptance) error {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE acceptances SET stage = $2, held_from = $3, responsible = $4, next_action = $5, due_at = $6,
		        dossier = $7, reason = $8, hold_reason = $9, built_at = $10, ready_at = $11, inspected_at = $12,
		        decided_at = $13, decided_by = NULLIF($14, '')::uuid, updated_at = $15
		 WHERE id = $1`,
		a.ID, string(a.Stage), string(a.HeldFrom), string(a.Responsible), a.NextAction, a.DueAt, a.Dossier, a.Reason,
		a.HoldReason, a.BuiltAt, a.ReadyAt, a.InspectedAt, a.DecidedAt, a.DecidedBy, a.UpdatedAt.UTC())
	if err != nil {
		return mapErr(err, "")
	}
	if tag.RowsAffected() == 0 {
		return mapErr(pgx.ErrNoRows, "")
	}
	return nil
}

func (r *AcceptanceRepository) AddEvent(ctx context.Context, e domain.AcceptanceEvent) (domain.AcceptanceEvent, error) {
	err := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO acceptance_events (acceptance_id, author_id, party, from_stage, to_stage, body, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id::text`,
		e.AcceptanceID, e.AuthorID, string(e.Party), string(e.FromStage), string(e.ToStage), e.Body, e.CreatedAt.UTC()).Scan(&e.ID)
	return e, mapErr(err, "")
}

func (r *AcceptanceRepository) Events(ctx context.Context, acceptanceID string) ([]domain.AcceptanceEvent, error) {
	rows, err := r.db.q(ctx).Query(ctx,
		`SELECT id::text, acceptance_id::text, author_id::text, party, from_stage, to_stage, body, created_at
		 FROM acceptance_events WHERE acceptance_id = $1 ORDER BY seq`, acceptanceID)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()
	out := []domain.AcceptanceEvent{}
	for rows.Next() {
		var e domain.AcceptanceEvent
		var party, from, to string
		if err := rows.Scan(&e.ID, &e.AcceptanceID, &e.AuthorID, &party, &from, &to, &e.Body, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning acceptance event: %w", err)
		}
		e.Party, e.FromStage, e.ToStage = domain.DisputeParty(party), domain.AcceptanceStage(from), domain.AcceptanceStage(to)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *AcceptanceRepository) List(ctx context.Context, f ports.AcceptanceFilter) ([]domain.Acceptance, int, error) {
	var where []string
	var args []any
	add := func(clause string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(clause, "?", fmt.Sprintf("$%d", len(args))))
	}
	if !f.AllSites {
		add("site_id = ANY(?::uuid[])", nonNil(f.SiteIDs))
	}
	if len(f.Stages) > 0 {
		stages := make([]string, len(f.Stages))
		for i, s := range f.Stages {
			stages[i] = string(s)
		}
		add("stage = ANY(?::text[])", stages)
	}
	if f.OpenOnly {
		where = append(where, "stage <> 'accepted'")
	}
	if f.OverdueAt != nil {
		add("(stage NOT IN ('accepted', 'on_hold') AND due_at < ?)", f.OverdueAt.UTC())
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	q := r.db.q(ctx)
	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM acceptances`+clause, args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err, "")
	}
	args = append(args, f.Limit, f.Offset)
	// Longest waiting for inspection first, then the oldest requests.
	rows, err := q.Query(ctx,
		`SELECT `+acceptanceColumns+` FROM acceptances`+clause+
			fmt.Sprintf(` ORDER BY ready_at NULLS LAST, requested_at LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, mapErr(err, "")
	}
	defer rows.Close()
	out := []domain.Acceptance{}
	for rows.Next() {
		a, err := scanAcceptance(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning acceptance: %w", err)
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}
