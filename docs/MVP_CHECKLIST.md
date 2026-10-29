# Coloplus MVP checklist

Backend status against the PRD (v1.1, 24 Oct 2026), section 12 "MVP build" and the acceptance tests AT01 to AT11. Updated 29 Oct 2026, after disputes, audit search, overdue-report alerts, acceptance and the site overview. Update this file in the same change that finishes an item.

- `[x]` built, lint clean, and the risky parts have tests
- `[ ]` not done. A tag says why: **partial** (some of it exists), **needs input** (waiting on a decision, credential or domain from outside), **new dependency** (needs your yes before I start)
- The PRD's "recommended first release" (section 13) is marked **FR** where an item belongs to it: site condition, a complete fault register, downtime, accountability and recovery updates, with energy and fuel estimates only where inputs are adequate.

## 1. Foundations

- [x] Hexagonal Go API, constructor injection, layer rules enforced by lint and an architecture test
- [x] Config read once, validated at boot, production guards (Swagger off, CORS list, rate limit, real mailer, https base URL)
- [x] One pgxpool, one HTTP client per provider, bounded worker pool, graceful shutdown order
- [x] Forward-only goose migrations (7), `api migrate` as a separate step
- [x] Cron scheduler with four jobs (email dispatch, incident alerts, report alerts, acceptance alerts)
- [x] Append-only audit log (database trigger), batched for low-risk events, written in the same transaction for approvals, corrections, publications, disputes
- [x] Rate limiting (tighter on sign-in and recovery), CORS allow-list, security headers, JSON-only writes (CSRF guard)
- [x] Swagger generated at build (`make swag`)
- [x] Dockerfile and a CI workflow exist (**not yet checked that CI is green on GitHub**)
- [ ] Production Docker Compose, Caddy, Cloudflare setup. **needs input** (domain, server access). Ask before touching infrastructure
- [ ] Deploy pipeline (ghcr.io, forced-command SSH, health-gated). **needs input**
- [ ] Backups to R2 and a restore exercise (AT11). **new dependency** (R2)
- [ ] UptimeRobot on `/healthcheck`. **needs input**

## 2. People and access (PRD 4)

- [x] Email and password sign-in, lockout after 5 failures, session in an HttpOnly cookie, deactivation takes effect on the next request
- [x] Roles map to permissions in the database: administrator, field reporter, supervisor, Airtel viewer, **Airtel reviewer** (new)
- [x] Site scope on every endpoint; out-of-scope ids answer 404 so ids cannot be probed
- [x] User administration and site assignments
- [x] Invitation and password-reset links by email (single use, expiring, a password change ends older sessions)
- [x] Authenticator-app (TOTP) setup and use, secrets encrypted at rest
- [ ] **Enforce** two-step sign-in for administrators, supervisors and all Airtel users. **partial** (optional today). Required before the pilot (DECISIONS 024)
- [ ] Recovery codes for a lost phone. Needed together with enforcement
- [ ] Model administrator role (drafts formulas; separate approver). Arrives with the formula engine
- [ ] Management / auditor role (read-only portfolio and audit scope)

## 3. Sites (PRD 6)

- [x] Site master record: immutable id, Airtel and Coloplus codes (duplicates refused), name, address, region and state, type, installed power sources, service window, freshness threshold, status (pending acceptance, active, suspended), commissioning date, coordinates
- [x] Site directory with search by either code and name, filters on state, region, status
- [ ] Asset capacities, reporting cadence, load profile, formula profile, effective dates on the site record. **partial**
- [ ] Directory filters on source, data quality, acceptance state, assigned owner. **partial**

## 4. Daily reporting and approval (PRD 6, 9)

- [x] Draft, submit, approve or return; the approver is never the author (service rule and database check)
- [x] Approved reports are immutable (triggers); a correction is a linked revision with a reason, and the original stays visible to Airtel until the correction is approved
- [x] Edits quote a version; a concurrent edit is a visible conflict
- [x] Interval validation: whole minutes, inside the reporting day (Africa/Lagos), no future, no served/outage overlap, a source for served time
- [x] Observed versus declared intervals; only observed counts
- [x] Airtel sees a published view of approved reports only (no notes, authors, review comments)
- [ ] Explanation required for late submissions and overrides
- [ ] Automatic splitting of events that cross midnight (DECISIONS 027). Reporter splits by hand today
- [ ] CSV/XLSX import: preview, row errors, duplicate detection, explicit commit, batch id and idempotency key. **new dependency** (an XLSX reader)
- [ ] Source hours, maintenance and open-issue fields on the daily summary. **partial**
- [ ] Load, generator, fuel and tank inputs (section 6 tables)
- [ ] Controlled reopening of a closed reporting period
- [ ] Provisional view of unapproved entries (only if Airtel and Coloplus jointly enable it). **needs input**

