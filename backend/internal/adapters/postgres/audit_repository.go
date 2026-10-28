package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"coloplus-backend/internal/domain"

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
