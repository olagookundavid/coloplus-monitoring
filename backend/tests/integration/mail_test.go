package integration

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
)

var tokenInLink = regexp.MustCompile(`token=([A-Za-z0-9_\-%]+)`)

func (s *stack) dispatch() {
	s.t.Helper()
	if _, err := s.notifications.Dispatch(context.Background()); err != nil {
		s.t.Fatal(err)
	}
}

// linkToken digs the one-time token out of the latest email to an address.
func (s *stack) linkToken(addr string) string {
	s.t.Helper()
	mails := s.mailer.to(addr)
	if len(mails) == 0 {
		s.t.Fatalf("no email was sent to %s", addr)
	}
	m := tokenInLink.FindStringSubmatch(mails[len(mails)-1].Text)
	if m == nil {
		s.t.Fatalf("no link in email to %s: %q", addr, mails[len(mails)-1].Text)
	}
	return m[1]
}

func (s *stack) exec(sql string, args ...any) {
	s.t.Helper()
	if _, err := s.pool.Exec(context.Background(), sql, args...); err != nil {
		s.t.Fatal(err)
	}
}

// Invitation and reset links are the way into an account, so they must be
// single-use, expire, not reveal who has an account, and end older sessions.
func TestAccount_LinksAreSingleUseExpireAndEndOldSessions(t *testing.T) {
	s := newStack(t)
	adminEmail, _ := s.seedUser(domain.RoleAdmin)
	admin := s.login(adminEmail)

	email := uniq("invitee") + "@coloplus.example"
	created := s.expect(s.call("POST", "/api/v1/users", map[string]any{"email": email, "full_name": "New Person", "role": "reporter"}, admin), 201)
	if created.Body["invited"] != true {
		t.Fatalf("a user created without a password should be invited: %s", created.Raw)
	}
	s.dispatch()
	tok := s.linkToken(email)

	// Cannot sign in before choosing a password; a weak one does not burn the link.
	s.expect(s.call("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": testPassword}, ""), 401)
	s.expect(s.call("POST", "/api/v1/auth/set-password", map[string]any{"token": tok, "password": "short"}, ""), 400)
	s.expect(s.call("POST", "/api/v1/auth/set-password", map[string]any{"token": tok, "password": testPassword}, ""), 200)
	s.expect(s.call("POST", "/api/v1/auth/set-password", map[string]any{"token": tok, "password": testPassword + "x"}, ""), 400) // used
	oldSession := s.login(email)
	s.expect(s.call("GET", "/api/v1/auth/me", nil, oldSession), 200)

	// A reset answers the same for an unknown address and sends nothing to it.
	s.expect(s.call("POST", "/api/v1/auth/forgot-password", map[string]any{"email": "nobody@coloplus.example"}, ""), 202)
	s.expect(s.call("POST", "/api/v1/auth/forgot-password", map[string]any{"email": email}, ""), 202)
	s.expect(s.call("POST", "/api/v1/auth/forgot-password", map[string]any{"email": email}, ""), 202) // within the cooldown
	s.dispatch()
	if n := len(s.mailer.to("nobody@coloplus.example")); n != 0 {
		t.Fatalf("an unknown address was emailed %d times", n)
	}
	if n := len(s.mailer.to(email)); n != 2 { // the invitation and one reset
		t.Fatalf("expected the invitation and a single reset email, got %d emails", n)
	}

	// Choosing a new password ends the session that existed before.
	time.Sleep(2100 * time.Millisecond) // beyond the clock tolerance
	newPassword := testPassword + "-new"
	s.expect(s.call("POST", "/api/v1/auth/set-password", map[string]any{"token": s.linkToken(email), "password": newPassword}, ""), 200)
	s.expect(s.call("GET", "/api/v1/auth/me", nil, oldSession), 401)
	s.expect(s.call("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": testPassword}, ""), 401)
	s.expect(s.call("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": newPassword}, ""), 200)

	// An expired link is refused.
	other := uniq("late") + "@coloplus.example"
	s.expect(s.call("POST", "/api/v1/users", map[string]any{"email": other, "full_name": "Late", "role": "reporter"}, admin), 201)
	s.dispatch()
	s.exec(`UPDATE user_tokens SET expires_at = now() - interval '1 hour'
	        WHERE user_id = (SELECT id FROM users WHERE email = $1)`, other)
	s.expect(s.call("POST", "/api/v1/auth/set-password", map[string]any{"token": s.linkToken(other), "password": testPassword}, ""), 400)
}

