package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// AuthService signs people in, with an optional authenticator-app step, and
// turns a verified token into the Actor that every other service checks.
type AuthService struct {
	users  ports.UserRepository
	hasher ports.PasswordHasher
	tokens ports.TokenIssuer
	totp   ports.TOTP
	box    ports.SecretBox
	audit  *AuditService
	clock  ports.Clock
	log    *slog.Logger
	ttl    time.Duration

	// dummyHash lets a lookup for an unknown email cost the same as a real
	// one, so response time does not reveal which emails have accounts.
	dummyHash string
}

func NewAuthService(
	users ports.UserRepository, hasher ports.PasswordHasher, tokens ports.TokenIssuer,
	totp ports.TOTP, box ports.SecretBox, audit *AuditService, clock ports.Clock,
	ttl time.Duration, log *slog.Logger,
) (*AuthService, error) {
	dummy, err := hasher.Hash("not-a-real-password-0000")
	if err != nil {
		return nil, fmt.Errorf("preparing auth service: %w", err)
	}
	return &AuthService{
		users: users, hasher: hasher, tokens: tokens, totp: totp, box: box,
		audit: audit, clock: clock, ttl: ttl, log: log, dummyHash: dummy,
	}, nil
}

// LoginResult is what a sign-in attempt produced. When MFARequired is set the
// password was right but a code is still needed; no token is issued.
type LoginResult struct {
	MFARequired bool
	Token       string
	ExpiresAt   time.Time
	User        domain.User
}

// Login checks the password and, if the account has two-step verification on,
// the authenticator code.
func (s *AuthService) Login(ctx context.Context, email, password, code string) (LoginResult, error) {
	now := s.clock.Now()
	email = domain.NormalizeEmail(email)

	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, apperrors.ErrNotFound) {
		s.hasher.Matches(s.dummyHash, password)
		return LoginResult{}, apperrors.ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, fmt.Errorf("looking up user: %w", err)
	}
	if user.IsLocked(now) {
		return LoginResult{}, apperrors.ErrAccountLocked
	}
	if !user.Active || !s.hasher.Matches(user.PasswordHash, password) {
		return LoginResult{}, s.failLogin(ctx, user, now, "bad_password")
	}

	if user.TOTPEnabled {
		if code == "" {
			return LoginResult{MFARequired: true}, nil
		}
		secret, err := s.box.Open(user.TOTPSecret)
		if err != nil {
			return LoginResult{}, fmt.Errorf("opening totp secret: %w", err)
		}
		if !s.totp.Verify(secret, code, now) {
			return LoginResult{}, s.failLogin(ctx, user, now, "bad_code")
		}
	}

	token, expires, err := s.tokens.Issue(user.ID, s.ttl)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issuing token: %w", err)
	}
	if err := s.users.RecordLoginSuccess(ctx, user.ID, now); err != nil {
		return LoginResult{}, fmt.Errorf("recording sign-in: %w", err)
	}
	_ = s.audit.Record(ctx, domain.AuditEvent{ActorID: user.ID, Action: "auth.login", EntityType: "user", EntityID: user.ID})
	return LoginResult{Token: token, ExpiresAt: expires, User: user}, nil
}

func (s *AuthService) failLogin(ctx context.Context, user domain.User, now time.Time, reason string) error {
	failed, lockedUntil := user.AfterFailedLogin(now)
	if err := s.users.RecordLoginFailure(ctx, user.ID, failed, lockedUntil); err != nil {
		s.log.Error("recording failed sign-in", "user_id", user.ID, "error", err)
	}
	_ = s.audit.Record(ctx, domain.AuditEvent{
		ActorID: user.ID, Action: "auth.login_failed", EntityType: "user", EntityID: user.ID,
		Metadata: map[string]any{"reason": reason, "locked": lockedUntil != nil},
	})
	if reason == "bad_code" {
		return apperrors.ErrInvalidCode
	}
	return apperrors.ErrInvalidCredentials
}

// Authenticate builds the Actor for a verified token's user. A deactivated
// user is refused here, so revoking access takes effect on the next request.
func (s *AuthService) Authenticate(ctx context.Context, userID string) (domain.Actor, error) {
	user, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, apperrors.ErrNotFound) || (err == nil && !user.Active) {
		return domain.Actor{}, apperrors.ErrUnauthorized
	}
	if err != nil {
		return domain.Actor{}, fmt.Errorf("loading user: %w", err)
	}

	perms, err := s.users.PermissionsForRole(ctx, user.Role)
	if err != nil {
		return domain.Actor{}, fmt.Errorf("loading permissions: %w", err)
	}
	actor := domain.Actor{
		UserID:      user.ID,
		Role:        user.Role,
		Permissions: make(map[domain.Permission]struct{}, len(perms)),
		SiteIDs:     map[string]struct{}{},
	}
	for _, p := range perms {
		actor.Permissions[p] = struct{}{}
	}
	if actor.Can(domain.PermSitesAll) {
		actor.AllSites = true
		return actor, nil
	}
	ids, err := s.users.SiteIDsForUser(ctx, user.ID)
	if err != nil {
		return domain.Actor{}, fmt.Errorf("loading site assignments: %w", err)
	}
	for _, id := range ids {
		actor.SiteIDs[id] = struct{}{}
	}
	return actor, nil
}

// Me returns the signed-in user's profile.
func (s *AuthService) Me(ctx context.Context, actor domain.Actor) (domain.User, error) {
	return s.users.GetByID(ctx, actor.UserID)
}

// MFASetup is a fresh authenticator secret waiting to be confirmed.
type MFASetup struct {
	Secret string
	URL    string
}

// SetupMFA creates a secret for the user. Two-step verification only turns on
// once EnableMFA proves the app produces a valid code.
func (s *AuthService) SetupMFA(ctx context.Context, actor domain.Actor) (MFASetup, error) {
	user, err := s.users.GetByID(ctx, actor.UserID)
	if err != nil {
		return MFASetup{}, err
	}
	if user.TOTPEnabled {
		return MFASetup{}, fmt.Errorf("%w: two-step verification is already on", apperrors.ErrConflict)
	}
	secret, err := s.totp.NewSecret()
	if err != nil {
		return MFASetup{}, err
	}
	sealed, err := s.box.Seal(secret)
	if err != nil {
		return MFASetup{}, fmt.Errorf("sealing totp secret: %w", err)
	}
	if err := s.users.SetTOTP(ctx, user.ID, sealed, false); err != nil {
		return MFASetup{}, err
	}
	return MFASetup{Secret: secret, URL: s.totp.URL(user.Email, secret)}, nil
}

// EnableMFA turns two-step verification on after checking a code.
func (s *AuthService) EnableMFA(ctx context.Context, actor domain.Actor, code string) error {
	user, err := s.users.GetByID(ctx, actor.UserID)
	if err != nil {
		return err
	}
	if user.TOTPEnabled {
		return nil
	}
	if user.TOTPSecret == "" {
		return invalid("start two-step setup first")
	}
	secret, err := s.box.Open(user.TOTPSecret)
	if err != nil {
		return fmt.Errorf("opening totp secret: %w", err)
	}
	if !s.totp.Verify(secret, code, s.clock.Now()) {
		return apperrors.ErrInvalidCode
	}
	if err := s.users.SetTOTP(ctx, user.ID, user.TOTPSecret, true); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, domain.AuditEvent{ActorID: user.ID, Action: "auth.mfa_enabled", EntityType: "user", EntityID: user.ID})
	return nil
}
