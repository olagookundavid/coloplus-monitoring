# Decisions log

Why things are the way they are. Append-only: to change a decision, add an entry that supersedes it. Read before changing architecture. Standing rules are promoted, one line each, into the relevant CLAUDE.md.

**Entry types:** *Fusewall standard* (inherited default) · *Product decision* · **Deviation from Fusewall standard** (needs the reason and the cost).

## Index
| # | Title | Type | Status |
|---|---|---|---|
| 001 | Hexagonal backend, ports and adapters | Fusewall standard | Accepted |
| 002 | Go + Gin + pgx + goose | Fusewall standard | Accepted |
| 003 | Self-hosted Postgres on shared Contabo VPS, DB + user per product | Fusewall standard | Accepted |
| 004 | Docker Compose, no Kubernetes | Fusewall standard | Accepted |
| 005 | Cloudflare proxy + Caddy + Origin Certificate | Fusewall standard | Accepted |
| 006 | CI/CD: GitHub Actions to ghcr.io to forced-command SSH deploy, health-gated | Fusewall standard | Accepted |
| 007 | Resend for email | Fusewall standard | Accepted |
| 008 | Paystack for payments, single webhook | Fusewall standard | Not applicable (see 015) |
| 009 | Cloudflare R2 for files and backups | Fusewall standard | Accepted (see 016) |
| 010 | No Redis by default | Fusewall standard | Accepted |
| 011 | Single pgxpool, env-sized; bounded injected worker pool | Fusewall standard | Accepted |
| 012 | Write-behind batching for audit/analytics only | Fusewall standard | Accepted (see 019) |
| 013 | Permission-based authorisation | Fusewall standard | Accepted |
| 014 | React + Vite + TanStack Query on Vercel | Fusewall standard | Accepted |
| 015 | No payments, no Paystack adapter | Deviation from Fusewall standard | Accepted |
| 016 | Private R2 bucket with short-lived, permission-checked download links | Deviation from Fusewall standard | Accepted |
| 017 | Coloplus shares the Contabo box and Postgres for the pilot | Product decision | Accepted |
| 018 | `pkg-common` lives in this repo as a shared folder | Product decision | Accepted |
| 019 | Audit: batched by default, transactional for approvals | Product decision | Accepted |
| 020 | Time zone: DB session pinned, domain maths in UTC | Product decision | Accepted |
| 021 | Swagger docs generated at build, not committed | Product decision | Accepted |
| 022 | Frontend folder reserved and empty | Product decision | Accepted |
| 023 | Migrations are a separate step: `api migrate` | Fusewall standard | Accepted |
| 024 | Sign-in: password + optional authenticator code, PASETO in an HttpOnly cookie | Product decision | Accepted |
| 025 | Roles map to permissions in the database; site scope is a separate check | Fusewall standard | Accepted |
| 026 | Daily report workflow, immutability and corrections | Product decision | Accepted |
| 027 | An interval must stay within one reporting day | Product decision | Accepted |
| 028 | Calculation provenance is returned on read, not yet stored | Product decision | Accepted (revisit) |
| 029 | First administrator is created from the command line | Product decision | Accepted |
| 030 | No invitation email yet: administrators set initial passwords | Product decision | Accepted (revisit) |
| 031 | Tests are minimal and aimed at critical risk | Product decision | Accepted |

## Template
### NNN: Title
- **Type:** Fusewall standard | Product decision | Deviation from Fusewall standard
- **Status:** Proposed | Accepted | Superseded by NNN
- **Context:** the forces and constraints.
- **Decision:** one or two sentences.
- **Alternatives considered:** and why not.
- **Cost of deviating** (deviations only): extra operations, loss of shared tooling.
- **Consequences:** easier / harder / must remember.
- **Revisit when:** the trigger that reopens this.

