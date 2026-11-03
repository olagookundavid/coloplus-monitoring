package integration

import (
	"testing"
	"time"
)

// Planned maintenance leaves the eligible time only when it was asked for in
// advance, within the cap, and a second person approved it; the person who
// asked can never approve their own.
func TestMaintenance_NoticeCapSecondPersonAndOverlap(t *testing.T) {
	w := newWorkflow(t)
	// Every administrator in the shared test database is told of a request;
	// clear those emails so they cannot crowd out other tests' mail.
	clear := func() { w.exec(`DELETE FROM notifications WHERE kind LIKE 'maintenance.%'`) }
	clear()
	t.Cleanup(clear)
	base := time.Now().In(w.loc).AddDate(0, 0, 4)
	d := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, w.loc)
	at := func(h int) string { return d.Add(time.Duration(h) * time.Hour).Format(time.RFC3339) }
	url := "/api/v1/sites/" + w.site + "/maintenance"
	ask := func(tok string, from, to int, reason string, want int) reply {
		t.Helper()
		return w.expect(w.call("POST", url, map[string]any{"starts_at": at(from), "ends_at": at(to), "reason": reason}, tok), want)
	}

	// Too soon, too long, and the customer cannot ask at all.
	soon := time.Now().Add(2 * time.Hour)
	w.expect(w.call("POST", url, map[string]any{"starts_at": soon.Format(time.RFC3339), "ends_at": soon.Add(time.Hour).Format(time.RFC3339), "reason": "Generator service"}, w.reporter), 400)
	ask(w.reporter, 0, 9, "Generator service", 400)
	ask(w.airtel, 0, 2, "Generator service", 403)

	first := ask(w.reporter, 0, 8, "Generator service", 201)
	id := first.str("id")
	// Over the monthly cap (8 + 8 > 12), and overlapping.
	ask(w.reporter, 8, 16, "More work", 400)
	ask(w.reporter, 2, 3, "Overlap", 409)

	// The person who asked cannot decide it; another person can, once.
	w.expect(w.call("POST", "/api/v1/maintenance/"+id+"/approve", nil, w.reporter), 403)
	own := ask(w.supervisor, 10, 11, "Battery check", 201)
	w.expect(w.call("POST", "/api/v1/maintenance/"+own.str("id")+"/approve", nil, w.supervisor), 403)
	w.expect(w.call("POST", "/api/v1/maintenance/"+own.str("id")+"/approve", nil, w.admin), 200)
	ok := w.expect(w.call("POST", "/api/v1/maintenance/"+id+"/approve", nil, w.supervisor), 200)
	if ok.str("status") != "approved" {
		t.Fatalf("approved = %s", ok.Raw)
	}
	w.expect(w.call("POST", "/api/v1/maintenance/"+id+"/approve", nil, w.admin), 409)

	// Cancelling needs a reason; the customer can see the window but not who asked.
	w.expect(w.call("POST", "/api/v1/maintenance/"+id+"/cancel", map[string]any{"note": "x"}, w.reporter), 400)
	seen := w.expect(w.call("GET", "/api/v1/maintenance?site_id="+w.site, nil, w.airtel), 200)
	if len(seen.list("windows")) != 2 || seen.list("windows")[0].(map[string]any)["requested_by"] != nil {
		t.Fatalf("customer view = %s", seen.Raw)
	}
	w.expect(w.call("POST", "/api/v1/maintenance/"+id+"/cancel", map[string]any{"note": "Work postponed"}, w.reporter), 200)
}

// The figures really change: an approved window comes out of the eligible time,
// and an outage still going at midnight is assumed to continue until a report
// says otherwise, shown as assumed and never as reported.
func TestFigures_MaintenanceExclusionAndMidnightCarryOver(t *testing.T) {
	w := newWorkflow(t)
	date1, at1 := w.reportDay(1)
	start, _ := time.Parse(time.RFC3339, at1(10, 0))
	end, _ := time.Parse(time.RFC3339, at1(12, 0))
	w.exec(`INSERT INTO maintenance_windows (site_id, starts_at, ends_at, reason, status, requested_by, requested_at, decided_by, decided_at)
	        VALUES ($1, $2, $3, 'Generator service', 'approved', $4, now(), (SELECT id FROM users WHERE email = $5), now())`,
		w.site, start, end, w.reporterID, w.supEmail)

	row := func(date string) map[string]any {
		t.Helper()
		r := w.expect(w.call("GET", "/api/v1/dashboard/overview?date="+date, nil, w.airtel), 200)
		for _, s := range r.list("sites") {
			if m := s.(map[string]any); m["site_id"] == w.site {
				return m["summary"].(map[string]any)
			}
		}
		t.Fatalf("site missing: %s", r.Raw)
		return nil
	}

	// Yesterday has no report: its window is assumed available, less the two hours.
	sum := row(date1)
	if sum["eligible_minutes"] != 1320.0 || sum["excluded_maintenance_minutes"] != 120.0 {
		t.Fatalf("maintenance not taken out: %v", sum)
	}

	// A whole-day outage two days ago, approved, runs to the end of that day.
	date2, _ := w.reportDay(2)
	id := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports",
		map[string]any{"report_date": date2, "power_status": "outage", "generator_hours": 0.0, "outage_reason": "Grid and generator down"}, w.reporter), 201).str("id")
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", map[string]any{"late_reason": "entered later"}, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)

	// Yesterday nobody reported: the outage is assumed to continue, as assumed outage.
	sum = row(date1)
	if sum["assumed_outage_minutes"] != 1320.0 || sum["outage_minutes"] != 0.0 {
		t.Fatalf("carried outage must be assumed, not reported: %v", sum)
	}
	if sum["reported_availability_pct"] != 0.0 {
		t.Fatalf("an assumed outage still counts against availability: %v", sum)
	}
	// Once yesterday is reported as served from midnight, nothing carries.
	_, body := w.goodDay(1)
	body["intervals"] = []any{iv(at1(0, 0), at1(24, 0), "served", "grid")}
	id = w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports", body, w.reporter), 201).str("id")
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)
	if sum = row(date1); sum["assumed_outage_minutes"] != 0.0 {
		t.Fatalf("a report from midnight settles it: %v", sum)
	}
}
