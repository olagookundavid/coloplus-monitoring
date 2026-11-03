# Coloplus MVP checklist

Backend status against the PRD (v1.1, 24 Oct 2026), section 12 "MVP build" and the acceptance tests AT01 to AT11. Updated 3 Nov 2026, after the go-live defaults (DECISIONS 068). Update this file in the same change that finishes an item.

- `[x]` built, lint clean, and the risky parts have tests
- `[~]` built in the recommended form; the rule or number still needs the customer's or the owner's manager's confirmation, or part of it is left out (the line says which)
- `[ ]` not done. A tag says why: **partial** (some of it exists), **needs input** (waiting on a decision, credential or domain from outside), **new dependency** (needs your yes before I start)
- The PRD's "recommended first release" (section 13) is marked **FR** where an item belongs to it: site condition, a complete fault register, downtime, accountability and recovery updates, with energy and fuel estimates only where inputs are adequate.

## 0. Readiness for first release (3 Nov 2026)

The backend is complete for the PRD's recommended first release (site condition, fault register, downtime, accountability and recovery updates). Every open question that only needed a setting now carries the recommended answer (DECISIONS 068). What stands between this and real use is not backend code:

1. **Resend sending domain verified and the live key in the server's `.env`** (A5, A6, B5). Without it no invitation or alert email leaves the system.
2. **Server `.env` edited** to take the new defaults (`WEEKLY_DIGEST_ENABLED=true`, `SLA_TARGET_PERCENT=99.5`, `RETENTION_ENABLED=true`), if it was copied from the old example.
3. **Backups of `coloplus` to R2 (B7) and one restore run (E6)**, then UptimeRobot (E3).
4. **Real sites loaded** (`POST /sites/import`) and users invited.
5. **The front end** covering the screens in `docs/FRONTEND.md`.

Not in the first release by decision: formula versions, estimated energy, XLSX/PDF, provisional view of unapproved entries.

## 1. Foundations

