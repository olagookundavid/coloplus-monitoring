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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"coloplus-backend/internal/adapters/mail"
	"coloplus-backend/internal/adapters/postgres"
	"coloplus-backend/internal/adapters/security"
	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/platform"
	"coloplus-backend/internal/ports"
	"coloplus-backend/internal/services"
	"coloplus-backend/internal/sql/migrations"
	httpapi "coloplus-backend/internal/transport/http"

	"pkg-common/middleware"
	"pkg-common/token"
	"pkg-common/totp"

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
	box     *security.SecretBox
	// authenticator holds each enrolled test user's TOTP secret by email.
	authenticator map[string]string

	mailer        *fakeMailer
	store         *fakeStore
	notifications *services.NotificationService
	incidents     *services.IncidentService
	corrections   *services.CorrectionService
	reportAlerts  *services.ReportAlertService
	audit         *services.AuditService
	acceptance    *services.AcceptanceService
	fuel          *services.FuelService
	digest        *services.DigestService
	anomalies     *services.AnomalyService
	disputes      *services.DisputeService
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
	return fmt.Sprintf("fake-%d-%d", time.Now().UnixNano(), len(m.sent)), nil
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
	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{URL: url, MaxConns: perfConns(), Timezone: "Africa/Lagos"})
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
	incidentCorrectionRepo := postgres.NewIncidentCorrectionRepository(db)
	disputeRepo := postgres.NewDisputeRepository(db)
	ticketRepo := postgres.NewTicketRepository(db)
	attachmentRepo := postgres.NewAttachmentRepository(db)
	fuelRepo := postgres.NewFuelRepository(db)
	acceptanceRepo := postgres.NewAcceptanceRepository(db)
	notificationRepo := postgres.NewNotificationRepository(db)
	userTokenRepo := postgres.NewUserTokenRepository(db)
	recoveryRepo := postgres.NewRecoveryCodeRepository(db)
	auditStore := postgres.NewAuditRepository(db)

	batcher, err := platform.NewBatcher(platform.BatcherConfig{
		Name: "audit", MaxItems: 20, Interval: 50 * time.Millisecond, QueueSize: 100,
	}, auditStore.InsertBatch, log)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = batcher.Close(context.Background()) })
	audit := services.NewAuditService(auditStore, batcher, clock, log)

	mailer := &fakeMailer{}
	alertRepo := postgres.NewAlertRepository(db)
	notifications := services.NewNotificationService(notificationRepo, mailer, audit, clock, log).WithAlerts(alertRepo).WithMailEvents(mustWebhook(t))
	auth, err := services.NewAuthService(userRepo, hasher, security.NewPasetoIssuer(maker),
		security.NewTOTP("Coloplus"), box, recoveryRepo, tx, notifications, audit, clock, time.Hour, log)
	if err != nil {
		t.Fatal(err)
	}
	accounts := services.NewAccountService(userRepo, userTokenRepo, hasher, tx, notifications, audit, clock, "https://app.test")
	maintenanceRepo := postgres.NewMaintenanceRepository(db)
	adjustments := services.NewAdjustments(maintenanceRepo, reportRepo, loc, true)
	directory := services.NewDirectoryService(postgres.NewDirectoryRepository(db), log)
	dashboard := services.NewDashboardService(siteRepo, reportRepo, incidentRepo, fuelRepo, clock, loc, dashboardTTL(), 24).WithAdjustments(adjustments)
	digest := services.NewDigestService(userRepo, postgres.NewDigestPreferenceRepository(db), auth, dashboard, incidentRepo, notifications, clock, loc, "https://app.test")
	anomalies := services.NewAnomalyService(siteRepo, reportRepo, incidentRepo, userRepo, notifications, clock, loc, "https://app.test")
	incidents := services.NewIncidentService(incidentRepo, siteRepo, userRepo, tx, audit, clock, notifications, loc, "https://app.test")
	corrections := services.NewCorrectionService(incidentCorrectionRepo, incidentRepo, tx, audit, clock)
	acceptance := services.NewAcceptanceService(acceptanceRepo, siteRepo, userRepo, tx, audit, clock, notifications, loc, "https://app.test").WithEscalation(7 * 24 * time.Hour)
	reportAlerts := services.NewReportAlertService(siteRepo, reportRepo, userRepo, notifications, clock, loc, "https://app.test")
	disputes := services.NewDisputeService(disputeRepo, reportRepo, siteRepo, userRepo, tx, audit, clock, notifications, loc, "https://app.test").WithResponseTime(72 * time.Hour)
	tickets := services.NewTicketService(ticketRepo, incidentRepo, siteRepo, userRepo, tx, audit, clock, notifications, "https://app.test")
	store := newFakeStore()
	periodRepo := postgres.NewPeriodClosureRepository(db)
	exportService := services.NewExportService(siteRepo, reportRepo, disputeRepo, incidentRepo, tx, audit, clock, loc).WithAcceptances(acceptanceRepo).WithSnapshots(postgres.NewIssuedExportRepository(db), store).WithRuns(postgres.NewCalculationRunRepository(db)).WithAdjustments(adjustments)
	periods := services.NewPeriodService(periodRepo, exportService, tx, audit, clock, loc, 5)
	attachments := services.NewAttachmentService(attachmentRepo, store, ticketRepo, incidentRepo, disputeRepo, acceptanceRepo,
		tx, audit, clock, services.AttachmentConfig{
			MaxBytes: 10 << 20, UploadTTL: 15 * time.Minute, DownloadTTL: 2 * time.Minute, PendingTTL: 24 * time.Hour,
		}).WithReports(reportRepo)
	fuel := services.NewFuelService(fuelRepo, siteRepo, reportRepo, userRepo, tx, audit, clock, notifications, loc, "https://app.test", 24)
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
		Users:         services.NewUserService(userRepo, siteRepo, hasher, tx, audit, clock, accounts, recoveryRepo, notifications),
		Sites:         services.NewSiteService(siteRepo, tx, audit),
		Reports:       services.NewReportService(reportRepo, siteRepo, tx, audit, clock, loc).WithPeriods(periodRepo).WithAdjustments(adjustments),
		Incidents:     incidents,
		Corrections:   corrections,
		Disputes:      disputes,
		Tickets:       tickets,
		Attachments:   attachments,
		Fuel:          fuel,
		Alerts:        services.NewAlertService(alertRepo, clock),
		Audit:         audit,
		Acceptance:    acceptance,
		SiteOverviews: services.NewSiteOverviewService(siteRepo, reportRepo, incidentRepo, disputeRepo, acceptanceRepo, clock, loc).WithAdjustments(adjustments),
		Maintenance:   services.NewMaintenanceService(maintenanceRepo, siteRepo, userRepo, tx, audit, clock, notifications, loc, "https://app.test", domain.MaintenanceRules{NoticeHours: 48, MaxHours: 8, MonthlyCapHours: 12}),
		Exports:       exportService,
		Periods:       periods,
		Dashboard:     dashboard,
		Directory:     directory,
		Digest:        digest,
	})
	return &stack{t: t, handler: handler, pool: pool, users: userRepo, hasher: hasher, loc: loc, batcher: batcher,
		box: box, authenticator: map[string]string{},
		store: store, mailer: mailer, notifications: notifications, incidents: incidents, corrections: corrections, reportAlerts: reportAlerts, audit: audit, acceptance: acceptance, fuel: fuel, digest: digest, anomalies: anomalies, disputes: disputes}
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
	email, id = s.seedUserWithoutMFA(role, siteIDs...)
	// Roles that must sign in with two steps are enrolled, as they would be
	// in real life, so the other tests exercise the normal sign-in.
	required, err := s.users.RoleRequiresMFA(context.Background(), role)
	if err != nil {
		s.t.Fatal(err)
	}
	if required {
		secret, err := totp.NewSecret()
		if err != nil {
			s.t.Fatal(err)
		}
		sealed, err := s.box.Seal(secret)
		if err != nil {
			s.t.Fatal(err)
		}
		if err := s.users.SetTOTP(context.Background(), id, sealed, true); err != nil {
			s.t.Fatal(err)
		}
		s.authenticator[email] = secret
	}
	return email, id
}

