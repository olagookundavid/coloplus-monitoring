package integration

import (
	"context"
	"testing"
)

// A stored run freezes a month. A later correction must not change it; it shows
// up when the run is compared with now. Only Coloplus staff who see every site
// can store or read runs, and the database refuses any edit.
func TestCalculationRun_FrozenCorrectionShowsInComparisonAndOnlyStaffSeeRuns(t *testing.T) {
	w := newWorkflow(t)
	// A region of its own keeps the run to this test's site.
	region := uniq("RunRegion")
	w.exec(`UPDATE sites SET region = $2 WHERE id = $1`, w.site, region)

	orig := w.draft(1)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/approve", nil, w.supervisor), 200)

	date, at := w.reportDay(1)
	month := date[:7]

	// The customer, field staff and a supervisor scoped to one site are refused.
	body := map[string]any{"month": month, "region": region}
	for _, tok := range []string{w.customer, w.reporter, w.supervisor} {
		w.expect(w.call("POST", "/api/v1/calculation-runs", body, tok), 403)
	}

	body["label"] = "close"
	run := w.expect(w.call("POST", "/api/v1/calculation-runs", body, w.admin), 201)
	id := run.str("id")
	frozen := w.expect(w.call("GET", "/api/v1/calculation-runs/"+id, nil, w.admin), 200)
	rows := frozen.list("rows")
	if len(rows) != 1 {
		t.Fatalf("run has %d rows, want the one site: %s", len(rows), frozen.Raw)
	}
	before := rows[0].(map[string]any)
	if before["outage_minutes"] != 30.0 || before["reports_fingerprint"] == "" || frozen.str("formula_version") == "" {
		t.Fatalf("frozen row = %v", before)
	}
	if cmp := w.expect(w.call("GET", "/api/v1/calculation-runs/"+id+"/compare", nil, w.admin), 200); len(cmp.list("changes")) != 0 {
		t.Fatalf("nothing has changed yet: %s", cmp.Raw)
	}

	// The outage is corrected from 30 to 45 minutes and approved.
	corr := w.expect(w.call("POST", "/api/v1/daily-reports/"+orig+"/corrections", map[string]any{"reason": "outage was 45 minutes"}, w.reporter), 201)
	corrID := corr.str("id")
	w.expect(w.call("PUT", "/api/v1/daily-reports/"+corrID, map[string]any{
		"row_version": corr.Body["row_version"],
		"intervals": []any{
			iv(at(0, 0), at(11, 30), "served", "grid"),
			iv(at(11, 30), at(12, 15), "outage", ""),
			iv(at(12, 15), at(23, 30), "served", "generator"),
		},
		"outage_reason": "Grid fault",
	}, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+corrID+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+corrID+"/approve", nil, w.supervisor), 200)

	// The run still says 30 minutes ...
	again := w.expect(w.call("GET", "/api/v1/calculation-runs/"+id, nil, w.admin), 200).list("rows")[0].(map[string]any)
	if again["outage_minutes"] != 30.0 {
		t.Fatalf("a stored run changed: %v", again)
	}
	// ... and the comparison shows the correction, with the reports flagged as changed.
	cmp := w.expect(w.call("GET", "/api/v1/calculation-runs/"+id+"/compare", nil, w.admin), 200)
	changes := cmp.list("changes")
	if len(changes) != 1 {
		t.Fatalf("comparison = %s", cmp.Raw)
	}
	ch := changes[0].(map[string]any)
	then, now := ch["then"].(map[string]any), ch["now"].(map[string]any)
	if then["outage_minutes"] != 30.0 || now["outage_minutes"] != 45.0 || ch["reports_changed"] != true {
		t.Fatalf("change = %v", ch)
	}

	// The database refuses to edit or remove a run.
	if _, err := w.pool.Exec(context.Background(), `UPDATE calculation_run_rows SET outage_minutes = 0 WHERE run_id = $1`, id); err == nil {
		t.Fatal("a stored run row must not be editable")
	}
	if _, err := w.pool.Exec(context.Background(), `DELETE FROM calculation_runs WHERE id = $1`, id); err == nil {
		t.Fatal("a stored run must not be removable")
	}
}
