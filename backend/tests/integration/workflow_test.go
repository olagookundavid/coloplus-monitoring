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
		"report_date":   date,
		"notes":         "internal: generator serviced",
		"outage_reason": "Grid failure at the site; generator took over",
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
	if sum["assumed_available_minutes"] != float64(30) {
		t.Fatalf("the 30 unreported minutes must be shown as assumed: %v", sum)
	}
	if sum["coverage_pct"] != 97.92 || sum["reported_availability_pct"] != 97.92 || sum["confirmed_served_share_pct"] != 95.83 {
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

// TestDailyReport_ToggleSubmitsWithGeneratorHours exercises the reporter's
// simplified day-level entry (power_status + generator_hours, no hand-entered
// intervals), for today's date specifically: this is the scenario the old
// "declared basis, full calendar day" design would have rejected as "an
// observation cannot end in the future", and that a declared (rather than
// observed) interval would have left permanently uncounted by Submit's
// "add at least one observed interval" rule.
func TestDailyReport_ToggleSubmitsWithGeneratorHours(t *testing.T) {
	w := newWorkflow(t)
	date, _ := w.reportDay(0) // today

	created := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports", map[string]any{
		"report_date": date, "power_status": "outage", "generator_hours": 3.5,
		"outage_reason": "Grid down since morning",
	}, w.reporter), 201)
	id := created.str("id")
	if created.str("generator_hours") != "3.5" {
		t.Fatalf("generator_hours = %s, want 3.5", created.str("generator_hours"))
	}

	submitted := w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", nil, w.reporter), 200)
	if submitted.str("status") != "submitted" {
		t.Fatalf("status = %s, want submitted: the day-level interval must count as observed", submitted.str("status"))
	}

	// A site is available unless someone says otherwise, so an outage day
	// cannot be submitted without saying why; a save without a reason keeps it.
	date3, _ := w.reportDay(1)
	noReason := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports", map[string]any{
		"report_date": date3, "power_status": "outage", "generator_hours": 2.0,
	}, w.reporter), 201)
	denied := w.expect(w.call("POST", "/api/v1/daily-reports/"+noReason.str("id")+"/submit", nil, w.reporter), 400)
	if !strings.Contains(denied.str("error"), "why the site was unavailable") {
		t.Fatalf("the refusal should ask for the reason: %s", denied.Raw)
	}
	saved := w.expect(w.call("PUT", "/api/v1/daily-reports/"+noReason.str("id"), map[string]any{
		"row_version": noReason.Body["row_version"], "power_status": "outage", "generator_hours": 2.0,
		"outage_reason": "Transformer fault",
	}, w.reporter), 200)
	again := w.expect(w.call("PUT", "/api/v1/daily-reports/"+noReason.str("id"), map[string]any{
		"row_version": saved.Body["row_version"], "power_status": "outage", "generator_hours": 2.5,
	}, w.reporter), 200)
	if again.str("outage_reason") != "Transformer fault" {
		t.Fatalf("saving without a reason must keep the one given: %s", again.Raw)
	}
	w.expect(w.call("POST", "/api/v1/daily-reports/"+noReason.str("id")+"/submit", nil, w.reporter), 200)

	// Missing generator hours on a day with an outage is refused.
	date2, _ := w.reportDay(2)
	w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports",
		map[string]any{"report_date": date2, "power_status": "outage"}, w.reporter), 400)
}

