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
- `internal/adapters/postgres`: pool, `DB.q(ctx)` (joins the tx in ctx), `TxManager`, repositories (users, sites, reports, incidents, disputes, acceptances, notifications, user tokens, audit), goose `Migrate`. `adapters/security`: bcrypt, PASETO issuer, TOTP, AES-GCM secret box. `adapters/mail`: Resend (plain HTTP) and a log-only mailer for development. `adapters/r2`: private file storage with hand-signed S3 links (standard library only).
- `internal/services`: Auth (login, MFA, `Authenticate`), User, Site, Report (workflow), Incident (fault register), Dispute (the customer and Coloplus), Ticket (support requests), Attachment (evidence files), Fuel (deliveries, readings, estimate), ReportAlert (overdue reports), Acceptance (commissioning), Digest (weekly email), SiteOverview (the site screen), Export (CSV with provenance), Notification (email outbox and dispatcher), Account (invitation and reset links), Dashboard, Audit, Directory (names for ids), Health.
- `internal/transport/http`: router, middleware chain, one error mapper (`errors.go`), handlers with swag annotations, mappers. Routes under `/api/v1`; `/healthcheck` (GET+HEAD).
- `internal/transport/jobs`: cron scheduler on 5-field specs. Jobs today: `dispatch-emails` (every minute), `incident-alerts` (every 10 minutes), `report-alerts` and `acceptance-alerts` (every 30 minutes), `attachment-gc` (daily; reports only unless `ATTACHMENT_GC_ENABLED`), `fuel-alerts` (every 30 minutes), `weekly-digest` (Monday 07:00; on by default, `WEEKLY_DIGEST_ENABLED=false` stops it), `retention` (daily; on by default, reports only if `RETENTION_ENABLED=false`), `anomaly-alerts` and `dispute-alerts` (every 30 minutes); specs come from `JOB_*_CRON`.
- `cmd/api`: composition root and shutdown order. Subcommands: `migrate`, `create-admin`, `reset-mfa <email>`, `logout-all`, `seed`, `seed-demo`, `check-storage`.

## API in one screen
Auth: `POST /auth/login|logout|forgot-password|set-password|mfa/setup|mfa/enable|mfa/recovery-codes`, `GET /auth/me`. Users (admin): `POST/GET /users`, `GET/PATCH /users/:id`, `POST /users/:id/invite|reset-mfa`, `POST /auth/logout-all` (signs everybody out; DECISIONS 072). Notifications (admin): `GET /notifications`, `POST /notifications/:id/retry`. Sites: `POST/GET /sites`, `GET/PUT /sites/:id`, `POST /sites/import` (CSV text in JSON; `dry_run`). Reports: `POST /sites/:id/daily-reports`, `GET /daily-reports`, `GET/PUT /daily-reports/:id`, `POST /daily-reports/:id/submit|approve|reject|corrections`. Incidents: `POST /sites/:id/incidents`, `GET /incidents[?open&overdue&status&severity&site_id&owner_id]`, `GET /incidents/summary`, `GET /incidents/:id`, `POST /incidents/:id/assign|transition|updates|publish`. Disputes: `POST /sites/:id/disputes`, `GET /disputes[?site_id&report_id&status&open]`, `GET /disputes/summary`, `GET /disputes/:id`, `POST /disputes/:id/comments|transition`. Tickets: `POST /sites/:id/tickets`, `GET /tickets[?site_id&status&category&priority&incident_id&open]`, `GET /tickets/summary|:id`, `POST /tickets/:id/comments|transition|incident`. Fuel: `POST/GET /sites/:id/fuel`, `POST /fuel/:id/void`. Attachments (owner: ticket, incident, dispute, report, acceptance): `POST /attachments` (ask for an upload link), `POST /attachments/:id/complete`, `GET /attachments?owner_type&owner_id`, `GET /attachments/:id/download`. Acceptance: `POST/GET /sites/:id/acceptance`, `POST /sites/:id/acceptance/transition`, `GET /acceptances[?stage&open&overdue]`. Site screen: `GET /sites/:id/overview?days=`. Exports (`exports.create`): `GET /exports/daily-summaries.csv?from&to[&site_id&region]`, `GET /exports/incidents.csv?from&to[&site_id&region]`. Issued exports: `GET /exports/issued`, `GET /exports/issued/:id/download`. Alerts (own only): `GET /alerts[?unread]`, `POST /alerts/:id/acknowledge`, `POST /alerts/acknowledge-all`. More exports: `GET /exports/monthly-performance.csv?month=`, `GET /exports/weekly-exceptions.csv?week_start=`, `GET /exports/commissioning-backlog.csv`. Planned maintenance: `POST /sites/:id/maintenance`, `GET /maintenance[?site_id&status&from&to]`, `POST /maintenance/:id/approve|reject|cancel`. Periods (administrators): `GET /periods`, `POST /periods/:month/close|reopen`. Calculation runs (Coloplus staff, all sites): `POST/GET /calculation-runs`, `GET /calculation-runs/:id|:id/compare`. Digest: `GET/PUT /digest/preference`. Audit (`audit.read`): `GET /audit[?actor_id&action&entity_type&entity_id&from&to]`. Dashboard: `GET /dashboard/overview?date=` or `?from&to` plus `&region` and `&quality=good|fair|poor`, `GET /dashboard/trend?from&to[&region&grain=day|week|month]`, `GET /dashboard/outages?from&to[&region&site_id]`, `GET /incidents/:id/outages`. Full detail in Swagger (`SWAGGER_ENABLED=true`, `/swagger/index.html`).

