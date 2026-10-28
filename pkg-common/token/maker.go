package token

import "time"

// Maker creates and verifies access tokens.
type Maker interface {
	// CreateToken issues a token for a user that is valid for duration.
	CreateToken(userID string, duration time.Duration) (string, *Payload, error)

	// VerifyToken checks that a token is genuine and unexpired.
	VerifyToken(token string) (*Payload, error)
}

// Payload is the data carried inside a token.
type Payload struct {
	UserID    string    `json:"user_id"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

// Valid reports whether the payload is still within its lifetime.
func (p *Payload) Valid(now time.Time) error {
	if now.After(p.ExpiredAt) {
		return ErrExpiredToken
	}
	return nil
}