## 003: Self-hosted Postgres on the shared Contabo VPS
- **Type:** Fusewall standard · **Status:** Accepted
- **Context:** cost and control over tuning and backups; one operator; several products on one box.
- **Decision:** one Postgres 18 container, one database and one least-privilege user per product, loopback-only port, nightly offsite backups.
- **Alternatives considered:** managed Postgres (more cost, less control); an instance per product (more RAM overhead).
- **Consequences:** shared blast radius and maintenance window; connection budgets per product (Coloplus: `DB_MAX_CONNS=8`, `DB_MIN_CONNS=1`); restore drills are our job.
- **Revisit when:** a product's load or compliance needs outgrow a shared instance, or `max_connections` is regularly near the limit.

## 011: Pools and sizes
- **Type:** Fusewall standard · **Status:** Accepted
- **Decision:** one pgxpool (8 max, 1 min; lifetime 1h, idle 5m, health check 30s, connect timeout 5s). One bounded worker pool (8 workers, 64 waiting submitters, then `ErrPoolOverloaded`). Rate limit 10 requests/s per client IP with burst 30. Access token lifetime 12h.
- **Consequences:** revisit the numbers after the pilot with real traffic; they are config, not code.

## 015: No payments, no Paystack adapter
- **Type:** Deviation from Fusewall standard · **Status:** Accepted
- **Context:** Coloplus is a monitoring portal for a business customer. The PRD has no purchases or payments.
- **Decision:** the `PaymentGateway` port, Paystack adapter, webhook and payment-return page are not built. The walking skeleton is "daily power report, validation, supervisor approval, availability on the dashboard" instead of "purchase with a payment".
- **Cost of deviating:** none; fewer moving parts. The standard's Paystack rules apply if payments are ever added.
- **Revisit when:** the product takes money.

## 016: Private R2 bucket with short-lived download links
- **Type:** Deviation from Fusewall standard · **Status:** Accepted
- **Context:** the standard serves R2 reads from a public URL. The PRD (AT01, evidence attachments) requires that a shared link must not bypass site permissions, and evidence belongs to Airtel's sites.
- **Decision:** the bucket is private. Uploads are presigned. Downloads are presigned with a short lifetime and issued only after a per-site permission check. A SHA-256 checksum is stored with each file. Files are virus-scanned before Airtel staff can open them.
- **Alternatives considered:** public bucket with unguessable names (a leaked link works forever, breaks AT01).
- **Cost of deviating:** a small download endpoint and a scan step; the shared "public URL from config" helper is not used.
- **Revisit when:** never expected to loosen.

## 017: Shared Contabo box and Postgres for the pilot
- **Type:** Product decision (owner) · **Status:** Accepted
- **Context:** the PRD targets 99.5% monthly availability. Showtime shares the box and has launch-day spikes; on 2026-10-19 a firewall change took production egress down.
- **Decision:** the pilot runs on the shared box with a per-product CPU, memory and connection budget. Dozzle has no login and shows every product's logs, so no customer data goes into logs, or Dozzle is put behind a login before go-live.
- **Consequences:** a Showtime incident can affect Coloplus.
- **Revisit when:** before the security review and wider rollout. Moving to its own small VPS then is a Deviation entry.

## 018: `pkg-common` is a shared folder in this repo
- **Type:** Product decision (owner) · **Status:** Accepted
- **Context:** Showtime's `pkg-common` hardcodes the cookie name `Showtime_AccessToken` and carries Redis, queue, tailscale and message-pipe code Coloplus does not need.
- **Decision:** a trimmed `pkg-common/` module sits at the repo root, consumed through `go.work`. It has tokens, logger and auth/recover/rate-limit middleware, with the cookie name passed in as config. Other apps in this repo use the same folder. Rules: no product names, only add (never change existing behaviour).
- **Alternatives considered:** a separate versioned repo (more overhead now; still the right move when a third product needs it); copying Showtime's module as is (leaks the product name and heavy dependencies).
- **Consequences:** this copy and Showtime's now diverge. Fixes made in one are not automatic in the other.
- **Revisit when:** a second product repo needs the same code; then extract to its own versioned module.

