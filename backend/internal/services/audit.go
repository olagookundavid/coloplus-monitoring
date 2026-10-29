package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// AuditService records who did what. Two paths exist on purpose:
//
//   - Record is batched and never fails the caller. Use it for events where
//     losing the last few seconds in a hard crash is acceptable.
//   - RecordInTx writes inside the caller's transaction. Use it where the
//     record must exist if and only if the change committed (approvals,
//     corrections, publications).
type AuditService struct {
	store ports.AuditStore
	queue ports.AuditQueue
	clock ports.Clock
	log   *slog.Logger
}

func NewAuditService(store ports.AuditStore, queue ports.AuditQueue, clock ports.Clock, log *slog.Logger) *AuditService {
	return &AuditService{store: store, queue: queue, clock: clock, log: log}
}

// Record queues the event for a batched write.
func (s *AuditService) Record(ctx context.Context, event domain.AuditEvent) error {
	event, err := s.prepare(event)
	if err != nil {
		return err
	}
	if !s.queue.Add(event) {
		s.log.Warn("audit queue did not accept event", "action", event.Action, "entity_type", event.EntityType)
	}
	return nil
}

// RecordInTx writes the event immediately on the transaction carried by ctx.
func (s *AuditService) RecordInTx(ctx context.Context, event domain.AuditEvent) error {
	event, err := s.prepare(event)
	if err != nil {
		return err
	}
	if err := s.store.Insert(ctx, event); err != nil {
		return fmt.Errorf("recording audit event %q: %w", event.Action, err)
	}
	return nil
}

func (s *AuditService) prepare(event domain.AuditEvent) (domain.AuditEvent, error) {
	if event.OccurredAt.IsZero() {
		event.OccurredAt = s.clock.Now()
	}
	if err := event.Validate(); err != nil {
		return event, fmt.Errorf("%w: %v", apperrors.ErrInvalidInput, err)
	}
	return event, nil
}

// SearchInput narrows an audit search.
type SearchAuditInput struct {
	ActorID    string
	Action     string
	EntityType string
	EntityID   string
	From, To   *time.Time
	Limit      int
	Offset     int
}

// Search reads the audit log for people with audit access, within their site
// scope. Reading the log is separate from editing anything.
func (s *AuditService) Search(ctx context.Context, actor domain.Actor, in SearchAuditInput) ([]domain.AuditEvent, int, error) {
	if !actor.Can(domain.PermAuditRead) {
		return nil, 0, apperrors.ErrForbidden
	}
	f := ports.AuditFilter{
		AllSites: actor.AllSites, SiteIDs: actor.SiteIDList(), ActorID: in.ActorID, Action: in.Action,
		EntityType: in.EntityType, EntityID: in.EntityID, From: in.From, To: in.To,
	}
	f.Limit, f.Offset = Page(in.Limit, in.Offset)
	return s.store.Search(ctx, f)
}
