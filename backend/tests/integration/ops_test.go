package integration

import (
	"context"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/services"
)

// A late report is stale data, never an outage: the people who owe it are told
// once a day, Airtel is not emailed, and the wording says power is not known
// to be out.
func TestReportAlerts_OverdueGoesToTheRightPeopleOnceAndNeverSaysOutage(t *testing.T) {
	w := newWorkflow(t)
	ctx := context.Background()

	// A brand-new site is not nagged before its first report is due.
	if _, err := w.reportAlerts.AlertOverdue(ctx); err != nil {
		t.Fatal(err)
	}
	w.dispatch()
	if got := len(w.mailer.to(w.repEmail)); got != 0 {
		t.Fatalf("a site created seconds ago was alerted (%d emails)", got)
	}

	w.exec(`UPDATE sites SET created_at = now() - interval '10 days' WHERE id = $1`, w.site)
	for i := 0; i < 2; i++ { // the second run must add nothing
		if _, err := w.reportAlerts.AlertOverdue(ctx); err != nil {
			t.Fatal(err)
		}
		w.dispatch()
	}
	for _, addr := range []string{w.repEmail, w.supEmail} {
		mails := w.mailer.to(addr)
		if len(mails) != 1 {
			t.Fatalf("%s got %d overdue emails, want 1", addr, len(mails))
		}
		if !strings.Contains(mails[0].Text, "Awaiting update") || !strings.Contains(mails[0].Text, "does not mean the power is out") {
			t.Fatalf("overdue wording: %q", mails[0].Text)
		}
	}
	if got := len(w.mailer.to(w.airEmail)); got != 0 {
		t.Fatalf("Airtel was emailed %d times about Coloplus's missing report", got)
	}
}

// Reading the audit log needs audit access, and a reader scoped to some sites
// sees only events about those sites.
func TestAudit_SearchNeedsAccessAndRespectsSiteScope(t *testing.T) {
	w := newWorkflow(t)
	inc := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "low_fuel", "severity": "minor", "impact": "no_outage", "title": "Fuel low",
		"started_at": ago(time.Hour),
	}, w.supervisor), 201).str("id")

	w.expect(w.call("GET", "/api/v1/audit", nil, w.supervisor), 403)
	w.expect(w.call("GET", "/api/v1/audit", nil, w.airtel), 403)
	found := w.expect(w.call("GET", "/api/v1/audit?entity_id="+inc+"&action=incident.reported", nil, w.admin), 200)
	if found.str("total") != "1" {
		t.Fatalf("admin search = %s", found.Raw)
	}
	w.expect(w.call("GET", "/api/v1/audit?from=yesterday", nil, w.admin), 400)

	reader := func(sites ...string) domain.Actor {
		ids := map[string]struct{}{}
		for _, id := range sites {
			ids[id] = struct{}{}
		}
		return domain.Actor{UserID: "reader", Permissions: map[domain.Permission]struct{}{domain.PermAuditRead: {}}, SiteIDs: ids}
	}
	in := servicesAuditInput(inc)
	if _, total, err := w.audit.Search(context.Background(), reader(w.site), in); err != nil || total != 1 {
		t.Fatalf("reader of the site: total=%d err=%v", total, err)
	}
	if _, total, err := w.audit.Search(context.Background(), reader(w.newSite(w.admin)), in); err != nil || total != 0 {
		t.Fatalf("reader of another site saw %d events, err=%v", total, err)
	}
}

func servicesAuditInput(entityID string) services.SearchAuditInput {
	return services.SearchAuditInput{EntityID: entityID}
}

