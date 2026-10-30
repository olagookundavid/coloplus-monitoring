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
	// recovery, tx and notifier serve the one-time recovery codes: using one
	// is a single transaction that also audits it and warns the owner.
	recovery ports.RecoveryCodeRepository
	tx       ports.TxManager
	notifier *NotificationService
	audit    *AuditService
	clock    ports.Clock
	log      *slog.Logger
	ttl      time.Duration

	// dummyHash lets a lookup for an unknown email cost the same as a real
	// one, so response time does not reveal which emails have accounts.
	dummyHash string
}

func NewAuthService(
	users ports.UserRepository, hasher ports.PasswordHasher, tokens ports.TokenIssuer,
	totp ports.TOTP, box ports.SecretBox, recovery ports.RecoveryCodeRepository, tx ports.TxManager,
	notifier *NotificationService, audit *AuditService, clock ports.Clock,
	ttl time.Duration, log *slog.Logger,
) (*AuthService, error) {
	dummy, err := hasher.Hash("not-a-real-password-0000")
	if err != nil {
		return nil, fmt.Errorf("preparing auth service: %w", err)
	}
	return &AuthService{
		users: users, hasher: hasher, tokens: tokens, totp: totp, box: box,
		recovery: recovery, tx: tx, notifier: notifier,
		audit: audit, clock: clock, ttl: ttl, log: log, dummyHash: dummy,
	}, nil
}

// LoginResult is what a sign-in attempt produced. When MFARequired is set the
// password was right but a code is still needed; no token is issued. When
// MFASetupRequired is set the person is signed in but may do nothing except
// turn two-step verification on.
type LoginResult struct {
	MFARequired      bool
	MFASetupRequired bool
	Token            string
	ExpiresAt        time.Time
	User             domain.User
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

	setupRequired := false
	if user.TOTPEnabled {
		if code == "" {
			return LoginResult{MFARequired: true}, nil
		}
		if err := s.checkSecondStep(ctx, user, code, now); err != nil {
			return LoginResult{}, err
		}
	} else if setupRequired, err = s.users.RoleRequiresMFA(ctx, user.Role); err != nil {
		return LoginResult{}, fmt.Errorf("checking two-step policy: %w", err)
	}

	token, expires, err := s.tokens.Issue(user.ID, s.ttl)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issuing token: %w", err)
	}
	if err := s.users.RecordLoginSuccess(ctx, user.ID, now); err != nil {
		return LoginResult{}, fmt.Errorf("recording sign-in: %w", err)
	}
	_ = s.audit.Record(ctx, domain.AuditEvent{ActorID: user.ID, Action: "auth.login", EntityType: "user", EntityID: user.ID})
	return LoginResult{Token: token, ExpiresAt: expires, User: user, MFASetupRequired: setupRequired}, nil
}

// checkSecondStep accepts an authenticator code, or one of the person's
// recovery codes. A wrong code counts towards the lockout like a wrong password.
func (s *AuthService) checkSecondStep(ctx context.Context, user domain.User, code string, now time.Time) error {
	if domain.LooksLikeRecoveryCode(code) {
		return s.useRecoveryCode(ctx, user, code, now)
	}
	secret, err := s.box.Open(user.TOTPSecret)
	if err != nil {
		return fmt.Errorf("opening totp secret: %w", err)
	}
	if !s.totp.Verify(secret, code, now) {
		return s.failLogin(ctx, user, now, "bad_code")
	}
	return nil
}

// useRecoveryCode spends a recovery code. The owner is audited and emailed
// each time: a code used by someone else must not go unnoticed.
func (s *AuthService) useRecoveryCode(ctx context.Context, user domain.User, code string, now time.Time) error {
	used := false
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		ok, err := s.recovery.Consume(ctx, user.ID, domain.HashRecoveryCode(code), now)
		if err != nil || !ok {
			return err
		}
		used = true
		left, err := s.recovery.Remaining(ctx, user.ID)
		if err != nil {
			return err
		}
		subject, body := recoveryCodeUsedEmail(user.FullName, left)
		if _, err := s.notifier.Enqueue(ctx, domain.Notification{
			Kind: "mfa-recovery-used", Recipient: user.Email, Subject: subject, Body: body,
			DedupeKey: fmt.Sprintf("mfa-recovery-used:%s:%d", user.ID, now.UnixNano()),
		}); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: user.ID, Action: "auth.recovery_code_used", EntityType: "user", EntityID: user.ID,
			Metadata: map[string]any{"remaining": left},
		})
	})
	if err != nil {
		return fmt.Errorf("using recovery code: %w", err)
	}
	if !used {
		return s.failLogin(ctx, user, now, "bad_code")
	}
	return nil
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

