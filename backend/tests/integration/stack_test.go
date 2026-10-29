package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"coloplus-backend/internal/adapters/postgres"
	"coloplus-backend/internal/adapters/security"
	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/platform"
	"coloplus-backend/internal/services"
	"coloplus-backend/internal/sql/migrations"
	httpapi "coloplus-backend/internal/transport/http"

	"pkg-common/middleware"
	"pkg-common/token"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	testPassword = "correct-horse-battery-9"
	cookieName   = "test_access"
)

var seq atomic.Int64

// uniq returns a value that never collides across test runs.
func uniq(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano()%1_000_000_000, seq.Add(1))
}

// stack is the real application, wired like cmd/api does it, on a real
// Postgres, driven through its HTTP handler.
type stack struct {
	t       *testing.T
	handler http.Handler
	pool    *pgxpool.Pool
	users   *postgres.UserRepository
	hasher  *security.BcryptHasher
	loc     *time.Location
	batcher *platform.Batcher[domain.AuditEvent]

	mailer        *fakeMailer
	notifications *services.NotificationService
	incidents     *services.IncidentService
	reportAlerts  *services.ReportAlertService
	audit         *services.AuditService
}

// fakeMailer records what would have been sent, and fails on demand.
type fakeMailer struct {
	mu   sync.Mutex
	sent []domain.Email
	// fail, when set, decides the result of a send.
	fail func(domain.Email) error
}

func (m *fakeMailer) Send(_ context.Context, mail domain.Email, _ string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		if err := m.fail(mail); err != nil {
			return "", err
		}
	}
	m.sent = append(m.sent, mail)
	return "fake-id", nil
}

// to returns the emails sent to an address.
func (m *fakeMailer) to(addr string) []domain.Email {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Email
	for _, e := range m.sent {
		if e.To == addr {
			out = append(out, e)
		}
	}
	return out
}

func newStack(t *testing.T) *stack {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := postgres.Migrate(ctx, url, migrations.FS, log); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{URL: url, MaxConns: 6, Timezone: "Africa/Lagos"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	loc, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		t.Fatal(err)
	}
	maker, _ := token.NewPasetoMaker("12345678901234567890123456789012")
	box, _ := security.NewSecretBox("abcdefghijklmnopqrstuvwxyz012345")
	hasher := security.NewBcryptHasher(4)
	clock := platform.SystemClock{}

	db := postgres.NewDB(pool)
	tx := postgres.NewTxManager(db)
	userRepo := postgres.NewUserRepository(db)
	siteRepo := postgres.NewSiteRepository(db)
	reportRepo := postgres.NewReportRepository(db)
	incidentRepo := postgres.NewIncidentRepository(db)
	disputeRepo := postgres.NewDisputeRepository(db)
	notificationRepo := postgres.NewNotificationRepository(db)
	userTokenRepo := postgres.NewUserTokenRepository(db)
	auditStore := postgres.NewAuditRepository(db)

	batcher, err := platform.NewBatcher(platform.BatcherConfig{
		Name: "audit", MaxItems: 20, Interval: 50 * time.Millisecond, QueueSize: 100,
	}, auditStore.InsertBatch, log)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = batcher.Close(context.Background()) })
	audit := services.NewAuditService(auditStore, batcher, clock, log)

	auth, err := services.NewAuthService(userRepo, hasher, security.NewPasetoIssuer(maker),
		security.NewTOTP("Coloplus"), box, audit, clock, time.Hour, log)
	if err != nil {
		t.Fatal(err)
	}
	mailer := &fakeMailer{}
	notifications := services.NewNotificationService(notificationRepo, mailer, audit, clock, log)
	accounts := services.NewAccountService(userRepo, userTokenRepo, hasher, tx, notifications, audit, clock, "https://app.test")
	incidents := services.NewIncidentService(incidentRepo, siteRepo, userRepo, tx, audit, clock, notifications, loc, "https://app.test")
	reportAlerts := services.NewReportAlertService(siteRepo, reportRepo, userRepo, notifications, clock, loc, "https://app.test")
	disputes := services.NewDisputeService(disputeRepo, reportRepo, siteRepo, userRepo, tx, audit, clock, notifications, loc, "https://app.test")
	rlCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	handler := httpapi.NewRouter(httpapi.Deps{
		Log: log, RateLimitCtx: rlCtx, TokenMaker: maker,
		RateLimit:     middleware.RateLimitConfig{Enabled: false},
		AuthRateLimit: middleware.RateLimitConfig{Enabled: false},
		Cookie:        httpapi.CookieConfig{Name: cookieName, SameSite: http.SameSiteLaxMode},
		Loc:           loc, Now: clock.Now,
		Health:   services.NewHealthService(db),
		Auth:     auth,
		Accounts: accounts, Notifications: notifications,
		Users:     services.NewUserService(userRepo, siteRepo, hasher, tx, audit, clock, accounts),
		Sites:     services.NewSiteService(siteRepo, tx, audit),
		Reports:   services.NewReportService(reportRepo, siteRepo, tx, audit, clock, loc),
		Incidents: incidents,
		Disputes:  disputes,
		Audit:     audit,
		Dashboard: services.NewDashboardService(siteRepo, reportRepo, clock, loc),
	})
	return &stack{t: t, handler: handler, pool: pool, users: userRepo, hasher: hasher, loc: loc, batcher: batcher,
		mailer: mailer, notifications: notifications, incidents: incidents, reportAlerts: reportAlerts, audit: audit}
}