// Email must survive a provider outage without being lost or sent twice,
// and a failure must be visible to an administrator.
func TestNotifications_RetryFailAndNeverDuplicate(t *testing.T) {
	s := newStack(t)
	adminEmail, _ := s.seedUser(domain.RoleAdmin)
	admin := s.login(adminEmail)
	repEmail, _ := s.seedUser(domain.RoleReporter)
	reporter := s.login(repEmail)
	ctx := context.Background()

	n := func(key string) domain.Notification {
		return domain.Notification{Kind: "test", Recipient: uniq("r") + "@coloplus.example", Subject: "hi", Body: "body", DedupeKey: key}
	}

	// The same reason to email is queued once.
	first := n(uniq("dup"))
	if ok, err := s.notifications.Enqueue(ctx, first); err != nil || !ok {
		t.Fatalf("first enqueue = %v, %v", ok, err)
	}
	if ok, _ := s.notifications.Enqueue(ctx, first); ok {
		t.Fatal("the same dedupe key was queued twice")
	}

	// A provider outage is retried later, not lost.
	outage := n(uniq("outage"))
	if _, err := s.notifications.Enqueue(ctx, outage); err != nil {
		t.Fatal(err)
	}
	s.mailer.fail = func(e domain.Email) error {
		if e.To == outage.Recipient {
			return apperrors.ErrUnavailable
		}
		return nil
	}
	if res, _ := s.notifications.Dispatch(ctx); res.Retrying < 1 {
		t.Fatalf("an outage should be retried, got %+v", res)
	}
	s.dispatch() // not due yet: nothing may be sent
	if len(s.mailer.to(outage.Recipient)) != 0 {
		t.Fatal("sent during the back-off")
	}
	s.mailer.fail = nil
	s.exec(`UPDATE notifications SET next_attempt_at = now() - interval '1 minute' WHERE recipient = $1`, outage.Recipient)
	s.dispatch()
	s.dispatch()
	if got := len(s.mailer.to(outage.Recipient)); got != 1 {
		t.Fatalf("recovered email was sent %d times, want 1", got)
	}
	var body string
	if err := s.pool.QueryRow(ctx, `SELECT body FROM notifications WHERE recipient = $1`, outage.Recipient).Scan(&body); err != nil || body != "" {
		t.Fatalf("body after sending = %q, %v; it can hold a link and must be cleared", body, err)
	}

	// Running out of attempts, or a refusal, ends in failed, visible to an admin.
	exhausted := n(uniq("exhausted"))
	rejected := n(uniq("rejected"))
	for _, x := range []domain.Notification{exhausted, rejected} {
		if _, err := s.notifications.Enqueue(ctx, x); err != nil {
			t.Fatal(err)
		}
	}
	s.exec(`UPDATE notifications SET attempts = $2 WHERE recipient = $1`, exhausted.Recipient, domain.MaxNotificationAttempts-1)
	s.mailer.fail = func(e domain.Email) error {
		if e.To == rejected.Recipient {
			return errors.Join(apperrors.ErrMailRejected, errors.New("bad address"))
		}
		if e.To == exhausted.Recipient {
			return apperrors.ErrUnavailable
		}
		return nil
	}
	s.dispatch()
	list := s.expect(s.call("GET", "/api/v1/notifications?status=failed&limit=200", nil, admin), 200)
	if !strings.Contains(list.Raw, exhausted.Recipient) || !strings.Contains(list.Raw, rejected.Recipient) {
		t.Fatalf("failed emails are not visible: %s", list.Raw)
	}
	if strings.Contains(list.Raw, "body") {
		t.Fatal("the message text must never be listed")
	}
	s.expect(s.call("GET", "/api/v1/notifications", nil, reporter), 403)

	// An administrator can send it again once the cause is fixed.
	s.mailer.fail = nil
	var id string
	if err := s.pool.QueryRow(ctx, `SELECT id::text FROM notifications WHERE recipient = $1`, rejected.Recipient).Scan(&id); err != nil {
		t.Fatal(err)
	}
	s.expect(s.call("POST", "/api/v1/notifications/"+id+"/retry", nil, admin), 202)
	s.expect(s.call("POST", "/api/v1/notifications/"+id+"/retry", nil, admin), 409) // no longer failed
	s.dispatch()
	if len(s.mailer.to(rejected.Recipient)) != 1 {
		t.Fatal("a retried email was not sent")
	}
}

