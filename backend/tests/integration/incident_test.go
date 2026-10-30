package integration

import (
	"strings"
	"testing"
	"time"
)

func ago(d time.Duration) string { return time.Now().Add(-d).UTC().Format(time.RFC3339) }

// The trust rules of the fault register: restoration is an observed time, it
// is verified by a second person with a new observation, a closed incident is
// frozen, and Airtel sees only what was published for them.
func TestIncident_RestorationNeedsAnObservationAndASecondPerson(t *testing.T) {
	w := newWorkflow(t)
	api := "/api/v1/incidents/"

	r := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "generator_failure", "severity": "critical", "impact": "outage",
		"title": "Generator failed to start", "description": "internal: fuel line leaking",
		"started_at": ago(3 * time.Hour),
	}, w.reporter), 201)
	id := r.str("id")

	// Out of scope, and invisible to Airtel until published.
	w.expect(w.call("GET", api+id, nil, w.ot), 404)
	w.expect(w.call("GET", api+id, nil, w.airtel), 404)
	w.expect(w.call("POST", api+id+"/updates", map[string]any{"body": "hello", "customer_visible": true}, w.supervisor), 409)

	// A field officer cannot run the workflow beyond recording restoration.
	w.expect(w.call("POST", api+id+"/transition", map[string]any{"status": "acknowledged"}, w.reporter), 403)

	// Work needs an owner.
	w.expect(w.call("POST", api+id+"/transition", map[string]any{"status": "in_progress"}, w.supervisor), 400)
	w.expect(w.call("POST", api+id+"/assign", map[string]any{"owner_id": w.reporterID, "next_action": "Bring fuel"}, w.supervisor), 200)
	if st := w.expect(w.call("GET", api+id, nil, w.supervisor), 200).str("incident", "status"); st != "assigned" {
		t.Fatalf("status after assigning = %q", st)
	}

	// Restoration is an observed time: not missing, not in the future.
	w.expect(w.call("POST", api+id+"/transition", map[string]any{"status": "restored"}, w.reporter), 400)
	w.expect(w.call("POST", api+id+"/transition", map[string]any{"status": "restored", "at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}, w.reporter), 400)
	restored := map[string]any{"status": "restored", "at": ago(time.Hour)}
	w.expect(w.call("POST", api+id+"/transition", restored, w.reporter), 200)
	w.expect(w.call("POST", api+id+"/transition", restored, w.reporter), 200) // a retry changes nothing

	// Verification needs a new observation, and cannot skip ahead.
	w.expect(w.call("POST", api+id+"/transition", map[string]any{"status": "closed", "root_cause": "x"}, w.supervisor), 409)
	w.expect(w.call("POST", api+id+"/transition", map[string]any{"status": "verified", "at": ago(10 * time.Minute)}, w.supervisor), 400)
	w.expect(w.call("POST", api+id+"/transition", map[string]any{"status": "verified", "at": ago(10 * time.Minute), "note": "Site called: lights on"}, w.supervisor), 200)

	// A closed incident needs a root cause, then cannot change.
	w.expect(w.call("POST", api+id+"/transition", map[string]any{"status": "closed"}, w.supervisor), 400)
	w.expect(w.call("POST", api+id+"/transition", map[string]any{"status": "closed", "root_cause": "Fuel line fatigue"}, w.supervisor), 200)
	w.expect(w.call("POST", api+id+"/updates", map[string]any{"body": "late note"}, w.supervisor), 409)
	if n := w.countAudit("incident.closed", id); n != 1 {
		t.Fatalf("closed audit rows = %d, want 1", n)
	}

	// Sharing with Airtel: only the published facts and customer updates.
	w.expect(w.call("POST", api+id+"/publish", nil, w.reporter), 403)
}

func TestIncident_AVerifierMustNotBeTheRestorer(t *testing.T) {
	w := newWorkflow(t)
	api := "/api/v1/incidents/"
	id := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "low_fuel", "severity": "minor", "impact": "no_outage", "title": "Fuel low",
		"started_at": ago(2 * time.Hour),
	}, w.supervisor), 201).str("id")

	w.expect(w.call("POST", api+id+"/transition", map[string]any{"status": "restored", "at": ago(time.Hour)}, w.supervisor), 200)
	r := w.call("POST", api+id+"/transition", map[string]any{"status": "verified", "at": ago(5 * time.Minute), "note": "checked"}, w.supervisor)
	w.expect(r, 400)

	// Reopening drops the earlier restoration, and the history keeps it.
	w.expect(w.call("POST", api+id+"/transition", map[string]any{"status": "in_progress"}, w.supervisor), 400) // no owner, no reason
	w.expect(w.call("POST", api+id+"/assign", map[string]any{"owner_id": w.reporterID}, w.supervisor), 200)
	w.expect(w.call("POST", api+id+"/transition", map[string]any{"status": "in_progress", "note": "Fuel ran out again"}, w.supervisor), 200)
	got := w.expect(w.call("GET", api+id, nil, w.supervisor), 200)
	if got.str("incident", "restored_at") != "" {
		t.Fatal("a reopened incident must not keep its restoration time")
	}
	if n := len(got.list("updates")); n < 4 {
		t.Fatalf("history has %d entries, want the whole trail", n)
	}
}

func TestIncident_AirtelSeesOnlyWhatIsPublishedForThem(t *testing.T) {
	w := newWorkflow(t)
	api := "/api/v1/incidents/"
	id := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "grid_loss", "severity": "major", "impact": "outage", "title": "Grid supply lost",
		"description": "internal: transformer blown", "started_at": ago(2 * time.Hour),
	}, w.reporter), 201).str("id")

	w.expect(w.call("POST", api+id+"/updates", map[string]any{"body": "internal chatter"}, w.reporter), 201)
	w.expect(w.call("POST", api+id+"/publish", nil, w.supervisor), 200)
	w.expect(w.call("POST", api+id+"/updates", map[string]any{"body": "Engineer on site, restoration expected tonight", "customer_visible": true}, w.supervisor), 201)
	// A reporter cannot speak to the customer.
	w.expect(w.call("POST", api+id+"/updates", map[string]any{"body": "x", "customer_visible": true}, w.reporter), 403)

	got := w.expect(w.call("GET", api+id, nil, w.airtel), 200)
	if got.Raw == "" || strings.Contains(got.Raw, "transformer") || strings.Contains(got.Raw, "internal chatter") || strings.Contains(got.Raw, "description") {
		t.Fatalf("Airtel saw internal detail: %s", got.Raw)
	}
	if !strings.Contains(got.Raw, "Engineer on site") {
		t.Fatalf("Airtel should see the customer update: %s", got.Raw)
	}
	list := w.expect(w.call("GET", "/api/v1/incidents", nil, w.airtel), 200)
	if len(list.list("published")) != 1 || len(list.list("incidents")) != 0 {
		t.Fatalf("Airtel list = %s", list.Raw)
	}
	w.expect(w.call("POST", api+id+"/assign", map[string]any{"owner_id": w.reporterID}, w.airtel), 403)
}
