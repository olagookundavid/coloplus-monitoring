// Package totp implements RFC 6238 time-based one-time passwords with the
// standard library only, so authenticator apps (Google Authenticator, Authy,
// 1Password) can be used for two-step login.
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // RFC 6238 authenticator apps use HMAC-SHA1.
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	period = 30 * time.Second
	digits = 6
)

var encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewSecret returns a random 160-bit secret as base32, the form authenticator
// apps expect.
func NewSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating totp secret: %w", err)
	}
	return encoding.EncodeToString(buf), nil
}

// Code returns the six-digit code for the secret at time t.
func Code(secret string, t time.Time) (string, error) {
	key, err := encoding.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", fmt.Errorf("decoding totp secret: %w", err)
	}
	return hotp(key, uint64(t.Unix())/uint64(period.Seconds())), nil
}

// Verify checks a code against the current 30-second window and one window
// either side, to allow for clock drift between phone and server.
func Verify(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != digits {
		return false
	}
	ok := false
	for _, drift := range []time.Duration{-period, 0, period} {
		want, err := Code(secret, now.Add(drift))
		if err != nil {
			return false
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			ok = true
		}
	}
	return ok
}

// URL builds the otpauth:// link that authenticator apps read from a QR code.
func URL(issuer, account, secret string) string {
	v := url.Values{}
	v.Set("secret", secret)
	v.Set("issuer", issuer)
	v.Set("algorithm", "SHA1")
	v.Set("digits", fmt.Sprint(digits))
	v.Set("period", fmt.Sprint(int(period.Seconds())))
	label := url.PathEscape(issuer + ":" + account)
	return "otpauth://totp/" + label + "?" + v.Encode()
}

func hotp(key []byte, counter uint64) string {
	mac := hmac.New(sha1.New, key)
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac.Write(msg[:])
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	bin := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])
	return fmt.Sprintf("%0*d", digits, bin%1_000_000)
}
