package mail

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// webhookTolerance is how far a webhook's timestamp may be from now. It stops
// an old, captured request from being played again.
const webhookTolerance = 5 * time.Minute

// ResendWebhook checks and reads the events Resend posts (delivered, bounced,
// complained). Resend signs them the Svix way: HMAC-SHA256 over
// "<id>.<timestamp>.<body>" with the secret from the webhook's settings.
type ResendWebhook struct{ secret []byte }

// NewResendWebhook takes the signing secret as Resend shows it ("whsec_...").
func NewResendWebhook(secret string) (*ResendWebhook, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(strings.TrimSpace(secret), "whsec_"))
	if err != nil || len(raw) == 0 {
		return nil, errors.New("resend webhook: the signing secret is not valid")
	}
	return &ResendWebhook{secret: raw}, nil
}

// Verify checks the signature and the timestamp, then reads the event. A bad
// signature is ErrUnauthorized and nothing in the body is trusted. Header names
// are lower case.
func (w *ResendWebhook) Verify(headers map[string]string, body []byte, now time.Time) (ports.MailEvent, error) {
	id, ts, sigs := headers["svix-id"], headers["svix-timestamp"], headers["svix-signature"]
	if id == "" || ts == "" || sigs == "" {
		return ports.MailEvent{}, apperrors.ErrUnauthorized
	}
	secs, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return ports.MailEvent{}, apperrors.ErrUnauthorized
	}
	if d := now.Sub(time.Unix(secs, 0)); d > webhookTolerance || d < -webhookTolerance {
		return ports.MailEvent{}, apperrors.ErrUnauthorized
	}

	mac := hmac.New(sha256.New, w.secret)
	mac.Write([]byte(id + "." + ts + "."))
	mac.Write(body)
	want := mac.Sum(nil)
	ok := false
	for _, part := range strings.Fields(sigs) {
		v, sig, found := strings.Cut(part, ",")
		if !found || v != "v1" {
			continue
		}
		got, err := base64.StdEncoding.DecodeString(sig)
		if err == nil && hmac.Equal(got, want) {
			ok = true
		}
	}
	if !ok {
		return ports.MailEvent{}, apperrors.ErrUnauthorized
	}

	var p struct {
		Type string `json:"type"`
		Data struct {
			EmailID string `json:"email_id"`
			Bounce  struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"bounce"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return ports.MailEvent{}, errors.New("resend webhook: the body is not valid JSON")
	}
	ev := ports.MailEvent{ProviderID: p.Data.EmailID}
	switch p.Type {
	case "email.bounced":
		ev.Kind = ports.MailBounced
		ev.Reason = strings.TrimSpace(p.Data.Bounce.Type + " " + p.Data.Bounce.Message)
	case "email.complained":
		ev.Kind = ports.MailComplained
		ev.Reason = "the recipient marked it as spam"
	default:
		ev.Kind = ports.MailIgnored
	}
	return ev, nil
}
