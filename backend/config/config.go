// Package config loads the typed, validated configuration. It is the only
// place that reads environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
	EnvTest        = "test"
)

type Config struct {
	App       App
	DB        DB
	Auth      Auth
	CORS      CORS
	RateLimit RateLimit
	Workers   Workers
	Audit     Audit
	Mail      Mail
	Jobs      Jobs
	Swagger   Swagger
}

type App struct {
	Name            string
	Env             string
	HTTPAddr        string
	ShutdownTimeout time.Duration
	LogLevel        string
	// ReportingTimezone defines the reporting day (Africa/Lagos). It is
	// separate from the DB session zone and used by the domain's day maths.
	ReportingTimezone string
}

type DB struct {
	URL      string
	MaxConns int32
	MinConns int32
	// Timezone pins every session; day boundaries are still computed in the
	// domain layer from UTC timestamps.
	Timezone string
}

type Auth struct {
	// TokenKey is the 32-character PASETO symmetric key.
	TokenKey       string
	CookieName     string
	AccessTokenTTL time.Duration
	// SecretsKey is the 32-character key that encrypts TOTP seeds at rest.
	SecretsKey string
	// CookieSameSite is lax, strict or none. none is for a front end served
	// from a different site than the API and requires a secure cookie.
	CookieSameSite string
	CookieSecure   bool
	CookieDomain   string
	BcryptCost     int
	// MFAIssuer is the name shown in authenticator apps.
	MFAIssuer string
}

type CORS struct {
	AllowOrigins []string
}

type RateLimit struct {
	Enabled bool
	Rps     float64
	Burst   int
	// AuthPerMinute and AuthBurst bound sign-in attempts per client IP.
	AuthPerMinute int
	AuthBurst     int
}

type Workers struct {
	Size      int
	QueueSize int
}

type Audit struct {
	BatchSize     int
	BatchInterval time.Duration
	QueueSize     int
}

// Mail configures outgoing email. Resend is the provider; log only prints
// emails and is for local development.
type Mail struct {
	Provider      string
	ResendAPIKey  string
	From          string
	RatePerSecond int
	Timeout       time.Duration
	// BaseURL is where the front end lives. Invitation and reset links, and
	// links to incidents in alerts, are built from it.
	BaseURL string
}

// Jobs holds the cron specs (five fields) of the scheduled work.
type Jobs struct {
	DispatchEmailsCron string
	IncidentAlertsCron string
}

type Swagger struct {
	Enabled bool
}

func (c Config) IsProduction() bool { return c.App.Env == EnvProduction }

// FromEnv loads a local .env file when present (development convenience) and
// then reads the process environment.
func FromEnv() (Config, error) {
	_ = godotenv.Load()
	return Load(os.LookupEnv)
}

