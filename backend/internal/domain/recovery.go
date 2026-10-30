package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	// RecoveryCodeCount is how many one-time codes a person gets at a time.
	RecoveryCodeCount = 10

	recoveryCodeLength   = 12
	recoveryCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 32 symbols, no look-alikes
)

// NewRecoveryCodes returns a fresh set, shown to the person once, written as
// XXXX-XXXX-XXXX.
func NewRecoveryCodes() ([]string, error) {
	codes := make([]string, RecoveryCodeCount)
	for i := range codes {
		b := make([]byte, recoveryCodeLength)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("generating recovery code: %w", err)
		}
		var sb strings.Builder
		for j, v := range b {
			if j > 0 && j%4 == 0 {
				sb.WriteByte('-')
			}
			// 256 is a multiple of 32, so the choice is uniform.
			sb.WriteByte(recoveryCodeAlphabet[int(v)%len(recoveryCodeAlphabet)])
		}
		codes[i] = sb.String()
	}
	return codes, nil
}

// NormalizeRecoveryCode drops spaces and dashes and upper-cases what a person
// typed, so formatting never costs them a code.
func NormalizeRecoveryCode(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	return strings.NewReplacer("-", "", " ", "").Replace(s)
}

// LooksLikeRecoveryCode tells a recovery code from a six-digit authenticator
// code.
func LooksLikeRecoveryCode(s string) bool {
	return len(NormalizeRecoveryCode(s)) == recoveryCodeLength
}

// HashRecoveryCode is what is stored in place of the code.
func HashRecoveryCode(s string) string {
	sum := sha256.Sum256([]byte(NormalizeRecoveryCode(s)))
	return hex.EncodeToString(sum[:])
}