## 5. Calculations (PRD 7, 8)

- [x] Daily metrics: eligible minutes, served, outage, unknown, coverage, reported availability, confirmed served share. Missing data is unknown, never available, never zero
- [x] Portfolio availability pooled by eligible minutes, not an average of percentages
- [x] "How calculated" payload: formula and version, input report and revision, window, time zone, warnings, plain-language explanation
- [ ] Store calculation runs with their inputs, so a result can be reproduced and queued/failed/superseded runs shown (DECISIONS 028). **partial** (returned on read, not stored)
- [ ] Formula versions, parameter sets, draft, test, independent approval, effective-dated activation (AT07)
- [ ] Source runtime; estimated energy; modelled fuel use; fuel balance and variance; battery autonomy (AT02, AT08). Each labelled as an estimate
- [ ] Weekly and monthly aggregation of the daily figures
- [ ] Planned-maintenance exclusions where the contract allows. **needs input** (contract)
- [ ] An SLA compliance figure. **needs input**: the PRD says to show operational metrics only until Airtel and Coloplus agree the treatment of unknown time

## 6. Portfolio and site screens, as API (PRD 5)

- [x] Portfolio overview for a day: active, pending and suspended counts, sites reported, not reported, awaiting update, outage reported, pooled availability with coverage, per-site condition (available, outage reported, awaiting update) **FR**
- [x] An overdue or missing entry never shows as current or available (AT03)
- [x] Site detail in one call (`GET /sites/:id/overview`): profile, condition, last days with visible gaps, open incidents, disputes, acceptance
- [ ] Period dashboards (week, month, custom range), data-quality indicator, stale-submission count
- [ ] Click-through from a card to its supporting sites and intervals

## 7. Incidents and actions (PRD 10)

- [x] Report, acknowledge, assign, in progress, restored, verified, closed, with an update history **FR**
- [x] Restoration needs an observed time; a target never counts. Verification needs a new observation from a different person than the one who recorded restoration. Closing needs a root cause. Closed is frozen
- [x] Faults with no outage kept apart from interruptions
- [x] Every incident has an owner (to work it), next action and a next-update time
- [x] Publication to Airtel does not wait for the daily report; Airtel sees published facts and customer updates only **FR**
- [x] Critical notice emailed to Airtel users on the site at publication; overdue-update email to the owner and supervisors, once per missed promise
- [ ] Link an incident to the outage intervals of daily reports without duplicating downtime. Also the automatic midnight split
- [ ] Evidence attachments on incidents. **new dependency** (R2)
- [ ] Link planned maintenance and approval references

## 8. Disputes (PRD 9)

- [x] Airtel reviewer raises a dispute about a site, a period, a published report or one interval, with a category and metric
- [x] Open, under review, awaiting evidence, agreed correction, resolved, closed
- [x] Coloplus's resolution and Airtel's acknowledgement are separate acts; withdrawal is not an agreement
- [x] An upheld dispute needs a real, later approved revision of that day (AT09)
- [x] Both parties' comments kept; internal Coloplus notes and staff ids never reach Airtel or their emails
- [x] Reports show `open_disputes`, so a disputed figure says so
- [ ] Evidence attachments on disputes. **new dependency** (R2); until then evidence is described in comments
- [ ] Dispute state carried into exports, separately identifiable. Waits for exports (section 12)
- [ ] Overdue-dispute alerts. **needs input** (an agreed response time)

## 9. Commissioning and acceptance (PRD 9)

