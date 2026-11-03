package integration

import (
	"context"
	"testing"

	"coloplus-backend/internal/domain"
)

// Every acceptance step gets its own deadline, an acceptance that stays late for
// long is also sent to the administrators (once), and a dispute left waiting on
// Coloplus past the response time is alerted once per wait.
func TestEscalation_AcceptanceDeadlinesAndOverdueDisputes(t *testing.T) {
	w := newWorkflow(t)
	ctx := context.Background()
	// The jobs look at every site in the shared test database; clear their mail
	// so it cannot crowd out other tests'.
	clear := func() {
		w.exec(`DELETE FROM notifications WHERE kind IN ('acceptance.escalated', 'dispute.overdue', 'acceptance.overdue')`)
		w.exec(`DELETE FROM alerts WHERE kind IN ('acceptance.escalated', 'dispute.overdue', 'acceptance.overdue')`)
	}
	clear()
	t.Cleanup(clear)
	mails := func(addr, kind string) int {
		t.Helper()
		var n int
		if err := w.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE recipient = $1 AND kind = $2`, addr, kind).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// A step with no date set is given the default for its stage.
	site := w.expect(w.call("POST", "/api/v1/sites", map[string]any{
		"airtel_code": uniq("AIR"), "coloplus_code": uniq("CP"), "name": "New site", "state": "Lagos",
		"power_sources": []string{"grid"}, "status": "pending_acceptance",
	}, w.admin), 201).str("id")
	api := "/api/v1/sites/" + site + "/acceptance"
	acceptSupEmail, _ := w.seedUser(domain.RoleSupervisor, site)
	acceptSup := w.login(acceptSupEmail)
	started := w.expect(w.call("POST", api, nil, acceptSup), 201)
	if started.str("due_at") == "" {
		t.Fatalf("a new acceptance should be given a deadline: %s", started.Raw)
	}
	built := w.expect(w.call("POST", api+"/transition", map[string]any{"stage": "built"}, acceptSup), 200)
	if built.str("due_at") == "" || built.str("due_at") == started.str("due_at") {
		t.Fatalf("each step has its own deadline: %s", built.Raw)
	}

	// Late by a day: the people responsible are told, the administrators are not yet.
	w.exec(`UPDATE acceptances SET due_at = now() - interval '1 day' WHERE site_id = $1`, site)
	for i := 0; i < 2; i++ {
		if _, err := w.acceptance.AlertOverdue(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if n := mails(w.adminEmail, "acceptance.escalated"); n != 0 {
		t.Fatalf("administrators were escalated to after one day: %d", n)
	}
	// Late by eight days: escalated once, however often it runs.
	w.exec(`UPDATE acceptances SET due_at = now() - interval '8 days' WHERE site_id = $1`, site)
	for i := 0; i < 2; i++ {
		if _, err := w.acceptance.AlertOverdue(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if n := mails(w.adminEmail, "acceptance.escalated"); n != 1 {
		t.Fatalf("administrator got %d escalations, want 1", n)
	}

	// A dispute waiting on Coloplus for four days is alerted once; a fresh one is not.
	revEmail, _ := w.seedUser(domain.RoleAirtelReviewer, w.site)
	reviewer := w.login(revEmail)
	id := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/disputes", map[string]any{
		"category": "missing_report", "title": "No report for the 3rd", "period_from": "2026-01-03", "period_to": "2026-01-03",
	}, reviewer), 201).str("id")
	run := func() {
		t.Helper()
		if _, err := w.disputes.AlertOverdue(ctx); err != nil {
			t.Fatal(err)
		}
	}
	run()
	if n := mails(w.supEmail, "dispute.overdue"); n != 0 {
		t.Fatalf("a new dispute is not overdue: %d", n)
	}
	w.exec(`UPDATE disputes SET updated_at = now() - interval '4 days' WHERE id = $1`, id)
	run()
	run()
	if n := mails(w.supEmail, "dispute.overdue"); n != 1 {
		t.Fatalf("supervisor got %d overdue dispute alerts, want 1", n)
	}
	if n := mails(revEmail, "dispute.overdue"); n != 0 {
		t.Fatal("the customer is not sent Coloplus's overdue alert")
	}
}