// TestDailyReport_MineFilterScopesToAuthor guards against a reporter's
// "my reports" view leaking another reporter's entries on the same site.
func TestDailyReport_MineFilterScopesToAuthor(t *testing.T) {
	w := newWorkflow(t)
	id1 := w.draft(1)

	secondEmail, _ := w.seedUser(domain.RoleReporter, w.site)
	second := w.login(secondEmail)
	date2, at2 := w.reportDay(2)
	id2 := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports", map[string]any{
		"report_date": date2,
		"intervals":   []any{iv(at2(0, 0), at2(1, 0), "served", "grid")},
	}, second), 201).str("id")

	mineFirst := w.expect(w.call("GET", "/api/v1/daily-reports?mine=true", nil, w.reporter), 200)
	first := mineFirst.list("reports")
	if len(first) != 1 {
		t.Fatalf("mine=true for the first reporter returned %d reports, want 1: %s", len(first), mineFirst.Raw)
	}
	if got, _ := first[0].(map[string]any)["id"].(string); got != id1 {
		t.Fatalf("mine=true returned report %s, want the caller's own %s", got, id1)
	}

	mineSecond := w.expect(w.call("GET", "/api/v1/daily-reports?mine=true", nil, second), 200)
	second2 := mineSecond.list("reports")
	if len(second2) != 1 {
		t.Fatalf("mine=true for the second reporter returned %d reports, want 1: %s", len(second2), mineSecond.Raw)
	}
	if got, _ := second2[0].(map[string]any)["id"].(string); got != id2 {
		t.Fatalf("mine=true returned report %s, want the caller's own %s", got, id2)
	}
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
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", map[string]any{"late_reason": "entered later"}, dual), 200)

	denied := w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, dual), 403)
	if !strings.Contains(denied.str("error"), "your own entry") {
		t.Fatalf("error = %s", denied.str("error"))
	}
	// A different reviewer can.
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)
}

// A site is available by default (owner direction, DECISIONS 051): time nobody
// reported counts as available, but it is always shown as assumed, the days
// without a report are counted beside it, the "Awaiting update" label stays,
// and a site can be set to the stricter reading where it is unknown instead.
func TestDashboard_UnreportedTimeIsAssumedAvailableAndSaysSo(t *testing.T) {
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
			s, _ := r["summary"].(map[string]any)
			if r["condition"] != "awaiting_update" || r["days_without_report"] != float64(1) || s == nil ||
				s["reported_availability_pct"] != 100.0 || s["assumed_available_minutes"] != float64(1440) ||
				s["confirmed_served_share_pct"] != 0.0 || s["served_minutes"] != float64(0) {
				t.Fatalf("a site nobody reported on is available by default, but assumed, awaiting update, and counted as a day without a report: %v", r)
			}
		}
	}

	// Pooled over eligible minutes: 2 sites x 1440, 1380 served, 30 outage.
	p, _ := ov.Body["portfolio"].(map[string]any)
	if p["eligible_minutes"] != float64(2880) || p["served_minutes"] != float64(1380) || p["unknown_minutes"] != float64(1470) {
		t.Fatalf("portfolio minutes = %v", p)
	}
	if p["coverage_pct"] != 48.96 || p["provisional"] != true || p["confirmed_served_share_pct"] != 47.92 {
		t.Fatalf("coverage and the confirmed share must still expose the missing site: %v", p)
	}
	// 2,850 of 2,880 minutes count as available, 1,470 of them assumed.
	if p["reported_availability_pct"] != 98.96 || p["assumed_available_minutes"] != float64(1470) || ov.Body["site_days_without_report"] != float64(1) {
		t.Fatalf("availability must say how much of it is assumed: %v / %s", p, ov.Raw)
	}

	// Set to the stricter reading, the silent site's time is unknown again.
	w.exec(`UPDATE sites SET unreported_time = 'unknown' WHERE id = $1`, silent)
	strict := w.expect(w.call("GET", "/api/v1/dashboard/overview?date="+date, nil, w.reporter), 200)
	for _, row := range strict.list("sites") {
		if r := row.(map[string]any); r["site_id"] == silent && r["summary"] != nil {
			t.Fatalf("under the strict reading a site with no report has no figure: %v", r)
		}
	}
	if sp, _ := strict.Body["portfolio"].(map[string]any); sp["reported_availability_pct"] != 97.92 {
		t.Fatalf("the unknown site must not count as available: %v", sp)
	}

	// An Airtel viewer assigned to the first site only sees that site.
	air := w.expect(w.call("GET", "/api/v1/dashboard/overview?date="+date, nil, w.airtel), 200)
	if len(air.list("sites")) != 1 {
		t.Fatalf("airtel viewer should see only their assigned site: %s", air.Raw)
	}
}

