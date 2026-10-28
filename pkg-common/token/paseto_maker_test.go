package token

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const testKey = "12345678901234567890123456789012"

func TestPasetoMaker_RoundTrip(t *testing.T) {
	maker, err := NewPasetoMaker(testKey)
	if err != nil {
		t.Fatal(err)
	}
	tok, _, err := maker.CreateToken("user-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	got, err := maker.VerifyToken(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.UserID != "user-1" {
		t.Fatalf("user id = %q, want user-1", got.UserID)
	}
}

func TestPasetoMaker_Rejects(t *testing.T) {
	maker, _ := NewPasetoMaker(testKey)
	other, _ := NewPasetoMaker(strings.Repeat("x", 32))

	expired, _, _ := maker.CreateToken("user-1", -time.Minute)
	foreign, _, _ := other.CreateToken("user-1", time.Minute)

	cases := []struct {
		name  string
		token string
		want  error
	}{
		{"expired", expired, ErrExpiredToken},
		{"signed with another key", foreign, ErrInvalidToken},
		{"garbage", "not-a-token", ErrInvalidToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := maker.VerifyToken(tc.token); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestNewPasetoMaker_KeySize(t *testing.T) {
	if _, err := NewPasetoMaker("short"); err == nil {
		t.Fatal("expected an error for a short key")
	}
}