// Load builds the configuration from a lookup function and reports every
// problem at once rather than the first one.
func Load(lookup func(string) (string, bool)) (Config, error) {
	r := &reader{lookup: lookup}
	env := r.str("APP_ENV", EnvDevelopment)

	cfg := Config{
		App: App{
			Name:              r.str("APP_NAME", "coloplus"),
			Env:               env,
			HTTPAddr:          r.str("HTTP_ADDR", ":8080"),
			ShutdownTimeout:   r.duration("SHUTDOWN_TIMEOUT", 20*time.Second),
			LogLevel:          r.str("LOG_LEVEL", "info"),
			ReportingTimezone: r.str("REPORTING_TIMEZONE", "Africa/Lagos"),
		},
		DB: DB{
			URL:      r.str("DATABASE_URL", ""),
			MaxConns: int32(r.integer("DB_MAX_CONNS", 8)),
			MinConns: int32(r.integer("DB_MIN_CONNS", 1)),
			Timezone: r.str("DB_TIMEZONE", "Africa/Lagos"),
		},
		Auth: Auth{
			TokenKey:       r.str("TOKEN_SYMMETRIC_KEY", ""),
			CookieName:     r.str("AUTH_COOKIE_NAME", ""),
			AccessTokenTTL: r.duration("ACCESS_TOKEN_TTL", 12*time.Hour),
			SecretsKey:     r.str("SECRETS_KEY", ""),
			CookieSameSite: strings.ToLower(r.str("AUTH_COOKIE_SAMESITE", "lax")),
			CookieSecure:   r.boolean("AUTH_COOKIE_SECURE", env == EnvProduction),
			CookieDomain:   r.str("AUTH_COOKIE_DOMAIN", ""),
			BcryptCost:     r.integer("BCRYPT_COST", 12),
			MFAIssuer:      r.str("MFA_ISSUER", "Coloplus"),
		},
		CORS: CORS{AllowOrigins: r.list("CORS_ALLOW_ORIGINS")},
		RateLimit: RateLimit{
			Enabled:       r.boolean("RATE_LIMIT_ENABLED", true),
			Rps:           float64(r.integer("RATE_LIMIT_RPS", 10)),
			Burst:         r.integer("RATE_LIMIT_BURST", 30),
			AuthPerMinute: r.integer("AUTH_RATE_LIMIT_PER_MINUTE", 10),
			AuthBurst:     r.integer("AUTH_RATE_LIMIT_BURST", 5),
		},
		Workers: Workers{
			Size:      r.integer("WORKER_POOL_SIZE", 8),
			QueueSize: r.integer("WORKER_POOL_QUEUE", 64),
		},
		Audit: Audit{
			BatchSize:     r.integer("AUDIT_BATCH_SIZE", 20),
			BatchInterval: r.duration("AUDIT_BATCH_INTERVAL", 5*time.Second),
			QueueSize:     r.integer("AUDIT_QUEUE_SIZE", 1000),
		},
		Mail: Mail{
			Provider:      strings.ToLower(r.str("MAIL_PROVIDER", "log")),
			ResendAPIKey:  r.str("RESEND_API_KEY", ""),
			From:          r.str("MAIL_FROM", ""),
			RatePerSecond: r.integer("MAIL_RATE_PER_SECOND", 8),
			Timeout:       r.duration("MAIL_TIMEOUT", 10*time.Second),
			BaseURL:       strings.TrimRight(r.str("APP_BASE_URL", "http://localhost:5173"), "/"),
		},
		Jobs: Jobs{
			DispatchEmailsCron: r.str("JOB_DISPATCH_EMAILS_CRON", "* * * * *"),
			IncidentAlertsCron: r.str("JOB_INCIDENT_ALERTS_CRON", "*/10 * * * *"),
		},
		Swagger: Swagger{Enabled: r.boolean("SWAGGER_ENABLED", false)},
	}

	errs := append(r.errs, cfg.validate()...)
	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %w", errors.Join(errs...))
	}
	return cfg, nil
}

