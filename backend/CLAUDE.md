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
- `internal/adapters/postgres`: pool, `DB.q(ctx)` (joins the tx in ctx), `TxManager`, repositories (users, sites, reports, incidents, audit), goose `Migrate`. `adapters/security`: bcrypt, PASETO issuer, TOTP, AES-GCM secret box.
- `internal/services`: Auth (login, MFA, `Authenticate`), User, Site, Report (workflow), Incident (fault register), Dashboard, Audit, Health.
- `internal/transport/http`: router, middleware chain, one error mapper (`errors.go`), handlers with swag annotations, mappers. Routes under `/api/v1`; `/healthcheck` (GET+HEAD).
- `internal/transport/jobs`: cron scheduler on 5-field specs.
- `cmd/api`: composition root and shutdown order. Subcommands: `migrate`, `create-admin`.

## API in one screen
Auth: `POST /auth/login|logout|mfa/setup|mfa/enable`, `GET /auth/me`. Users (admin): `POST/GET /users`, `GET/PATCH /users/:id`. Sites: `POST/GET /sites`, `GET/PUT /sites/:id`. Reports: `POST /sites/:id/daily-reports`, `GET /daily-reports`, `GET/PUT /daily-reports/:id`, `POST /daily-reports/:id/submit|approve|reject|corrections`. Incidents: `POST /sites/:id/incidents`, `GET /incidents[?open&overdue&status&severity&site_id&owner_id]`, `GET /incidents/summary`, `GET /incidents/:id`, `POST /incidents/:id/assign|transition|updates|publish`. Dashboard: `GET /dashboard/overview?date=`. Full detail in Swagger (`SWAGGER_ENABLED=true`, `/swagger/index.html`).

## Domain rules to keep
- Missing or overdue data is **unknown / awaiting update**, never zero, never available. Percentages are `nil` when not applicable.
- Only `observed` intervals count; `declared` ones are kept and ignored. Served and outage intervals may not overlap. Intervals stay inside one reporting day (decision 027).
- Incidents: restoration is an observed time, never a target; verification needs a new observation from someone other than the person who recorded restoration; closing needs a root cause; closed is frozen; Airtel sees only published incidents and customer updates (DECISIONS 032).
- Approved reports are immutable (DB triggers). A change is a linked correction revision. The approver is never the author.
- Airtel users get `PublishedReportDTO`, never the internal DTO. Out-of-scope ids return 404.
- Add a new permission as a migration row plus a constant in `domain/user.go`; map it to roles in `role_permissions`.

## First administrator
`ADMIN_PASSWORD=... go run ./cmd/api create-admin boss@example.com Full Name` after `make migrate`.

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
- **Resend** (`Mailer`): addresses from config only; paced at 8/s; retry on rate-limit; sent from the worker pool; a failed email never fails the business transaction; sender domain = link domain.
- **R2** (`ObjectStore`): **private bucket**, presigned upload, permission-checked short-lived presigned download (DECISIONS 016); record a SHA-256 checksum; GC behind an env flag, dry-run by default.

## Time
Stored UTC. The DB session zone is pinned from `DB_TIMEZONE` (Africa/Lagos). Day boundaries and interval maths are done in `domain` from UTC timestamps with an injected `Clock`, never with SQL date functions.

## Concurrency, errors, lifecycle
- `FOR UPDATE` in a consistent order; `SKIP LOCKED` to claim jobs; idempotency key per external reference.
- Cron: `cron.ParseStandard` (5-field); a bad spec is fatal at boot; jobs only call services.
- Shutdown: HTTP, then scheduler + jobs ctx, then batchers + worker pool, then DB.

## Security
- Authorise by permission, not role name. Rate-limit globally (`RATE_LIMIT_*`), tighter on auth. CORS allow-list, never `*`. Swagger off in production (config refuses to boot otherwise). No customer data in logs: Dozzle has no login (DECISIONS 017).
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
