package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/ports"

	"github.com/jackc/pgx/v5"
)

// AuditRepository implements ports.AuditStore.
type AuditRepository struct {
	db *DB
}

func NewAuditRepository(db *DB) *AuditRepository { return &AuditRepository{db: db} }

var auditColumns = []string{"occurred_at", "actor_id", "action", "entity_type", "entity_id", "metadata"}

// Insert writes one event on the transaction carried by ctx, if any.
func (r *AuditRepository) Insert(ctx context.Context, e domain.AuditEvent) error {
	meta, err := marshalMetadata(e.Metadata)
	if err != nil {
		return err
	}
	_, err = r.db.q(ctx).Exec(ctx,
		`INSERT INTO audit_log (occurred_at, actor_id, action, entity_type, entity_id, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		e.OccurredAt, e.ActorID, e.Action, e.EntityType, e.EntityID, meta)
	if err != nil {
		return fmt.Errorf("inserting audit event: %w", err)
	}
	return nil
}

// InsertBatch writes many events with a single COPY.
func (r *AuditRepository) InsertBatch(ctx context.Context, events []domain.AuditEvent) error {
	rows := make([][]any, 0, len(events))
	for _, e := range events {
		meta, err := marshalMetadata(e.Metadata)
		if err != nil {
			return err
		}
		rows = append(rows, []any{e.OccurredAt, e.ActorID, e.Action, e.EntityType, e.EntityID, meta})
	}
	_, err := r.db.q(ctx).CopyFrom(ctx, pgx.Identifier{"audit_log"}, auditColumns, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("copying %d audit events: %w", len(events), err)
	}
	return nil
}

func marshalMetadata(m map[string]any) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encoding audit metadata: %w", err)
	}
	return b, nil
}

// Search reads the log back. The site scope matches the site_id that site-
// related events carry in their metadata.
func (r *AuditRepository) Search(ctx context.Context, f ports.AuditFilter) ([]domain.AuditEvent, int, error) {
	var where []string
	var args []any
	add := func(clause string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(clause, "?", fmt.Sprintf("$%d", len(args))))
	}
	if !f.AllSites {
		add("metadata->>'site_id' = ANY(?::text[])", nonNil(f.SiteIDs))
	}
	if f.ActorID != "" {
		add("actor_id = ?", f.ActorID)
	}
	if f.Action != "" {
		add("action = ?", f.Action)
	}
	if f.EntityType != "" {
		add("entity_type = ?", f.EntityType)
	}
	if f.EntityID != "" {
		add("entity_id = ?", f.EntityID)
	}
	if f.From != nil {
		add("occurred_at >= ?", f.From.UTC())
	}
	if f.To != nil {
		add("occurred_at < ?", f.To.UTC())
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	q := r.db.q(ctx)
	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM audit_log`+clause, args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err, "")
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := q.Query(ctx,
		`SELECT id, occurred_at, actor_id, action, entity_type, entity_id, metadata FROM audit_log`+clause+
			fmt.Sprintf(` ORDER BY occurred_at DESC, id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, mapErr(err, "")
	}
	defer rows.Close()

	out := []domain.AuditEvent{}
	for rows.Next() {
		var e domain.AuditEvent
		var meta []byte
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.ActorID, &e.Action, &e.EntityType, &e.EntityID, &meta); err != nil {
			return nil, 0, fmt.Errorf("scanning audit event: %w", err)
		}
		if err := json.Unmarshal(meta, &e.Metadata); err != nil {
			return nil, 0, fmt.Errorf("decoding audit metadata: %w", err)
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}