// The site screen in one call must not weaken any rule: a day with no
// approved report is a gap rather than zero, Airtel sees only published
// incidents, and an outsider cannot probe a site.
func TestSiteOverview_GapsStayGapsAndAirtelSeesOnlyPublished(t *testing.T) {
	w := newWorkflow(t)
	orig := w.draft(1)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/approve", nil, w.supervisor), 200)

	open := func(title string) string {
		return w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
			"category": "grid_loss", "severity": "major", "impact": "outage", "title": title,
			"description": "internal: transformer", "started_at": ago(time.Hour),
		}, w.supervisor), 201).str("id")
	}
	open("Internal only")
	shared := open("Shared with Airtel")
	w.expect(w.call("POST", "/api/v1/incidents/"+shared+"/publish", nil, w.supervisor), 200)

	url := "/api/v1/sites/" + w.site + "/overview?days=3"
	staff := w.expect(w.call("GET", url, nil, w.supervisor), 200)
	days := staff.list("days")
	if len(days) != 3 {
		t.Fatalf("days = %d, want 3: %s", len(days), staff.Raw)
	}
	if _, has := days[0].(map[string]any)["summary"]; has {
		t.Fatalf("today has no approved report and must be a gap, not zero: %v", days[0])
	}
	if _, has := days[1].(map[string]any)["summary"]; !has {
		t.Fatalf("yesterday's approved report is missing: %v", days[1])
	}
	if staff.str("open_incidents") != "2" || len(staff.list("incidents")) != 2 {
		t.Fatalf("staff should see both incidents: %s", staff.Raw)
	}

	air := w.expect(w.call("GET", url, nil, w.airtel), 200)
	if len(air.list("published_incidents")) != 1 || air.Body["incidents"] != nil || strings.Contains(air.Raw, "transformer") || strings.Contains(air.Raw, "Internal only") {
		t.Fatalf("Airtel sees more than the published incident: %s", air.Raw)
	}
	w.expect(w.call("GET", url, nil, w.ot), 404)
}

// An export must not turn a gap into a figure, must carry the dispute state,
// stay inside the caller's sites, and leave an audit trail.
func TestExport_DailySummariesKeepGapsDisputesAndScope(t *testing.T) {
	w := newWorkflow(t)
	revEmail, _ := w.seedUser(domain.RoleAirtelReviewer, w.site)
	reviewer := w.login(revEmail)
	elseEmail, _ := w.seedUser(domain.RoleAirtelReviewer, w.newSite(w.admin))
	elsewhere := w.login(elseEmail)

	orig := w.draft(1)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/approve", nil, w.supervisor), 200)
	w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/disputes", map[string]any{
		"report_id": orig, "category": "wrong_metric", "title": "Figure looks low",
	}, reviewer), 201)

	from, _ := w.reportDay(2)
	to, _ := w.reportDay(1)
	url := "/api/v1/exports/daily-summaries.csv?from=" + from + "&to=" + to

	rows := func(r reply) [][]string {
		t.Helper()
		recs, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(r.Raw, "\xef\xbb\xbf"))).ReadAll()
		if err != nil {
			t.Fatalf("not valid CSV: %v\n%s", err, r.Raw)
		}
		return recs
	}
	check := func(tok string) {
		recs := rows(w.expect(w.call("GET", url, nil, tok), 200))
		col := map[string]int{}
		for i, h := range recs[0] {
			col[h] = i
		}
		var gap, day []string
		for _, rec := range recs[1:] {
			if rec[col["site_id"]] != w.site {
				continue
			}
			switch rec[col["date"]] {
			case from:
				gap = rec
			case to:
				day = rec
			}
		}
		if gap == nil || day == nil {
			t.Fatalf("expected a row for each day of the site: %v", recs)
		}
		if gap[col["report_status"]] != "no_approved_report" || gap[col["served_minutes"]] != "" ||
			gap[col["outage_minutes"]] != "" || gap[col["coverage_percent"]] != "" || gap[col["confirmed_served_share_percent"]] != "" {
			t.Fatalf("a day without a report has nothing confirmed, got %v", gap)
		}
		// Available by default: the figure is there, and marked as assumed.
		if gap[col["reported_availability_percent"]] != "100.00" || gap[col["assumed_available_minutes"]] != "1440" {
			t.Fatalf("an unreported day is assumed available and the export must say so, got %v", gap)
		}
		if day[col["report_status"]] != "approved" || day[col["report_id"]] != orig || day[col["served_minutes"]] != "1380" ||
			day[col["open_disputes"]] != "1" || day[col["formula_version"]] == "" || day[col["assumptions"]] == "" {
			t.Fatalf("approved day lost its provenance: %v", day)
		}
	}
	check(w.supervisor)
	check(w.airtel)

	w.expect(w.call("GET", url, nil, w.ot), 403)                         // a reporter has no export right
	w.expect(w.call("GET", url+"&site_id="+w.site, nil, elsewhere), 404) // not their site
	for _, rec := range rows(w.expect(w.call("GET", url, nil, elsewhere), 200))[1:] {
		if rec[0] == w.site {
			t.Fatalf("export leaked a site outside the caller's scope: %v", rec)
		}
	}
	w.expect(w.call("GET", "/api/v1/exports/daily-summaries.csv?from="+from+"&to=2999-01-01", nil, w.supervisor), 400)
	if n := w.countAudit("export.daily_summaries", "daily_summaries:"+from+":"+to); n < 2 {
		t.Fatalf("exports were not audited: %d", n)
	}
}

