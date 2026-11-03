package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// DigestPreferenceRepository implements ports.DigestPreferenceRepository.
type DigestPreferenceRepository struct{ db *DB }

func NewDigestPreferenceRepository(db *DB) *DigestPreferenceRepository {
	return &DigestPreferenceRepository{db: db}
}

func (r *DigestPreferenceRepository) OptedOut(ctx context.Context) (map[string]bool, error) {
	rows, err := r.db.q(ctx).Query(ctx, `SELECT user_id::text FROM digest_preferences WHERE NOT weekly`)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning digest preference: %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

func (r *DigestPreferenceRepository) Weekly(ctx context.Context, userID string) (bool, error) {
	var on bool
	err := r.db.q(ctx).QueryRow(ctx, `SELECT weekly FROM digest_preferences WHERE user_id = $1`, userID).Scan(&on)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	return on, mapErr(err, "")
}

func (r *DigestPreferenceRepository) SetWeekly(ctx context.Context, userID string, on bool) error {
	_, err := r.db.q(ctx).Exec(ctx,
		`INSERT INTO digest_preferences (user_id, weekly) VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET weekly = EXCLUDED.weekly, updated_at = now()`, userID, on)
	return mapErr(err, "")
}
