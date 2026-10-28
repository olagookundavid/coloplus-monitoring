package postgres

import (
	"context"
	"fmt"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"

	"github.com/jackc/pgx/v5"
)

// UserRepository implements ports.UserRepository.
type UserRepository struct{ db *DB }

func NewUserRepository(db *DB) *UserRepository { return &UserRepository{db: db} }

const userColumns = `id::text, email, full_name, password_hash, role_id, is_active, totp_secret, totp_enabled,
	failed_logins, locked_until, last_login_at, created_at, updated_at`

func scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	var role string
	err := row.Scan(&u.ID, &u.Email, &u.FullName, &u.PasswordHash, &role, &u.Active, &u.TOTPSecret, &u.TOTPEnabled,
		&u.FailedLogins, &u.LockedUntil, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	u.Role = domain.Role(role)
	return u, err
}

func (r *UserRepository) Create(ctx context.Context, u domain.User) (domain.User, error) {
	row := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO users (email, full_name, password_hash, role_id, is_active)
		 VALUES ($1, $2, $3, $4, $5) RETURNING `+userColumns,
		u.Email, u.FullName, u.PasswordHash, string(u.Role), u.Active)
	created, err := scanUser(row)
	if err != nil {
		return domain.User{}, mapErr(err, "an account with this email already exists")
	}
	return created, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	u, err := scanUser(r.db.q(ctx).QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	return u, mapErr(err, "")
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	u, err := scanUser(r.db.q(ctx).QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE lower(email) = lower($1)`, email))
	return u, mapErr(err, "")
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.q(ctx).Query(ctx, `SELECT `+userColumns+` FROM users ORDER BY full_name, email`)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Update(ctx context.Context, u domain.User) error {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE users SET full_name = $2, role_id = $3, is_active = $4, password_hash = $5, updated_at = now()
		 WHERE id = $1`,
		u.ID, u.FullName, string(u.Role), u.Active, u.PasswordHash)
	if err != nil {
		return mapErr(err, "")
	}
	if tag.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

func (r *UserRepository) RecordLoginSuccess(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.q(ctx).Exec(ctx,
		`UPDATE users SET failed_logins = 0, locked_until = NULL, last_login_at = $2 WHERE id = $1`, id, at)
	return err
}

func (r *UserRepository) RecordLoginFailure(ctx context.Context, id string, failed int, lockedUntil *time.Time) error {
	_, err := r.db.q(ctx).Exec(ctx,
		`UPDATE users SET failed_logins = $2, locked_until = $3 WHERE id = $1`, id, failed, lockedUntil)
	return err
}

func (r *UserRepository) SetTOTP(ctx context.Context, id, sealedSecret string, enabled bool) error {
	_, err := r.db.q(ctx).Exec(ctx,
		`UPDATE users SET totp_secret = $2, totp_enabled = $3, updated_at = now() WHERE id = $1`, id, sealedSecret, enabled)
	return err
}

func (r *UserRepository) PermissionsForRole(ctx context.Context, role domain.Role) ([]domain.Permission, error) {
	rows, err := r.db.q(ctx).Query(ctx, `SELECT permission_id FROM role_permissions WHERE role_id = $1`, string(role))
	if err != nil {
		return nil, fmt.Errorf("loading permissions: %w", err)
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, domain.Permission(p))
	}
	return perms, rows.Err()
}

func (r *UserRepository) SiteIDsForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.q(ctx).Query(ctx, `SELECT site_id::text FROM user_sites WHERE user_id = $1 ORDER BY site_id`, userID)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SetUserSites replaces a user's assignments in one statement pair. Call it
// inside a transaction so a failure leaves the old assignments intact.
func (r *UserRepository) SetUserSites(ctx context.Context, userID string, siteIDs []string) error {
	q := r.db.q(ctx)
	if _, err := q.Exec(ctx, `DELETE FROM user_sites WHERE user_id = $1`, userID); err != nil {
		return mapErr(err, "")
	}
	if len(siteIDs) == 0 {
		return nil
	}
	_, err := q.Exec(ctx,
		`INSERT INTO user_sites (user_id, site_id) SELECT $1, unnest($2::uuid[]) ON CONFLICT DO NOTHING`, userID, siteIDs)
	return mapErr(err, "")
}