// The incident register must not show Airtel an unpublished incident, must
// not invent a duration before restoration is recorded, and stays inside the
// caller's sites and region.
func TestExport_IncidentRegisterHidesUnpublishedAndKeepsScope(t *testing.T) {
	w := newWorkflow(t)
	elseEmail, _ := w.seedUser(domain.RoleAirtelReviewer, w.newSite(w.admin))
	elsewhere := w.login(elseEmail)
	w.exec(`UPDATE sites SET region = 'Lagos Island', responsible_team = 'Island field team' WHERE id = $1`, w.site)

	open := func(title string) string {
		return w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
			"category": "grid_loss", "severity": "major", "impact": "outage", "title": title,
			"description": "internal: transformer", "started_at": ago(time.Hour),
		}, w.supervisor), 201).str("id")
	}
	open("Internal only")
	w.expect(w.call("POST", "/api/v1/incidents/"+open("Shared with Airtel")+"/publish", nil, w.supervisor), 200)

	from, _ := w.reportDay(1)
	to := time.Now().In(w.loc).Format("2006-01-02")
	url := "/api/v1/exports/incidents.csv?from=" + from + "&to=" + to
	rows := func(tok, q string) [][]string {
		t.Helper()
		r := w.expect(w.call("GET", url+q, nil, tok), 200)
		recs, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(r.Raw, "\xef\xbb\xbf"))).ReadAll()
		if err != nil {
			t.Fatalf("not valid CSV: %v\n%s", err, r.Raw)
		}
		return recs
	}

	air := rows(w.airtel, "")
	if len(air) != 2 || strings.Contains(strings.Join(air[1], "|"), "Internal only") || strings.Contains(strings.Join(air[1], "|"), "transformer") {
		t.Fatalf("Airtel must get the published incident only: %v", air)
	}
	col := map[string]int{}
	for i, h := range air[0] {
		col[h] = i
	}
	if air[1][col["duration_minutes"]] != "" || air[1][col["restored_at_utc"]] != "" || air[1][col["responsible_team"]] != "Island field team" {
		t.Fatalf("no restoration recorded, so no duration: %v", air[1])
	}
	if got := len(rows(w.supervisor, "")); got != 3 {
		t.Fatalf("staff should see both incidents, got %d rows", got-1)
	}
	if got := len(rows(w.supervisor, "&region=abuja")); got != 1 {
		t.Fatalf("a region with no sites must give no incidents, got %d rows", got-1)
	}
	if got := len(rows(elsewhere, "")); got != 1 {
		t.Fatalf("export leaked an incident outside the caller's sites: %d rows", got-1)
	}
	w.expect(w.call("GET", url+"&site_id="+w.site, nil, elsewhere), 404)
	w.expect(w.call("GET", url, nil, w.ot), 403)
	if n := w.countAudit("export.incident_register", "incident_register:"+from+":"+to); n < 3 {
		t.Fatalf("exports were not audited: %d", n)
	}
}