- [x] Stages: requested, built, ready for inspection, inspected, accepted, rejected, on hold (with a reason, resuming to where it was)
- [x] Coloplus submits the dossier; Airtel records its own review; nothing else counts as acceptance, and inspection comes before acceptance (AT10)
- [x] Reasons, responsible party (whose move), next action, due date; each side emailed when the other acts
- [x] Age since ready for inspection, reported apart from operational performance
- [x] Commissioning backlog (`GET /acceptances`), longest-waiting first
- [x] Acceptance-overdue alert, once per missed due time; a hold stops the clock
- [ ] KPI inclusion rule. **needs input**: built as a proposed default (Airtel's acceptance activates a pending site; pending sites stay out of the KPIs). Confirm with Airtel
- [ ] Documents attached to the dossier. **new dependency** (R2)
- [ ] Escalation beyond the second email; per-milestone due dates

## 10. Alerts and email (PRD 10)

- [x] Email outbox: written with the change, sent by a job; retried 1m, 5m, 30m, 2h; failures visible to administrators and retryable; never sent twice
- [x] Resend adapter (plain HTTP, paced); log-only mailer for development
- [ ] Verified sending domain and live Resend key. **needs input** (domain)
- [ ] In-app alert centre with acknowledgement
- [x] Overdue-report alert to the site's reporters and supervisors, once a day while late; says "stale or missing", never "outage"
- [ ] Daily and weekly digests; repeat reminders; approved recipient lists; per-user preferences
- [ ] Fuel and runtime anomaly alerts, low fuel / autonomy alerts. Wait for the fuel model
- [ ] Bounce handling (Resend webhooks)

## 11. Evidence and files (PRD 6)

- [ ] Private R2 bucket, presigned upload, permission-checked short-lived download, SHA-256 checksum, scoped to site and period. **new dependency**
- [ ] Attach to reports, incidents, disputes, acceptance dossiers

## 12. Reports and exports (PRD 10)

- [ ] Daily site summaries, weekly exception report, monthly performance pack, incident register, maintenance log, commissioning backlog
- [ ] CSV and XLSX export; printable or PDF. **new dependency** (XLSX, PDF)
- [ ] Every export carries site ids, period, time zone, units, basis, coverage, review state, formula and report versions, generation time, assumptions, and dispute state
- [ ] Immutable report snapshots with version history; "never overwrite an issued report"
- [ ] Asynchronous generation with visible progress

## 13. Audit (PRD 11)

- [x] Authentication, approvals, corrections, publications, disputes, user changes recorded; append-only
- [x] Audit search (`GET /audit`): by actor, action, entity, time; needs `audit.read`; site-scoped readers see only their sites
- [ ] Audit of exports (waits for exports)
- [ ] 24-month online retention and a deletion policy. **needs input** (joint approval)

## 14. Non-functional targets (PRD 11) and AT11

- [ ] Dashboard p95 under 3 s at 1,000 sites and 50 concurrent users. Not measured
- [ ] Approved entry reflected on the dashboard within 60 s. Meets it by construction today (computed on read); re-check once calculation runs are stored
- [ ] Backup restore exercise (RPO 24 h, RTO 8 h)
- [ ] Notification-failure test, access-control test at scale

## 15. Acceptance tests

| ID | What it proves | Status |
|---|---|---|
| AT01 | Airtel cannot reach another customer's site, attachment, report or API response | Sites, reports, incidents, disputes: tested. Attachments and exports: not built |
| AT02 | A known input gives the expected energy estimate with its inputs and formula version | Not started (no energy model) |
| AT03 | An overdue or missing entry shows stale or unknown, never available or zero | Done, tested |
| AT04 | Overlapping, duplicate, cross-midnight intervals give correct unions and allocations | Unions, overlap rejection and coverage: done. Cross-midnight splitting: not built |
| AT05 | Re-submitting an import or draft creates no duplicate; concurrent edits conflict | Concurrent edits: done, tested. Imports: not built |
| AT06 | Reporter cannot self-approve; a correction keeps the original and makes a new revision | Done, tested |
| AT07 | Formula activation needs approval; old results reproduce exactly | Not started |
| AT08 | Fuel movements reconcile; missing values block a false variance | Not started |
| AT09 | Airtel can raise and track a dispute; exports keep provenance and dispute state | Disputes: done, tested. Exports: not built |
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
- Routes the emailed links assume: `/accept-invite`, `/reset-password`, `/incidents/:id`, `/disputes/:id`, `/sites/:id`, `/sites/:id/acceptance`
