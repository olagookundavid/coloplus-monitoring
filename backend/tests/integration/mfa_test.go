package integration

import (
	"context"
	"strings"
	"testing"

	"coloplus-backend/internal/domain"
)

// Two-step sign-in is required by role. Until a required person has turned it
// on they hold no permissions at all, a recovery code works exactly once and
// warns its owner, new codes retire the old ones, and a reset is done by
// someone else.
func TestMFA_RequiredRolesAreLockedToSetupAndRecoveryCodesWorkOnce(t *testing.T) {
	s := newStack(t)
	adminEmail, adminID := s.seedUser(domain.RoleAdmin)
	admin := s.login(adminEmail)
	admin2Email, _ := s.seedUser(domain.RoleAdmin)
	admin2 := s.login(admin2Email)

	// A supervisor who has not set it up signs in, but can do nothing else.
	supEmail, supID := s.seedUserWithoutMFA(domain.RoleSupervisor)
	first := s.expect(s.call("POST", "/api/v1/auth/login", map[string]any{"email": supEmail, "password": testPassword}, ""), 200)
	tok := first.str("access_token")
	if first.Body["mfa_setup_required"] != true {
		t.Fatalf("login should say setup is required: %s", first.Raw)
	}
	me := s.expect(s.call("GET", "/api/v1/auth/me", nil, tok), 200)
	if me.Body["mfa_setup_required"] != true || len(me.list("permissions")) != 0 {
		t.Fatalf("until setup the supervisor must hold no permissions: %s", me.Raw)
	}
	s.expect(s.call("GET", "/api/v1/sites", nil, tok), 403)
	s.expect(s.call("GET", "/api/v1/daily-reports", nil, tok), 403)

	setup := s.expect(s.call("POST", "/api/v1/auth/mfa/setup", nil, tok), 200)
	s.authenticator[supEmail] = setup.str("secret")
	enabled := s.expect(s.call("POST", "/api/v1/auth/mfa/enable", map[string]any{"code": s.codeFor(supEmail)}, tok), 200)
	codes := enabled.list("recovery_codes")
	if len(codes) != domain.RecoveryCodeCount {
		t.Fatalf("recovery codes = %d, want %d", len(codes), domain.RecoveryCodeCount)
	}
	s.expect(s.call("GET", "/api/v1/sites", nil, tok), 200) // the same session now works
	var stored string
	if err := s.pool.QueryRow(context.Background(),
		`SELECT string_agg(code_hash, ',') FROM mfa_recovery_codes WHERE user_id = $1`, supID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	for _, c := range codes {
		if strings.Contains(stored, c.(string)) {
			t.Fatal("recovery codes must be stored hashed, never in clear")
		}
	}

	// A recovery code signs in once, however it is typed, and warns its owner.
	signIn := func(code string) reply {
		return s.call("POST", "/api/v1/auth/login", map[string]any{"email": supEmail, "password": testPassword, "totp_code": code}, "")
	}
	typed := strings.ToLower(strings.ReplaceAll(codes[0].(string), "-", ""))
	s.expect(signIn(typed), 200)
	s.expect(signIn(codes[0].(string)), 401)
	s.expect(signIn(codes[1].(string)), 200)
	if _, err := s.notifications.Dispatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := s.mailer.to(supEmail); len(got) != 2 || !strings.Contains(got[0].Subject, "recovery code") {
		t.Fatalf("the owner should be warned each time a recovery code is used: %+v", got)
	}
	if n := s.countAudit("auth.recovery_code_used", supID); n != 2 {
		t.Fatalf("recovery use audited %d times, want 2", n)
	}

	// New codes need a current authenticator code and retire the old set.
	tok = s.login(supEmail)
	s.expect(s.call("POST", "/api/v1/auth/mfa/recovery-codes", map[string]any{"code": "000000"}, tok), 401)
	s.expect(s.call("POST", "/api/v1/auth/mfa/recovery-codes", map[string]any{"code": codes[2]}, tok), 401) // a recovery code is not enough
	fresh := s.expect(s.call("POST", "/api/v1/auth/mfa/recovery-codes", map[string]any{"code": s.codeFor(supEmail)}, tok), 200).list("recovery_codes")
	s.expect(signIn(codes[2].(string)), 401)
	s.expect(signIn(fresh[0].(string)), 200)

	// Resetting is done by another administrator, never by oneself. After it
	// the old session is limited to setup again and the codes are gone.
	s.expect(s.call("POST", "/api/v1/users/"+adminID+"/reset-mfa", nil, admin), 403)
	s.expect(s.call("POST", "/api/v1/users/"+supID+"/reset-mfa", nil, tok), 403)
	s.expect(s.call("POST", "/api/v1/users/00000000-0000-0000-0000-000000000000/reset-mfa", nil, admin2), 404)
	s.expect(s.call("POST", "/api/v1/users/"+supID+"/reset-mfa", nil, admin2), 204)
	s.expect(s.call("GET", "/api/v1/sites", nil, tok), 403)
	again := s.expect(s.call("POST", "/api/v1/auth/login", map[string]any{"email": supEmail, "password": testPassword}, ""), 200)
	if again.Body["mfa_setup_required"] != true {
		t.Fatalf("after a reset the person must set it up again: %s", again.Raw)
	}
	if n := s.countAudit("auth.mfa_reset", supID); n != 1 {
		t.Fatalf("reset audited %d times, want 1", n)
	}

	// A role that does not require it can carry on without.
	repEmail, _ := s.seedUserWithoutMFA(domain.RoleReporter)
	rep := s.expect(s.call("POST", "/api/v1/auth/login", map[string]any{"email": repEmail, "password": testPassword}, ""), 200)
	if rep.Body["mfa_setup_required"] == true {
		t.Fatalf("a reporter does not need two-step sign-in: %s", rep.Raw)
	}
}