// Management/auditor reads reports and the audit log inside its sites and can
// change nothing: audit access is separate from editing rights.
func TestAuditorRole_ReadsInScopeAndChangesNothing(t *testing.T) {
	w := newWorkflow(t)
	email, _ := w.seedUser(domain.RoleAuditor, w.site)
	auditor := w.login(email)

	id := w.draft(1)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)

	w.expect(w.call("GET", "/api/v1/daily-reports/"+id, nil, auditor), 200)
	w.expect(w.call("GET", "/api/v1/audit", nil, auditor), 200)
	w.expect(w.call("GET", "/api/v1/sites/"+w.site, nil, auditor), 200)
	w.expect(w.call("GET", "/api/v1/sites/"+w.newSite(w.admin), nil, auditor), 404)

	_, body := w.goodDay(3)
	w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports", body, auditor), 403)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/corrections", map[string]any{"reason": "x"}, auditor), 403)
	w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "low_fuel", "severity": "minor", "impact": "no_outage", "title": "x", "started_at": ago(time.Hour),
	}, auditor), 403)
	w.expect(w.call("GET", "/api/v1/users", nil, auditor), 403)
}

// The drill-down behind an outage card lists each reported outage once, traced
// to its report; the reason is Coloplus's; an incident links to the outage
// during it without double counting; and "Airtel" cannot reach an unpublished
// incident's outage.
func TestDrilldown_OutagesAreTracedCountedOnceAndScoped(t *testing.T) {
	w := newWorkflow(t)
	id := w.draft(1) // an outage of 30 minutes at 11:30
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)
	day, at := w.reportDay(1)

	url := "/api/v1/dashboard/outages?from=" + day + "&to=" + day
	staff := w.expect(w.call("GET", url, nil, w.supervisor), 200)
	list := staff.list("outages")
	if len(list) != 1 || staff.Body["total_minutes"] != 30.0 {
		t.Fatalf("one 30-minute outage expected: %s", staff.Raw)
	}
	first := list[0].(map[string]any)
	if first["report_id"] != id || first["minutes"] != 30.0 || first["reason"] == nil {
		t.Fatalf("an outage must trace to its report and carry its reason for staff: %v", first)
	}
	if cust := w.expect(w.call("GET", url, nil, w.airtel), 200); strings.Contains(cust.Raw, "Grid failure") || strings.Contains(cust.Raw, `"reason"`) {
		t.Fatalf("the reason is Coloplus's own: %s", cust.Raw)
	}
	w.expect(w.call("GET", url+"&site_id="+w.newSite(w.admin), nil, w.airtel), 404)

	// An incident that started before the outage links to it, once.
	inc := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "grid_loss", "severity": "major", "impact": "outage", "title": "Grid loss", "started_at": at(11, 0),
	}, w.supervisor), 201).str("id")
	o := w.expect(w.call("GET", "/api/v1/incidents/"+inc+"/outages", nil, w.supervisor), 200)
	if o.Body["total_minutes"] != 30.0 || o.Body["open"] != true || len(o.list("outages")) != 1 {
		t.Fatalf("the incident should link to the 30 minutes of outage during it: %s", o.Raw)
	}
	w.expect(w.call("GET", "/api/v1/incidents/"+inc+"/outages", nil, w.airtel), 404) // not published
	w.expect(w.call("POST", "/api/v1/incidents/"+inc+"/publish", nil, w.supervisor), 200)
	w.expect(w.call("GET", "/api/v1/incidents/"+inc+"/outages", nil, w.airtel), 200)
	w.expect(w.call("GET", "/api/v1/incidents/"+inc+"/outages", nil, w.ot), 404)
}

