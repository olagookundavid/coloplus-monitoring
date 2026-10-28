package totp

import (
	"strings"
	"testing"
	"time"
)

// RFC 6238 appendix B test secret ("12345678901234567890") in base32.
const rfcSecret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

func TestCode_RFC6238Vectors(t *testing.T) {
	// The RFC lists 8-digit codes; the last six digits are the 6-digit code.
	cases := map[int64]string{
		59:         "287082",
		1111111109: "081804",
		1111111111: "050471",
		1234567890: "005924",
		2000000000: "279037",
	}
	for unix, want := range cases {
		got, err := Code(rfcSecret, time.Unix(unix, 0))
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("t=%d: code = %s, want %s", unix, got, want)
		}
	}
}

func TestVerify(t *testing.T) {
	now := time.Unix(1234567890, 0)
	good, _ := Code(rfcSecret, now)
	prev, _ := Code(rfcSecret, now.Add(-30*time.Second))
	old, _ := Code(rfcSecret, now.Add(-5*time.Minute))

	cases := []struct {
		name string
		code string
		want bool
	}{
		{"current window", good, true},
		{"previous window (clock drift)", prev, true},
		{"five minutes old", old, false},
		{"wrong code", "000000", false},
		{"too short", "123", false},
		{"empty", "", false},
		{"surrounding spaces are tolerated", " " + good + " ", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Verify(rfcSecret, tc.code, now); got != tc.want {
				t.Fatalf("Verify = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestVerify_BadSecret(t *testing.T) {
	if Verify("not base32 !!", "123456", time.Now()) {
		t.Fatal("a malformed secret must never verify")
	}
}

func TestNewSecretAndURL(t *testing.T) {
	s1, err := NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	s2, _ := NewSecret()
	if s1 == s2 || len(s1) != 32 {
		t.Fatalf("secrets should be unique 32-char base32 strings, got %q and %q", s1, s2)
	}
	u := URL("Coloplus", "ada@example.com", s1)
	for _, want := range []string{"otpauth://totp/", "secret=" + s1, "issuer=Coloplus"} {
		if !strings.Contains(u, want) {
			t.Errorf("url %q is missing %q", u, want)
		}
	}
}
