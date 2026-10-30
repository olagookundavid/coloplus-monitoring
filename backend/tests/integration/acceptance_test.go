package integration

import (
	"context"
	"strings"
	"testing"

	"coloplus-backend/internal/domain"
)

func countSubject(mails []domain.Email, part string) int {
	n := 0
	for _, m := range mails {
		if strings.Contains(m.Subject, part) {
			n++
		}
	}
	return n
}

// Acceptance is Airtel's decision alone: Coloplus builds and submits a
// dossier, Airtel inspects and accepts or rejects, and only that acceptance
// brings a pending site into service. A hold stops the clock, and a missed
// due time is alerted once.
func TestAcceptance_OnlyAirtelAcceptsAndThatBringsTheSiteIntoService(t *testing.T) {
	w := newWorkflow(t)
	ctx := context.Background()

	site := w.expect(w.call("POST", "/api/v1/sites", map[string]any{
		"airtel_code": uniq("AIR"), "coloplus_code": uniq("CP"), "name": "New site", "state": "Lagos",
		"power_sources": []string{"grid"}, "status": "pending_acceptance",
	}, w.admin), 201).str("id")
	supEmail, _ := w.seedUser(domain.RoleSupervisor, site)
	revEmail, _ := w.seedUser(domain.RoleAirtelReviewer, site)
	viewEmail, _ := w.seedUser(domain.RoleAirtelViewer, site)
	repEmail, _ := w.seedUser(domain.RoleReporter, site)
	sup, reviewer, viewer, rep := w.login(supEmail), w.login(revEmail), w.login(viewEmail), w.login(repEmail)

	api := "/api/v1/sites/" + site + "/acceptance"
	step := func(tok string, body map[string]any) reply { return w.call("POST", api+"/transition", body, tok) }
	stage := func() string { return w.expect(w.call("GET", api, nil, reviewer), 200).str("acceptance", "stage") }

	// Only Coloplus managers start it, only for a pending site, only once.
	w.expect(w.call("POST", api, nil, rep), 403)
	w.expect(w.call("POST", api, nil, reviewer), 403)
	w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/acceptance", nil, w.supervisor), 409) // already active
	w.expect(w.call("POST", api, nil, sup), 201)
	w.expect(w.call("POST", api, nil, sup), 409)
	w.expect(w.call("GET", api, nil, w.ot), 404)

	// Coloplus cannot skip ahead, and cannot accept for Airtel at any point.
	w.expect(step(sup, map[string]any{"stage": "accepted"}), 409)
	w.expect(step(sup, map[string]any{"stage": "ready_for_inspection", "note": "dossier"}), 409)
	w.expect(step(sup, map[string]any{"stage": "built"}), 200)
	w.expect(step(sup, map[string]any{"stage": "ready_for_inspection"}), 400) // the dossier is required
	w.expect(step(sup, map[string]any{"stage": "ready_for_inspection", "note": "Built, tested, photos in the folder."}), 200)
	w.expect(step(sup, map[string]any{"stage": "ready_for_inspection", "note": "Built, tested, photos in the folder."}), 200) // a retry changes nothing
	w.expect(step(sup, map[string]any{"stage": "inspected"}), 409)
	w.expect(step(sup, map[string]any{"stage": "accepted"}), 409)
	w.expect(step(viewer, map[string]any{"stage": "inspected"}), 403)
	if r := w.expect(w.call("GET", api, nil, viewer), 200); r.str("acceptance", "waiting_on") != "airtel" {
		t.Fatalf("after the dossier it should wait on Airtel: %s", r.Raw)
	}

	// A missed due time is alerted once, to whoever the step waits on and to Coloplus.
	w.exec(`UPDATE acceptances SET due_at = now() - interval '1 hour' WHERE site_id = $1`, site)
	for i := 0; i < 2; i++ {
		if _, err := w.acceptance.AlertOverdue(ctx); err != nil {
			t.Fatal(err)
		}
		w.dispatch()
	}
	for _, addr := range []string{revEmail, supEmail} {
		if n := countSubject(w.mailer.to(addr), "Acceptance action overdue"); n != 1 {
			t.Fatalf("%s got %d overdue alerts, want 1", addr, n)
		}
	}

	// Airtel must inspect before accepting, and must say why it rejects.
	w.expect(step(reviewer, map[string]any{"stage": "accepted"}), 409)
	w.expect(step(reviewer, map[string]any{"stage": "rejected"}), 400)
	w.expect(step(reviewer, map[string]any{"stage": "rejected", "note": "Battery bank not installed."}), 200)
	if st := w.expect(w.call("GET", "/api/v1/sites/"+site, nil, w.admin), 200).str("status"); st != "pending_acceptance" {
		t.Fatalf("a rejected site must stay pending, got %s", st)
	}
	w.expect(step(sup, map[string]any{"stage": "built"}), 400) // say what was fixed
	w.expect(step(sup, map[string]any{"stage": "built", "note": "Batteries installed."}), 200)
	w.expect(step(sup, map[string]any{"stage": "ready_for_inspection", "note": "Resubmitted with battery test."}), 200)
	w.expect(step(reviewer, map[string]any{"stage": "inspected", "note": "Walked the site."}), 200)

	// A hold needs a reason, stops everything until resumed, and resumes to where it was.
	w.expect(step(sup, map[string]any{"stage": "on_hold"}), 400)
	w.expect(step(sup, map[string]any{"stage": "on_hold", "note": "Waiting for Airtel's access card."}), 200)
	w.expect(step(reviewer, map[string]any{"stage": "accepted"}), 409)
	w.expect(step(reviewer, map[string]any{"stage": "built"}), 409)
	w.expect(step(reviewer, map[string]any{"stage": "inspected"}), 200)
	if st := stage(); st != "inspected" {
		t.Fatalf("stage after resuming = %s", st)
	}

	// Only now, by Airtel, is the site accepted, and only now does it enter service.
	w.expect(step(sup, map[string]any{"stage": "accepted"}), 409)
	accepted := w.expect(step(reviewer, map[string]any{"stage": "accepted", "note": "Accepted."}), 200)
	if accepted.str("decided_at") == "" {
		t.Fatalf("acceptance not recorded: %s", accepted.Raw)
	}
	got := w.expect(w.call("GET", "/api/v1/sites/"+site, nil, w.admin), 200)
	if got.str("status") != "active" || got.str("commissioning_date") == "" {
		t.Fatalf("accepted site = %s", got.Raw)
	}
	w.expect(step(sup, map[string]any{"stage": "on_hold", "note": "x"}), 409)
	if _, err := w.pool.Exec(ctx, `UPDATE acceptances SET next_action = 'x' WHERE site_id = $1`, site); err == nil {
		t.Fatal("an accepted site's record could be changed")
	}

	w.dispatch()
	if len(w.mailer.to(revEmail)) == 0 || len(w.mailer.to(supEmail)) == 0 {
		t.Fatal("each side should be told when the other acts")
	}
	if n := w.countAudit("site.activated", site); n != 1 {
		t.Fatalf("site.activated audit rows = %d, want 1", n)
	}
}