func (c Config) validate() []error {
	var errs []error
	bad := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	switch c.App.Env {
	case EnvDevelopment, EnvProduction, EnvTest:
	default:
		bad("APP_ENV must be development, production or test, got %q", c.App.Env)
	}
	if c.DB.URL == "" {
		bad("DATABASE_URL is required")
	}
	if c.DB.MaxConns < 1 || c.DB.MinConns < 0 || c.DB.MinConns > c.DB.MaxConns {
		bad("DB_MIN_CONNS (%d) must be between 0 and DB_MAX_CONNS (%d), and DB_MAX_CONNS must be at least 1", c.DB.MinConns, c.DB.MaxConns)
	}
	if _, err := time.LoadLocation(c.DB.Timezone); err != nil {
		bad("DB_TIMEZONE %q is not a known time zone", c.DB.Timezone)
	}
	if len(c.Auth.TokenKey) != 32 {
		bad("TOKEN_SYMMETRIC_KEY must be exactly 32 characters")
	}
	if c.Auth.CookieName == "" {
		bad("AUTH_COOKIE_NAME is required")
	}
	if len(c.Auth.SecretsKey) != 32 {
		bad("SECRETS_KEY must be exactly 32 characters")
	}
	if c.Auth.SecretsKey != "" && c.Auth.SecretsKey == c.Auth.TokenKey {
		bad("SECRETS_KEY must differ from TOKEN_SYMMETRIC_KEY")
	}
	switch c.Auth.CookieSameSite {
	case "lax", "strict":
	case "none":
		if !c.Auth.CookieSecure {
			bad("AUTH_COOKIE_SAMESITE=none requires AUTH_COOKIE_SECURE=true")
		}
	default:
		bad("AUTH_COOKIE_SAMESITE must be lax, strict or none, got %q", c.Auth.CookieSameSite)
	}
	if c.Auth.BcryptCost < 4 || c.Auth.BcryptCost > 14 {
		bad("BCRYPT_COST must be between 4 and 14")
	}
	if _, err := time.LoadLocation(c.App.ReportingTimezone); err != nil {
		bad("REPORTING_TIMEZONE %q is not a known time zone", c.App.ReportingTimezone)
	}
	if c.RateLimit.AuthPerMinute < 1 || c.RateLimit.AuthBurst < 1 {
		bad("AUTH_RATE_LIMIT_PER_MINUTE and AUTH_RATE_LIMIT_BURST must be at least 1")
	}
	if c.Workers.Size < 1 || c.Workers.QueueSize < 0 {
		bad("WORKER_POOL_SIZE must be at least 1 and WORKER_POOL_QUEUE must not be negative")
	}
	if c.Audit.BatchSize < 1 || c.Audit.QueueSize < c.Audit.BatchSize || c.Audit.BatchInterval <= 0 {
		bad("AUDIT_BATCH_SIZE must be at least 1, AUDIT_QUEUE_SIZE at least AUDIT_BATCH_SIZE, AUDIT_BATCH_INTERVAL positive")
	}
	if c.RateLimit.Enabled && (c.RateLimit.Rps <= 0 || c.RateLimit.Burst < 1) {
		bad("RATE_LIMIT_RPS and RATE_LIMIT_BURST must be positive when the rate limit is enabled")
	}
	switch c.Mail.Provider {
	case "log":
	case "resend":
		if c.Mail.ResendAPIKey == "" || c.Mail.From == "" {
			bad("MAIL_PROVIDER=resend needs RESEND_API_KEY and MAIL_FROM")
		}
	default:
		bad("MAIL_PROVIDER must be log or resend, got %q", c.Mail.Provider)
	}
	if c.Mail.RatePerSecond < 1 || c.Mail.Timeout <= 0 {
		bad("MAIL_RATE_PER_SECOND must be at least 1 and MAIL_TIMEOUT positive")
	}
	if !strings.HasPrefix(c.Mail.BaseURL, "http://") && !strings.HasPrefix(c.Mail.BaseURL, "https://") {
		bad("APP_BASE_URL must start with http:// or https://")
	}
	for _, origin := range c.CORS.AllowOrigins {
		if origin == "*" {
			bad("CORS_ALLOW_ORIGINS must list explicit origins, never *")
		}
	}
	if c.IsProduction() {
		if len(c.CORS.AllowOrigins) == 0 {
			bad("CORS_ALLOW_ORIGINS is required in production")
		}
		if !c.RateLimit.Enabled {
			bad("RATE_LIMIT_ENABLED must be true in production")
		}
		if !c.Auth.CookieSecure {
			bad("AUTH_COOKIE_SECURE must be true in production")
		}
		if c.Auth.BcryptCost < 10 {
			bad("BCRYPT_COST must be at least 10 in production")
		}
		if c.Swagger.Enabled {
			bad("SWAGGER_ENABLED must be false in production")
		}
		if c.Mail.Provider != "resend" {
			bad("MAIL_PROVIDER must be resend in production")
		}
		if !strings.HasPrefix(c.Mail.BaseURL, "https://") {
			bad("APP_BASE_URL must be https in production")
		}
	}
	return errs
}

// reader collects parse errors so Load can report them together.
type reader struct {
	lookup func(string) (string, bool)
	errs   []error
}

func (r *reader) str(key, def string) string {
	if v, ok := r.lookup(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

func (r *reader) integer(key string, def int) int {
	v := r.str(key, "")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s must be a whole number, got %q", key, v))
		return def
	}
	return n
}

func (r *reader) boolean(key string, def bool) bool {
	v := r.str(key, "")
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s must be true or false, got %q", key, v))
		return def
	}
	return b
}

func (r *reader) duration(key string, def time.Duration) time.Duration {
	v := r.str(key, "")
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s must be a duration such as 5s or 2m, got %q", key, v))
		return def
	}
	return d
}

func (r *reader) list(key string) []string {
	v := r.str(key, "")
	if v == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(v, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