// sessionSkew tolerates a small clock difference between this process and the
// database, which stamps password changes made by an administrator.
const sessionSkew = 2 * time.Second

// Authenticate builds the Actor for a verified token's user. A deactivated
// user is refused here, so revoking access takes effect on the next request,
// and so is a token issued before the person's password last changed.
func (s *AuthService) Authenticate(ctx context.Context, userID string, issuedAt time.Time) (domain.Actor, error) {
	user, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, apperrors.ErrNotFound) || (err == nil && !user.Active) {
		return domain.Actor{}, apperrors.ErrUnauthorized
	}
	if err != nil {
		return domain.Actor{}, fmt.Errorf("loading user: %w", err)
	}
	if issuedAt.Add(sessionSkew).Before(user.PasswordChangedAt) {
		return domain.Actor{}, apperrors.ErrUnauthorized
	}

	// A role that needs two-step sign-in holds no permissions until the
	// person has turned it on, so every service refuses them by default.
	if !user.TOTPEnabled {
		required, err := s.users.RoleRequiresMFA(ctx, user.Role)
		if err != nil {
			return domain.Actor{}, fmt.Errorf("checking two-step policy: %w", err)
		}
		if required {
			return domain.Actor{
				UserID: user.ID, Role: user.Role, MFASetupRequired: true,
				Permissions: map[domain.Permission]struct{}{}, SiteIDs: map[string]struct{}{},
			}, nil
		}
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

// EnableMFA turns two-step verification on after checking a code and returns
// the recovery codes. They are shown once: only their hashes are kept.
func (s *AuthService) EnableMFA(ctx context.Context, actor domain.Actor, code string) ([]string, error) {
	user, err := s.users.GetByID(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}
	if user.TOTPEnabled {
		return nil, nil
	}
	if user.TOTPSecret == "" {
		return nil, invalid("start two-step setup first")
	}
	now := s.clock.Now()
	if user.IsLocked(now) {
		return nil, apperrors.ErrAccountLocked
	}
	secret, err := s.box.Open(user.TOTPSecret)
	if err != nil {
		return nil, fmt.Errorf("opening totp secret: %w", err)
	}
	if !s.totp.Verify(secret, code, now) {
		return nil, s.failLogin(ctx, user, now, "bad_code")
	}
	codes, err := domain.NewRecoveryCodes()
	if err != nil {
		return nil, err
	}
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.users.SetTOTP(ctx, user.ID, user.TOTPSecret, true); err != nil {
			return err
		}
		if err := s.recovery.Replace(ctx, user.ID, hashAll(codes)); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{ActorID: user.ID, Action: "auth.mfa_enabled", EntityType: "user", EntityID: user.ID})
	})
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// NewRecoveryCodes replaces the person's recovery codes with a fresh set. It
// needs a current authenticator code, so a stolen session cannot mint codes.
func (s *AuthService) NewRecoveryCodes(ctx context.Context, actor domain.Actor, code string) ([]string, error) {
	user, err := s.users.GetByID(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}
	if !user.TOTPEnabled {
		return nil, invalid("turn on two-step verification first")
	}
	now := s.clock.Now()
	if user.IsLocked(now) {
		return nil, apperrors.ErrAccountLocked
	}
	secret, err := s.box.Open(user.TOTPSecret)
	if err != nil {
		return nil, fmt.Errorf("opening totp secret: %w", err)
	}
	if !s.totp.Verify(secret, code, now) {
		return nil, s.failLogin(ctx, user, now, "bad_code")
	}
	codes, err := domain.NewRecoveryCodes()
	if err != nil {
		return nil, err
	}
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.recovery.Replace(ctx, user.ID, hashAll(codes)); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{ActorID: user.ID, Action: "auth.recovery_codes_renewed", EntityType: "user", EntityID: user.ID})
	})
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// RecoveryCodesLeft is how many unused recovery codes the person has.
func (s *AuthService) RecoveryCodesLeft(ctx context.Context, actor domain.Actor) (int, error) {
	return s.recovery.Remaining(ctx, actor.UserID)
}

func hashAll(codes []string) []string {
	out := make([]string, len(codes))
	for i, c := range codes {
		out[i] = domain.HashRecoveryCode(c)
	}
	return out
}
