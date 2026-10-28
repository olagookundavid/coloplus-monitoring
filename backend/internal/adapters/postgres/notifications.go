package postgres

import (
	"context"
	"fmt"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"

	"github.com/jackc/pgx/v5"
)

// NotificationRepository implements ports.NotificationStore.
type NotificationRepository struct{ db *DB }

func NewNotificationRepository(db *DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

const notificationColumns = `id::text, kind, recipient, subject, body, status, attempts, next_attempt_at,
	last_error, provider_id, dedupe_key, created_at, sent_at`

func scanNotification(row pgx.Row) (domain.Notification, error) {
	var n domain.Notification
	var status string
	err := row.Scan(&n.ID, &n.Kind, &n.Recipient, &n.Subject, &n.Body, &status, &n.Attempts, &n.NextAttemptAt,
		&n.LastError, &n.ProviderID, &n.DedupeKey, &n.CreatedAt, &n.SentAt)
	n.Status = domain.NotificationStatus(status)
	return n, err
}

func (r *NotificationRepository) Enqueue(ctx context.Context, n domain.Notification) (bool, error) {
	tag, err := r.db.q(ctx).Exec(ctx,
		`INSERT INTO notifications (kind, recipient, subject, body, dedupe_key, next_attempt_at)
		 VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (dedupe_key) DO NOTHING`,
		n.Kind, n.Recipient, n.Subject, n.Body, n.DedupeKey, n.NextAttemptAt.UTC())
	if err != nil {
		return false, mapErr(err, "")
	}
	return tag.RowsAffected() == 1, nil
}

func (r *NotificationRepository) ClaimDue(ctx context.Context, now time.Time, limit int, lease time.Duration) ([]domain.Notification, error) {
	rows, err := r.db.q(ctx).Query(ctx,
		`UPDATE notifications SET attempts = attempts + 1, next_attempt_at = $2
		 WHERE id IN (SELECT id FROM notifications
		              WHERE status = 'pending' AND next_attempt_at <= $1
		              ORDER BY next_attempt_at LIMIT $3 FOR UPDATE SKIP LOCKED)
		 RETURNING `+notificationColumns, now.UTC(), now.Add(lease).UTC(), limit)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()

	out := []domain.Notification{}
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning notification: %w", err)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *NotificationRepository) MarkSent(ctx context.Context, id, providerID string, at time.Time) error {
	_, err := r.db.q(ctx).Exec(ctx,
		`UPDATE notifications SET status = 'sent', sent_at = $3, provider_id = $2, body = '', last_error = ''
		 WHERE id = $1`, id, providerID, at.UTC())
	return mapErr(err, "")
}

func (r *NotificationRepository) MarkRetry(ctx context.Context, id, reason string, next time.Time) error {
	_, err := r.db.q(ctx).Exec(ctx,
		`UPDATE notifications SET last_error = $2, next_attempt_at = $3 WHERE id = $1`, id, reason, next.UTC())
	return mapErr(err, "")
}

func (r *NotificationRepository) MarkFailed(ctx context.Context, id, reason string) error {
	_, err := r.db.q(ctx).Exec(ctx, `UPDATE notifications SET status = 'failed', last_error = $2 WHERE id = $1`, id, reason)
	return mapErr(err, "")
}

func (r *NotificationRepository) Requeue(ctx context.Context, id string, at time.Time) error {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE notifications SET status = 'pending', attempts = 0, next_attempt_at = $2, last_error = ''
		 WHERE id = $1 AND status = 'failed'`, id, at.UTC())
	if err != nil {
		return mapErr(err, "")
	}
	if tag.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

func (r *NotificationRepository) GetByID(ctx context.Context, id string) (domain.Notification, error) {
	n, err := scanNotification(r.db.q(ctx).QueryRow(ctx, `SELECT `+notificationColumns+` FROM notifications WHERE id = $1`, id))
	return n, mapErr(err, "")
}

func (r *NotificationRepository) List(ctx context.Context, f ports.NotificationFilter) ([]domain.Notification, int, error) {
	where, args := "", []any{}
	if f.Status != "" {
		where, args = " WHERE status = $1", append(args, string(f.Status))
	}
	var total int
	if err := r.db.q(ctx).QueryRow(ctx, `SELECT count(*) FROM notifications`+where, args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err, "")
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.q(ctx).Query(ctx,
		`SELECT `+notificationColumns+` FROM notifications`+where+
			fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, mapErr(err, "")
	}
	defer rows.Close()

	out := []domain.Notification{}
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning notification: %w", err)
		}
		out = append(out, n)
	}
	return out, total, rows.Err()
}
