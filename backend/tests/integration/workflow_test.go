package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"coloplus-backend/internal/domain"

	"pkg-common/totp"
)

// workflow is the cast of a typical day: an admin, a site, a field reporter
// on that site, a supervisor, an Airtel viewer, and an outsider.
type workflow struct {
	*stack
	site                                    string
	admin, reporter, supervisor, airtel, ot string // tokens
	reporterID                              string
	repEmail, supEmail, airEmail            string
}

func newWorkflow(t *testing.T) *workflow {
	s := newStack(t)
	adminEmail, _ := s.seedUser(domain.RoleAdmin)
	admin := s.login(adminEmail)
	site := s.newSite(admin)

	repEmail, repID := s.seedUser(domain.RoleReporter, site)
	supEmail, _ := s.seedUser(domain.RoleSupervisor, site)
	airEmail, _ := s.seedUser(domain.RoleAirtelViewer, site)
	outEmail, _ := s.seedUser(domain.RoleReporter) // assigned to nothing

	return &workflow{
		stack: s, site: site, admin: admin, reporterID: repID,
		reporter: s.login(repEmail), supervisor: s.login(supEmail), airtel: s.login(airEmail), ot: s.login(outEmail),
		repEmail: repEmail, supEmail: supEmail, airEmail: airEmail,
	}
}

// goodDay: 23 hours served, 30 minutes outage, 30 minutes left unknown.
func (w *workflow) goodDay(daysAgo int) (date string, body map[string]any) {
	date, at := w.reportDay(daysAgo)
	return date, map[string]any{
		"report_date": date,
		"notes":       "internal: generator serviced",
		"intervals": []any{
			iv(at(0, 0), at(11, 30), "served", "grid"),
			iv(at(11, 30), at(12, 0), "outage", ""),
			iv(at(12, 0), at(23, 30), "served", "generator"),
		},
	}
}

func (w *workflow) draft(daysAgo int) string {
	_, body := w.goodDay(daysAgo)
	return w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports", body, w.reporter), 201).str("id")
}

func TestDailyReport_FullApprovalFlow(t *testing.T) {
	w := newWorkflow(t)
	id := w.draft(1)

	// Airtel cannot see an unapproved report, and neither can an outsider.
	w.expect(w.call("GET", "/api/v1/daily-reports/"+id, nil, w.airtel), 404)
	w.expect(w.call("GET", "/api/v1/daily-reports/"+id, nil, w.ot), 404)

	// Submitting twice is safe (a retry after a dropped connection).
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", nil, w.reporter), 200)
	again := w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", nil, w.reporter), 200)
	if again.str("status") != "submitted" {
		t.Fatalf("status after a repeat submit = %s", again.str("status"))
	}

	// The reporter cannot approve; the supervisor can.
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.reporter), 403)
	approved := w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", map[string]any{"comment": "ok"}, w.supervisor), 200)
	if approved.str("status") != "approved" {
		t.Fatalf("status = %s", approved.str("status"))
	}

	// Approved reports can no longer be edited or resubmitted.
	w.expect(w.call("PUT", "/api/v1/daily-reports/"+id, map[string]any{"row_version": 99, "intervals": []any{}}, w.reporter), 409)

	// Airtel now sees the published view: facts, not internal commentary.
	pub := w.expect(w.call("GET", "/api/v1/daily-reports/"+id, nil, w.airtel), 200)
	if pub.Body["published"] == nil || pub.Body["report"] != nil {
		t.Fatalf("an Airtel user must receive the published view only: %s", pub.Raw)
	}
	for _, leak := range []string{"internal: generator serviced", "author_id", "reviewer_id", "review_comment", "notes"} {
		if strings.Contains(pub.Raw, leak) {
			t.Fatalf("published view leaks %q: %s", leak, pub.Raw)
		}
	}

	// The summary reproduces the PRD's worked example.
	sum, _ := pub.Body["summary"].(map[string]any)
	if sum["served_minutes"] != float64(1380) || sum["outage_minutes"] != float64(30) || sum["unknown_minutes"] != float64(30) {
		t.Fatalf("summary minutes wrong: %v", sum)
	}
	if sum["coverage_pct"] != 97.92 || sum["reported_availability_pct"] != 97.87 || sum["confirmed_served_share_pct"] != 95.83 {
		t.Fatalf("summary percentages wrong: %v", sum)
	}
	calc, _ := pub.Body["calculation"].(map[string]any)
	if calc["formula"] != "daily-availability" || calc["input_report_id"] != id {
		t.Fatalf("calculation provenance missing: %v", calc)
	}

	// Airtel's list holds approved reports only.
	list := w.expect(w.call("GET", "/api/v1/daily-reports?site_id="+w.site, nil, w.airtel), 200)
	if len(list.list("published")) != 1 || list.Body["reports"] != nil {
		t.Fatalf("airtel list = %s", list.Raw)
	}

	if n := w.countAudit("report.approved", id); n != 1 {
		t.Fatalf("approval audit rows = %d, want 1", n)
	}
	if n := w.countAudit("report.submitted", id); n != 1 {
		t.Fatalf("submission audit rows = %d, want exactly 1 even though it was submitted twice", n)
	}
}

