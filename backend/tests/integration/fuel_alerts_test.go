package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"coloplus-backend/internal/domain"
)

// A low tank is emailed once per episode to the people who record fuel, never
// to the customer; the customer and the dashboard see the same estimate; a
// reading far from the estimate is flagged for review rather than corrected;
// and the dashboard says how much of a site rests on reports.
func TestFuel_LowAlertsVarianceAndDashboardRows(t *testing.T) {
	w := newWorkflow(t)
	audEmail, _ := w.seedUser(domain.RoleAuditor, w.site)
	auditor := w.login(audEmail)
	w.exec(`UPDATE sites SET consumption_lph = 1, tank_capacity_litres = 2000 WHERE id = $1`, w.site)
	fuelURL := "/api/v1/sites/" + w.site + "/fuel"
	_, at4 := w.reportDay(4)
	post := func(body map[string]any, tok string, want int) reply {
		t.Helper()
		return w.expect(w.call("POST", fuelURL, body, tok), want)
	}

	// 40 litres delivered four days ago, 20 hours of generator since: about 20 left.
	post(map[string]any{"kind": "delivery", "litres": 40, "occurred_at": at4(9, 0)}, w.reporter, 201)
	report := func(daysAgo int, body map[string]any) {
		t.Helper()
		date, _ := w.reportDay(daysAgo)
		body["report_date"] = date
		id := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports", body, w.reporter), 201).str("id")
		w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", map[string]any{"late_reason": "entered later"}, w.reporter), 200)
		w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)
	}
	report(4, map[string]any{"power_status": "served"})
	report(3, map[string]any{"power_status": "outage", "generator_hours": 10.0, "outage_reason": "Grid down"})
	report(2, map[string]any{"power_status": "outage", "generator_hours": 10.0, "outage_reason": "Grid down"})
	report(1, map[string]any{"power_status": "served"})

	// The estimate is low (about 20 hours left, at or under 24), and the flag says so.
	est := w.expect(w.call("GET", fuelURL, nil, w.airtel), 200).Body["estimate"].(map[string]any)
	if est["litres"] != 20.0 || est["low"] != true || est["hours_remaining"] != 20.0 {
		t.Fatalf("40 L less 20 hours should be a low 20 L estimate: %v", est)
	}

	// The alert goes to the people who record fuel, once, and not to the customer.
	alert := func() int {
		t.Helper()
		n, err := w.fuel.AlertLow(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		w.dispatch()
		return n
	}
	alert()
	for _, addr := range []string{w.repEmail, w.supEmail} {
		mails := w.mailer.to(addr)
		if len(mails) != 1 || !strings.Contains(mails[0].Subject, "Fuel running low") || !strings.Contains(mails[0].Text, "estimate") {
			t.Fatalf("%s should get one low-fuel email that says it is an estimate: %v", addr, mails)
		}
	}
	if got := len(w.mailer.to(w.airEmail)); got != 0 {
		t.Fatalf("the customer was emailed %d times about low fuel", got)
	}
	alert()
	if got := len(w.mailer.to(w.repEmail)); got != 1 {
		t.Fatalf("an unchanged low tank must not be repeated, got %d emails", got)
	}

	// The same estimate is on the dashboard row for those who may see fuel, and
	// not for the auditor, who may not.
	date, _ := w.reportDay(1)
	row := func(tok string) map[string]any {
		for _, r := range w.expect(w.call("GET", "/api/v1/dashboard/overview?date="+date, nil, tok), 200).list("sites") {
			if m := r.(map[string]any); m["site_id"] == w.site {
				return m
			}
		}
		t.Fatal("site missing from the dashboard")
		return nil
	}
	if f, _ := row(w.airtel)["fuel"].(map[string]any); f == nil || f["litres"] != 20.0 || f["low"] != true || f["basis"] != "estimate" {
		t.Fatalf("the dashboard row should carry the low estimate: %v", row(w.airtel)["fuel"])
	}
	if row(auditor)["fuel"] != nil {
		t.Fatalf("the auditor has no fuel access and must not get it from the dashboard: %v", row(auditor)["fuel"])
	}

	// A tank reading close to the estimate is fine; one far from it is flagged,
	// and the estimate then restarts from the reading either way.
	_, at0 := w.reportDay(0)
	_ = at0
	near := post(map[string]any{"kind": "reading", "litres": 18, "occurred_at": ago(30 * time.Minute)}, w.reporter, 201)
	v, _ := near.Body["variance"].(map[string]any)
	if v == nil || v["estimated_before_litres"] != 20.0 || v["difference_litres"] != -2.0 || v["review_warning"] != false {
		t.Fatalf("18 L found against 20 expected is a small gap: %s", near.Raw)
	}
	far := post(map[string]any{"kind": "reading", "litres": 500, "occurred_at": ago(10 * time.Minute)}, w.reporter, 201)
	if v, _ := far.Body["variance"].(map[string]any); v == nil || v["review_warning"] != true {
		t.Fatalf("500 L found against about 18 expected must be flagged for review: %s", far.Raw)
	}
	if e := w.expect(w.call("GET", fuelURL, nil, w.airtel), 200).Body["estimate"].(map[string]any); e["litres"] != 500.0 || e["low"] != false {
		t.Fatalf("the reading wins and the estimate restarts from it: %v", e)
	}
	alert()
	if got := len(w.mailer.to(w.repEmail)); got != 1 {
		t.Fatalf("a refilled tank must not alert, got %d emails", got)
	}

	// Data quality: four of the last six days were reported.
	from, _ := w.reportDay(6)
	q := w.expect(w.call("GET", "/api/v1/dashboard/overview?from="+from+"&to="+date, nil, w.airtel), 200)
	dq, _ := q.Body["data_quality"].(map[string]any)
	if dq["reported_days"] != 4.0 || dq["days"] != 6.0 || dq["completeness_pct"] != 66.7 || dq["level"] != "poor" || dq["late_reports"] != 3.0 {
		t.Fatalf("4 of 6 days reported, 3 of them late, is poor quality: %v", dq)
	}
}