## Domain rules to keep
- A site is **available by default** (DECISIONS 051): time nobody reported as an outage counts as available but is always **assumed, never confirmed** (`assumed_available_minutes`; the confirmed share counts only reported time), and the days without a report are counted beside every figure. A site with `unreported_time = unknown` keeps the strict reading, where unreported time is neither up nor down and its figure is `nil`. Overdue data still shows as **awaiting update**, never zero. Marking an outage needs a reason (`outage_reason`) to submit. Percentages are `nil` when not applicable.
- Only `observed` intervals count; `declared` ones are kept and ignored. Served and outage intervals may not overlap. Intervals stay inside one reporting day (decision 027).
- Incidents: restoration is an observed time, never a target; verification needs a new observation from someone other than the person who recorded restoration; closing needs a root cause; closed is frozen; the customer sees only published incidents and customer updates (DECISIONS 032).
- Disputes: Coloplus's resolution and the customer's acknowledgement are separate acts; an upheld dispute about a report needs a later approved revision of that day; the customer never sees internal comments or staff ids; a closed dispute is frozen (DECISIONS 034).
- Tickets: a request from the customer, not a fault record. The customer never sees internal notes, staff ids, or a linked incident until it is published to them; the priority is the requester's wish, not an incident severity; the contact address typed into a ticket is never emailed (only the requester's account is); a retried request with the same `client_key` returns the same ticket (DECISIONS 046).
- Exports are issued, not just sent: each CSV is entered in an immutable register with its SHA-256 (`X-Export-Id`, `X-Content-SHA256`) and its exact bytes are kept when storage is on; a failed copy never blocks the export, a failed fingerprint does (DECISIONS 058).
- Site import is all or nothing and blank cells never overwrite on an update; a stored calculation run can never be edited, a correction shows only when it is compared with now (DECISIONS 059, 061). The weekly digest shows a person only what their own dashboard would, with assumed time named as assumed (058).
- Figures take two facts from outside a day's own report (DECISIONS 062): approved planned maintenance leaves the eligible time, and an outage still going at the end of a day is assumed to continue (one day, `assumed_outage_minutes`, never confirmed, not coverage). Every service that shows a figure builds it through `Adjustments`, so a figure never differs between screens: call `adjust.For(...)` and `summarise`, never `domain.Summarise` directly for a figure that is shown. A maintenance window needs notice, a monthly cap and a second person's approval. A closed month takes no report, correction or approval until an administrator reopens it with a reason (062). The SLA status always has the confirmed-only reading beside it (061).
- Jobs that must look at every site page through all of them (`listAllActiveSites`); a job that read only the first page would silently skip the rest of a large portfolio.
- Alerts: every alert emailed to a person is also kept for them in the alert centre until they acknowledge it; a person sees and clears only their own; account emails (invitations, resets) never come here (DECISIONS 056).
- Fuel: the tank level is always an **estimate** (basis `estimate`), worked out from entries and the site's agreed litres per hour; never shown without an agreed rate; a day with no approved report is unknown fuel use, not zero, so the level is flagged provisional and is an upper bound; entries are never edited, only voided with a reason; notes and who recorded are Coloplus-only (DECISIONS 050). A tank reading is compared with the estimate before it and flagged, never corrected; low-fuel emails go once per episode to those who record fuel, never to the customer (DECISIONS 052).
- Attachments: the file never passes through the API (upload link, direct PUT, confirm). Who may attach or see a file follows the ticket, incident, dispute or acceptance it hangs on; a Coloplus file is internal unless shared, and the customer never sees an internal one or any file of an unpublished incident; a download link lasts 2 minutes and is audited; a stored file that is not the declared size and SHA-256 is discarded; the storage key is random, never the file name (DECISIONS 047).
- Acceptance: only the customer's own decision is acceptance, an inspection comes first, and acceptance (not Coloplus's say-so) activates a pending site (DECISIONS 036).
- The customer sees a site's responsible team, never the person working an incident (DECISIONS 044).
- Data quality is built only from what is on record (reported days, late reports, corrections, unconfirmed share); the level is a summary of those numbers, with proposed thresholds (DECISIONS 052).
- Dashboard answers are shared between identical requests and kept `DASHBOARD_CACHE_TTL` (10 s); the key is the exact site scope, so keep the permission check before the lookup (DECISIONS 048). Speed test: `go test -tags perf -run TestPerf ./tests/integration/` against an empty scratch database.
- Dashboard: figures over a period are summed minutes (never an average of daily percentages); an unreported day is unknown and a trend day nobody reported is null; `current_source` is named only while the report is current and the last interval was served (DECISIONS 043).
- Exports: a day with no approved report stays a row with no figures (never zero); every row carries revision, basis, coverage, formula version, dispute state and assumptions; each export is audited (DECISIONS 037). The incident register shows the customer published incidents only and leaves duration blank until restoration is recorded (DECISIONS 044).
- Two-step sign-in: required by `roles.mfa_required` (admin, supervisor, viewer, reviewer). A required person who has not enrolled holds **no permissions** (`Actor.MFASetupRequired`) until they enable it; never add a route that works for such an actor beyond `/auth/me` and the `mfa` setup (DECISIONS 038).
- Approved reports are immutable (DB triggers). A change is a linked correction revision. The approver is never the author.
- Customer users (the `viewer` and `reviewer` roles) get `PublishedReportDTO`, never the internal DTO. Out-of-scope ids return 404.
- Screens show names, never ids (DECISIONS 070): a response object that carries a user or site id gets a companion `..._name` field tagged `names:"SiteID:site"` (or `:user`), and handlers answer through `a.respondNamed`, which fills every name in one batch query per kind. A name is only filled when its id is present, so what a mapper hides from the customer stays hidden. Audit events are named in `audit_handlers.go` (`DirectoryService.EntityLabels`; a new audit entity type gets a line in `labelQueries` in `adapters/postgres/directory.go`).
- Add a new permission as a migration row plus a constant in `domain/user.go`; map it to roles in `role_permissions`.

