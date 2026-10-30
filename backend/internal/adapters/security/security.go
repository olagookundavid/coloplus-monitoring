// Package security adapts password hashing, tokens, TOTP and secret
// encryption to the ports the services use.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"pkg-common/token"
	"pkg-common/totp"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher implements ports.PasswordHasher.
type BcryptHasher struct{ cost int }

func NewBcryptHasher(cost int) *BcryptHasher { return &BcryptHasher{cost: cost} }

func (h *BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(b), nil
}

func (h *BcryptHasher) Matches(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// PasetoIssuer implements ports.TokenIssuer.
type PasetoIssuer struct{ maker token.Maker }

func NewPasetoIssuer(maker token.Maker) *PasetoIssuer { return &PasetoIssuer{maker: maker} }

func (i *PasetoIssuer) Issue(userID string, ttl time.Duration) (string, time.Time, error) {
	tok, payload, err := i.maker.CreateToken(userID, ttl)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("creating token: %w", err)
	}
	return tok, payload.ExpiredAt, nil
}

// TOTP implements ports.TOTP. Issuer is the name shown in authenticator apps.
type TOTP struct{ issuer string }

func NewTOTP(issuer string) *TOTP { return &TOTP{issuer: issuer} }

func (t *TOTP) NewSecret() (string, error) { return totp.NewSecret() }

func (t *TOTP) Verify(secret, code string, now time.Time) bool { return totp.Verify(secret, code, now) }

func (t *TOTP) URL(account, secret string) string { return totp.URL(t.issuer, account, secret) }

// SecretBox implements ports.SecretBox with AES-256-GCM.
type SecretBox struct{ aead cipher.AEAD }

func NewSecretBox(key string) (*SecretBox, error) {
	if len(key) != 32 {
		return nil, errors.New("secret box key must be exactly 32 bytes")
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating gcm: %w", err)
	}
	return &SecretBox{aead: aead}, nil
}

func (b *SecretBox) Seal(plaintext string) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("creating nonce: %w", err)
	}
	sealed := b.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (b *SecretBox) Open(sealed string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(sealed)
	if err != nil {
		return "", fmt.Errorf("decoding sealed value: %w", err)
	}
	if len(raw) < b.aead.NonceSize() {
		return "", errors.New("sealed value is too short")
	}
	nonce, ct := raw[:b.aead.NonceSize()], raw[b.aead.NonceSize():]
	plain, err := b.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("opening sealed value: %w", err)
	}
	return string(plain), nil
}
