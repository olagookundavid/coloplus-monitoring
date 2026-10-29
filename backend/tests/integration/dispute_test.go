package integration

import (
	"context"
	"strings"
	"testing"

	"coloplus-backend/internal/domain"
)

// The trust rules of disputes: only Airtel's reviewers raise them, only Coloplus
// settles them, an upheld one ends in a real correction, Airtel's agreement is
// its own act, Airtel never sees Coloplus's internal side, and a closed
// dispute is frozen.
func TestDispute_BothSidesAndTheTrustRules(t *testing.T) {
	w := newWorkflow(t)
	api := "/api/v1/disputes/"

	revEmail, _ := w.seedUser(domain.RoleAirtelReviewer, w.site)
	reviewer := w.login(revEmail)
	otherSite := w.newSite(w.admin)
	elseEmail, _ := w.seedUser(domain.RoleAirtelReviewer, otherSite)
	elsewhere := w.login(elseEmail)

	// One published report, with an outage interval to dispute, and one that is still a draft.
	orig := w.draft(1)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/approve", nil, w.supervisor), 200)
	draftOnly := w.draft(2)

	published := w.expect(w.call("GET", "/api/v1/daily-reports/"+orig, nil, reviewer), 200)
	var outageID string
	for _, v := range published.Body["published"].(map[string]any)["intervals"].([]any) {
		if iv := v.(map[string]any); iv["state"] == "outage" {
			outageID = iv["id"].(string)
		}
	}
	if outageID == "" {
		t.Fatalf("no outage interval in %s", published.Raw)
	}

	raise := func(body map[string]any, tok string) reply {
		return w.call("POST", "/api/v1/sites/"+w.site+"/disputes", body, tok)
	}
	dispute := map[string]any{
		"report_id": orig, "interval_id": outageID, "category": "wrong_duration",
		"title": "Outage shown longer than it was", "description": "Power was back at 11:45.",
	}

	// Only a reviewer raises, only within their sites, and only about a published report.
	w.expect(raise(dispute, w.airtel), 403)
	w.expect(raise(dispute, w.supervisor), 403)
	w.expect(raise(dispute, elsewhere), 404)
	w.expect(raise(map[string]any{"report_id": draftOnly, "category": "other", "title": "x"}, reviewer), 400)
	w.expect(raise(map[string]any{"report_id": orig, "interval_id": "00000000-0000-0000-0000-000000000000", "category": "other", "title": "x"}, reviewer), 400)

	id := w.expect(raise(dispute, reviewer), 201).str("id")
	w.expect(raise(dispute, reviewer), 409) // the same thing is not disputed twice at once

	// A disputed figure keeps showing it.
	if n := w.expect(w.call("GET", "/api/v1/daily-reports/"+orig, nil, w.airtel), 200).Body["open_disputes"]; n != float64(1) {
		t.Fatalf("open_disputes on the report = %v, want 1", n)
	}

	// Scope: the viewer reads it, a stranger cannot, a reporter has no business with it.
	w.expect(w.call("GET", api+id, nil, w.airtel), 200)
	w.expect(w.call("GET", api+id, nil, elsewhere), 404)
	w.expect(w.call("GET", api+id, nil, w.reporter), 403)

	// Each side may only make its own moves.
	step := func(tok string, body map[string]any) reply { return w.call("POST", api+id+"/transition", body, tok) }
	w.expect(step(reviewer, map[string]any{"status": "under_review", "note": "x"}), 409) // Airtel cannot start Coloplus's review
	w.expect(step(w.airtel, map[string]any{"status": "closed", "note": "x"}), 403)       // a viewer cannot act at all
	w.expect(step(w.supervisor, map[string]any{"status": "under_review"}), 200)
	if owner := w.expect(w.call("GET", api+id, nil, w.supervisor), 200).str("dispute", "owner_id"); owner == "" {
		t.Fatal("taking a dispute for review should make it the supervisor's")
	}
	w.expect(step(w.supervisor, map[string]any{"status": "closed", "note": "x"}), 409) // Coloplus cannot close it for Airtel

	// Internal notes stay inside Coloplus.
	const secret = "internal: we think the logger drifted"
	w.expect(w.call("POST", api+id+"/comments", map[string]any{"body": secret, "internal": true}, reviewer), 403)
	w.expect(w.call("POST", api+id+"/comments", map[string]any{"body": secret, "internal": true}, w.supervisor), 201)
	w.expect(w.call("POST", api+id+"/comments", map[string]any{"body": "We are checking the site log."}, w.supervisor), 201)

	// Evidence requested; an Airtel comment hands it back.
	w.expect(step(w.supervisor, map[string]any{"status": "awaiting_evidence"}), 400) // say what is needed
	w.expect(step(w.supervisor, map[string]any{"status": "awaiting_evidence", "note": "Send the site's generator log."}), 200)
	w.expect(w.call("POST", api+id+"/comments", map[string]any{"body": "Log attached to the shared folder."}, reviewer), 201)
	if st := w.expect(w.call("GET", api+id, nil, w.supervisor), 200).str("dispute", "status"); st != "under_review" {
		t.Fatalf("status after Airtel's evidence = %q, want under_review", st)
	}

	// Upheld means a real correction: agreed first, then a later approved revision of that day.
	w.expect(step(w.supervisor, map[string]any{"status": "resolved", "outcome": "upheld", "note": "Corrected."}), 400)
	w.expect(step(w.supervisor, map[string]any{"status": "agreed_correction", "note": "Outage was 15 minutes."}), 200)
	w.expect(step(w.supervisor, map[string]any{"status": "resolved", "outcome": "not_upheld", "note": "No."}), 400) // cannot walk away from an agreed correction
	w.expect(step(w.supervisor, map[string]any{"status": "resolved", "outcome": "upheld", "note": "Corrected."}), 400)
	w.expect(step(w.supervisor, map[string]any{"status": "resolved", "outcome": "upheld", "note": "Corrected.", "resolution_report_id": orig}), 400)

	_, at := w.reportDay(1)
	corr := w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/corrections", map[string]any{"reason": "outage was 15 minutes"}, w.reporter), 201)
	corrID := corr.str("id")
	w.expect(w.call("PUT", "/api/v1/daily-reports/"+corrID, map[string]any{
		"row_version": corr.Body["row_version"], "notes": "corrected",
		"intervals": []any{
			iv(at(0, 0), at(11, 30), "served", "grid"),
			iv(at(11, 30), at(11, 45), "outage", ""),
			iv(at(11, 45), at(23, 30), "served", "generator"),
		},
	}, w.reporter), 200)
	w.expect(step(w.supervisor, map[string]any{"status": "resolved", "outcome": "upheld", "note": "Corrected.", "resolution_report_id": corrID}), 400) // not approved yet
	w.expect(w.call("POST", "/api/v1/daily-reports/"+corrID+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+corrID+"/approve", nil, w.supervisor), 200)
	resolved := w.expect(step(w.supervisor, map[string]any{"status": "resolved", "outcome": "upheld", "note": "Corrected.", "resolution_report_id": corrID}), 200)
	if resolved.str("outcome") != "upheld" || resolved.str("acknowledged_at") != "" {
		t.Fatalf("Coloplus's decision is not Airtel's agreement: %s", resolved.Raw)
	}

	// Airtel does not have to accept it: reopening drops the resolution but keeps the history.
	w.expect(step(reviewer, map[string]any{"status": "under_review"}), 400) // say why
	reopened := w.expect(step(reviewer, map[string]any{"status": "under_review", "note": "Still 15 minutes too long."}), 200)
	if reopened.str("outcome") != "" || reopened.str("reopen_count") != "1" {
		t.Fatalf("after reopening: %s", reopened.Raw)
	}
	w.expect(step(w.supervisor, map[string]any{"status": "agreed_correction", "note": "Rechecking the end time."}), 200)
	w.expect(step(w.supervisor, map[string]any{"status": "resolved", "outcome": "upheld", "note": "Rechecked and unchanged.", "resolution_report_id": corrID}), 200)

	// Airtel's agreement is its own act, and retrying it changes nothing.
	acked := w.expect(step(reviewer, map[string]any{"status": "closed"}), 200)
	if acked.str("acknowledged_at") == "" || acked.str("status") != "closed" {
		t.Fatalf("acknowledgement not recorded: %s", acked.Raw)
	}
	w.expect(step(reviewer, map[string]any{"status": "closed"}), 200)
	w.expect(w.call("POST", api+id+"/comments", map[string]any{"body": "late"}, reviewer), 409)
	w.expect(step(w.supervisor, map[string]any{"status": "under_review"}), 409)

	// Airtel never saw the internal note, staff ids, or an email with them.
	seen := w.expect(w.call("GET", api+id, nil, reviewer), 200)
	if strings.Contains(seen.Raw, secret) || strings.Contains(seen.Raw, "owner_id") || strings.Contains(seen.Raw, "author_id") {
		t.Fatalf("Airtel's view leaks Coloplus's side: %s", seen.Raw)
	}
	if staff := w.expect(w.call("GET", api+id, nil, w.supervisor), 200); !strings.Contains(staff.Raw, secret) {
		t.Fatalf("Coloplus lost its internal note: %s", staff.Raw)
	}
	w.dispatch()
	if len(w.mailer.to(w.supEmail)) == 0 || len(w.mailer.to(revEmail)) == 0 {
		t.Fatal("each side should have been emailed about the other's moves")
	}
	for _, m := range w.mailer.to(revEmail) {
		if strings.Contains(m.Text, secret) {
			t.Fatalf("an internal note was emailed to Airtel: %q", m.Text)
		}
	}

	// A closed dispute and its history are frozen in the database itself.
	if _, err := w.pool.Exec(context.Background(), `UPDATE disputes SET title = 'changed' WHERE id = $1`, id); err == nil {
		t.Fatal("a closed dispute could be changed")
	}
	if _, err := w.pool.Exec(context.Background(), `DELETE FROM dispute_events WHERE dispute_id = $1`, id); err == nil {
		t.Fatal("dispute history could be deleted")
	}
}

// Withdrawing needs a reason and is final; it is not an agreement.
func TestDispute_WithdrawingClosesItWithoutAnyAgreement(t *testing.T) {
	w := newWorkflow(t)
	revEmail, _ := w.seedUser(domain.RoleAirtelReviewer, w.site)
	reviewer := w.login(revEmail)

	id := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/disputes", map[string]any{
		"category": "missing_report", "title": "No report for the 3rd", "period_from": "2026-01-03", "period_to": "2026-01-03",
	}, reviewer), 201).str("id")

	step := func(body map[string]any) reply {
		return w.call("POST", "/api/v1/disputes/"+id+"/transition", body, reviewer)
	}
	w.expect(step(map[string]any{"status": "closed"}), 400)
	closed := w.expect(step(map[string]any{"status": "closed", "note": "Found it, our mistake."}), 200)
	if closed.str("outcome") != "withdrawn" || closed.str("acknowledged_at") != "" {
		t.Fatalf("withdrawn dispute = %s", closed.Raw)
	}
}
