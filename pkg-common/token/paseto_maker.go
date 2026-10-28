package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/aead/chacha20poly1305"
	"github.com/o1egl/paseto"
)

var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
)

// PasetoMaker issues and verifies PASETO v2 local tokens.
type PasetoMaker struct {
	paseto       *paseto.V2
	symmetricKey []byte
	now          func() time.Time
}

// NewPasetoMaker builds a maker from a symmetric key of exactly 32 bytes.
func NewPasetoMaker(symmetricKey string) (Maker, error) {
	if len(symmetricKey) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("invalid key size: must be exactly %d characters", chacha20poly1305.KeySize)
	}
	return &PasetoMaker{
		paseto:       paseto.NewV2(),
		symmetricKey: []byte(symmetricKey),
		now:          time.Now,
	}, nil
}

func (m *PasetoMaker) CreateToken(userID string, duration time.Duration) (string, *Payload, error) {
	now := m.now()
	payload := &Payload{UserID: userID, IssuedAt: now, ExpiredAt: now.Add(duration)}
	tok, err := m.paseto.Encrypt(m.symmetricKey, payload, nil)
	return tok, payload, err
}

func (m *PasetoMaker) VerifyToken(tok string) (*Payload, error) {
	payload := &Payload{}
	if err := m.paseto.Decrypt(tok, m.symmetricKey, payload, nil); err != nil {
		return nil, ErrInvalidToken
	}
	if err := payload.Valid(m.now()); err != nil {
		return nil, err
	}
	return payload, nil
}