## First administrator
`ADMIN_PASSWORD=... go run ./cmd/api create-admin boss@example.com Full Name` after `make migrate`. Administrators must set up two-step sign-in at their first login (`/auth/mfa/setup`, then `/auth/mfa/enable`; with curl until the front end exists). Locked out with no administrator left: `go run ./cmd/api reset-mfa boss@example.com`.

## Sample data
`make seed` (dev only) inserts two users per role (six roles), six sites (two with an agreed fuel rate), and every other table spread across the states it can be in: daily reports from draft through rejected and superseded; incidents across all seven statuses plus two incident corrections (approved and rejected) and a published one; disputes across all six statuses; support tickets open, in progress and resolved; site acceptance across accepted (which activates the site), requested, built, ready for inspection, rejected and on hold, plus one site suspended outright; planned maintenance requested, approved, rejected and cancelled; fuel deliveries, a tank reading (so a variance can be worked out) and a voided entry; a stored calculation run; and a closed reporting month plus a closed-then-reopened one. It goes through the real services, so it carries a real audit trail, and every alert-generating action also lands in the in-app alert centre for its recipients. Safe to run more than once: every step looks for what already exists and, for incidents (append-only, so a partially-finished one from an earlier failed run can't be deleted and retried) resumes from whatever status it is already at rather than skipping it outright. One email is written directly into the outbox as already `failed`, since the development log mailer always succeeds and the admin email log needs a real example to retry. Not seeded: evidence attachments, since local development has no `R2_*` configured and the upload endpoints answer 503 without it. Prints each seeded login and the shared password once, at the end.

`api seed-demo --reporter=<email> --supervisors=<email>,<email> --reviewer=<email>` (DECISIONS 073) is the one to run on a server database: it only attaches sample history (past days on the three TEST-LAG sites, plus incidents, a dispute and a ticket of each kind, maintenance) to people who are already invited, sets no password, queues no email and leaves today empty. It stops with a plain message if someone is not invited, has the wrong role or is not assigned to the three sites. It shares its wiring with `seed` through `newSeeder`; never point `seed` itself at a database that will be kept.

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
- **R2** (`ObjectStore`, built): **private bucket**, presigned upload, permission-checked short-lived presigned download (DECISIONS 016, 047); a SHA-256 checksum is signed into the upload and recorded; GC behind `ATTACHMENT_GC_ENABLED`, dry-run by default. Optional: without `R2_*` the evidence endpoints answer 503 and everything else works. Run `go run ./cmd/api check-storage` after setting the keys.

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
- **Do test:** the trust rules (approval, self-approval, immutability, site scope, what the customer can see), the PRD metrics and that assumed time is never shown as confirmed, auth and lockout, concurrency and idempotency, transactions and audit atomicity, shutdown/flush behaviour, layer imports, and anything that has broken before.
- **Do not test:** getters, DTO mappers, simple field validation, config defaults, happy-path CRUD, library behaviour, anything the compiler or lint already guarantees.
- Prefer a single integration test through the HTTP stack (`tests/integration`, real Postgres) over several small ones. Fakes for ports only when a real database cannot show the failure.
- A bug found later gets one regression test. No tests for coverage's sake.
- `make test` is the unit gate; `make test/integration` needs `make db/up`. Swagger docs (`backend/docs`) are generated by `make swag` and not committed; the contract the front end reads is `docs/api/swagger.json` (`make openapi`), committed: refresh it in any change that touches routes or DTOs.
