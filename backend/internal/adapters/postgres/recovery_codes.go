package postgres

import (
	"context"
	"fmt"
	"time"
)

// RecoveryCodeRepository implements ports.RecoveryCodeRepository.
type RecoveryCodeRepository struct{ db *DB }

func NewRecoveryCodeRepository(db *DB) *RecoveryCodeRepository {
	return &RecoveryCodeRepository{db: db}
}

func (r *RecoveryCodeRepository) Replace(ctx context.Context, userID string, hashes []string) error {
	q := r.db.q(ctx)
	if _, err := q.Exec(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id = $1`, userID); err != nil {
		return mapErr(err, "")
	}
	if len(hashes) == 0 {
		return nil
	}
	if _, err := q.Exec(ctx,
		`INSERT INTO mfa_recovery_codes (user_id, code_hash) SELECT $1, unnest($2::text[])`, userID, hashes); err != nil {
		return mapErr(err, "")
	}
	return nil
}

func (r *RecoveryCodeRepository) Consume(ctx context.Context, userID, hash string, at time.Time) (bool, error) {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE mfa_recovery_codes SET used_at = $3
		 WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL`, userID, hash, at)
	if err != nil {
		return false, mapErr(err, "")
	}
	return tag.RowsAffected() == 1, nil
}

func (r *RecoveryCodeRepository) Remaining(ctx context.Context, userID string) (int, error) {
	var n int
	if err := r.db.q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM mfa_recovery_codes WHERE user_id = $1 AND used_at IS NULL`, userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("counting recovery codes: %w", err)
	}
	return n, nil
}

func (r *RecoveryCodeRepository) DeleteAll(ctx context.Context, userID string) error {
	if _, err := r.db.q(ctx).Exec(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id = $1`, userID); err != nil {
		return mapErr(err, "")
	}
	return nil
}
