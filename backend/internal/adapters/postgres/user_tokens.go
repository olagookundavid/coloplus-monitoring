package postgres

import (
	"context"
	"time"

	"coloplus-backend/internal/domain"
)

// UserTokenRepository implements ports.UserTokenRepository.
type UserTokenRepository struct{ db *DB }

func NewUserTokenRepository(db *DB) *UserTokenRepository { return &UserTokenRepository{db: db} }

const userTokenColumns = `id::text, user_id::text, purpose, expires_at, used_at, created_at`

func (r *UserTokenRepository) Create(ctx context.Context, t domain.UserToken, hash string) (domain.UserToken, error) {
	var purpose string
	err := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO user_tokens (user_id, purpose, token_hash, expires_at) VALUES ($1, $2, $3, $4)
		 RETURNING `+userTokenColumns, t.UserID, string(t.Purpose), hash, t.ExpiresAt.UTC()).
		Scan(&t.ID, &t.UserID, &purpose, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)
	t.Purpose = domain.TokenPurpose(purpose)
	return t, mapErr(err, "")
}

func (r *UserTokenRepository) GetByHash(ctx context.Context, hash string) (domain.UserToken, error) {
	var t domain.UserToken
	var purpose string
	err := r.db.q(ctx).QueryRow(ctx,
		`SELECT `+userTokenColumns+` FROM user_tokens WHERE token_hash = $1 FOR UPDATE`, hash).
		Scan(&t.ID, &t.UserID, &purpose, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)
	t.Purpose = domain.TokenPurpose(purpose)
	return t, mapErr(err, "")
}

func (r *UserTokenRepository) MarkUsed(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.q(ctx).Exec(ctx, `UPDATE user_tokens SET used_at = $2 WHERE id = $1`, id, at.UTC())
	return mapErr(err, "")
}

func (r *UserTokenRepository) RevokeOutstanding(ctx context.Context, userID string, purpose domain.TokenPurpose, at time.Time) error {
	_, err := r.db.q(ctx).Exec(ctx,
		`UPDATE user_tokens SET used_at = $3 WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL`,
		userID, string(purpose), at.UTC())
	return mapErr(err, "")
}

func (r *UserTokenRepository) CountSince(ctx context.Context, userID string, purpose domain.TokenPurpose, since time.Time) (int, error) {
	var n int
	err := r.db.q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM user_tokens WHERE user_id = $1 AND purpose = $2 AND created_at >= $3`,
		userID, string(purpose), since.UTC()).Scan(&n)
	return n, mapErr(err, "")
}