## 019: Audit is batched by default, transactional for approvals
- **Type:** Product decision (owner) · **Status:** Accepted
- **Context:** the owner chose batching so every audit event does not hold a connection. Graceful shutdown flushes the batcher before the pool closes, so only a hard kill can lose events.
- **Decision:** `AuditService.Record` batches (20 events or 5 seconds, queue of 1,000). The accepted risk is losing up to 5 seconds of batched events on a hard kill. `RecordInTx` writes inside the business transaction and is the recommended path for approvals, corrections, publications and exports, because there the record must exist exactly when the change does. Which path each feature uses is decided when that feature is built.
- **Consequences:** `audit_log` is append-only at the database level (trigger).
- **Revisit when:** a hard-kill loss is ever observed, or an auditor asks for stronger guarantees.

## 020: Time zone
- **Type:** Product decision · **Status:** Accepted
- **Decision:** timestamps are stored UTC. Each DB session is pinned to `DB_TIMEZONE` (default `Africa/Lagos`). Day boundaries and interval arithmetic are computed in the domain layer from UTC timestamps with an injected `Clock`, so the maths is unit-testable and does not depend on the session zone.

## 021: Swagger docs generated at build
- **Type:** Product decision · **Status:** Accepted
- **Decision:** `backend/docs` is generated by `swag init` (`make swag`, the Dockerfile and CI) and gitignored, as in Showtime. The UI is served only when `SWAGGER_ENABLED=true`; the app refuses to boot in production with it on.
- **Consequences:** a fresh clone needs `make swag` once before it compiles.

