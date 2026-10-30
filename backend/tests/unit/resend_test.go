package unit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"coloplus-backend/internal/adapters/mail"
	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
)

// The outbox decides what to do from how the adapter classifies a failure,
// so the split between "try again later" and "will never work" is the part
// worth pinning down.
func TestResend_ClassifiesFailures(t *testing.T) {
	var gotAuth, gotKey string
	var gotBody map[string]any
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotKey = r.Header.Get("Authorization"), r.Header.Get("Idempotency-Key")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"id":"abc123"}`))
	}))
	defer srv.Close()

	r, err := mail.NewResend(mail.ResendConfig{APIKey: "key", From: "Coloplus <a@b.example>", BaseURL: srv.URL, RatePerSecond: 1000})
	if err != nil {
		t.Fatal(err)
	}
	send := func() (string, error) {
		return r.Send(context.Background(), domain.Email{To: "x@y.example", Subject: "s", Text: "t"}, "dedupe-1")
	}

	id, err := send()
	if err != nil || id != "abc123" {
		t.Fatalf("success = %q, %v", id, err)
	}
	if gotAuth != "Bearer key" || gotKey != "dedupe-1" || gotBody["subject"] != "s" {
		t.Fatalf("request = auth %q key %q body %v", gotAuth, gotKey, gotBody)
	}

	for code, want := range map[int]error{
		http.StatusTooManyRequests:     apperrors.ErrUnavailable,
		http.StatusInternalServerError: apperrors.ErrUnavailable,
		http.StatusUnprocessableEntity: apperrors.ErrMailRejected,
		http.StatusForbidden:           apperrors.ErrMailRejected,
	} {
		status = code
		if _, err := send(); !errors.Is(err, want) {
			t.Errorf("status %d: error %v, want %v", code, err, want)
		}
	}
}
