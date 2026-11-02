package integration

import (
	"testing"
	"time"
)

// The reporter's "my sites" grid must never fabricate a figure, must show a
// visible gap for a site with no report, and must scope open-incident counts
// to the reporter's own sites, not every incident in the system.
func TestMySites_PooledAvailabilityAndIncidentScope(t *testing.T) {
	w := newWorkflow(t)

	// A second site assigned to the same reporter that never reports.
	silent := w.newSite(w.admin)
	w.expect(w.call("PATCH", "/api/v1/users/"+w.reporterID, map[string]any{"site_ids": []string{w.site, silent}}, w.admin), 200)

	// w.site gets one approved report with an outage, 3 days ago -- inside
	// both the 7-day bar window and the 30-day pooling window.
	id := w.draft(3)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", map[string]any{"late_reason": "entered later"}, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)

	// One open incident on w.site only.
	w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "generator_failure", "severity": "critical", "impact": "outage",
		"title": "Generator failed to start", "description": "fuel line leaking",
		"started_at": ago(3 * time.Hour),
	}, w.reporter), 201)

	res := w.expect(w.call("GET", "/api/v1/dashboard/my-sites", nil, w.reporter), 200)
	rows := res.list("sites")
	if len(rows) != 2 {
		t.Fatalf("expected 2 sites, got %d: %s", len(rows), res.Raw)
	}

	var reported, neverReported map[string]any
	for _, row := range rows {
		r := row.(map[string]any)
		switch r["site_id"] {
		case w.site:
			reported = r
		case silent:
			neverReported = r
		}
	}
	if reported == nil || neverReported == nil {
		t.Fatalf("missing expected sites in response: %s", res.Raw)
	}

	// The silent site: never fabricate availability, every bar is a gap, no
	// incidents leak onto it.
	if neverReported["availability_pct_30d"] != nil {
		t.Fatalf("a never-reported site's 30-day availability must be null, got %v", neverReported["availability_pct_30d"])
	}
	for _, b := range neverReported["bars"].([]any) {
		bar := b.(map[string]any)
		if bar["color"] != "red" {
			t.Fatalf("a never-reported site's bars must all be red, got %v", neverReported["bars"])
		}
	}
	if neverReported["open_incidents"] != float64(0) {
		t.Fatalf("the silent site must show 0 open incidents, got %v", neverReported["open_incidents"])
	}

	// The reporting site: a real availability figure, at least one amber bar
	// (the outage day), and its incident count scoped to itself only -- not
	// every open incident in the system (the AllSites-filter bug this test
	// guards against).
	if reported["availability_pct_30d"] == nil {
		t.Fatalf("the reporting site must have a non-null 30-day availability: %s", res.Raw)
	}
	amber := false
	for _, b := range reported["bars"].([]any) {
		bar := b.(map[string]any)
		if bar["color"] == "amber" {
			amber = true
		}
	}
	if !amber {
		t.Fatalf("the reporting site's bars must show the outage day as amber: %v", reported["bars"])
	}
	if reported["open_incidents"] != float64(1) {
		t.Fatalf("the reporting site must show exactly its own 1 open incident, got %v", reported["open_incidents"])
	}
}