// Over a period the missing days stay unknown, a day nobody reported is a gap
// in the trend rather than a line, a region narrows the sites, and the site
// names a source only while its report is current.
func TestDashboard_PeriodRegionAndTrendKeepGaps(t *testing.T) {
	w := newWorkflow(t)
	to, _ := w.goodDay(1)
	from, _ := w.reportDay(2)

	silent := w.newSite(w.admin)
	w.expect(w.call("PATCH", "/api/v1/users/"+w.reporterID, map[string]any{"site_ids": []string{w.site, silent}}, w.admin), 200)
	w.exec(`UPDATE sites SET region = 'Lagos Island' WHERE id = $1`, w.site)
	w.exec(`UPDATE sites SET region = 'Abuja' WHERE id = $1`, silent)

	id := w.draft(1)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)

	ov := w.expect(w.call("GET", "/api/v1/dashboard/overview?from="+from+"&to="+to+"&region=lagos+island", nil, w.reporter), 200)
	rows := ov.list("sites")
	if len(rows) != 1 || ov.Body["days"] != float64(2) || len(ov.list("regions")) != 2 {
		t.Fatalf("the region must narrow the sites and still list every region: %s", ov.Raw)
	}
	row := rows[0].(map[string]any)
	sum, _ := row["summary"].(map[string]any)
	// Two days of 1440 minutes; only yesterday was reported.
	if row["days_reported"] != float64(1) || row["current_source"] != "generator" ||
		sum["eligible_minutes"] != float64(2880) || sum["served_minutes"] != float64(1380) || sum["unknown_minutes"] != float64(1470) ||
		sum["assumed_available_minutes"] != float64(1470) || row["days_without_report"] != float64(1) {
		t.Fatalf("a period must pool the days, with the unreported day assumed and counted: %v", row)
	}

	all := w.expect(w.call("GET", "/api/v1/dashboard/overview?from="+from+"&to="+to, nil, w.reporter), 200)
	for _, r := range all.list("sites") {
		if m := r.(map[string]any); m["site_id"] == silent && (m["current_source"] != nil || m["days_reported"] != float64(0) || m["days_without_report"] != float64(2)) {
			t.Fatalf("a site that never reported has no source and two days without a report: %v", m)
		}
	}

	tr := w.expect(w.call("GET", "/api/v1/dashboard/trend?from="+from+"&to="+to, nil, w.reporter), 200)
	pts := tr.list("points")
	if len(pts) != 2 {
		t.Fatalf("one point per day: %s", tr.Raw)
	}
	first := pts[0].(map[string]any)["summary"].(map[string]any)
	last := pts[1].(map[string]any)["summary"].(map[string]any)
	// A day nobody reported is assumed available, and says so: nothing is
	// confirmed and no site reported.
	if first["reported_availability_pct"] != 100.0 || first["assumed_available_minutes"] != float64(2880) ||
		first["confirmed_served_share_pct"] != 0.0 || pts[0].(map[string]any)["sites_reported"] != float64(0) {
		t.Fatalf("a day nobody reported is assumed available, never confirmed: %v", pts[0])
	}
	if last["reported_availability_pct"] == nil || pts[1].(map[string]any)["sites_reported"] != float64(1) {
		t.Fatalf("yesterday was reported: %v", pts[1])
	}

	// Weekly and monthly points pool the same days: none lost, none counted twice.
	for _, grain := range []string{"week", "month"} {
		g := w.expect(w.call("GET", "/api/v1/dashboard/trend?from="+from+"&to="+to+"&grain="+grain, nil, w.reporter), 200)
		days, reported, minutes := 0.0, 0.0, 0.0
		for _, p := range g.list("points") {
			pt := p.(map[string]any)
			days += pt["days"].(float64)
			reported += pt["site_days_reported"].(float64)
			minutes += pt["summary"].(map[string]any)["eligible_minutes"].(float64)
		}
		if g.str("grain") != grain || days != 2 || reported != 1 || minutes != 2*2*1440 {
			t.Fatalf("%s points must cover the 2 days and 2 sites exactly once: days=%v reported=%v minutes=%v: %s", grain, days, reported, minutes, g.Raw)
		}
	}
	w.expect(w.call("GET", "/api/v1/dashboard/trend?from="+from+"&to="+to+"&grain=year", nil, w.reporter), 400)
	w.expect(w.call("GET", "/api/v1/dashboard/trend?from="+from, nil, w.reporter), 400)
	w.expect(w.call("GET", "/api/v1/dashboard/overview?date="+to+"&from="+from+"&to="+to, nil, w.reporter), 400)
	w.expect(w.call("GET", "/api/v1/dashboard/overview?from="+from+"&to=2999-01-01", nil, w.reporter), 400)
	w.expect(w.call("GET", "/api/v1/dashboard/trend?from=2020-01-01&to="+to, nil, w.reporter), 400)
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

	// Directory filters narrow the list; who is assigned where stays Coloplus's own.
	if got := w.expect(w.call("GET", "/api/v1/sites?source=solar", nil, w.admin), 200).list("sites"); len(got) != 0 {
		for _, g := range got {
			if g.(map[string]any)["id"] == w.site {
				t.Fatalf("a grid and generator site matched the solar filter")
			}
		}
	}
	if got := w.expect(w.call("GET", "/api/v1/sites?source=grid&assigned_to="+w.reporterID, nil, w.supervisor), 200).list("sites"); len(got) != 1 {
		t.Fatalf("the reporter's one site should match, got %d", len(got))
	}
	w.expect(w.call("GET", "/api/v1/sites?source=wind", nil, w.admin), 400)
	w.expect(w.call("GET", "/api/v1/sites?assigned_to="+w.reporterID, nil, w.airtel), 403)

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
	s.expect(s.call("POST", "/api/v1/auth/mfa/enable", map[string]any{"code": code}, tok), 200)

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

