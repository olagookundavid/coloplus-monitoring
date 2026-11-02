package postgres

import (
	"context"
	"fmt"
	"time"

	"coloplus-backend/internal/domain"
)

// AlertRepository implements ports.AlertRepository.
type AlertRepository struct{ db *DB }

func NewAlertRepository(db *DB) *AlertRepository { return &AlertRepository{db: db} }

func (r *AlertRepository) Add(ctx context.Context, userID, kind, title, body, dedupeKey string, at time.Time) (bool, error) {
	tag, err := r.db.q(ctx).Exec(ctx,
		`INSERT INTO alerts (user_id, kind, title, body, dedupe_key, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (user_id, dedupe_key) DO NOTHING`,
		userID, kind, title, body, dedupeKey, at.UTC())
	if err != nil {
		return false, mapErr(err, "")
	}
	return tag.RowsAffected() == 1, nil
}

func (r *AlertRepository) List(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]domain.Alert, int, int, error) {
	q := r.db.q(ctx)
	var total, unread int
	if err := q.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE $2 OR acknowledged_at IS NULL), count(*) FILTER (WHERE acknowledged_at IS NULL)
		 FROM alerts WHERE user_id = $1`, userID, !unreadOnly).Scan(&total, &unread); err != nil {
		return nil, 0, 0, mapErr(err, "")
	}
	where := ""
	if unreadOnly {
		where = " AND acknowledged_at IS NULL"
	}
	rows, err := q.Query(ctx,
		`SELECT id::text, user_id::text, kind, title, body, created_at, acknowledged_at FROM alerts
		 WHERE user_id = $1`+where+` ORDER BY seq DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, 0, mapErr(err, "")
	}
	defer rows.Close()
	out := []domain.Alert{}
	for rows.Next() {
		var a domain.Alert
		if err := rows.Scan(&a.ID, &a.UserID, &a.Kind, &a.Title, &a.Body, &a.CreatedAt, &a.AcknowledgedAt); err != nil {
			return nil, 0, 0, fmt.Errorf("scanning alert: %w", err)
		}
		out = append(out, a)
	}
	return out, total, unread, rows.Err()
}

func (r *AlertRepository) Acknowledge(ctx context.Context, userID, alertID string, at time.Time) (bool, error) {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE alerts SET acknowledged_at = COALESCE(acknowledged_at, $3) WHERE id = $2::uuid AND user_id = $1`,
		userID, alertID, at.UTC())
	if err != nil {
		return false, mapErr(err, "")
	}
	return tag.RowsAffected() == 1, nil
}

func (r *AlertRepository) AcknowledgeAll(ctx context.Context, userID string, at time.Time) (int, error) {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE alerts SET acknowledged_at = $2 WHERE user_id = $1 AND acknowledged_at IS NULL`, userID, at.UTC())
	if err != nil {
		return 0, mapErr(err, "")
	}
	return int(tag.RowsAffected()), nil
}