// reply is a decoded response.
type reply struct {
	Code   int
	Body   map[string]any
	Raw    string
	Header http.Header
}

// call sends a JSON request, with a bearer token when one is given.
func (s *stack) call(method, path string, body any, tok string) reply {
	s.t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			s.t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	} else if method != http.MethodGet {
		rd = strings.NewReader("{}")
	}
	req := httptest.NewRequest(method, path, rd)
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	s.handler.ServeHTTP(w, req)

	r := reply{Code: w.Code, Raw: w.Body.String(), Header: w.Header()}
	_ = json.Unmarshal(w.Body.Bytes(), &r.Body)
	return r
}

func (r reply) str(keys ...string) string {
	var cur any = r.Body
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = m[k]
	}
	if cur == nil {
		return ""
	}
	return fmt.Sprint(cur)
}

func (r reply) list(key string) []any {
	l, _ := r.Body[key].([]any)
	return l
}

func (s *stack) expect(r reply, code int) reply {
	s.t.Helper()
	if r.Code != code {
		s.t.Fatalf("status = %d, want %d. body: %s", r.Code, code, r.Raw)
	}
	return r
}

// seedUser inserts a user directly, as a stand-in for the first-admin command.
func (s *stack) seedUser(role domain.Role, siteIDs ...string) (email, id string) {
	s.t.Helper()
	ctx := context.Background()
	hash, err := s.hasher.Hash(testPassword)
	if err != nil {
		s.t.Fatal(err)
	}
	email = uniq(string(role)) + "@coloplus.example"
	u, err := s.users.Create(ctx, domain.User{Email: email, FullName: "Test " + string(role), PasswordHash: hash, Role: role, Active: true})
	if err != nil {
		s.t.Fatal(err)
	}
	if len(siteIDs) > 0 {
		if err := s.users.SetUserSites(ctx, u.ID, siteIDs); err != nil {
			s.t.Fatal(err)
		}
	}
	return email, u.ID
}

// login signs in and returns the access token.
func (s *stack) login(email string) string {
	s.t.Helper()
	r := s.expect(s.call("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": testPassword}, ""), 200)
	return r.str("access_token")
}

func (s *stack) newSite(adminToken string, sources ...string) string {
	s.t.Helper()
	if len(sources) == 0 {
		sources = []string{"grid", "generator"}
	}
	r := s.expect(s.call("POST", "/api/v1/sites", map[string]any{
		"airtel_code": uniq("AIR"), "coloplus_code": uniq("CP"), "name": "Test site", "state": "Lagos",
		"power_sources": sources, "status": "active",
	}, adminToken), 201)
	return r.str("id")
}

// reportDay is yesterday in Lagos, so intervals are never in the future.
func (s *stack) reportDay(daysAgo int) (date string, at func(h, m int) string) {
	y, m, d := time.Now().In(s.loc).AddDate(0, 0, -daysAgo).Date()
	date = fmt.Sprintf("%04d-%02d-%02d", y, m, d)
	return date, func(h, min int) string { return time.Date(y, m, d, h, min, 0, 0, s.loc).Format(time.RFC3339) }
}

func iv(start, end, state, source string) map[string]any {
	m := map[string]any{"start": start, "end": end, "state": state}
	if source != "" {
		m["source"] = source
	}
	return m
}

func (s *stack) countAudit(action, entityID string) int {
	s.t.Helper()
	_ = s.batcher.Close(context.Background()) // flush anything batched
	var n int
	if err := s.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_log WHERE action = $1 AND entity_id = $2`, action, entityID).Scan(&n); err != nil {
		s.t.Fatal(err)
	}
	return n
}
