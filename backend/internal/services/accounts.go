package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

const (
	inviteTTL = 72 * time.Hour
	resetTTL  = time.Hour
	// A second reset is not mailed within this window, so the form cannot be
	// used to flood someone's inbox.
	resetCooldown = 5 * time.Minute
)

// AccountService issues the one-time links that let a person choose or
// recover a password, and redeems them.
type AccountService struct {
	users    ports.UserRepository
	tokens   ports.UserTokenRepository
	hasher   ports.PasswordHasher
	tx       ports.TxManager
	notifier *NotificationService
	audit    *AuditService
	clock    ports.Clock
	baseURL  string
}

// NewAccountService builds the service. baseURL is where the front end lives;
// links point at /accept-invite and /reset-password under it.
func NewAccountService(
	users ports.UserRepository, tokens ports.UserTokenRepository, hasher ports.PasswordHasher,
	tx ports.TxManager, notifier *NotificationService, audit *AuditService, clock ports.Clock, baseURL string,
) *AccountService {
	return &AccountService{
		users: users, tokens: tokens, hasher: hasher, tx: tx, notifier: notifier,
		audit: audit, clock: clock, baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("creating token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *AccountService) link(path, raw string) string {
	return s.baseURL + path + "?token=" + url.QueryEscape(raw)
}

// issue makes a token and queues the email that carries it. It must run in a
// transaction, so the link exists if and only if the change that caused it
// committed.
func (s *AccountService) issue(ctx context.Context, user domain.User, purpose domain.TokenPurpose) error {
	raw, err := newToken()
	if err != nil {
		return err
	}
	now := s.clock.Now()
	ttl, path, kind := inviteTTL, "/accept-invite", "account.invite"
	if purpose == domain.PurposeReset {
		ttl, path, kind = resetTTL, "/reset-password", "account.reset"
	}
	if err := s.tokens.RevokeOutstanding(ctx, user.ID, purpose, now); err != nil {
		return err
	}
	tok, err := s.tokens.Create(ctx, domain.UserToken{UserID: user.ID, Purpose: purpose, ExpiresAt: now.Add(ttl)}, hashToken(raw))
	if err != nil {
		return err
	}
	subject, body := inviteEmail(user.FullName, s.link(path, raw), ttl)
	if purpose == domain.PurposeReset {
		subject, body = resetEmail(user.FullName, s.link(path, raw), ttl)
	}
	_, err = s.notifier.Enqueue(ctx, domain.Notification{
		Kind: kind, Recipient: user.Email, Subject: subject, Body: body, DedupeKey: kind + ":" + tok.ID,
	})
	return err
}

// Invite sends a new or still-unregistered person a link to choose their
// password. Call it inside the transaction that created the user.
func (s *AccountService) Invite(ctx context.Context, user domain.User) error {
	return s.issue(ctx, user, domain.PurposeInvite)
}

// RequestReset mails a reset link when the address belongs to an active
// account. It answers the same way either way, so the form cannot be used to
// find out who has an account.
func (s *AccountService) RequestReset(ctx context.Context, email string) error {
	user, err := s.users.GetByEmail(ctx, domain.NormalizeEmail(email))
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !user.Active {
		return nil
	}
	now := s.clock.Now()
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		recent, err := s.tokens.CountSince(ctx, user.ID, domain.PurposeReset, now.Add(-resetCooldown))
		if err != nil || recent > 0 {
			return err
		}
		if err := s.issue(ctx, user, domain.PurposeReset); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			Action: "auth.reset_requested", EntityType: "user", EntityID: user.ID,
		})
	})
}

// SetPassword redeems an invitation or reset link. The link works once.
// Choosing a password ends every session that existed before, and clears any
// sign-in lockout.
func (s *AccountService) SetPassword(ctx context.Context, rawToken, password string) error {
	if err := domain.ValidatePassword(password); err != nil {
		return invalid("%s", err.Error())
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		now := s.clock.Now()
		tok, err := s.tokens.GetByHash(ctx, hashToken(strings.TrimSpace(rawToken)))
		if errors.Is(err, apperrors.ErrNotFound) {
			return apperrors.ErrInvalidLink
		}
		if err != nil {
			return err
		}
		if !tok.Usable(now) {
			return apperrors.ErrInvalidLink
		}
		user, err := s.users.GetByID(ctx, tok.UserID)
		if err != nil {
			return err
		}
		if !user.Active {
			return apperrors.ErrInvalidLink
		}
		if err := s.users.SetPassword(ctx, user.ID, hash, now); err != nil {
			return err
		}
		if err := s.tokens.MarkUsed(ctx, tok.ID, now); err != nil {
			return err
		}
		// Any other outstanding link for this person is now stale.
		for _, p := range []domain.TokenPurpose{domain.PurposeInvite, domain.PurposeReset} {
			if err := s.tokens.RevokeOutstanding(ctx, user.ID, p, now); err != nil {
				return err
			}
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: user.ID, Action: "auth.password_set", EntityType: "user", EntityID: user.ID,
			Metadata: map[string]any{"via": string(tok.Purpose)},
		})
	})
}
