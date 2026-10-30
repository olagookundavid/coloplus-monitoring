package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pkg-common/token"

	"github.com/gin-gonic/gin"
)

const testKey = "12345678901234567890123456789012"

func init() { gin.SetMode(gin.TestMode) }

func serve(h gin.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	r := gin.New()
	r.GET("/x", h, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuth(t *testing.T) {
	maker, _ := token.NewPasetoMaker(testKey)
	good, _, _ := maker.CreateToken("u1", time.Minute)
	expired, _, _ := maker.CreateToken("u1", -time.Minute)
	cfg := AuthConfig{CookieName: "app_token"}

	cases := []struct {
		name string
		req  func() *http.Request
		want int
	}{
		{"no token", func() *http.Request { return httptest.NewRequest("GET", "/x", nil) }, 401},
		{"bearer header", func() *http.Request {
			r := httptest.NewRequest("GET", "/x", nil)
			r.Header.Set("Authorization", "Bearer "+good)
			return r
		}, 204},
		{"cookie named by config", func() *http.Request {
			r := httptest.NewRequest("GET", "/x", nil)
			r.AddCookie(&http.Cookie{Name: "app_token", Value: good})
			return r
		}, 204},
		{"cookie with another name is ignored", func() *http.Request {
			r := httptest.NewRequest("GET", "/x", nil)
			r.AddCookie(&http.Cookie{Name: "other", Value: good})
			return r
		}, 401},
		{"expired", func() *http.Request {
			r := httptest.NewRequest("GET", "/x", nil)
			r.Header.Set("Authorization", "Bearer "+expired)
			return r
		}, 401},
		{"query token off by default", func() *http.Request {
			return httptest.NewRequest("GET", "/x?token="+good, nil)
		}, 401},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := serve(Auth(maker, cfg), tc.req()).Code; got != tc.want {
				t.Fatalf("status = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestRateLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := RateLimit(ctx, RateLimitConfig{Enabled: true, Rps: 0.001, Burst: 2})

	statuses := make([]int, 3)
	for i := range statuses {
		req := httptest.NewRequest("GET", "/x", nil)
		req.Header.Set("CF-Connecting-IP", "203.0.113.9")
		statuses[i] = serve(h, req).Code
	}
	if statuses[0] != 204 || statuses[1] != 204 || statuses[2] != http.StatusTooManyRequests {
		t.Fatalf("statuses = %v, want [204 204 429]", statuses)
	}

	other := httptest.NewRequest("GET", "/x", nil)
	other.Header.Set("CF-Connecting-IP", "203.0.113.10")
	if got := serve(h, other).Code; got != 204 {
		t.Fatalf("a different client was limited: %d", got)
	}
}
