package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"coloplus-backend/internal/domain"
)

func sha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Evidence must stay inside the thing it belongs to: the store must hold the
// file that was described, a Coloplus-only file never reaches the customer, an
// unpublished incident's files are invisible to the customer, nobody outside the
// site can reach any of it, and every download link is accounted for.
func TestAttachments_OnlyTheRightPeopleSeeTheRightFiles(t *testing.T) {
	w := newWorkflow(t)
	elseEmail, _ := w.seedUser(domain.RoleReviewer, w.newSite(w.admin))
	elsewhere := w.login(elseEmail)

	ticket := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/tickets", map[string]any{
		"category": "equipment_fault", "priority": "high", "subject": "Panel damaged", "description": "See photo",
	}, w.customer), 201).str("id")

	body := func(owner, id, name, typ string, size int64, digest string, shared bool) map[string]any {
		return map[string]any{
			"owner_type": owner, "owner_id": id, "filename": name, "content_type": typ,
			"size_bytes": size, "sha256": digest, "shared": shared,
		}
	}
	// send runs the three steps; the browser's PUT is the fake store's put.
	send := func(tok string, req map[string]any) (id string) {
		t.Helper()
		r := w.expect(w.call("POST", "/api/v1/attachments", req, tok), 201)
		att := r.Body["attachment"].(map[string]any)
		id = att["id"].(string)
		key := strings.TrimPrefix(r.Body["upload"].(map[string]any)["url"].(string), "https://store.test/up/")
		if strings.Contains(key, "panel") || strings.Contains(key, ".jpg") {
			t.Fatalf("the file's name is in its storage key: %s", key)
		}
		w.store.put(key, int64(req["size_bytes"].(int64)), req["sha256"].(string))
		w.expect(w.call("POST", "/api/v1/attachments/"+id+"/complete", nil, tok), 200)
		return id
	}

	// What can be sent at all.
	const photo = "photo-bytes"
	good := body("ticket", ticket, "panel.jpg", "image/jpeg", int64(len(photo)), sha(photo), false)
	w.expect(w.call("POST", "/api/v1/attachments", body("ticket", ticket, "run.html", "text/html", 10, sha("x"), false), w.customer), 400)
	w.expect(w.call("POST", "/api/v1/attachments", body("ticket", ticket, "panel.png", "image/jpeg", 10, sha("x"), false), w.customer), 400)
	// A script is refused whatever it claims to be: by its type, and by its name.
	for _, s := range [][2]string{
		{"run.py", "text/x-python"}, {"run.sh", "application/x-sh"}, {"run.exe", "application/octet-stream"},
		{"run.py", "text/plain"}, {"run.sh", "application/pdf"}, {"run.js", "image/png"},
	} {
		w.expect(w.call("POST", "/api/v1/attachments", body("ticket", ticket, s[0], s[1], 10, sha("x"), false), w.customer), 400)
	}
	w.expect(w.call("POST", "/api/v1/attachments", body("ticket", ticket, "big.pdf", "application/pdf", 11<<20, sha("x"), false), w.customer), 400)
	w.expect(w.call("POST", "/api/v1/attachments", good, elsewhere), 404) // another site's ticket
	w.expect(w.call("POST", "/api/v1/attachments", good, w.ot), 403)      // a reporter has no ticket rights

	// The store must hold exactly what was described.
	r := w.expect(w.call("POST", "/api/v1/attachments", good, w.customer), 201)
	pending := r.Body["attachment"].(map[string]any)["id"].(string)
	pendingKey := strings.TrimPrefix(r.Body["upload"].(map[string]any)["url"].(string), "https://store.test/up/")
	w.expect(w.call("POST", "/api/v1/attachments/"+pending+"/complete", nil, w.customer), 400) // nothing uploaded yet
	w.store.put(pendingKey, 3, sha(photo))                                                     // the wrong size arrived
	w.expect(w.call("POST", "/api/v1/attachments/"+pending+"/complete", nil, w.customer), 400) // so it is discarded
	if len(w.store.deleted) == 0 || w.store.deleted[len(w.store.deleted)-1] != pendingKey {
		t.Fatalf("a mismatched upload was kept in storage")
	}
	w.expect(w.call("POST", "/api/v1/attachments/"+pending+"/complete", nil, w.customer), 409) // and cannot be revived
	if l := w.expect(w.call("GET", "/api/v1/attachments?owner_type=ticket&owner_id="+ticket, nil, w.customer), 200); len(l.list("attachments")) != 0 {
		t.Fatalf("an unconfirmed or rejected file is listed: %s", l.Raw)
	}

	mine := send(w.customer, good)
	other := body("ticket", ticket, "note.txt", "text/plain", 5, sha("hello"), false)
	r = w.expect(w.call("POST", "/api/v1/attachments", other, w.customer), 201)
	w.expect(w.call("POST", "/api/v1/attachments/"+r.Body["attachment"].(map[string]any)["id"].(string)+"/complete", nil, w.supervisor), 404) // only the sender confirms

	// Coloplus sends one file kept internal and one shared.
	internal := send(w.supervisor, body("ticket", ticket, "driver-notes.txt", "text/plain", 5, sha("notes"), false))
	shared := send(w.supervisor, body("ticket", ticket, "refill.pdf", "application/pdf", 6, sha("refill"), true))

	ids := func(tok string) map[string]bool {
		out := map[string]bool{}
		for _, a := range w.expect(w.call("GET", "/api/v1/attachments?owner_type=ticket&owner_id="+ticket, nil, tok), 200).list("attachments") {
			out[a.(map[string]any)["id"].(string)] = true
		}
		return out
	}
	if got := ids(w.customer); !got[mine] || !got[shared] || got[internal] || len(got) != 2 {
		t.Fatalf("The customer should see its own file and the shared one only: %v", got)
	}
	if got := ids(w.supervisor); !got[mine] || !got[shared] || !got[internal] {
		t.Fatalf("Coloplus should see all three: %v", got)
	}
	w.expect(w.call("GET", "/api/v1/attachments?owner_type=ticket&owner_id="+ticket, nil, elsewhere), 404)

	w.expect(w.call("GET", "/api/v1/attachments/"+internal+"/download", nil, w.customer), 404)
	w.expect(w.call("GET", "/api/v1/attachments/"+mine+"/download", nil, elsewhere), 404)
	dl := w.expect(w.call("GET", "/api/v1/attachments/"+internal+"/download", nil, w.supervisor), 200)
	if !strings.HasPrefix(dl.str("url"), "https://store.test/dl/attachments/") || dl.str("filename") != "driver-notes.txt" {
		t.Fatalf("download = %s", dl.Raw)
	}
	w.expect(w.call("GET", "/api/v1/attachments/"+shared+"/download", nil, w.customer), 200)
	if n := w.countAudit("attachment.downloaded", shared); n != 1 {
		t.Fatalf("a download link was issued without an audit record: %d", n)
	}
	if n := w.countAudit("attachment.added", mine); n != 1 {
		t.Fatalf("an upload was not audited: %d", n)
	}

	// An incident's files are invisible to the customer until it is published, and
	// The customer never attaches to Coloplus's fault record.
	inc := w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/incidents", map[string]any{
		"category": "low_fuel", "severity": "minor", "impact": "no_outage", "title": "Fuel low", "started_at": ago(time.Hour),
	}, w.supervisor), 201).str("id")
	send(w.supervisor, body("incident", inc, "tank.jpg", "image/jpeg", 4, sha("tank"), true))
	send(w.supervisor, body("incident", inc, "internal.txt", "text/plain", 4, sha("priv"), false))
	w.expect(w.call("GET", "/api/v1/attachments?owner_type=incident&owner_id="+inc, nil, w.customer), 404)
	w.expect(w.call("POST", "/api/v1/incidents/"+inc+"/publish", nil, w.supervisor), 200)
	if l := w.expect(w.call("GET", "/api/v1/attachments?owner_type=incident&owner_id="+inc, nil, w.customer), 200); len(l.list("attachments")) != 1 {
		t.Fatalf("The customer should see only the shared incident file: %s", l.Raw)
	}
	w.expect(w.call("POST", "/api/v1/attachments", body("incident", inc, "x.jpg", "image/jpeg", 4, sha("zzzz"), true), w.customer), 403)
}

