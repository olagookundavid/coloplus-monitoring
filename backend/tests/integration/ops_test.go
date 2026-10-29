package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/services"
)

// A late report is stale data, never an outage: the people who owe it are told
// once a day, Airtel is not emailed, and the wording says power is not known
// to be out.
func TestReportAlerts_OverdueGoesToTheRightPeopleOnceAndNeverSaysOutage(t *testing.T) {
	w := newWorkflow(t)
	ctx := context.Background()

	// A brand-new site is not nagged before its first report is due.
	if _, err := w.reportAlerts.AlertOverdue(ctx); err != nil {
		t.Fatal(err)
	}
	w.dispatch()
	if got := len(w.mailer.to(w.repEmail)); got != 0 {
		t.Fatalf("a site created seconds ago was alerted (%d emails)", got)
	}

	w.exec(`UPDATE sites SET created_at = now() - interval '10 days' WHERE id = $1`, w.site)
	for i := 0; i < 2; i++ { // the second run must add nothing
		if _, err := w.reportAlerts.AlertOverdue(ctx); err != nil {
			t.Fatal(err)
		}
		w.dispatch()
	}
	for _, addr := range []string{w.repEmail, w.supEmail} {
		mails := w.mailer.to(addr)
		if len(mails) != 1 {
			t.Fatalf("%s got %d overdue emails, want 1", addr, len(mails))
		}
		if !strings.Contains(mails[0].Text, "Awaiting update") || !strings.Contains(mails[0].Text, "does not mean the power is out") {
			t.Fatalf("overdue wording: %q", mails[0].Text)
		}
	}
	if got := len(w.mailer.to(w.airEmail)); got != 0 {
		t.Fatalf("Airtel was emailed %d times about Coloplus's missing report", got)
	}
}

// Reading the audit log needs audit access, and a reader scoped to some sites
// sees only events about those sites.
func TestAudit_SearchNeedsAccessAndRespectsSiteScope(t *testing.T) {
	w := newWorkflow(t)
	inc := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "low_fuel", "severity": "minor", "impact": "no_outage", "title": "Fuel low",
		"started_at": ago(time.Hour),
	}, w.supervisor), 201).str("id")

	w.expect(w.call("GET", "/api/v1/audit", nil, w.supervisor), 403)
	w.expect(w.call("GET", "/api/v1/audit", nil, w.airtel), 403)
	found := w.expect(w.call("GET", "/api/v1/audit?entity_id="+inc+"&action=incident.reported", nil, w.admin), 200)
	if found.str("total") != "1" {
		t.Fatalf("admin search = %s", found.Raw)
	}
	w.expect(w.call("GET", "/api/v1/audit?from=yesterday", nil, w.admin), 400)

	reader := func(sites ...string) domain.Actor {
		ids := map[string]struct{}{}
		for _, id := range sites {
			ids[id] = struct{}{}
		}
		return domain.Actor{UserID: "reader", Permissions: map[domain.Permission]struct{}{domain.PermAuditRead: {}}, SiteIDs: ids}
	}
	in := servicesAuditInput(inc)
	if _, total, err := w.audit.Search(context.Background(), reader(w.site), in); err != nil || total != 1 {
		t.Fatalf("reader of the site: total=%d err=%v", total, err)
	}
	if _, total, err := w.audit.Search(context.Background(), reader(w.newSite(w.admin)), in); err != nil || total != 0 {
		t.Fatalf("reader of another site saw %d events, err=%v", total, err)
	}
}

func servicesAuditInput(entityID string) services.SearchAuditInput {
	return services.SearchAuditInput{EntityID: entityID}
}
