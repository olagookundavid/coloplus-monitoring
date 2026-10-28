package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

const (
	dispatchBatch = 25
	// A claimed email is not picked up again until its lease runs out, which
	// covers a crash between claiming and sending.
	dispatchLease = 5 * time.Minute
	// One run sends at most this many batches, so a backlog cannot hold the
	// scheduler for long.
	dispatchMaxBatches = 8
)

// NotificationService owns the email outbox. Business code queues an email
// inside its own transaction; the dispatcher sends it afterwards. A provider
// outage therefore never fails a business action and never loses an email.
type NotificationService struct {
	store  ports.NotificationStore
	mailer ports.Mailer
	audit  *AuditService
	clock  ports.Clock
	log    *slog.Logger
}

func NewNotificationService(store ports.NotificationStore, mailer ports.Mailer, audit *AuditService, clock ports.Clock, log *slog.Logger) *NotificationService {
	return &NotificationService{store: store, mailer: mailer, audit: audit, clock: clock, log: log}
}

// Enqueue queues one email. The same dedupe key is only ever queued once, so
// callers can run again after a failure or a retry without sending twice.
// Call it inside the transaction of the change that caused the email.
func (s *NotificationService) Enqueue(ctx context.Context, n domain.Notification) (bool, error) {
	if n.Recipient == "" || n.Subject == "" || n.DedupeKey == "" || n.Kind == "" {
		return false, errors.New("a notification needs a kind, recipient, subject and dedupe key")
	}
	// Due now, by this process's clock, which is the one the dispatcher uses.
	n.NextAttemptAt = s.clock.Now()
	return s.store.Enqueue(ctx, n)
}

// EnqueueFor queues the same email for several people, one dedupe key each.
func (s *NotificationService) EnqueueFor(ctx context.Context, users []domain.User, kind, key, subject, body string) (int, error) {
	queued := 0
	for _, u := range users {
		ok, err := s.Enqueue(ctx, domain.Notification{
			Kind: kind, Recipient: u.Email, Subject: subject, Body: body, DedupeKey: key + ":" + u.ID,
		})
		if err != nil {
			return queued, err
		}
		if ok {
			queued++
		}
	}
	return queued, nil
}

// DispatchResult says what one dispatcher run did.
type DispatchResult struct {
	Sent, Retrying, Failed int
}

// Dispatch sends what is due. A transient failure is retried with a growing
// delay; a refusal, or running out of attempts, marks the email failed so it
// shows up in the admin list.
func (s *NotificationService) Dispatch(ctx context.Context) (DispatchResult, error) {
	var res DispatchResult
	for batch := 0; batch < dispatchMaxBatches; batch++ {
		due, err := s.store.ClaimDue(ctx, s.clock.Now(), dispatchBatch, dispatchLease)
		if err != nil {
			return res, fmt.Errorf("claiming notifications: %w", err)
		}
		for _, n := range due {
			if ctx.Err() != nil {
				return res, nil
			}
			s.deliver(ctx, n, &res)
		}
		if len(due) < dispatchBatch {
			break
		}
	}
	return res, nil
}

func (s *NotificationService) deliver(ctx context.Context, n domain.Notification, res *DispatchResult) {
	id, err := s.mailer.Send(ctx, domain.Email{To: n.Recipient, Subject: n.Subject, Text: n.Body}, n.DedupeKey)
	if err == nil {
		if err := s.store.MarkSent(ctx, n.ID, id, s.clock.Now()); err != nil {
			s.log.Error("email sent but not recorded", "notification_id", n.ID, "error", err)
		}
		res.Sent++
		return
	}
	if ctx.Err() != nil {
		return // shutting down; the lease lets the next run pick it up
	}

	reason := truncate(err.Error(), 500)
	if errors.Is(err, apperrors.ErrMailRejected) {
		s.fail(ctx, n, reason, res)
		return
	}
	delay, more := domain.RetryDelay(n.Attempts)
	if !more {
		s.fail(ctx, n, reason, res)
		return
	}
	if err := s.store.MarkRetry(ctx, n.ID, reason, s.clock.Now().Add(delay)); err != nil {
		s.log.Error("scheduling email retry", "notification_id", n.ID, "error", err)
	}
	res.Retrying++
}

func (s *NotificationService) fail(ctx context.Context, n domain.Notification, reason string, res *DispatchResult) {
	if err := s.store.MarkFailed(ctx, n.ID, reason); err != nil {
		s.log.Error("marking email failed", "notification_id", n.ID, "error", err)
	}
	s.log.Warn("email could not be delivered", "notification_id", n.ID, "kind", n.Kind, "attempts", n.Attempts)
	_ = s.audit.Record(ctx, domain.AuditEvent{
		Action: "notification.failed", EntityType: "notification", EntityID: n.ID,
		Metadata: map[string]any{"kind": n.Kind, "attempts": n.Attempts},
	})
	res.Failed++
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n]
	}
	return s
}

// List shows delivery status to administrators. The body is never returned.
func (s *NotificationService) List(ctx context.Context, actor domain.Actor, status domain.NotificationStatus, limit, offset int) ([]domain.Notification, int, error) {
	if !actor.Can(domain.PermNotificationsRead) {
		return nil, 0, apperrors.ErrForbidden
	}
	limit, offset = Page(limit, offset)
	return s.store.List(ctx, ports.NotificationFilter{Status: status, Limit: limit, Offset: offset})
}

// Retry puts a failed email back in the queue with a fresh set of attempts.
func (s *NotificationService) Retry(ctx context.Context, actor domain.Actor, id string) error {
	if !actor.Can(domain.PermNotificationsRead) {
		return apperrors.ErrForbidden
	}
	if err := s.store.Requeue(ctx, id, s.clock.Now()); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return conflict("only an email that failed can be retried")
		}
		return err
	}
	_ = s.audit.Record(ctx, domain.AuditEvent{
		ActorID: actor.UserID, Action: "notification.retried", EntityType: "notification", EntityID: id,
	})
	return nil
}