## 022: Frontend folder reserved and empty
- **Type:** Product decision · **Status:** Accepted
- **Decision:** the backend is built first. `frontend/` holds only a placeholder; the front-end engineer adds the React app and its `CLAUDE.md` (the standard's section 9.3).

## 023: Migrations are a separate step
- **Type:** Fusewall standard · **Status:** Accepted
- **Decision:** `api migrate` applies pending goose migrations and exits. The deploy script runs it before starting the new image. The server never migrates itself on start.

## 024: Sign-in
- **Type:** Product decision · **Status:** Accepted
- **Context:** the PRD requires two-step verification for privileged and Airtel users, revocation, session expiry and an audit of sign-ins. The standard names PASETO and bcrypt but no TOTP library.
- **Decision:** email + password (bcrypt, cost 12, 12 to 72 characters, one letter and one number). Optional authenticator-app step using RFC 6238 TOTP written with the standard library in `pkg-common/totp` (tested against the RFC vectors), so no new dependency. TOTP seeds are stored encrypted (AES-256-GCM, `SECRETS_KEY`). The PASETO access token (12h) is set as an HttpOnly cookie named by `AUTH_COOKIE_NAME` and also returned for API clients. Every authenticated request reloads the user, so deactivation takes effect immediately. Five failed attempts lock the account for 15 minutes; sign-in also has its own per-IP limit. State-changing requests must be `application/json`, which doubles as a CSRF guard.
- **Consequences:** two-step is available but **not yet enforced**: a policy that requires it for Airtel and privileged roles, and recovery codes for lost phones, are still to build. Losing `SECRETS_KEY` disables everyone's two-step login.
- **Revisit when:** before the pilot (enforce two-step), or when Airtel SSO is agreed.

## 025: Roles, permissions and site scope
- **Type:** Fusewall standard · **Status:** Accepted
- **Decision:** code checks permissions (`reports.review`, `sites.manage` and so on), never role names. `role_permissions` in the database maps the four MVP roles (administrator, field reporter, supervisor, Airtel viewer). Site scope is separate: administrators hold `sites.all`; everyone else sees only assigned sites. An out-of-scope id answers "not found", not "forbidden", so ids cannot be probed. Administrators hold no `reports.review`, so they cannot approve.
- **Consequences:** the read-only management/auditor role and the model administrator role are later phases and arrive as new rows, not code changes.

## 026: Daily report workflow
- **Type:** Product decision · **Status:** Accepted
- **Decision:** draft, submitted, approved, rejected (returned), superseded. The approver must differ from the author (also a database CHECK). Approving a correction supersedes the report it corrects in the same transaction. An approved or superseded report, and its intervals, cannot be changed or deleted: database triggers enforce it, not only application code. Edits quote a `row_version`; a stale one is a conflict. Submitting twice is a no-op. Airtel users get a separate published view (no notes, author or review comments) of approved reports only. Approvals, returns, submissions and corrections write their audit row in the same transaction (decision 019), with before and after minutes for corrections.
- **Consequences:** Airtel keeps seeing the original until a correction is approved.

## 027: An interval stays within one reporting day
- **Type:** Product decision · **Status:** Accepted
- **Context:** the PRD says to split cross-midnight events across reporting periods without changing the original incident.
- **Decision:** for now an interval must lie inside the reporting day (Africa/Lagos), in whole minutes, and may not end in the future. The reporter enters the part after midnight in the next day's report, and the error says so. Incidents (which will hold the unsplit event) do not exist yet.
- **Revisit when:** the incident register is built; automatic splitting belongs there.

## 028: Provenance is returned, not yet stored
- **Type:** Product decision · **Status:** Accepted (revisit)
- **Decision:** the daily metrics (S, U, D, X, coverage, reported availability, confirmed served share) are pure functions in `domain`, computed on read from the approved report revision, and returned with a `calculation` block (formula `daily-availability` v1, input report id and revision, window, time zone, warnings). Only observed intervals count; declared ones are shown but ignored. Nothing is persisted as a calculation run yet.
- **Consequences:** results are reproducible because inputs are immutable, but the PRD's stored calculation runs, report snapshots and period reopening are still to build.
- **Revisit when:** monthly reports (Advanced 2) need frozen, versioned results.

## 029: First administrator from the command line
- **Type:** Product decision · **Status:** Accepted
- **Decision:** `ADMIN_PASSWORD=... api create-admin <email> <name>` creates the first administrator, reading the password from the environment so it never reaches shell history. It refuses once any administrator exists. Everyone else is created from the portal.

## 030: No invitation email yet
- **Type:** Product decision · **Status:** Accepted (revisit)
- **Decision:** administrators set an initial password when creating a user and tell them out of band. The Resend adapter, invitation and password-reset emails come next, with the Mailer port.
- **Revisit when:** the Resend adapter is built; then replace initial passwords with invitation links.

## 031: Tests are minimal and aimed at critical risk
- **Type:** Product decision (owner) · **Status:** Accepted
- **Context:** the owner wants feature delivery first and expects testers to catch obvious defects. The Fusewall standard's testing section is a ceiling, not a target.
- **Decision:** write tests only for what is critical or has already gone wrong: trust and security rules, the PRD metrics, concurrency and atomicity, shutdown behaviour. Skip mappers, simple validation, defaults and happy-path CRUD. The suite was cut from about 70 to 44 tests on 2026-10-28.
- **Consequences:** more bugs will reach manual testing; that is accepted. Anything not tested is said so in the hand-over note rather than covered later by bulk tests.
- **Revisit when:** a release goes to Airtel at scale, or defects found in testing show a pattern that a test would have stopped.

## 032: Incident register rules
- **Type:** Product decision · **Status:** Accepted (confirm with Coloplus Operations)
- **Decision:** stages are reported, acknowledged, assigned, in progress, restored, verified, closed. Restoration needs the time power was actually restored; a restoration target is a plan and never counts. Verification needs a described new observation and its time, and must come from someone other than the person who recorded restoration (DB check as well as service rule). Closing needs a root cause. A closed incident cannot change (trigger); history is append-only. Work cannot start without an owner. A field officer may record restoration; acknowledging, assigning, verifying, closing, publishing and writing to Airtel are supervisor actions.
- **Airtel visibility:** Airtel sees an incident only once a supervisor publishes it, which does not wait for the daily report to be approved (PRD: critical notices must not wait). They see title, category, severity, stage, times and the updates written for them, not the description, internal notes, owner, author or root cause. Whether to share root cause is to be agreed with Airtel.
- **Not built yet:** linking incidents to the outage intervals of a daily report, evidence attachments (needs the R2 adapter), overdue alerts by email (needs Resend), planned-maintenance exclusions.
