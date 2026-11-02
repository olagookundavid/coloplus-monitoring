package integration

import (
	"strings"
	"testing"
	"time"

	"coloplus-backend/internal/domain"
)

// A ticket is Airtel's request and Coloplus's work. Airtel must never see
// Coloplus's internal notes or an incident that has not been published to
// them, nobody outside the site can reach it, a retried request makes one
// ticket, and the contact address typed into a ticket is never emailed.
func TestTickets_AirtelRaisesColoplusWorksAndNothingInternalLeaks(t *testing.T) {
	w := newWorkflow(t)
	elseEmail, _ := w.seedUser(domain.RoleAirtelReviewer, w.newSite(w.admin))
	elsewhere := w.login(elseEmail)

	raise := map[string]any{
		"category": "low_fuel", "priority": "high", "subject": "Diesel refill", "description": "Tank is low",
		"contact_email": "someone.else@external.example", "client_key": "k-1",
	}
	url := "/api/v1/sites/" + w.site + "/tickets"
	w.expect(w.call("POST", url, raise, w.reporter), 403) // field staff do not raise tickets
	w.expect(w.call("POST", url, raise, elsewhere), 404)  // not their site
	bad := map[string]any{"category": "weather", "priority": "high", "subject": "x", "description": "x"}
	w.expect(w.call("POST", url, bad, w.airtel), 400)

	first := w.expect(w.call("POST", url, raise, w.airtel), 201)
	id := first.str("id")
	if again := w.expect(w.call("POST", url, raise, w.airtel), 201); again.str("id") != id {
		t.Fatalf("a retried request made a second ticket: %s vs %s", again.str("id"), id)
	}
	if first.str("status") != "open" || first.Body["raised_by"] != nil {
		t.Fatalf("a new ticket is open and Airtel does not get staff ids: %s", first.Raw)
	}
	w.dispatch()
	if got := len(w.mailer.to(w.supEmail)); got != 1 {
		t.Fatalf("the supervisor should hear of the new ticket once, got %d", got)
	}

	w.expect(w.call("GET", "/api/v1/tickets/"+id, nil, elsewhere), 404)
	w.expect(w.call("GET", "/api/v1/tickets/"+id, nil, w.ot), 403)
	if l := w.expect(w.call("GET", "/api/v1/tickets", nil, elsewhere), 200); len(l.list("tickets")) != 0 {
		t.Fatalf("a ticket leaked to another site's user: %s", l.Raw)
	}

	// Coloplus works it: an internal note stays inside, a reply goes out.
	post := func(path string, body map[string]any, tok string, want int) reply {
		t.Helper()
		return w.expect(w.call("POST", "/api/v1/tickets/"+id+path, body, tok), want)
	}
	post("/comments", map[string]any{"body": "internal: driver is on leave", "internal": true}, w.airtel, 403)
	post("/comments", map[string]any{"body": "internal: driver is on leave", "internal": true}, w.supervisor, 201)
	post("/comments", map[string]any{"body": "A refill is booked for tomorrow."}, w.supervisor, 201)
	post("/transition", map[string]any{"status": "in_progress"}, w.airtel, 409)
	post("/transition", map[string]any{"status": "in_progress"}, w.supervisor, 200)
	post("/transition", map[string]any{"status": "resolved"}, w.supervisor, 400) // needs a note
	w.dispatch()
	mails := w.mailer.to(w.airEmail)
	if len(mails) != 2 || strings.Contains(mails[0].Text+mails[1].Text, "driver is on leave") {
		t.Fatalf("Airtel should get the reply and the status change, never the internal note: %d mails", len(mails))
	}
	if got := len(w.mailer.to("someone.else@external.example")); got != 0 {
		t.Fatalf("the contact address was emailed %d times", got)
	}

	// An incident is Coloplus's record: Airtel hears of it only once published.
	inc := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "low_fuel", "severity": "minor", "impact": "no_outage", "title": "Fuel low", "started_at": ago(time.Hour),
	}, w.supervisor), 201).str("id")
	other := w.newSite(w.admin)
	w.expect(w.call("PATCH", "/api/v1/users/"+w.reporterID, map[string]any{"site_ids": []string{w.site, other}}, w.admin), 200)
	farInc := w.expect(w.call("POST", "/api/v1/sites/"+other+"/incidents", map[string]any{
		"category": "low_fuel", "severity": "minor", "impact": "no_outage", "title": "Elsewhere", "started_at": ago(time.Hour),
	}, w.reporter), 201).str("id")
	post("/incident", map[string]any{"incident_id": farInc}, w.supervisor, 400)
	post("/incident", map[string]any{"incident_id": inc}, w.airtel, 403)
	post("/incident", map[string]any{"incident_id": inc}, w.supervisor, 200)

	if v := w.expect(w.call("GET", "/api/v1/tickets/"+id, nil, w.airtel), 200); v.Body["ticket"].(map[string]any)["incident_id"] != nil ||
		strings.Contains(v.Raw, "driver is on leave") || strings.Contains(v.Raw, "Linked to") {
		t.Fatalf("Airtel sees what is internal: %s", v.Raw)
	}
	if v := w.expect(w.call("GET", "/api/v1/tickets/"+id, nil, w.supervisor), 200); v.Body["ticket"].(map[string]any)["incident_id"] != inc ||
		!strings.Contains(v.Raw, "driver is on leave") {
		t.Fatalf("staff should see the link and the internal note: %s", v.Raw)
	}
	w.expect(w.call("POST", "/api/v1/incidents/"+inc+"/publish", nil, w.supervisor), 200)
	if v := w.expect(w.call("GET", "/api/v1/tickets/"+id, nil, w.airtel), 200); v.Body["ticket"].(map[string]any)["incident_id"] != inc {
		t.Fatalf("a published incident should show on the ticket: %s", v.Raw)
	}

	// Resolve, then Airtel can reopen only with a reason.
	post("/transition", map[string]any{"status": "resolved", "note": "Refilled."}, w.supervisor, 200)
	post("/transition", map[string]any{"status": "open"}, w.airtel, 400)
	reopened := post("/transition", map[string]any{"status": "open", "note": "Still low."}, w.airtel, 200)
	if reopened.str("status") != "open" || reopened.Body["resolution"] != nil || reopened.Body["reopen_count"] != float64(1) {
		t.Fatalf("a reopened ticket is open again with no resolution: %s", reopened.Raw)
	}
	if n := w.countAudit("ticket.incident_linked", id); n != 1 {
		t.Fatalf("linking was not audited: %d", n)
	}
}
