package integration

import (
	"context"
	"strings"
	"testing"

	"coloplus-backend/internal/domain"
)

// The weekly digest goes to each person once a week, shows only what their own
// screen shows, and stops when they turn it off.
func TestDigest_OncePerWeekScopedAndOptOut(t *testing.T) {
	w := newWorkflow(t)
	// The job writes for every user in the shared test database; clear those
	// rows so they cannot crowd out other tests' mail.
	clear := func() { w.exec(`DELETE FROM notifications WHERE kind = 'digest.weekly'`) }
	clear()
	t.Cleanup(clear)
	w.exec(`UPDATE sites SET consumption_lph = 1, tank_capacity_litres = 2000 WHERE id = $1`, w.site)
	offEmail, _ := w.seedUser(domain.RoleSupervisor, w.site)
	off := w.login(offEmail)
	w.expect(w.call("PUT", "/api/v1/digest/preference", map[string]any{"weekly": false}, off), 200)
	if got := w.expect(w.call("GET", "/api/v1/digest/preference", nil, off), 200); got.str("weekly") != "false" {
		t.Fatalf("preference = %s", got.Raw)
	}

	mails := func(addr string) []string {
		t.Helper()
		rows, err := w.pool.Query(context.Background(), `SELECT subject || E'\n' || body FROM notifications WHERE recipient = $1 AND kind = 'digest.weekly'`, addr)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var m string
			if err := rows.Scan(&m); err != nil {
				t.Fatal(err)
			}
			out = append(out, m)
		}
		return out
	}

	for i := 0; i < 2; i++ { // the second run must add nothing
		if _, err := w.digest.SendWeekly(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(mails(w.supEmail)); n != 1 {
		t.Fatalf("supervisor got %d digests, want exactly 1", n)
	}
	if n := len(mails(offEmail)); n != 0 {
		t.Fatalf("someone who turned it off still got %d", n)
	}
	if n := len(mails("nobody@coloplus.example")); n != 0 {
		t.Fatal("no address should have received this")
	}

	staff, customer := mails(w.supEmail)[0], mails(w.airEmail)
	if len(customer) != 1 {
		t.Fatalf("customer got %d digests, want 1", len(customer))
	}
	// A site with no reports is assumed, never confirmed; the customer is
	// never told about fuel or internal incident figures.
	if !strings.Contains(staff, "Confirmed by reports") || !strings.Contains(staff, "assumption") {
		t.Fatalf("figures must say what is confirmed and what is assumed:\n%s", staff)
	}
	if strings.Contains(customer[0], "critical") || strings.Contains(customer[0], "Fuel") {
		t.Fatalf("the customer's digest shows staff-only information:\n%s", customer[0])
	}
}
