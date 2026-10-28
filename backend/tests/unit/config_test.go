package unit

import (
	"strings"
	"testing"

	"coloplus-backend/config"
)

func lookupFrom(env map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := env[k]; return v, ok }
}

func validEnv() map[string]string {
	return map[string]string{
		"DATABASE_URL":        "postgres://u:p@localhost:5432/db",
		"TOKEN_SYMMETRIC_KEY": "12345678901234567890123456789012",
		"AUTH_COOKIE_NAME":    "coloplus_access",
		"SECRETS_KEY":         "abcdefghijklmnopqrstuvwxyz012345",
	}
}

func TestConfig_ReportsEveryProblemAtOnce(t *testing.T) {
	_, err := config.Load(lookupFrom(map[string]string{
		"TOKEN_SYMMETRIC_KEY": "short",
		"DB_MAX_CONNS":        "lots",
	}))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"DATABASE_URL", "TOKEN_SYMMETRIC_KEY", "AUTH_COOKIE_NAME", "SECRETS_KEY", "DB_MAX_CONNS"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s: %v", want, err)
		}
	}
}

func TestConfig_ProductionGuards(t *testing.T) {
	env := validEnv()
	env["APP_ENV"] = "production"
	env["SWAGGER_ENABLED"] = "true"
	env["RATE_LIMIT_ENABLED"] = "false"
	_, err := config.Load(lookupFrom(env))
	if err == nil {
		t.Fatal("expected production guards to fail")
	}
	for _, want := range []string{"CORS_ALLOW_ORIGINS", "RATE_LIMIT_ENABLED", "SWAGGER_ENABLED", "MAIL_PROVIDER"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s: %v", want, err)
		}
	}

	env["SWAGGER_ENABLED"] = "false"
	env["RATE_LIMIT_ENABLED"] = "true"
	env["CORS_ALLOW_ORIGINS"] = "https://app.example.com"
	env["MAIL_PROVIDER"] = "resend"
	env["RESEND_API_KEY"] = "re_test"
	env["MAIL_FROM"] = "Coloplus <alerts@example.com>"
	env["APP_BASE_URL"] = "https://app.example.com"
	if _, err := config.Load(lookupFrom(env)); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
}
