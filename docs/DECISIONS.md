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
| 032 | Incident register rules | Product decision | Accepted (confirm with Operations) |
| 033 | Email through an outbox; Resend over plain HTTP; invitations and reset links | Architecture | Accepted |
| 034 | Disputes: two parties, two acts, a real correction | Product decision | Accepted (confirm with Airtel) |
| 035 | Audit search and the overdue-report alert | Product decision | Accepted |
| 036 | Commissioning and acceptance; the site overview | Product decision | Accepted (confirm the KPI rule with Airtel) |
| 037 | CSV exports with provenance, no new dependency | Product decision | Accepted |
| 038 | Two-step sign-in enforced by role, with recovery codes | Product decision | Accepted (confirm the role list) |
| 039 | Frontend conventions: Fusewall defaults + Coloplus trust/provenance rules, provisional brand tokens | Product decision | Accepted |
| 040 | Custom toast component instead of react-hot-toast | Deviation from Fusewall standard | Accepted |
| 041 | Joi for client-side auth-form validation | Product decision | Accepted |

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
- **Consequences:** (enforcement and recovery codes were added later, see 038) two-step was available but **not enforced**: a policy that requires it for Airtel and privileged roles, and recovery codes for lost phones, are still to build. Losing `SECRETS_KEY` disables everyone's two-step login.
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
- **Type:** Product decision · **Status:** Superseded by 033
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

