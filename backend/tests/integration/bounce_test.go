package integration

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A bounce reported by the email provider marks that email as not delivered
// and shows it to the administrator, and only a genuine, fresh webhook can do
// that: nobody can mark real emails undelivered, or replay an old report.
func TestBounceWebhook_MarksTheEmailAndRefusesForgeries(t *testing.T) {
	w := newWorkflow(t)
	w.expect(w.call("POST", "/api/v1/sites/"+w.site+"/tickets", map[string]any{
		"category": "other", "priority": "low", "subject": "Bounce test", "description": "x",
	}, w.airtel), 201)
	w.dispatch()

	var notifID, providerID string
	if err := w.pool.QueryRow(context.Background(),
		`SELECT id::text, provider_id FROM notifications WHERE recipient = $1 AND status = 'sent' ORDER BY created_at DESC LIMIT 1`,
		w.supEmail).Scan(&notifID, &providerID); err != nil || providerID == "" {
		t.Fatalf("the supervisor's email should have been sent: %v", err)
	}

	secret, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(webhookSecret, "whsec_"))
	post := func(body []byte, id string, at time.Time, sign bool) int {
		ts := strconv.FormatInt(at.Unix(), 10)
		mac := hmac.New(sha256.New, secret)
		mac.Write([]byte(id + "." + ts + "."))
		mac.Write(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/resend", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("svix-id", id)
		req.Header.Set("svix-timestamp", ts)
		if sign {
			req.Header.Set("svix-signature", "v1,"+base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		} else {
			req.Header.Set("svix-signature", "v1,AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
		}
		rec := httptest.NewRecorder()
		w.handler.ServeHTTP(rec, req)
		return rec.Code
	}
	event, _ := json.Marshal(map[string]any{
		"type": "email.bounced",
		"data": map[string]any{"email_id": providerID, "bounce": map[string]any{"type": "Permanent", "message": "mailbox does not exist"}},
	})

	status := func() (string, string) {
		var st, why string
		if err := w.pool.QueryRow(context.Background(), `SELECT status, last_error FROM notifications WHERE id = $1::uuid`, notifID).Scan(&st, &why); err != nil {
			t.Fatal(err)
		}
		return st, why
	}

	now := time.Now()
	if code := post(event, "msg_forged", now, false); code != 401 {
		t.Fatalf("a forged webhook must be refused, got %d", code)
	}
	if code := post(event, "msg_old", now.Add(-time.Hour), true); code != 401 {
		t.Fatalf("a replayed old webhook must be refused, got %d", code)
	}
	if st, _ := status(); st != "sent" {
		t.Fatalf("a refused webhook changed the email: %s", st)
	}

	if code := post(event, "msg_ok", now, true); code != 204 {
		t.Fatalf("a genuine bounce should be accepted, got %d", code)
	}
	if st, why := status(); st != "failed" || !strings.Contains(why, "mailbox does not exist") {
		t.Fatalf("the bounced email must be marked not delivered with the reason: %s / %s", st, why)
	}
	// Repeating it, or an event we do not act on, or one for an unknown email, is harmless.
	other, _ := json.Marshal(map[string]any{"type": "email.delivered", "data": map[string]any{"email_id": providerID}})
	unknown, _ := json.Marshal(map[string]any{"type": "email.bounced", "data": map[string]any{"email_id": "nope"}})
	for i, body := range [][]byte{event, other, unknown} {
		if code := post(body, fmt.Sprintf("msg_more_%d", i), now, true); code != 204 {
			t.Fatalf("event %d should be accepted and ignored, got %d", i, code)
		}
	}
	if n := w.countAudit("notification.bounced", notifID); n != 1 {
		t.Fatalf("the bounce was audited %d times, want once", n)
	}
}
