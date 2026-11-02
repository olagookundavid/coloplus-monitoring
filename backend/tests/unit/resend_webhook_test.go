package unit

import (
	"errors"
	"testing"
	"time"

	"coloplus-backend/internal/adapters/mail"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// The webhook check is run against the worked example published by Svix, the
// scheme Resend uses. A forged or replayed bounce could otherwise mark real
// emails as undelivered, or hide that they were.
func TestResendWebhook_VerifiesTheSvixExampleAndRefusesTheRest(t *testing.T) {
	wh, err := mail.NewResendWebhook("whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw")
	if err != nil {
		t.Fatal(err)
	}
	headers := map[string]string{
		"svix-id":        "msg_p5jXN8AQM9LWM0D4loKWxJek",
		"svix-timestamp": "1614265330",
		"svix-signature": "v1,g0hM9SsE+OTPJTGt/tmIKtSyZlE3uFJELVlNIOLJ1OE=",
	}
	body := []byte(`{"test": 2432232314}`)
	sent := time.Unix(1614265330, 0)

	ev, err := wh.Verify(headers, body, sent.Add(time.Minute))
	if err != nil || ev.Kind != ports.MailIgnored {
		t.Fatalf("the published example must verify (an unknown event is ignored): %v %+v", err, ev)
	}
	for name, tc := range map[string]struct {
		h    map[string]string
		body []byte
		now  time.Time
	}{
		"changed body":  {headers, []byte(`{"test": 2432232315}`), sent},
		"replayed late": {headers, body, sent.Add(time.Hour)},
		"no signature":  {map[string]string{"svix-id": headers["svix-id"], "svix-timestamp": headers["svix-timestamp"]}, body, sent},
		"wrong secret":  {map[string]string{"svix-id": headers["svix-id"], "svix-timestamp": headers["svix-timestamp"], "svix-signature": "v1,AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}, body, sent},
	} {
		if _, err := wh.Verify(tc.h, tc.body, tc.now); !errors.Is(err, apperrors.ErrUnauthorized) {
			t.Fatalf("%s must be refused as unauthorised, got %v", name, err)
		}
	}
}