// Evidence on a daily report is Coloplus's own whatever the sender asks for, the
// customer can neither add nor see it, and an approved report takes no more.
func TestAttachments_ReportEvidenceIsInternalAndFrozenOnApproval(t *testing.T) {
	w := newWorkflow(t)
	id := w.draft(1)
	const photo = "meter-photo"
	req := map[string]any{
		"owner_type": "report", "owner_id": id, "filename": "meter.jpg", "content_type": "image/jpeg",
		"size_bytes": int64(len(photo)), "sha256": sha(photo), "shared": true, // asking to share is ignored
	}
	w.expect(w.call("POST", "/api/v1/attachments", req, w.customer), 403)
	w.expect(w.call("POST", "/api/v1/attachments", req, w.ot), 404) // a reporter on no site cannot find it

	r := w.expect(w.call("POST", "/api/v1/attachments", req, w.reporter), 201)
	att := r.Body["attachment"].(map[string]any)
	if att["internal"] != true {
		t.Fatalf("report evidence must stay internal: %v", att)
	}
	key := strings.TrimPrefix(r.Body["upload"].(map[string]any)["url"].(string), "https://store.test/up/")
	w.store.put(key, int64(len(photo)), sha(photo))
	w.expect(w.call("POST", "/api/v1/attachments/"+att["id"].(string)+"/complete", nil, w.reporter), 200)

	list := "/api/v1/attachments?owner_type=report&owner_id=" + id
	if got := w.expect(w.call("GET", list, nil, w.supervisor), 200); len(got.list("attachments")) != 1 {
		t.Fatalf("staff should see the file: %s", got.Raw)
	}
	w.expect(w.call("GET", list, nil, w.customer), 403)

	// Once approved the report is frozen: no more evidence.
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/submit", nil, w.reporter), 200)
	w.expect(w.call("POST", "/api/v1/daily-reports/"+id+"/approve", nil, w.supervisor), 200)
	w.expect(w.call("POST", "/api/v1/attachments", req, w.reporter), 409)
}