## 033: Email through an outbox; Resend over plain HTTP; invitations and reset links
- **Type:** Architecture · **Status:** Accepted
- **Decision:** every email is a row in `notifications`, written in the same transaction as the change that caused it, and sent afterwards by a dispatcher job (every minute). So a Resend outage never fails or blocks a business action, nothing is lost, and no third-party call happens with a transaction open. A dedupe key makes queueing idempotent; claiming uses `FOR UPDATE SKIP LOCKED` with a lease, so a crash or a second instance does not double-send; Resend also gets the dedupe key as its idempotency key. Transient failures retry after 1m, 5m, 30m, 2h; a refusal or the fifth failure marks the email `failed`, which administrators can list and retry (`GET /notifications`, `POST /notifications/:id/retry`). The message text is cleared once sent and never returned by the API, because it can hold a one-time link.
- **Provider:** Resend is called with the standard library (`net/http`, one client, timeout, paced at `MAIL_RATE_PER_SECOND`), so no SDK dependency was added. `MAIL_PROVIDER=log` prints emails, links included, for local development and is refused in production. The sender domain is still to be verified in Resend; until then production cannot start with `resend` configured without `RESEND_API_KEY` and `MAIL_FROM`.
- **Accounts:** a user created without a password is invited by email (72 h, single use); `POST /auth/forgot-password` mails a reset link (1 h, single use, never reveals whether an address has an account, one per 5 minutes per person); `POST /auth/set-password` redeems either. Only a hash of the token is stored. Choosing or resetting a password ends every earlier session (`users.password_changed_at` against the token's issue time, 2 s clock tolerance). Setting a password by hand (admin) is still allowed.
- **Alerts built on it:** an overdue promised incident update emails the owner and the site's supervisors once per missed promise (every 10 minutes check); publishing a critical incident emails the customer's users on that site at once, with only published facts.
- **Not built yet:** repeat reminders on a schedule, daily digests, overdue-report alerts, per-user alert preferences, an alert centre in the app, bounce handling from Resend webhooks.
- **Needs from outside:** a verified Resend sending domain, and the front end's base URL and routes (`/accept-invite`, `/reset-password`, `/incidents/:id`).

## 034: Disputes: two parties, two acts, a real correction
- **Type:** Product decision · **Status:** Accepted (confirm with Airtel's operational reviewer)
- **Decision:** a new role, **Airtel reviewer** (`airtel_reviewer`), is an Airtel viewer who can also raise disputes, comment, withdraw, reopen and acknowledge. Three permissions: `disputes.read` (both sides and administrators), `disputes.raise` (Airtel reviewer), `disputes.manage` (supervisor). Which side a person speaks for comes from those permissions, never from a role name.
- **What can be disputed:** a site, a period, a published report, or one interval of it. Only an approved report can be disputed, since that is all Airtel sees. The same thing cannot be disputed twice while the first is open (unique index; 409), so a double-click or a retry lands on the existing one.
- **States:** open, under review, awaiting evidence, agreed correction, resolved, closed. Coloplus takes it (open to under review), asks for evidence, agrees a correction, or resolves. Airtel withdraws (closed), supplies evidence or reopens a resolution (back to under review, with a reason), or acknowledges a resolution (closed). An Airtel comment on a dispute waiting for their evidence hands it back to Coloplus. Repeating a step changes nothing.
- **Two acts, kept apart:** Coloplus's decision (`resolved`, with an outcome and who) and Airtel's agreement (`acknowledged_at`) are separate columns and separate steps. A resolved dispute is not closed until Airtel acknowledges it; a withdrawn one is closed with no agreement at all.
- **Upheld means a real correction:** an upheld dispute must pass through "agreed correction" and, if it was about a report, name a later approved revision of the same site and day. The report workflow does the correcting (decision 026); the dispute only points at it. A dispute never changes the operational record.
- **Visibility:** both parties' comments are kept and shown to both. Coloplus can mark a comment internal; Airtel never sees it, nor staff ids (owner, resolver, authors), and internal notes are never emailed. Reports show `open_disputes` so a disputed figure always says so. Each public event emails the other side, scoped to the site, through the outbox (decision 033).
- **Frozen:** a closed dispute cannot change, and the history is append-only, both by trigger.
- **Not built yet:** evidence attachments (needs the R2 adapter; until then evidence is described in comments), dispute state in exports (no exports yet), overdue-dispute alerts (needs an agreed response time), a read-only management/auditor role.

## 035: Audit search and the overdue-report alert
- **Type:** Product decision · **Status:** Accepted
- **Audit search:** `GET /audit` filters by actor, exact action, entity type and id, and a time range, newest first, paged. It needs `audit.read`, which administrators hold and which is separate from any editing right. A reader without all-site access sees only events whose metadata carries one of their sites, so user and sign-in events (no site) stay with administrators. Reading the log is not itself logged yet.
- **Overdue-report alert:** every 30 minutes a job emails each active site's reporters and supervisors when the site is "awaiting update" by the same rule the dashboard uses (`domain.Condition`), once per site per day while it stays late. A site younger than its freshness window with no report is not nagged. The email says the data is stale or missing and that it does not mean the power is out (PRD section 10). Airtel is not emailed.
- **Not built yet:** repeat intervals other than daily, an in-app alert centre, digests, acknowledgement of alerts.

## 036: Commissioning and acceptance; the site overview
- **Type:** Product decision · **Status:** Accepted (confirm the KPI inclusion rule with Airtel)
- **Decision:** one acceptance process per site, with stages requested, built, ready for inspection, inspected, accepted, rejected, on hold. Coloplus (`acceptance.manage`, supervisors) requests, builds, and declares the site ready by submitting a dossier. Airtel (`acceptance.review`, the reviewer) inspects, then accepts or rejects with a reason. Either side can put it on hold with a reason and resume it to the stage it left. Nothing Coloplus does counts as Airtel's acceptance, and a site cannot be accepted without being inspected first. Accepted is final (trigger). History is append-only. Age since "ready for inspection" is reported on its own (`ready_age_hours`), apart from operational performance.
- **The KPI rule (a proposed default, not an agreed one):** when Airtel accepts, a site still *pending acceptance* becomes *active* in the same transaction, which is what brings it into the service KPIs; a suspended site is left alone. Until that site is accepted it stays out of the KPIs and shows separately, as the dashboard already did. Past reports are not re-counted. This is the PRD's open decision "acceptance inclusion" and must be confirmed.
- **Alerts:** each stage change emails the other party (not internal rework). A missed due time emails whoever the step waits on, and Coloplus's supervisors, once per missed due time (`acceptance-alerts`, every 30 minutes). A hold stops the clock.
- **Site overview:** `GET /sites/:id/overview?days=` returns the profile, condition and as-of time, the last days (a day with no approved report is a visible gap, never zero or available), open incidents, open disputes and the acceptance process in one call. Each section follows the same permission and scope rules as its own endpoint: Airtel sees published incidents only, and an out-of-scope site is a 404.
- **Not built yet:** documents on the dossier (needs R2), an escalation path beyond the second email, per-milestone due dates.

## 037: CSV exports with provenance, no new dependency
- **Type:** Product decision · **Status:** Accepted
- **Decision:** `GET /exports/daily-summaries.csv?from&to[&site_id]` gives one row per site and reporting day (approved reports only), using the standard library only. It needs its own right, `exports.create` (administrator, supervisor, Airtel viewer and reviewer; not the reporter), apart from reading the same figures on screen. At most 92 days and 100,000 rows per file; a bigger ask is refused with a message, not truncated.
- **Provenance on every row:** site ids and codes, date, time zone, service window, report id and revision, approval time, whether it is a correction, minutes (eligible, served, outage, unknown) and percentages (coverage, reported availability, confirmed served share), provisional flag, basis, declared intervals ignored, open and total disputes, formula and version, generation time and the assumptions in words. Repeating them per row is deliberate: a row pasted into another sheet keeps its meaning.
- **Gaps stay gaps:** a day with no approved report is still a row, marked `no_approved_report`, with eligible and unknown minutes (the whole window is unknown) and every served, outage and percentage cell blank, never zero (PRD trust rule). Dispute state counts disputes against any revision of that day, plus period-only disputes covering it, so a correction does not hide the dispute behind it.
- **Audit:** each export writes `export.daily_summaries` in a transaction, with range, site count and row count. Site scope comes from the same actor as every other read; an out-of-scope `site_id` is a 404.
- **Safe to open:** text cells that start with `=`, `+`, `-` or `@` get a leading apostrophe so a spreadsheet never runs them as a formula; a byte order mark makes Excel read accents correctly.
- **Not built yet:** XLSX and PDF (new dependencies, waiting for a yes), the incident register, weekly and monthly packs, immutable stored snapshots, asynchronous generation for large files. The file is built in memory, which the row cap makes safe for now.

## 038: Two-step sign-in enforced by role, with recovery codes
- **Type:** Product decision · **Status:** Accepted (confirm the role list with Airtel). Completes the "not yet enforced" consequence of 024.
- **Decision:** the PRD asks for MFA for privileged users and Airtel access. A role flag, `roles.mfa_required`, marks administrator, supervisor, Airtel viewer and Airtel reviewer; reporters are not required (they can still turn it on). Code asks the flag, never a role name.
- **Enforcement by construction:** a person whose role requires it and who has not turned it on signs in (so they can enrol), but `Authenticate` gives them an actor with **no permissions** and `MFASetupRequired`. Every service already refuses an actor without the right permission, so nothing else needs a route allow-list; only `/auth/me`, `/auth/mfa/setup`, `/auth/mfa/enable` and logout work. The same session becomes fully useful the moment they enable it. A reset puts them back in this state at once, even before older sessions expire.
- **Recovery codes:** ten codes of 12 characters (60 bits, no look-alike letters), shown once when two-step is enabled, only SHA-256 hashes stored, each spends once (an atomic update, safe under concurrent requests). A code is accepted in the same field as the authenticator code, in any case and with or without dashes. A wrong code, of either kind, counts towards the lockout. Using one is audited and emailed to its owner in the same transaction. Making a new set (`POST /auth/mfa/recovery-codes`) needs a current authenticator code and retires the old set.
- **Lost phone and codes:** `POST /users/:id/reset-mfa` (users.manage) turns it off, deletes the codes, ends older sessions and emails the person; they enrol again at next sign-in. An administrator cannot reset their own, so a stolen admin session cannot remove the second step. If no administrator is left, `api reset-mfa <email>` does the same from the server console (audited, `via: console`).
- **Contract change for the front end:** `POST /auth/mfa/enable` now answers 200 with `recovery_codes` (it was 204). `/auth/me` and the login response carry `mfa_setup_required`; `/auth/me` also gives `recovery_codes_left`.
- **Known gaps:** a TOTP code can be replayed within its 30-second window (no used-step memory yet); the "ends older sessions" rule keeps the existing 2-second clock-skew tolerance; recovery codes are hashed with plain SHA-256 (adequate for 60 random bits behind a database that is not public, but a keyed hash would be stronger). Airtel SSO would replace all of this (024's revisit trigger).

## 039: Frontend conventions and provisional brand tokens
- **Type:** Product decision · **Status:** Accepted
- **Context:** `frontend/CLAUDE.md` written before any app code (022), adapting the sibling Fusewall product Showtime's frontend conventions for consistency across the owner's projects, plus PRD §5/§7/§10 trust, provenance and export requirements, plus a stakeholder HTML mockup (provisional, no official Coloplus brand kit yet).
- **Decision:** Keep the Fusewall frontend defaults unchanged (Heroicons, DataTable on TanStack Table, no-card tables, round corners, TanStack Query, ConfirmDialog, barrel exports). Add Coloplus-specific rules with no Showtime equivalent: status/trust language (no fabricated availability, text+icon with color, visible gaps), basis + review-state badges with a "how calculated" drill-down, export disclosure panels, and a row-navigation-vs-`RowActions` split driven by `usePermissions()`. One React app, two permission-gated lazy route groups (`pages/customer/`, `pages/ops/`) rather than two deployments. Brand palette and corner radius taken provisionally from the mockup (teal/mint primary, amber warning, red danger, round ~8-12px corners, Airtel red reserved for their co-brand logo only), wired through a Tailwind v4 `@theme` token layer so only tokens, never components, change when a real kit lands.
- **Alternatives considered:** two separate frontend apps/deployments for the two audiences (more isolation, but duplicates `DataTable`/`ConfirmDialog`/dashboard chrome/domain types and the Fusewall single-Vercel-project hosting default, for a split Showtime already proves unnecessary with its own admin/team-head/player-portal/seller areas in one app). Staying abstract on brand tokens until an official kit exists (rejected: leaves no concrete default to build against now).
- **Consequences:** a brand-kit swap later touches only the `@theme` block, not components. A second telco customer, if ever onboarded, needs the customer name/co-brand logo parametrized rather than hardcoded to Airtel — not needed for the pilot, flagged here so it isn't forgotten.
- **Revisit when:** an official Coloplus/Fusewall brand kit arrives, or a second Airtel-equivalent customer is onboarded.

## 040: Custom toast component instead of react-hot-toast
- **Type:** Deviation from Fusewall standard · **Status:** Accepted
- **Context:** `react-hot-toast` is named in the Fusewall architecture standard's frontend default stack and was carried into `frontend/CLAUDE.md` §2 when that file was first written (039). Building the auth screens, the owner chose a custom in-house toast component instead.
- **Decision:** toasts are implemented as `src/utils/toast.ts` (a small subscribable store, no external dependency) + `src/components/ui/Toast.tsx`/`ToastViewport.tsx`, mounted once in `main.tsx`. Call-site shape (`toast.success(message)` etc.) matches what `react-hot-toast` would have offered, so no page-level code reads differently because of this choice.
- **Alternatives considered:** `react-hot-toast` (the Fusewall default; rejected by the owner for this product).
- **Cost of deviating:** one more small UI primitive to maintain in this codebase instead of an external package; no loss of capability for what this product needs (tone-based toasts, auto-dismiss, dismissible).
- **Consequences:** `frontend/CLAUDE.md` §2 updated in the same change to name the custom helper instead of `react-hot-toast`.
- **Revisit when:** never expected to revert; if a future need (e.g. promise-based toast chaining, complex stacking) outgrows the in-house version, re-evaluate then.

## 041: Joi for client-side auth-form validation
- **Type:** Product decision · **Status:** Accepted
- **Context:** the Fusewall architecture standard's frontend default stack names no validation library at all. Building the three auth forms (Login, Forgot password, Accept invite), the owner chose Joi over a zero-dependency hand-rolled alternative.
- **Decision:** `src/utils/authValidation.ts` defines one Joi schema per form (`loginSchema`, `forgotPasswordSchema`, `acceptInviteSchema`) plus a `getFieldErrors()` adapter converting Joi's `ValidationError` into the `Record<string, string>` field-error map the shared `Field` component expects.
- **Alternatives considered:** hand-rolled validator functions (zero dependencies, originally recommended for three small forms); `zod`/`yup` (lighter-weight schema libraries more common in browser bundles — not chosen).
- **Consequences:** Joi is heavier than `zod`/`yup` in bundle size (built for server/Hapi use originally, not browser-first) — noted, not a blocker for a pilot-scale auth flow. Future larger forms (the ops portal's entry/approval/formula screens) can reuse the same Joi pattern rather than re-deciding a validation library from scratch.
- **Revisit when:** bundle size becomes a measured problem, or a future form's needs (e.g. deeply dynamic field arrays) outgrow what's comfortable in Joi.