// The report packs carry the same provenance as the other exports, list only
// what they should, stay inside the caller's sites and leave an audit trail.
func TestReportPacks_MonthlyWeeklyAndBacklog(t *testing.T) {
	w := newWorkflow(t)
	revEmail, _ := w.seedUser(domain.RoleAirtelReviewer, w.site)
	reviewer := w.login(revEmail)
	elseEmail, _ := w.seedUser(domain.RoleAirtelReviewer, w.newSite(w.admin))
	elsewhere := w.login(elseEmail)

	id := w.draft(1)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)
	w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "grid_loss", "severity": "major", "impact": "outage", "title": "Grid loss", "started_at": ago(time.Hour),
	}, w.supervisor), 201)

	yesterday, _ := w.reportDay(1)
	day, _ := time.Parse("2006-01-02", yesterday)
	month := day.Format("2006-01")
	monday := day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7)).Format("2006-01-02")

	table := func(r reply) ([]string, [][]string) {
		recs, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(r.Raw, "\xef\xbb\xbf"))).ReadAll()
		if err != nil || len(recs) == 0 {
			t.Fatalf("not valid CSV: %v\n%s", err, r.Raw)
		}
		return recs[0], recs[1:]
	}
	find := func(head []string, rows [][]string, siteID string) map[string]string {
		for _, r := range rows {
			if r[0] == siteID {
				m := map[string]string{}
				for i, h := range head {
					m[h] = r[i]
				}
				return m
			}
		}
		return nil
	}

	head, rows := table(w.expect(w.call("GET", "/api/v1/exports/monthly-performance.csv?month="+month, nil, reviewer), 200))
	m := find(head, rows, w.site)
	if m == nil || m["days_reported"] == "0" || m["assumed_available_minutes"] == "" || m["formula_version"] == "" ||
		m["availability_basis"] != "available by default" || m["assumptions"] == "" || m["source_minutes"] == "" {
		t.Fatalf("the monthly row must carry its figures and how they were reached: %v", m)
	}
	if _, others := table(w.expect(w.call("GET", "/api/v1/exports/monthly-performance.csv?month="+month, nil, elsewhere), 200)); len(others) != 1 || others[0][0] == w.site {
		t.Fatalf("a pack must stay inside the caller's sites: %v", others)
	}
	w.expect(w.call("GET", "/api/v1/exports/monthly-performance.csv?month=2999-01", nil, reviewer), 400)
	w.expect(w.call("GET", "/api/v1/exports/monthly-performance.csv?month=last", nil, reviewer), 400)

	head, rows = table(w.expect(w.call("GET", "/api/v1/exports/weekly-exceptions.csv?week_start="+monday, nil, reviewer), 200))
	e := find(head, rows, w.site)
	if e == nil || !strings.Contains(e["exceptions"], "outage") || e["outage_minutes"] != "30" || e["sites_in_view"] != "1" {
		t.Fatalf("the week's outage should be an exception: %v", e)
	}
	w.expect(w.call("GET", "/api/v1/exports/weekly-exceptions.csv?week_start="+yesterday+"x", nil, reviewer), 400)
	if day.Weekday() != time.Monday {
		w.expect(w.call("GET", "/api/v1/exports/weekly-exceptions.csv?week_start="+yesterday, nil, reviewer), 400) // not a Monday
	}

	head, _ = table(w.expect(w.call("GET", "/api/v1/exports/commissioning-backlog.csv", nil, reviewer), 200))
	if head[0] != "site_id" || len(head) < 10 {
		t.Fatalf("the backlog export lost its header: %v", head)
	}
	w.expect(w.call("GET", "/api/v1/exports/commissioning-backlog.csv", nil, w.ot), 403) // a reporter has no export right
	_ = w.countAudit("export.monthly_performance", "-")                                  // flushes the audit batch
	var n int
	if err := w.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_log WHERE action IN ('export.monthly_performance', 'export.weekly_exceptions', 'export.commissioning_backlog')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n < 3 {
		t.Fatalf("the packs were not all audited: %d", n)
	}
}
