package integration

import (
	"testing"
	"time"

	"coloplus-backend/internal/domain"
)

// The trust rules of an incident correction: it never touches money or
// reports, but it still gets a second person, a transactional audit row, and
// a kept original -- plus the one rule unique to this entity, that the same
// incident and field cannot have two requests open at once.
func TestIncidentCorrection_ApprovalIsAuditedAndKeepsTheOriginal(t *testing.T) {
	w := newWorkflow(t)
	incidentID := mustReport(w)
	api := "/api/v1/incidents/" + incidentID + "/corrections"
	corrAPI := "/api/v1/incident-corrections/"
	body := map[string]any{"field": "restored_at", "corrected_value": ago(90 * time.Minute), "reason": "Ticket shows an earlier restore time"}

	r := w.expect(w.call("POST", api, body, w.supervisor), 201)
	corrID := r.str("id")
	if r.str("status") != "requested" {
		t.Fatalf("status = %q, want requested", r.str("status"))
	}
	original := r.str("original_value")

	// A second open request for the same incident and field is refused.
	w.expect(w.call("POST", api, body, w.supervisor), 409)

	// Nobody approves their own request, however privileged.
	w.expect(w.call("POST", corrAPI+corrID+"/approve", nil, w.supervisor), 403)

	// A reporter has no incidents.manage at all.
	w.expect(w.call("POST", corrAPI+corrID+"/approve", nil, w.reporter), 403)

	// A second supervisor approves.
	supEmail2, _ := w.seedUser(domain.RoleSupervisor, w.site)
	supervisor2 := w.login(supEmail2)

	before := w.expect(w.call("GET", "/api/v1/incidents/"+incidentID, nil, w.supervisor), 200).str("incident", "restored_at")
	w.expect(w.call("POST", corrAPI+corrID+"/approve", nil, supervisor2), 200)
	after := w.expect(w.call("GET", "/api/v1/incidents/"+incidentID, nil, w.supervisor), 200).str("incident", "restored_at")
	if after == before {
		t.Fatalf("approving the correction did not change restored_at (still %q)", after)
	}

	if n := w.countAudit("incident.correction_approved", corrID); n != 1 {
		t.Fatalf("correction_approved audit rows = %d, want 1", n)
	}

	got := w.expect(w.call("GET", corrAPI+corrID, nil, w.supervisor), 200)
	if got.str("status") != "approved" {
		t.Fatalf("status after approval = %q", got.str("status"))
	}
	if got.str("original_value") != original {
		t.Fatalf("original_value changed after approval: got %q, want %q", got.str("original_value"), original)
	}

	// A decided correction cannot be approved again.
	w.expect(w.call("POST", corrAPI+corrID+"/approve", nil, supervisor2), 409)
}

// mustReport logs an open, restored incident and returns its id.
func mustReport(w *workflow) string {
	w.t.Helper()
	id := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "grid_loss", "severity": "major", "impact": "outage", "title": "Grid supply lost",
		"started_at": ago(3 * time.Hour),
	}, w.reporter), 201).str("id")
	w.expect(w.call("POST", "/api/v1/incidents/"+id+"/transition",
		map[string]any{"status": "restored", "at": ago(time.Hour)}, w.reporter), 200)
	return id
}