func TestDailyReport_ConcurrentEditsConflict(t *testing.T) {
	w := newWorkflow(t)
	id := w.draft(1)
	got := w.expect(w.call("GET", "/api/v1/daily-reports/"+id, nil, w.reporter), 200)
	rep, _ := got.Body["report"].(map[string]any)
	version := rep["row_version"]

	body := func(notes string) map[string]any {
		return map[string]any{"row_version": version, "notes": notes, "intervals": rep["intervals"]}
	}
	w.expect(w.call("PUT", "/api/v1/daily-reports/"+id, body("first edit"), w.reporter), 200)
	// A second edit based on the same, now stale, version must not win silently.
	stale := w.expect(w.call("PUT", "/api/v1/daily-reports/"+id, body("second edit"), w.reporter), 409)
	if !strings.Contains(stale.str("error"), "reload") {
		t.Fatalf("the conflict should tell the user to reload: %s", stale.Raw)
	}
}

func TestDailyReport_ValidationAndScope(t *testing.T) {
	w := newWorkflow(t)
	date, at := w.reportDay(1)
	create := func(tok string, intervals []any) reply {
		return w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports",
			map[string]any{"report_date": date, "intervals": intervals}, tok)
	}

	// Contradictory served/outage overlap is blocked with a worded reason.
	bad := w.expect(create(w.reporter, []any{
		iv(at(1, 0), at(5, 0), "served", "grid"), iv(at(4, 0), at(6, 0), "outage", ""),
	}), 400)
	if !strings.Contains(bad.str("error"), "contradict") {
		t.Fatalf("error = %s", bad.str("error"))
	}

	// A reporter not assigned to the site cannot see or write to it.
	w.expect(create(w.ot, []any{iv(at(1, 0), at(2, 0), "served", "grid")}), 404)
	// A supervisor does not create reports; an Airtel user never does.
	w.expect(create(w.supervisor, nil), 403)
	w.expect(create(w.airtel, nil), 403)

	// A future date is refused.
	w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports",
		map[string]any{"report_date": "2999-01-01", "intervals": []any{}}, w.reporter), 400)

	// One report in progress per site and day.
	w.expect(create(w.reporter, []any{iv(at(1, 0), at(2, 0), "served", "grid")}), 201)
	w.expect(create(w.reporter, []any{iv(at(1, 0), at(2, 0), "served", "grid")}), 409)

	// Submitting with nothing observed is refused.
	date2, _ := w.reportDay(2)
	empty := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports",
		map[string]any{"report_date": date2, "intervals": []any{}}, w.reporter), 201)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+empty.str("id")+"/submit", nil, w.reporter), 400)
}