// A kept dashboard answer must never reach someone whose sites differ: the
// cache is keyed by the exact set of sites a person may see.
func TestDashboard_KeptAnswersNeverCrossSiteScopes(t *testing.T) {
	t.Setenv("DASHBOARD_CACHE_TTL", "1m")
	w := newWorkflow(t)
	otherSite := w.newSite(w.admin)
	email, _ := w.seedUser(domain.RoleAirtelViewer, otherSite)
	other := w.login(email)
	date, _ := w.reportDay(1)
	url := "/api/v1/dashboard/overview?date=" + date

	siteIDs := func(tok string) string {
		var ids []string
		for _, r := range w.expect(w.call("GET", url, nil, tok), 200).list("sites") {
			ids = append(ids, r.(map[string]any)["site_id"].(string))
		}
		return strings.Join(ids, ",")
	}
	// Ask in both orders and twice, so each answer is served from the cache.
	for i := 0; i < 2; i++ {
		if got := siteIDs(w.airtel); got != w.site {
			t.Fatalf("round %d: the first Airtel user should see only %s, got %s", i, w.site, got)
		}
		if got := siteIDs(other); got != otherSite {
			t.Fatalf("round %d: the second Airtel user should see only %s, got %s", i, otherSite, got)
		}
	}
}

// A report that arrives after its site would already show "Awaiting update"
// must say why; on time it needs nothing, and the reason stays with the report.
func TestReportSubmit_LateNeedsAReasonOnTimeDoesNot(t *testing.T) {
	w := newWorkflow(t)

	onTime := w.draft(1)
	if r := w.expect(w.call("POST", "/api/v1/daily-reports/"+onTime+"/submit", nil, w.reporter), 200); r.Body["late_reason"] != nil {
		t.Fatalf("an on-time report carries no late reason: %s", r.Raw)
	}

	late := w.draft(4)
	denied := w.expect(w.call("POST", "/api/v1/daily-reports/"+late+"/submit", nil, w.reporter), 400)
	if !strings.Contains(denied.str("error"), "late") {
		t.Fatalf("the refusal should say the report is late: %s", denied.Raw)
	}
	w.expect(w.call("POST", "/api/v1/daily-reports/"+late+"/submit", map[string]any{"late_reason": "  "}, w.reporter), 400)
	r := w.expect(w.call("POST", "/api/v1/daily-reports/"+late+"/submit", map[string]any{"late_reason": "No network at the site until Tuesday"}, w.reporter), 200)
	if r.str("late_reason") != "No network at the site until Tuesday" {
		t.Fatalf("the reason must stay with the report: %s", r.Raw)
	}
	if got := w.expect(w.call("GET", "/api/v1/daily-reports/"+late, nil, w.supervisor), 200).Body["report"].(map[string]any)["late_reason"]; got == nil {
		t.Fatal("the approver must see why it was late")
	}
}