// seedUserWithoutMFA creates a user who has not set up two-step verification.
func (s *stack) seedUserWithoutMFA(role domain.Role, siteIDs ...string) (email, id string) {
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

// codeFor is the authenticator code an enrolled test user would read off their phone.
func (s *stack) codeFor(email string) string {
	s.t.Helper()
	code, err := totp.Code(s.authenticator[email], time.Now())
	if err != nil {
		s.t.Fatal(err)
	}
	return code
}

// login signs in and returns the access token.
func (s *stack) login(email string) string {
	s.t.Helper()
	body := map[string]any{"email": email, "password": testPassword}
	if _, ok := s.authenticator[email]; ok {
		body["totp_code"] = s.codeFor(email)
	}
	r := s.expect(s.call("POST", "/api/v1/auth/login", body, ""), 200)
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

// fakeStore stands in for R2: it remembers what a browser would have uploaded,
// so a test can send a file, send a wrong one, or send nothing.
type fakeStore struct {
	mu      sync.Mutex
	objects map[string]ports.ObjectInfo
	deleted []string
	puts    []string
}

func newFakeStore() *fakeStore { return &fakeStore{objects: map[string]ports.ObjectInfo{}} }

// put is the browser's PUT to the signed link.
func (f *fakeStore) put(key string, size int64, sha string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = ports.ObjectInfo{SizeBytes: size, SHA256Hex: sha}
}

func (f *fakeStore) PresignUpload(req ports.UploadRequest) (ports.PresignedUpload, error) {
	return ports.PresignedUpload{
		Method: "PUT", URL: "https://store.test/up/" + req.Key,
		Headers: map[string]string{"Content-Type": req.ContentType}, ExpiresAt: time.Now().Add(req.TTL),
	}, nil
}

func (f *fakeStore) PresignDownload(req ports.DownloadRequest) (string, time.Time, error) {
	return "https://store.test/dl/" + req.Key, time.Now().Add(req.TTL), nil
}

func (f *fakeStore) Stat(_ context.Context, key string) (ports.ObjectInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	info, ok := f.objects[key]
	if !ok {
		return ports.ObjectInfo{}, apperrors.ErrNotFound
	}
	return info, nil
}

func (f *fakeStore) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.objects, key)
	f.deleted = append(f.deleted, key)
	return nil
}

// perfConns sizes the test pool; a speed test can widen it with PERF_CONNS.
func perfConns() int32 {
	if n, err := strconv.Atoi(os.Getenv("PERF_CONNS")); err == nil && n > 0 {
		return int32(n)
	}
	return 6
}

// dashboardTTL keeps the dashboard uncached so every test sees its own data at
// once; a speed test sets DASHBOARD_CACHE_TTL to measure the cached path.
func dashboardTTL() time.Duration {
	d, _ := time.ParseDuration(os.Getenv("DASHBOARD_CACHE_TTL"))
	return d
}

// webhookSecret is the signing secret the test stack accepts bounce webhooks with.
const webhookSecret = "whsec_dGVzdC13ZWJob29rLXNlY3JldC0xMjM0NTY3ODkwMTI="

func mustWebhook(t *testing.T) *mail.ResendWebhook {
	t.Helper()
	wh, err := mail.NewResendWebhook(webhookSecret)
	if err != nil {
		t.Fatal(err)
	}
	return wh
}

func (f *fakeStore) Put(_ context.Context, key, _ string, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = ports.ObjectInfo{SizeBytes: int64(len(data))}
	f.puts = append(f.puts, key)
	return nil
}