// Alerts go to the people who can act, once, and the customer only ever
// receives the published facts of a critical incident.
func TestIncident_AlertsReachTheRightPeopleOnce(t *testing.T) {
	w := newWorkflow(t)
	api := "/api/v1/incidents/"
	strangerEmail, _ := w.seedUser(domain.RoleSupervisor) // supervises no site
	otherAirtel, _ := w.seedUser(domain.RoleAirtelViewer) // sees no site

	id := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "generator_failure", "severity": "critical", "impact": "outage", "title": "Generator down",
		"description": "internal: bearings seized", "started_at": ago(3 * time.Hour),
	}, w.reporter), 201).str("id")
	w.expect(w.call("POST", api+id+"/assign", map[string]any{"owner_id": w.reporterID, "next_action": "Dispatch engineer"}, w.supervisor), 200)

	// Nothing is overdue until a promised update is missed.
	if n, err := w.incidents.AlertOverdue(context.Background()); err != nil || n != 0 {
		t.Fatalf("alerts before anything is overdue = %d, %v", n, err)
	}
	w.exec(`UPDATE incidents SET next_update_due = now() - interval '1 hour' WHERE id = $1`, id)
	n, err := w.incidents.AlertOverdue(context.Background())
	if err != nil || n < 2 {
		t.Fatalf("overdue alerts queued = %d, %v; want at least the owner and the site supervisor", n, err)
	}
	if again, _ := w.incidents.AlertOverdue(context.Background()); again != 0 {
		t.Fatalf("the same missed update was alerted again (%d)", again)
	}
	w.dispatch()
	if len(w.mailer.to(w.repEmail)) != 1 || len(w.mailer.to(w.supEmail)) != 1 {
		t.Fatal("the owner and the site's supervisor should each get one alert")
	}
	if len(w.mailer.to(strangerEmail)) != 0 || len(w.mailer.to(w.airEmail)) != 0 {
		t.Fatal("people outside the site, and the customer, must not get internal alerts")
	}

	// Publishing a critical incident tells the customer's people on that site
	// at once, with nothing internal in it.
	w.expect(w.call("POST", api+id+"/publish", nil, w.supervisor), 200)
	w.dispatch()
	got := w.mailer.to(w.airEmail)
	if len(got) != 1 || !strings.Contains(got[0].Text, "Generator down") {
		t.Fatalf("the customer should get one notice of the critical incident, got %+v", got)
	}
	if strings.Contains(got[0].Text, "bearings") || strings.Contains(got[0].Text, "engineer") {
		t.Fatalf("the customer notice leaked internal detail: %s", got[0].Text)
	}
	if len(w.mailer.to(otherAirtel)) != 0 {
		t.Fatal("a customer user who cannot see the site must not be notified")
	}
	w.expect(w.call("POST", api+id+"/publish", nil, w.supervisor), 200) // repeat
	w.dispatch()
	if len(w.mailer.to(w.airEmail)) != 1 {
		t.Fatal("publishing twice sent a second notice")
	}
}
