# Backend guide

## Layers
| Layer | May import | Must not import |
|---|---|---|
| `domain` | stdlib | everything else |
| `ports` | `domain` | adapters, services, transport |
| `services` | `domain`, `ports`, `platform`, `errors` | pgx, gin, net/http, vendor SDKs, SQL |
| `adapters/*` | `ports`, `domain`, its driver/SDK | services, transport |
| `transport/*` | `services`, `dto`, `errors`, `pkg-common` | adapters, pgx |
| `cmd/*` | everything | none |

`depguard` (`.golangci.yml`) and `tests/unit/architecture_test.go` enforce this. A violation is a bug even if it compiles.

## Adding a feature (in this order)
migration -> `domain` type + pure rules + tests -> `ports` interface -> `adapters/postgres` + integration test -> `services` use case + fake-port tests -> `dto` -> `transport` handler + route + permission + swag annotations -> frontend.

## What exists
- `config/`: typed config, read once, validated at boot, reports every problem at once. Only place that reads env vars.
- `internal/domain`: users, roles, permissions, `Actor` (permissions + site scope), sites, daily reports, incidents and their workflow rules, interval validation, the PRD's daily metrics, `Condition` (available / outage reported / awaiting update).
- `internal/platform`: `Batcher[T]`, `WorkerPool`, `SystemClock`.
- `internal/adapters/postgres`: pool, `DB.q(ctx)` (joins the tx in ctx), `TxManager`, repositories (users, sites, reports, incidents, disputes, acceptances, notifications, user tokens, audit), goose `Migrate`. `adapters/security`: bcrypt, PASETO issuer, TOTP, AES-GCM secret box. `adapters/mail`: Resend (plain HTTP) and a log-only mailer for development.
- `internal/services`: Auth (login, MFA, `Authenticate`), User, Site, Report (workflow), Incident (fault register), Dispute (Airtel and Coloplus), ReportAlert (overdue reports), Acceptance (commissioning), SiteOverview (the site screen), Export (CSV with provenance), Notification (email outbox and dispatcher), Account (invitation and reset links), Dashboard, Audit, Health.
- `internal/transport/http`: router, middleware chain, one error mapper (`errors.go`), handlers with swag annotations, mappers. Routes under `/api/v1`; `/healthcheck` (GET+HEAD).
- `internal/transport/jobs`: cron scheduler on 5-field specs. Jobs today: `dispatch-emails` (every minute), `incident-alerts` (every 10 minutes), `report-alerts` and `acceptance-alerts` (every 30 minutes); specs come from `JOB_*_CRON`.
- `cmd/api`: composition root and shutdown order. Subcommands: `migrate`, `create-admin`, `reset-mfa <email>`.

## API in one screen
Auth: `POST /auth/login|logout|forgot-password|set-password|mfa/setup|mfa/enable|mfa/recovery-codes`, `GET /auth/me`. Users (admin): `POST/GET /users`, `GET/PATCH /users/:id`, `POST /users/:id/invite|reset-mfa`. Notifications (admin): `GET /notifications`, `POST /notifications/:id/retry`. Sites: `POST/GET /sites`, `GET/PUT /sites/:id`. Reports: `POST /sites/:id/daily-reports`, `GET /daily-reports`, `GET/PUT /daily-reports/:id`, `POST /daily-reports/:id/submit|approve|reject|corrections`. Incidents: `POST /sites/:id/incidents`, `GET /incidents[?open&overdue&status&severity&site_id&owner_id]`, `GET /incidents/summary`, `GET /incidents/:id`, `POST /incidents/:id/assign|transition|updates|publish`. Disputes: `POST /sites/:id/disputes`, `GET /disputes[?site_id&report_id&status&open]`, `GET /disputes/summary`, `GET /disputes/:id`, `POST /disputes/:id/comments|transition`. Acceptance: `POST/GET /sites/:id/acceptance`, `POST /sites/:id/acceptance/transition`, `GET /acceptances[?stage&open&overdue]`. Site screen: `GET /sites/:id/overview?days=`. Exports (`exports.create`): `GET /exports/daily-summaries.csv?from&to[&site_id]`. Audit (`audit.read`): `GET /audit[?actor_id&action&entity_type&entity_id&from&to]`. Dashboard: `GET /dashboard/overview?date=`. Full detail in Swagger (`SWAGGER_ENABLED=true`, `/swagger/index.html`).

## Domain rules to keep
- Missing or overdue data is **unknown / awaiting update**, never zero, never available. Percentages are `nil` when not applicable.
- Only `observed` intervals count; `declared` ones are kept and ignored. Served and outage intervals may not overlap. Intervals stay inside one reporting day (decision 027).
- Incidents: restoration is an observed time, never a target; verification needs a new observation from someone other than the person who recorded restoration; closing needs a root cause; closed is frozen; Airtel sees only published incidents and customer updates (DECISIONS 032).
- Disputes: Coloplus's resolution and Airtel's acknowledgement are separate acts; an upheld dispute about a report needs a later approved revision of that day; Airtel never sees internal comments or staff ids; a closed dispute is frozen (DECISIONS 034).
- Acceptance: only Airtel's own decision is acceptance, an inspection comes first, and acceptance (not Coloplus's say-so) activates a pending site (DECISIONS 036).
- Exports: a day with no approved report stays a row with no figures (never zero); every row carries revision, basis, coverage, formula version, dispute state and assumptions; each export is audited (DECISIONS 037).
- Two-step sign-in: required by `roles.mfa_required` (admin, supervisor, Airtel). A required person who has not enrolled holds **no permissions** (`Actor.MFASetupRequired`) until they enable it; never add a route that works for such an actor beyond `/auth/me` and the `mfa` setup (DECISIONS 038).
- Approved reports are immutable (DB triggers). A change is a linked correction revision. The approver is never the author.
- Airtel users get `PublishedReportDTO`, never the internal DTO. Out-of-scope ids return 404.
- Add a new permission as a migration row plus a constant in `domain/user.go`; map it to roles in `role_permissions`.

