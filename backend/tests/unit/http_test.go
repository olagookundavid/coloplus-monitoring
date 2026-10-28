package unit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
		Cookie:       httpapi.CookieConfig{Name: "test_token"},
		Loc:          time.UTC,
		Now:          time.Now,
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

func TestWritesMustBeJSON(t *testing.T) {
	h := testRouter(t, fakePinger{}, middleware.RateLimitConfig{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("email=a&password=b"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "application/json") {
		t.Fatalf("a form post must be refused: %d %s", w.Code, w.Body.String())
	}
}

func TestProtectedRoutesNeedAToken(t *testing.T) {
	h := testRouter(t, fakePinger{}, middleware.RateLimitConfig{})
	for _, path := range []string{"/api/v1/auth/me", "/api/v1/sites", "/api/v1/daily-reports", "/api/v1/users", "/api/v1/dashboard/overview"} {
		if got := do(h, http.MethodGet, path).Code; got != http.StatusUnauthorized {
			t.Errorf("GET %s without a token = %d, want 401", path, got)
		}
	}
}
