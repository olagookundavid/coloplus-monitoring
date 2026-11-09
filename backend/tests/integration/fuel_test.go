package integration

import (
	"strings"
	"testing"
	"time"

	"coloplus-backend/internal/domain"
)

// The fuel level is an estimate from what people entered and the agreed rate.
// It must follow the arithmetic, never invent a number without a rate, keep
// internal notes from the customer, hide voided entries from the customer, and
// stay inside the caller's sites.
func TestFuel_EstimateFollowsTheRateAndNothingInternalLeaks(t *testing.T) {
	w := newWorkflow(t)
	elseEmail, _ := w.seedUser(domain.RoleViewer, w.newSite(w.admin))
	elsewhere := w.login(elseEmail)
	fuelURL := "/api/v1/sites/" + w.site + "/fuel"

	// No agreed rate: unavailable, with the reason, and never a number.
	st := w.expect(w.call("GET", fuelURL, nil, w.customer), 200)
	est := st.Body["estimate"].(map[string]any)
	if est["available"] != false || est["litres"] != nil || est["reason"] == "" {
		t.Fatalf("with no agreed rate there must be no number: %s", st.Raw)
	}

	w.exec(`UPDATE sites SET consumption_lph = 1, tank_capacity_litres = 2000 WHERE id = $1`, w.site)

	// A delivery of 1,000 litres four days ago; the generator then ran 10 + 10 hours.
	_, at4 := w.reportDay(4)
	post := func(body map[string]any, tok string, want int) reply {
		t.Helper()
		return w.expect(w.call("POST", fuelURL, body, tok), want)
	}
	post(map[string]any{"kind": "delivery", "litres": 1000, "occurred_at": at4(9, 0)}, w.customer, 403)       // customers do not record
	post(map[string]any{"kind": "delivery", "litres": 5000, "occurred_at": at4(9, 0)}, w.reporter, 400)       // more than the tank holds
	post(map[string]any{"kind": "delivery", "litres": 1000, "occurred_at": ago(-time.Hour)}, w.reporter, 400) // the future
	first := post(map[string]any{
		"kind": "delivery", "litres": 1000, "occurred_at": at4(9, 0), "note": "internal: driver Ade", "client_key": "k1",
	}, w.reporter, 201)
	if again := post(map[string]any{"kind": "delivery", "litres": 1000, "occurred_at": at4(9, 0), "client_key": "k1"}, w.reporter, 201); again.str("id") != first.str("id") {
		t.Fatalf("a retried entry made a second delivery")
	}
	w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/fuel", map[string]any{"kind": "delivery", "litres": 1, "occurred_at": at4(9, 0)}, w.ot), 404)

	// Reported days: four ago (grid all day), then two outage days of 10 hours, then yesterday on grid.
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

	level := func(tok string) map[string]any {
		return w.expect(w.call("GET", fuelURL, nil, tok), 200).Body["estimate"].(map[string]any)
	}
	if e := level(w.customer); e["available"] != true || e["litres"] != 980.0 || e["basis"] != "estimate" ||
		e["provisional"] != false || e["assumed_empty_start"] != true {
		t.Fatalf("1,000 L less 20 hours at 1 L/h should be an estimated 980: %v", e)
	}

	// A day nobody reported is unknown, not zero use: flagged, and only an upper bound.
	w.exec(`UPDATE daily_reports SET status = 'superseded' WHERE site_id = $1 AND report_date = (SELECT report_date FROM daily_reports WHERE site_id = $1 ORDER BY report_date DESC LIMIT 1 OFFSET 1)`, w.site)
	if e := level(w.supervisor); e["unreported_days"] != 1.0 || e["provisional"] != true {
		t.Fatalf("a missing day must be flagged: %v", e)
	}

	// Coloplus sees notes and who recorded; the customer sees neither.
	if raw := w.expect(w.call("GET", fuelURL, nil, w.customer), 200).Raw; strings.Contains(raw, "driver Ade") || strings.Contains(raw, "recorded_by") {
		t.Fatalf("the customer sees Coloplus-only detail: %s", raw)
	}
	if raw := w.expect(w.call("GET", fuelURL, nil, w.supervisor), 200).Raw; !strings.Contains(raw, "driver Ade") {
		t.Fatalf("Coloplus should see the note: %s", raw)
	}

	// A mistaken entry is voided with a reason, stays on record for Coloplus,
	// vanishes for the customer, and no longer counts.
	id := first.str("id")
	w.expect(w.call("POST", "/api/v1/fuel/"+id+"/void", map[string]any{"reason": "typo"}, w.reporter), 403)
	w.expect(w.call("POST", "/api/v1/fuel/"+id+"/void", map[string]any{"reason": "x"}, w.supervisor), 400)
	w.expect(w.call("POST", "/api/v1/fuel/"+id+"/void", map[string]any{"reason": "Entered against the wrong site"}, w.supervisor), 200)
	if e := level(w.customer); e["available"] != false {
		t.Fatalf("with its only entry voided there is nothing to estimate from: %v", e)
	}
	if n := len(w.expect(w.call("GET", fuelURL, nil, w.customer), 200).list("entries")); n != 0 {
		t.Fatalf("a voided entry reached the customer")
	}
	if n := len(w.expect(w.call("GET", fuelURL, nil, w.supervisor), 200).list("entries")); n != 1 {
		t.Fatalf("Coloplus should still see the voided entry")
	}
	w.expect(w.call("GET", fuelURL, nil, elsewhere), 404)
	if c := w.countAudit("fuel.voided", id); c != 1 {
		t.Fatalf("voiding was not audited: %d", c)
	}
}