## First administrator
`ADMIN_PASSWORD=... go run ./cmd/api create-admin boss@example.com Full Name` after `make migrate`. Administrators must set up two-step sign-in at their first login (`/auth/mfa/setup`, then `/auth/mfa/enable`; with curl until the front end exists). Locked out with no administrator left: `go run ./cmd/api reset-mfa boss@example.com`.

## Sample data
`make seed` (dev only) inserts one user per role, a handful of sites, daily reports moving through their workflow, an incident, a dispute, and sites at different acceptance stages — through the real services, so it carries a real audit trail. Safe to run more than once; it looks for what already exists before creating anything. Prints each seeded login and the shared password once, at the end.

## Pooling
- pgxpool from env: `DB_MAX_CONNS` (default 8), `DB_MIN_CONNS` (1); lifetime 1h, idle 5m, health 30s, connect 5s. Shared Postgres: keep this product's share small and documented.
- No pooler today, so default protocol. A pooler in transaction mode needs `QueryExecModeSimpleProtocol` and a DECISIONS entry.
- One `http.Client` per provider adapter with timeouts. Never `http.DefaultClient`.
- Worker pool: injected, `WORKER_POOL_SIZE` (8) workers, `WORKER_POOL_QUEUE` (64) waiters, then `ErrPoolOverloaded`; `recover()` per task; drained on shutdown.

## Transactions and audit
- Services call `TxManager.WithTx(ctx, fn)`; repositories called with fn's ctx join the transaction. Never pass a `pgx.Tx` around.
- Audit: `Record` goes through the batcher (20 events or 5s, queue 1000, 250ms block then drop-and-count). `RecordInTx` writes inside the caller's transaction. Use it for approvals, corrections, publications, exports.
- `audit_log` is append-only at the database level (trigger). Do not add UPDATE/DELETE paths.

## Providers (each behind a port, added with the first feature that needs it)
- **Resend** (`Mailer`, built): plain `net/http`, no SDK; one client with a timeout; paced at `MAIL_RATE_PER_SECOND`; a single attempt per call, because retries belong to the outbox. Business code never sends: it calls `NotificationService.Enqueue` inside its transaction (DECISIONS 033). Transient failures wrap `ErrUnavailable`, refusals `ErrMailRejected`. Sender domain = link domain where possible.
- **R2** (`ObjectStore`): **private bucket**, presigned upload, permission-checked short-lived presigned download (DECISIONS 016); record a SHA-256 checksum; GC behind an env flag, dry-run by default.

## Time
Stored UTC. The DB session zone is pinned from `DB_TIMEZONE` (Africa/Lagos). Day boundaries and interval maths are done in `domain` from UTC timestamps with an injected `Clock`, never with SQL date functions.

## Concurrency, errors, lifecycle
- `FOR UPDATE` in a consistent order; `SKIP LOCKED` to claim jobs; idempotency key per external reference.
- Cron: `cron.ParseStandard` (5-field); a bad spec is fatal at boot; jobs only call services.
- Shutdown: HTTP, then scheduler + jobs ctx, then batchers + worker pool, then DB.

## Security
- Authorise by permission, not role name. Rate-limit globally (`RATE_LIMIT_*`), tighter on auth. CORS allow-list, never `*`. Swagger off in production (config refuses to boot otherwise). No customer data in logs: Dozzle has no login (DECISIONS 017).
- Invitation and reset links: random 32-byte token, only its SHA-256 stored, single use, expiring; the forgot-password answer never depends on whether the address exists. A password change ends older sessions.
- Token cookie name comes from `AUTH_COOKIE_NAME`; `pkg-common` has no product names.

## Migrations
- goose, numbered, embedded, forward-only, expand then migrate then contract; never edit an applied one; run `api migrate` before deploy.

## Tests: minimal on purpose
The owner's priority is building the product; testers will catch obvious problems. So:
- **Do test:** the trust rules (approval, self-approval, immutability, site scope, what Airtel can see), the PRD metrics and "unknown is never available", auth and lockout, concurrency and idempotency, transactions and audit atomicity, shutdown/flush behaviour, layer imports, and anything that has broken before.
- **Do not test:** getters, DTO mappers, simple field validation, config defaults, happy-path CRUD, library behaviour, anything the compiler or lint already guarantees.
- Prefer a single integration test through the HTTP stack (`tests/integration`, real Postgres) over several small ones. Fakes for ports only when a real database cannot show the failure.
- A bug found later gets one regression test. No tests for coverage's sake.
- `make test` is the unit gate; `make test/integration` needs `make db/up`. Swagger docs (`backend/docs`) are generated by `make swag` and not committed.
