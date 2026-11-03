package integration

import (
	"context"
	"strings"
	"testing"
)

// A report whose numbers do not fit the site is flagged for a look, once, to the
// people who review it; nothing about the figures changes and a fitting report
// is left alone.
func TestAnomalies_FlaggedOnceAndOnlyWhenTheNumbersDoNotFit(t *testing.T) {
	w := newWorkflow(t)
	// The job looks at every site in the shared test database; clear its mail so
	// it cannot crowd out other tests'.
	clear := func() {
		w.exec(`DELETE FROM notifications WHERE kind = 'report.anomaly'`)
		w.exec(`DELETE FROM alerts WHERE kind = 'report.anomaly'`)
	}
	clear()
	t.Cleanup(clear)

	// goodDay has a 30-minute outage and generator time; no incident was opened.
	id := w.draft(1)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)

	mails := func(addr string) (out []string) {
		t.Helper()
		rows, err := w.pool.Query(context.Background(), `SELECT body FROM notifications WHERE recipient = $1 AND kind = 'report.anomaly'`, addr)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var b string
			if err := rows.Scan(&b); err != nil {
				t.Fatal(err)
			}
			out = append(out, b)
		}
		return out
	}
	run := func() {
		t.Helper()
		if _, err := w.anomalies.Check(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	run()
	run() // the second run adds nothing
	got := mails(w.supEmail)
	if len(got) != 1 || !strings.Contains(got[0], "no incident was opened") {
		t.Fatalf("an outage with no incident should be flagged once: %v", got)
	}
	if len(mails(w.repEmail)) != 0 || len(mails(w.airEmail)) != 0 {
		t.Fatal("only the people who review reports are told, never the customer")
	}

	// With an incident opened for the day, and then no generator at the site,
	// only the generator flag is raised.
	clear()
	_, at := w.reportDay(1)
	w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "grid_loss", "severity": "major", "impact": "outage", "title": "Grid outage",
		"started_at": at(11, 30),
	}, w.supervisor), 201)
	w.exec(`UPDATE sites SET power_sources = '{grid}' WHERE id = $1`, w.site)
	run()
	got = mails(w.supEmail)
	if len(got) != 1 || !strings.Contains(got[0], "no generator installed") || strings.Contains(got[0], "no incident") {
		t.Fatalf("expected only the generator flag: %v", got)
	}
}
