package integration

import (
	"context"
	"testing"
	"time"
)

// A closed month takes no new report, correction or approval until an
// administrator reopens it and says why; only administrators can do either,
// and a month cannot be closed before the grace period has passed.
func TestPeriods_ClosedMonthRefusesReportsUntilReopened(t *testing.T) {
	w := newWorkflow(t)
	now := time.Now().In(w.loc)
	old := time.Date(now.Year(), now.Month()-3, 1, 0, 0, 0, 0, w.loc)
	month := old.Format("2006-01")
	day := old.AddDate(0, 0, 9)
	// Leave the shared test database as found, even if an assertion fails.
	t.Cleanup(func() {
		_, _ = w.pool.Exec(context.Background(), `UPDATE period_closures SET reopened_by = closed_by, reopened_at = now(), reopen_reason = 'test cleanup' WHERE month = $1 AND reopened_at IS NULL`, old.Format("2006-01-02"))
	})
	_, _ = w.pool.Exec(context.Background(), `UPDATE period_closures SET reopened_by = closed_by, reopened_at = now(), reopen_reason = 'test cleanup' WHERE month = $1 AND reopened_at IS NULL`, old.Format("2006-01-02"))

	draft := func(want int) {
		t.Helper()
		w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/daily-reports", map[string]any{
			"report_date": day.Format("2006-01-02"), "power_status": "served",
		}, w.reporter), want)
	}

	// Not before the grace period, and not by anyone but an administrator.
	current := now.Format("2006-01")
	w.expect(w.call("POST", "/api/v1/periods/"+current+"/close", nil, w.admin), 400)
	w.expect(w.call("POST", "/api/v1/periods/"+month+"/close", nil, w.supervisor), 403)
	w.expect(w.call("POST", "/api/v1/periods/"+month+"/close", nil, w.customer), 403)

	w.expect(w.call("POST", "/api/v1/periods/"+month+"/close", nil, w.admin), 201)
	w.expect(w.call("POST", "/api/v1/periods/"+month+"/close", nil, w.admin), 409)
	draft(409)

	// A reason is required to reopen.
	w.expect(w.call("POST", "/api/v1/periods/"+month+"/reopen", map[string]any{"reason": "x"}, w.admin), 400)
	w.expect(w.call("POST", "/api/v1/periods/"+month+"/reopen", map[string]any{"reason": "a late report turned up"}, w.supervisor), 403)
	re := w.expect(w.call("POST", "/api/v1/periods/"+month+"/reopen", map[string]any{"reason": "a late report turned up"}, w.admin), 200)
	if re.str("locked") != "false" || re.str("reopen_reason") != "a late report turned up" {
		t.Fatalf("reopened = %s", re.Raw)
	}
	draft(201)
	// Reopening a month that is not closed is not found.
	w.expect(w.call("POST", "/api/v1/periods/"+month+"/reopen", map[string]any{"reason": "again for no reason"}, w.admin), 404)
}