- [x] Hexagonal Go API, constructor injection, layer rules enforced by lint and an architecture test
- [x] Config read once, validated at boot, production guards (Swagger off, CORS list, rate limit, real mailer, https base URL)
- [x] One pgxpool, one HTTP client per provider, bounded worker pool, graceful shutdown order
- [x] Forward-only goose migrations (7), `api migrate` as a separate step
- [x] Cron scheduler with four jobs (email dispatch, incident alerts, report alerts, acceptance alerts)
- [x] Append-only audit log (database trigger), batched for low-risk events, written in the same transaction for approvals, corrections, publications, disputes
- [x] Rate limiting (tighter on sign-in and recovery), CORS allow-list, security headers, JSON-only writes (CSRF guard)
- [x] Swagger generated at build (`make swag`); `make openapi` writes the committed contract `docs/api/swagger.json` for the front end (Swagger 2.0)
- [x] Site has a responsible team (a team name, shown to Airtel; never a person)
- [x] Dockerfile and a CI workflow; CI is green on GitHub (lint, unit, integration against Postgres, build, deploy)
- [x] Production Docker Compose, Caddy and Cloudflare are live at `api.remotems.com` (front end `www.remotems.com`)
- [x] Deploy pipeline: a push to main tests, builds to ghcr.io and deploys over a forced-command SSH key, running migrations first; a docs-only or front-end-only push does not deploy
- [ ] Backups to R2 and a restore exercise (AT11): step B7 in `deploy/CHECKLIST.md` (extend the server's backup script to dump `coloplus`) is the owner's to do on the server
- [ ] UptimeRobot on `/healthcheck`: step E3 in `deploy/CHECKLIST.md`, the owner's to do

## 2. People and access (PRD 4)

- [x] Email and password sign-in, lockout after 5 failures, session in an HttpOnly cookie, deactivation takes effect on the next request
- [x] Roles map to permissions in the database: administrator, field reporter, supervisor, Airtel viewer, **Airtel reviewer** (new)
- [x] Site scope on every endpoint; out-of-scope ids answer 404 so ids cannot be probed
- [x] User administration and site assignments
- [x] Invitation and password-reset links by email (single use, expiring, a password change ends older sessions)
- [x] Authenticator-app (TOTP) setup and use, secrets encrypted at rest
- [x] **Enforced** two-step sign-in for administrators, supervisors and all Airtel users: until enrolled they hold no permissions (DECISIONS 038). Confirm the role list with Airtel
- [x] Recovery codes for a lost phone (one-time, hashed, emailed on use), regenerate, administrator or console reset of a lost second step
- [ ] Model administrator role (drafts formulas; separate approver). Arrives with the formula engine
- [x] Management / auditor role (`auditor`: sites, reports and audit log within assigned sites, nothing editable, two-step sign-in required; DECISIONS 045)

## 3. Sites (PRD 6)

- [x] Site master record: immutable id, Airtel and Coloplus codes (duplicates refused), name, address, region and state, type, installed power sources, service window, freshness threshold, status (pending acceptance, active, suspended), commissioning date, coordinates
- [x] Site directory with search by either code and name, filters on state, region, status
- [ ] Asset capacities, reporting cadence, load profile, formula profile, effective dates on the site record. **partial**
- [x] Directory filters on power source, acceptance stage and assigned person (staff only)
- [x] Data-quality filter: `GET /dashboard/overview?quality=good|fair|poor` (DECISIONS 067), with the proposed thresholds

## 4. Daily reporting and approval (PRD 6, 9)

- [x] Draft, submit, approve or return; the approver is never the author (service rule and database check)
- [x] Approved reports are immutable (triggers); a correction is a linked revision with a reason, and the original stays visible to Airtel until the correction is approved
- [x] Edits quote a version; a concurrent edit is a visible conflict
- [x] Interval validation: whole minutes, inside the reporting day (Africa/Lagos), no future, no served/outage overlap, a source for served time
- [x] Observed versus declared intervals; only observed counts
- [x] Airtel sees a published view of approved reports only (no notes, authors, review comments)
- [x] Explanation required for late submissions (a correction already requires its reason) (DECISIONS 049)
- [~] Events that cross midnight: an outage still going at the end of a day is assumed to continue into the next until it reports otherwise, shown as assumed outage (DECISIONS 062; switch `MIDNIGHT_CARRY_OVER`). Automatic splitting of one entry into two days' reports is not built, since it writes into another day's report; the reporter still splits by hand
- [x] CSV site import: dry run, row-by-row errors, duplicate detection on the Airtel code, all-or-nothing commit, safe to repeat (`POST /sites/import`, DECISIONS 059). XLSX and importing reports or users not built; XLSX **needs a new dependency**
- [x] Source hours on the daily summary (`source_minutes`, in the API and the CSV export)
- [x] Planned maintenance on the daily summary (`excluded_maintenance_minutes`); open issues are on the incident register
- [x] Fuel the manual way: per-site agreed litres per hour and tank size, engineers record deliveries and tank readings, the level is estimated from reported generator hours and labelled an estimate (DECISIONS 050)
- [ ] Load and generator-asset inputs beyond runtime hours (section 6 tables)
- [x] Closing and reopening a month: administrators only, after a grace period, reopening needs a reason, closing stores the month's run (`/periods`, DECISIONS 064). The rule is the recommended one and awaits confirmation
- [ ] Provisional view of unapproved entries (only if Airtel and Coloplus jointly enable it). **needs input**

## 5. Calculations (PRD 7, 8)

- [x] Daily metrics: eligible minutes, served, outage, unknown, coverage, reported availability, confirmed served share. Missing data is unknown, never available, never zero
- [x] Portfolio availability pooled by eligible minutes, not an average of percentages
- [x] "How calculated" payload: formula and version, input report and revision, window, time zone, warnings, plain-language explanation
- [x] Calculation runs stored with formula version and a fingerprint of the reports behind them; comparison with today (`/calculation-runs`, DECISIONS 061). Month-end automation and queued/failed run states not built (runs are computed in the request)
- [ ] Formula versions, parameter sets, draft, test, independent approval, effective-dated activation (AT07)
- [ ] Source runtime; estimated energy; modelled fuel use; fuel balance and variance; battery autonomy (AT02, AT08). Each labelled as an estimate
- [x] Weekly and monthly aggregation: `GET /dashboard/trend?grain=week|month`, pooled by eligible minutes (DECISIONS 055)
- [x] Planned-maintenance exclusions, with notice, a cap and a second person's approval (DECISIONS 062). Awaits confirmation of the rules and whether the customer must agree
- [~] SLA status (met, at risk, missed) beside the confirmed-only reading, target 99.5% by default (`SLA_TARGET_PERCENT`, DECISIONS 063, 068). The target and the treatment of assumed time await confirmation

## 6. Portfolio and site screens, as API (PRD 5)

- [x] Portfolio overview for a day: active, pending and suspended counts, sites reported, not reported, awaiting update, outage reported, pooled availability with coverage, per-site condition (available, outage reported, awaiting update) **FR**
- [x] An overdue or missing entry shows "Awaiting update", never as current. Under the owner's default-available rule its time counts as available but is shown as assumed, never confirmed (AT03, DECISIONS 051)
- [x] Site detail in one call (`GET /sites/:id/overview`): profile, condition, last days with visible gaps, open incidents, disputes, acceptance
- [x] Period dashboards: `GET /dashboard/overview?from&to&region` pools the days by eligible minutes, `GET /dashboard/trend` gives one point per day with gaps as null (DECISIONS 043)
- [x] Stale count on the dashboard (`sites_awaiting_update`, `sites_not_reported`)
- [x] Data-quality indicator on the dashboard (reported days, late reports, corrections, unconfirmed share; level good/fair/poor with proposed thresholds, DECISIONS 052)
- [~] Data-quality thresholds are built as proposals (DECISIONS 052); the customer still has to agree them
- [x] Click-through: `GET /dashboard/outages` lists the outage intervals behind the figures, traced to their reports (DECISIONS 055)

## 7. Incidents and actions (PRD 10)

- [x] Report, acknowledge, assign, in progress, restored, verified, closed, with an update history **FR**
- [x] Restoration needs an observed time; a target never counts. Verification needs a new observation from a different person than the one who recorded restoration. Closing needs a root cause. Closed is frozen
- [x] Faults with no outage kept apart from interruptions
- [x] Every incident has an owner (to work it), next action and a next-update time
- [x] Publication to Airtel does not wait for the daily report; Airtel sees published facts and customer updates only **FR**
- [x] Critical notice emailed to Airtel users on the site at publication; overdue-update email to the owner and supervisors, once per missed promise
- [x] Link an incident to the outage intervals of daily reports without duplicating downtime: `GET /incidents/:id/outages` (DECISIONS 055)
- [~] Midnight: carry-over built (DECISIONS 062); automatic split into the neighbouring day not built
- [x] Evidence attachments on incidents (DECISIONS 047)
- [x] Support tickets: Airtel raises a request about a site, Coloplus works it (open, in progress, resolved), comments with internal notes, reopen, retry-safe creation, link to the incident opened for it (DECISIONS 046)
- [x] Ticket attachments (DECISIONS 047)
- [x] Planned maintenance is a record with its request, approver and note (`/maintenance`, DECISIONS 062)
- [x] Reporter can request a correction to when a fault started or was restored, with the original kept and a second person deciding it (decision 042). Request/list/get built and permission-gated; approve/reject built but no supervisor UI yet

## 8. Disputes (PRD 9)

- [x] Airtel reviewer raises a dispute about a site, a period, a published report or one interval, with a category and metric
- [x] Open, under review, awaiting evidence, agreed correction, resolved, closed
- [x] Coloplus's resolution and Airtel's acknowledgement are separate acts; withdrawal is not an agreement
- [x] An upheld dispute needs a real, later approved revision of that day (AT09)
- [x] Both parties' comments kept; internal Coloplus notes and staff ids never reach Airtel or their emails
- [x] Reports show `open_disputes`, so a disputed figure says so
- [x] Evidence attachments on disputes (DECISIONS 047)
- [x] Dispute state carried into exports (open and total disputes per site-day, corrections do not hide them)
- [~] Overdue-dispute alerts: Coloplus gets 72 hours (`DISPUTE_RESPONSE_HOURS`) once a dispute waits on it; one alert per wait (DECISIONS 067). The time is a proposal

## 9. Commissioning and acceptance (PRD 9)

- [x] Stages: requested, built, ready for inspection, inspected, accepted, rejected, on hold (with a reason, resuming to where it was)
- [x] Coloplus submits the dossier; Airtel records its own review; nothing else counts as acceptance, and inspection comes before acceptance (AT10)
- [x] Reasons, responsible party (whose move), next action, due date; each side emailed when the other acts
- [x] Age since ready for inspection, reported apart from operational performance
- [x] Commissioning backlog (`GET /acceptances`), longest-waiting first
- [x] Acceptance-overdue alert, once per missed due time; a hold stops the clock
- [ ] KPI inclusion rule. **needs input**: built as a proposed default (Airtel's acceptance activates a pending site; pending sites stay out of the KPIs). Confirm with Airtel
- [x] Documents attached to the acceptance dossier (DECISIONS 047)
- [~] Every acceptance step has a deadline (a default per stage when none is set) and an acceptance overdue for 7 days also goes to the administrators once (DECISIONS 067). The days are proposals

## 10. Alerts and email (PRD 10)

- [x] Email outbox: written with the change, sent by a job; retried 1m, 5m, 30m, 2h; failures visible to administrators and retryable; never sent twice
- [x] Resend adapter (plain HTTP, paced); log-only mailer for development
- [ ] Verified sending domain and live Resend key. **needs input** (domain)
- [x] In-app alert centre with acknowledgement: `GET /alerts`, `POST /alerts/:id/acknowledge`, `POST /alerts/acknowledge-all` (DECISIONS 056)
- [x] Overdue-report alert to the site's reporters and supervisors, once a day while late; says "stale or missing", never "outage"
- [x] Weekly digest email with a per-user opt-out (DECISIONS 060). Daily digests, repeat reminders and approved recipient lists not built
- [x] Low-fuel alert by email (once per episode, to those who record fuel; DECISIONS 052)
- [~] Report anomaly alerts: generator hours at a site with no generator, more generator hours than the service window, an outage with no incident opened. Emailed once per report to the people who review it, as "worth a look" (DECISIONS 066). The rules are proposals; no statistical or per-site baselines
- [x] Bounce handling: Resend webhook `POST /webhooks/resend` (signature-checked), a bounced or spam-reported email is marked not delivered in the admin list (DECISIONS 057). Needs the webhook created in Resend and `RESEND_WEBHOOK_SECRET` set

## 11. Evidence and files (PRD 6)

- [x] Private R2 bucket, presigned upload, permission-checked short-lived download (2 minutes), SHA-256 checksum, scoped to the site, internal files hidden from Airtel, every download audited; `api check-storage` proves the bucket works. Tested against the live bucket on 2 November: upload with the signed link, checksum enforced, signed download and delete all passed
- [x] Abandoned-upload cleanup, off by default (`ATTACHMENT_GC_ENABLED`), reports what it would delete
- [x] No virus scanner, by decision: only an allow-list of file types is accepted (photos, PDF, text, CSV, Word, Excel), the name's extension must match the type, and files are always downloaded, never displayed (DECISIONS 047)
- [x] Evidence on daily reports (`owner_type=report`): Coloplus-only whatever is asked, never visible to the customer, closed once the report is approved (DECISIONS 066)

## 12. Reports and exports (PRD 10)

- [x] Daily site summaries, incident register, weekly exception report, monthly performance pack and commissioning backlog (CSV; DECISIONS 056)
- [x] Maintenance log: every window with who asked, who decided and why (`GET /maintenance`)
- [x] CSV export of daily summaries; gaps stay blank rows
- [x] CSV export of the incident register (`/exports/incidents.csv`); Airtel gets published incidents only, duration stays blank until restoration is recorded (DECISIONS 044)
- [ ] XLSX export; printable or PDF. Deferred by decision: CSV is enough until a customer asks. **new dependency** (XLSX, PDF)
- [x] Every export carries site ids, period, time zone, units, basis, coverage, review state, formula and report versions, generation time, assumptions, and dispute state (CSV daily summaries, DECISIONS 037)
- [x] Issued reports are never overwritten: every export is entered in an immutable register with its SHA-256, and the exact file is kept when storage is on (`GET /exports/issued`; DECISIONS 058). Approved daily reports were already immutable, with linked revisions
- [ ] Asynchronous generation with visible progress: not needed while exports take well under a second; revisit if a pack grows large

## 13. Audit (PRD 11)

- [x] Authentication, approvals, corrections, publications, disputes, user changes recorded; append-only
- [x] Audit search (`GET /audit`): by actor, action, entity, time; needs `audit.read`; site-scoped readers see only their sites
- [x] Audit of exports (`export.daily_summaries`, `export.incident_register`, in a transaction)
- [~] Retention: emails are anonymised after 90 days (on by default, DECISIONS 068); reports and the audit log are kept; deleting evidence and a 24-month archive are not built (DECISIONS 065). The period needs agreeing

## 14. Non-functional targets (PRD 11) and AT11

- [x] Dashboard p95 under 3 s at 1,000 sites and 50 concurrent users: measured on a laptop, in-process (no network), 30 days of reports per site. One person: 30-day overview 0.3 s. 50 requests at the same instant: worst p95 0.52 s coalesced, 0.12 s with the 10 s cache (it was 3.8 s before; DECISIONS 048). Re-run on the live server (0.75 CPU) with `go test -tags perf` against a scratch database before relying on it
- [x] Approved entry reflected on the dashboard within 60 s: figures are computed on read, plus at most `DASHBOARD_CACHE_TTL` (10 s) of reuse; stored runs are snapshots and sit outside the live path, so nothing waits on them
- [ ] Backup restore exercise (RPO 24 h, RTO 8 h). Script written (`deploy/restore-test.sh`); needs one run on the server and its time recorded here
- [ ] Notification-failure test, access-control test at scale

## 15. Acceptance tests

| ID | What it proves | Status |
|---|---|---|
| AT01 | Airtel cannot reach another customer's site, attachment, report or API response | Sites, reports, incidents, disputes, tickets, attachments, exports: tested |
| AT02 | A known input gives the expected energy estimate with its inputs and formula version | Not started (no energy model) |
| AT03 | An overdue or missing entry shows stale ("Awaiting update"); unreported time is assumed, never confirmed (owner's default-available rule, DECISIONS 051) | Done, tested. Changed from the PRD wording; the customer should agree to it |
| AT04 | Overlapping, duplicate, cross-midnight intervals give correct unions and allocations | Unions, overlap rejection and coverage: done. Cross-midnight splitting: not built |
| AT05 | Re-submitting an import or draft creates no duplicate; concurrent edits conflict | Concurrent edits: done, tested. Site import: done, tested (reports are not imported, by design) |
| AT06 | Reporter cannot self-approve; a correction keeps the original and makes a new revision | Done, tested |
| AT07 | Formula activation needs approval; old results reproduce exactly | Not started |
| AT08 | Fuel movements reconcile; missing values block a false variance | Not started |
| AT09 | Airtel can raise and track a dispute; exports keep provenance and dispute state | Disputes: done, tested. Exports: CSV done with provenance and dispute state, tested; XLSX and PDF open |
| AT10 | Acceptance states stay separate from operational status | Done, tested. KPI rule is a default awaiting Airtel's confirmation |
| AT11 | Load, access control, notification failure and backup restore meet the pilot targets | Not started |

## 16. Waiting on people, not code (PRD 13)

- [ ] What is monitored: power to a load, building supply, or telecom service
- [ ] Reporting cadence and stale thresholds per site or record type
- [ ] Contractual availability: windows, acceptance inclusion, maintenance exclusions, unknown time, aggregation
- [ ] Initial formulas and parameters (load profiles, fuel curves) validated by the Coloplus technical owner
- [ ] Evidence required per entry type; whether incidents are visible to Airtel before approval; approval turnaround
- [ ] Hosting, identity provider, alert channels, retention, support and recovery targets
- [ ] Airtel nominates its operational reviewer
- [ ] A reference dataset with hand-calculated expected results (PRD 12, before build)
- [ ] The Resend sending domain verified; the front-end base URL
- [ ] Whether root cause is shared with Airtel (DECISIONS 032)

## 17. Front end (separate engineer)

- [ ] React app and its `CLAUDE.md` in `frontend/`
- Sign-in screens: login answers `mfa_required` (ask for the authenticator or a recovery code in the same `totp_code` field) or `ok` with `mfa_setup_required` (send them to setup: `/auth/mfa/setup`, show the QR, `/auth/mfa/enable`, then show the ten `recovery_codes` once with a "I have saved these" step)
- Routes the emailed links assume: `/accept-invite`, `/reset-password`, `/incidents/:id`, `/disputes/:id`, `/sites/:id`, `/sites/:id/acceptance`