func TestDailyReport_CorrectionKeepsTheOriginalUntilApproved(t *testing.T) {
	w := newWorkflow(t)
	orig := w.draft(1)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/approve", nil, w.supervisor), 200)

	// A new draft for an approved day must go through a correction.
	date, at := w.reportDay(1)
	w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports",
		map[string]any{"report_date": date, "intervals": []any{iv(at(1, 0), at(2, 0), "served", "grid")}}, w.reporter), 409)

	w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/corrections", map[string]any{"reason": "x"}, w.reporter), 400)
	corr := w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/corrections",
		map[string]any{"reason": "outage was 45 minutes, not 30"}, w.reporter), 201)
	corrID := corr.str("id")
	if corr.str("revision") != "2" || corr.str("supersedes_id") != orig || corr.str("status") != "draft" {
		t.Fatalf("correction = %s", corr.Raw)
	}

	// Fix the outage to 45 minutes, submit.
	w.expect(w.call("PUT", "/api/v1/daily-reports/"+corrID, map[string]any{
		"row_version": corr.Body["row_version"], "notes": "corrected",
		"intervals": []any{
			iv(at(0, 0), at(11, 30), "served", "grid"),
			iv(at(11, 30), at(12, 15), "outage", ""),
			iv(at(12, 15), at(23, 30), "served", "generator"),
		},
	}, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+corrID+"/submit", nil, w.reporter), 200)

	// Until the correction is approved, Airtel still sees the original.
	seen := w.expect(w.call("GET", "/api/v1/daily-reports?site_id="+w.site, nil, w.airtel), 200)
	if pubs := seen.list("published"); len(pubs) != 1 || pubs[0].(map[string]any)["id"] != orig {
		t.Fatalf("airtel should still see only the original: %s", seen.Raw)
	}

	w.expect(w.call("POST", "/api/v1/daily-reports/"+corrID+"/approve", nil, w.supervisor), 200)

	// Now the original is superseded, the correction is live, and nothing was overwritten.
	old := w.expect(w.call("GET", "/api/v1/daily-reports/"+orig, nil, w.supervisor), 200)
	if old.str("report", "status") != "superseded" {
		t.Fatalf("original status = %s, want superseded", old.str("report", "status"))
	}
	seen = w.expect(w.call("GET", "/api/v1/daily-reports?site_id="+w.site, nil, w.airtel), 200)
	if pubs := seen.list("published"); len(pubs) != 1 || pubs[0].(map[string]any)["id"] != corrID {
		t.Fatalf("airtel should now see the correction: %s", seen.Raw)
	}
	live := w.expect(w.call("GET", "/api/v1/daily-reports/"+corrID, nil, w.airtel), 200)
	if live.Body["summary"].(map[string]any)["outage_minutes"] != float64(45) {
		t.Fatalf("corrected outage = %v", live.Body["summary"])
	}

	// The audit record carries the before and after values.
	var meta string
	if err := w.pool.QueryRow(context.Background(),
		`SELECT metadata::text FROM audit_log WHERE action = 'report.approved' AND entity_id = $1`, corrID).Scan(&meta); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"before"`, `"after"`, orig, "outage was 45 minutes"} {
		if !strings.Contains(meta, want) {
			t.Fatalf("correction audit metadata missing %s: %s", want, meta)
		}
	}
}

func TestDailyReport_AnApprovedRecordIsImmutableInTheDatabase(t *testing.T) {
	w := newWorkflow(t)
	id := w.draft(1)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)

	ctx := context.Background()
	for name, stmt := range map[string]string{
		"change the report":    `UPDATE daily_reports SET notes = 'tampered' WHERE id = $1`,
		"delete the report":    `DELETE FROM daily_reports WHERE id = $1`,
		"change an interval":   `UPDATE power_intervals SET end_at = end_at + interval '1 hour' WHERE report_id = $1`,
		"delete the intervals": `DELETE FROM power_intervals WHERE report_id = $1`,
		"add an interval": `INSERT INTO power_intervals (report_id, start_at, end_at, state, source)
		                    SELECT $1, now() - interval '3 hour', now() - interval '2 hour', 'served', 'grid'`,
	} {
		if _, err := w.pool.Exec(ctx, stmt, id); err == nil {
			t.Errorf("%s: the database must refuse this", name)
		}
	}
}

func TestSelfApprovalIsBlocked(t *testing.T) {
	w := newWorkflow(t)
	ctx := context.Background()

	// A role that can both write and review reports: the dual-hat case.
	if _, err := w.pool.Exec(ctx, `INSERT INTO roles (id, name) VALUES ('test_dual', 'Test dual role') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx, `INSERT INTO role_permissions (role_id, permission_id)
		VALUES ('test_dual','sites.read'),('test_dual','reports.create'),('test_dual','reports.read'),('test_dual','reports.review')
		ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	email, _ := w.seedUser("test_dual", w.site)
	dual := w.login(email)

	_, body := w.goodDay(3)
	id := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports", body, dual), 201).str("id")
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", nil, dual), 200)

	denied := w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, dual), 403)
	if !strings.Contains(denied.str("error"), "your own entry") {
		t.Fatalf("error = %s", denied.str("error"))
	}
	// A different reviewer can.
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)
}

func TestDashboard_MissingReportsAreUnknownNeverAvailable(t *testing.T) {
	w := newWorkflow(t)
	date, _ := w.goodDay(1)

	// A second active site that never reports.
	silent := w.newSite(w.admin)
	w.expect(w.call("PATCH", "/api/v1/users/"+w.reporterID, map[string]any{"site_ids": []string{w.site, silent}}, w.admin), 200)

	id := w.draft(1)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)

	// The reporter's scope is exactly the two sites; the dashboard shows both.
	ov := w.expect(w.call("GET", "/api/v1/dashboard/overview?date="+date, nil, w.reporter), 200)
	if ov.Body["active_sites"] != float64(2) || ov.Body["sites_reported"] != float64(1) || ov.Body["sites_not_reported"] != float64(1) {
		t.Fatalf("counts wrong: %s", ov.Raw)
	}
	if ov.Body["sites_awaiting_update"] != float64(1) {
		t.Fatalf("the silent site must be awaiting update: %s", ov.Raw)
	}
	for _, row := range ov.list("sites") {
		r := row.(map[string]any)
		if r["site_id"] == silent {
			if r["condition"] != "awaiting_update" || r["summary"] != nil {
				t.Fatalf("a site with no report must be awaiting update with no summary: %v", r)
			}
		}
	}

	// Pooled over eligible minutes: 2 sites x 1440, 1380 served, 30 outage.
	p, _ := ov.Body["portfolio"].(map[string]any)
	if p["eligible_minutes"] != float64(2880) || p["served_minutes"] != float64(1380) || p["unknown_minutes"] != float64(1470) {
		t.Fatalf("portfolio minutes = %v", p)
	}
	if p["coverage_pct"] != 48.96 || p["provisional"] != true {
		t.Fatalf("portfolio coverage must expose the missing site: %v", p)
	}

	// An Airtel viewer assigned to the first site only sees that site.
	air := w.expect(w.call("GET", "/api/v1/dashboard/overview?date="+date, nil, w.airtel), 200)
	if len(air.list("sites")) != 1 {
		t.Fatalf("airtel viewer should see only their assigned site: %s", air.Raw)
	}
}

func TestSites_ScopeSearchAndDuplicates(t *testing.T) {
	w := newWorkflow(t)

	// A site the reporter is not assigned to.
	other := w.newSite(w.admin)

	list := w.expect(w.call("GET", "/api/v1/sites", nil, w.reporter), 200)
	if len(list.list("sites")) != 1 || list.list("sites")[0].(map[string]any)["id"] != w.site {
		t.Fatalf("a reporter lists only assigned sites: %s", list.Raw)
	}
	// A guessed id outside scope reveals nothing, and a malformed one is a plain 404.
	w.expect(w.call("GET", "/api/v1/sites/"+other, nil, w.reporter), 404)
	w.expect(w.call("GET", "/api/v1/sites/not-a-uuid", nil, w.reporter), 404)

	// Only administrators write to the register.
	w.expect(w.call("POST", "/api/v1/sites", map[string]any{"airtel_code": "X"}, w.reporter), 403)
	w.expect(w.call("PUT", "/api/v1/sites/"+w.site, map[string]any{"airtel_code": "X"}, w.supervisor), 403)

	// Duplicate Airtel site codes are rejected.
	site := w.expect(w.call("GET", "/api/v1/sites/"+w.site, nil, w.admin), 200)
	dup := w.expect(w.call("POST", "/api/v1/sites", map[string]any{
		"airtel_code": strings.ToLower(site.str("airtel_code")), "coloplus_code": uniq("CP"), "name": "Dup", "state": "Lagos",
		"power_sources": []string{"grid"},
	}, w.admin), 409)
	if !strings.Contains(dup.str("error"), "already exists") {
		t.Fatalf("error = %s", dup.str("error"))
	}

	// Validation is worded, and lists everything wrong at once.
	bad := w.expect(w.call("POST", "/api/v1/sites", map[string]any{"airtel_code": "", "power_sources": []string{"wind"}}, w.admin), 400)
	for _, want := range []string{"Airtel site code", "state is required", "power source"} {
		if !strings.Contains(bad.str("error"), want) {
			t.Errorf("error %q should mention %q", bad.str("error"), want)
		}
	}

	// Search by code.
	found := w.expect(w.call("GET", "/api/v1/sites?q="+site.str("coloplus_code"), nil, w.admin), 200)
	if len(found.list("sites")) != 1 {
		t.Fatalf("search by code found %d sites", len(found.list("sites")))
	}
}

func TestAuth_LockoutDeactivationAndMFA(t *testing.T) {
	s := newStack(t)
	email, id := s.seedUser(domain.RoleReporter)

	// Unknown email and wrong password look identical.
	a := s.call("POST", "/api/v1/auth/login", map[string]any{"email": "nobody@coloplus.example", "password": testPassword}, "")
	b := s.call("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": "wrong-password-1"}, "")
	if a.Code != 401 || b.Code != 401 || a.str("error") != b.str("error") {
		t.Fatalf("login failures must be indistinguishable: %d %q vs %d %q", a.Code, a.str("error"), b.Code, b.str("error"))
	}

	// MFA: set up, confirm with a real code, then login needs a code.
	tok := s.login(email)
	setup := s.expect(s.call("POST", "/api/v1/auth/mfa/setup", nil, tok), 200)
	secret := setup.str("secret")
	if !strings.HasPrefix(setup.str("otpauth_url"), "otpauth://totp/") {
		t.Fatalf("otpauth url = %s", setup.str("otpauth_url"))
	}
	var stored string
	if err := s.pool.QueryRow(context.Background(), `SELECT totp_secret FROM users WHERE id = $1`, id).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == "" || stored == secret || strings.Contains(stored, secret) {
		t.Fatal("the TOTP secret must be stored encrypted, never in clear")
	}
	s.expect(s.call("POST", "/api/v1/auth/mfa/enable", map[string]any{"code": "000000"}, tok), 401)

	code, _ := totp.Code(secret, time.Now())
	s.expect(s.call("POST", "/api/v1/auth/mfa/enable", map[string]any{"code": code}, tok), 204)

	step1 := s.expect(s.call("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": testPassword}, ""), 200)
	if step1.str("status") != "mfa_required" || step1.str("access_token") != "" {
		t.Fatalf("with two-step on, a password alone must not yield a token: %s", step1.Raw)
	}
	s.expect(s.call("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": testPassword, "totp_code": "000000"}, ""), 401)
	code, _ = totp.Code(secret, time.Now())
	ok := s.expect(s.call("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": testPassword, "totp_code": code}, ""), 200)
	if ok.str("access_token") == "" {
		t.Fatalf("a correct code must sign in: %s", ok.Raw)
	}

	// The cookie is HttpOnly and named by config.
	req := s.call("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": testPassword, "totp_code": code}, "")
	if c := req.Header.Get("Set-Cookie"); !strings.Contains(c, cookieName+"=") || !strings.Contains(c, "HttpOnly") {
		t.Fatalf("Set-Cookie = %q", c)
	}

	// Deactivating a user cuts the token they already hold.
	adminEmail, _ := s.seedUser(domain.RoleAdmin)
	admin := s.login(adminEmail)
	s.expect(s.call("GET", "/api/v1/auth/me", nil, ok.str("access_token")), 200)
	s.expect(s.call("PATCH", "/api/v1/users/"+id, map[string]any{"is_active": false}, admin), 200)
	s.expect(s.call("GET", "/api/v1/auth/me", nil, ok.str("access_token")), 401)

	// An administrator cannot lock themselves out.
	var adminID string
	adminMe := s.expect(s.call("GET", "/api/v1/auth/me", nil, admin), 200)
	adminID = adminMe.str("user", "id")
	s.expect(s.call("PATCH", "/api/v1/users/"+adminID, map[string]any{"is_active": false}, admin), 400)
}

func TestAuth_AccountLocksAfterRepeatedFailures(t *testing.T) {
	s := newStack(t)
	email, _ := s.seedUser(domain.RoleReporter)

	for i := 0; i < domain.MaxFailedLogins; i++ {
		s.expect(s.call("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": "wrong-password-1"}, ""), 401)
	}
	// Even the right password is refused while locked.
	locked := s.expect(s.call("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": testPassword}, ""), 429)
	if !strings.Contains(locked.str("error"), "try again") {
		t.Fatalf("error = %s", locked.str("error"))
	}
}
