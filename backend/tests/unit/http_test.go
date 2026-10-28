package unit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"coloplus-backend/internal/services"
	httpapi "coloplus-backend/internal/transport/http"

	"pkg-common/middleware"
	"pkg-common/token"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func testRouter(t *testing.T, pinger fakePinger, rl middleware.RateLimitConfig) http.Handler {
	t.Helper()
	maker, err := token.NewPasetoMaker("12345678901234567890123456789012")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return httpapi.NewRouter(httpapi.Deps{
		Log:          discardLogger(),
		RateLimit:    rl,
		RateLimitCtx: ctx,
		TokenMaker:   maker,
		CookieName:   "test_token",
		Health:       services.NewHealthService(pinger),
	})
}

func do(h http.Handler, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestHealthcheck(t *testing.T) {
	up := testRouter(t, fakePinger{}, middleware.RateLimitConfig{})
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		if got := do(up, method, "/healthcheck").Code; got != http.StatusOK {
			t.Errorf("%s /healthcheck = %d, want 200", method, got)
		}
	}

	down := testRouter(t, fakePinger{err: errors.New("no route to host")}, middleware.RateLimitConfig{})
	w := do(down, http.MethodGet, "/healthcheck")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if strings.Contains(w.Body.String(), "no route") {
		t.Fatalf("driver detail leaked: %s", w.Body.String())
	}
}

func TestHealthcheckIsNotRateLimited(t *testing.T) {
	h := testRouter(t, fakePinger{}, middleware.RateLimitConfig{Enabled: true, Rps: 0.001, Burst: 1})
	for i := 0; i < 5; i++ {
		if got := do(h, http.MethodGet, "/healthcheck").Code; got != http.StatusOK {
			t.Fatalf("request %d = %d, want 200", i, got)
		}
	}
}

func TestPingAndSecurityHeaders(t *testing.T) {
	w := do(testRouter(t, fakePinger{}, middleware.RateLimitConfig{}), http.MethodGet, "/api/v1/ping")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "pong") {
		t.Fatalf("ping = %d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("X-Request-ID") == "" {
		t.Fatalf("missing headers: %v", w.Header())
	}
}

func TestUnknownRouteIsAWordedNotFound(t *testing.T) {
	w := do(testRouter(t, fakePinger{}, middleware.RateLimitConfig{}), http.MethodGet, "/api/v1/nope")
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "couldn't find") {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}
