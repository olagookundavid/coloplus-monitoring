package integration

import (
	"strings"
	"testing"

	"coloplus-backend/internal/domain"
)

// Alerts wait in the alert centre until the person acknowledges them. A person
// sees and clears only their own, and somebody who has not finished setting up
// two-step sign-in cannot use the centre.
func TestAlertCentre_OwnAlertsOnlyAndAcknowledgement(t *testing.T) {
	w := newWorkflow(t)

	w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/tickets", map[string]any{
		"category": "low_fuel", "priority": "high", "subject": "Diesel refill", "description": "Tank is low",
	}, w.airtel), 201)

	// The supervisor and the reporter on the site... only those with a say in tickets are told.
	sup := w.expect(w.call("GET", "/api/v1/alerts?unread=true", nil, w.supervisor), 200)
	alerts := sup.list("alerts")
	if len(alerts) != 1 || sup.Body["unread"] != 1.0 || !strings.Contains(alerts[0].(map[string]any)["title"].(string), "Diesel refill") {
		t.Fatalf("the supervisor should have one open alert about the ticket: %s", sup.Raw)
	}
	id := alerts[0].(map[string]any)["id"].(string)
	if got := w.expect(w.call("GET", "/api/v1/alerts", nil, w.airtel), 200).Body["unread"]; got != 0.0 {
		t.Fatalf("the person who raised the ticket is not alerted to their own request: %v", got)
	}

	// Nobody else can see or clear it.
	w.expect(w.call("POST", "/api/v1/alerts/"+id+"/acknowledge", nil, w.airtel), 404)
	w.expect(w.call("POST", "/api/v1/alerts/"+id+"/acknowledge", nil, w.reporter), 404)
	if got := w.expect(w.call("GET", "/api/v1/alerts", nil, w.supervisor), 200).Body["unread"]; got != 1.0 {
		t.Fatalf("another person's attempt must change nothing: %v", got)
	}

	// The owner acknowledges, once or twice; the alert stays on record.
	w.expect(w.call("POST", "/api/v1/alerts/"+id+"/acknowledge", nil, w.supervisor), 204)
	w.expect(w.call("POST", "/api/v1/alerts/"+id+"/acknowledge", nil, w.supervisor), 204)
	after := w.expect(w.call("GET", "/api/v1/alerts", nil, w.supervisor), 200)
	if after.Body["unread"] != 0.0 || after.Body["total"] != 1.0 || after.list("alerts")[0].(map[string]any)["acknowledged_at"] == nil {
		t.Fatalf("acknowledged alerts stay listed, with the time: %s", after.Raw)
	}
	if n := len(w.expect(w.call("GET", "/api/v1/alerts?unread=true", nil, w.supervisor), 200).list("alerts")); n != 0 {
		t.Fatalf("an acknowledged alert is not unread: %d", n)
	}
	if got := w.expect(w.call("POST", "/api/v1/alerts/acknowledge-all", nil, w.supervisor), 200).Body["acknowledged"]; got != 0.0 {
		t.Fatalf("nothing was left to acknowledge: %v", got)
	}

	// Account emails are not alerts, and an unfinished two-step setup holds no access.
	email, _ := w.seedUserWithoutMFA(domain.RoleAirtelViewer, w.site)
	w.expect(w.call("GET", "/api/v1/alerts", nil, w.login(email)), 403)
}
