# What the front end still has to build

For the front-end engineer. The backend is ahead of the app: everything below exists, is tested and is in `docs/api/swagger.json` (the contract; regenerate with `make openapi`). This file says what to build against it, what changed under screens that already exist, and the rules that must show on screen. Your own standing rules are in `frontend/CLAUDE.md`; where this file says "show X", the trust rules in §11 to §13 of that file apply.

Reasons for each change are in `docs/DECISIONS.md` (numbers in brackets). Tick items off here as you finish them.

## 0. Do these first: they change screens that already exist

These are not new features. The backend now answers differently, and today's screens will show wrong or incomplete numbers without them.

- [x] **Outage reason on the daily report.** A report with an outage cannot be submitted without `outage_reason` (text). It goes on the create body (`POST /sites/:id/daily-reports`) and on the save body (`PUT /daily-reports/:id`). A day with an outage also needs `generator_hours`. The 400 says which is missing. (051)
- [x] **Late reports.** If a report is submitted after its site would already show "Awaiting update", `POST /daily-reports/:id/submit` answers 400 and asks for a reason. Send it as `{ "late_reason": "..." }`. Show the reason on the report afterwards (`late_reason`). Corrections never need one. (049)
- [x] **Assumed time must look assumed.** A site is available by default now. Every summary carries `assumed_available_minutes`; reported availability includes it, the confirmed share does not. Wherever you show an availability figure, show the confirmed share and the assumed hours beside it, and never present assumed time as confirmed. Also show `days_without_report` (per site row) and `site_days_without_report` (portfolio) next to every figure. A site can be set to the strict reading (`unreported_time: "unknown"`); then its availability is `null`, which you already show as `—`. (051)
- [x] **Two new minutes in every summary** (`SummaryDTO`): `excluded_maintenance_minutes` (planned maintenance, already taken out of `eligible_minutes`) and `assumed_outage_minutes` (an outage assumed to continue past midnight; it lowers reported availability, is not confirmed, and is not part of coverage). Show both whenever they are above zero, labelled as planned maintenance and as assumed outage. (062)
- [x] **Formula version is now "3".** If the "How calculated" panel hard-codes a version or an explanation, read them from the response (`calculation.version`, `calculation.explanation`) instead.
- [x] **Data quality** on every dashboard row and for the portfolio: `data_quality` = `{ level: good|fair|poor, days, reported_days, completeness_pct, late_reports, corrections, unconfirmed_share_pct }`. Show the level with the numbers beside it so it can be explained. (052)
- [x] **Responsible team** (`responsible_team`) is on every site and dashboard row: show it to everyone, including Airtel. Never show who is working an incident to Airtel. (044)
- [x] **Customer sees no internal fields.** `outage_reason`, who asked for or decided maintenance, fuel, internal notes and staff ids are Coloplus-only. The API already leaves them out for Airtel; do not build UI that expects them in the customer portal. (Fuel was leaking into the customer portal — `Overview.tsx`'s fuel widget/column and `SiteAttentionBanner`'s low/empty-fuel wording — removed.)

## 1. New screens and features

- [x] **Coloplus's side of disputes, tickets and acceptance (supervisors).** The supervisor role holds `disputes.manage`, `tickets.manage` and `acceptance.manage`. Disputes (`pages/ops/SupervisorDisputes.tsx`) and tickets (`pages/ops/SupervisorTickets.tsx`) are now built: list + detail, the transitions `under_review`, `awaiting_evidence`, `agreed_correction` (with `resolution_report_id`), `resolved` (with `outcome`), `in_progress`, internal comments, and linking a ticket to an incident. Acceptance was already built, embedded in `pages/ops/SupervisorSites.tsx`'s site detail panel (`POST /sites/:id/acceptance`, then `built`, `ready_for_inspection`, hold and resume) — `docs/SESSION_RUNBOOK.md`'s "no Coloplus-side screens" note was stale on that point.

### Site form and site list
- [x] New site fields on create/edit (`PUT /sites/:id`): `responsible_team`, `unreported_time` (`available` default, or `unknown`), `consumption_lph` (litres per generator hour), `tank_capacity_litres`. The owner will set the fuel rate and tank size here per site.
- [x] **Import sites from CSV** (`POST /sites/import`). The API is JSON only, so read the chosen file in the browser and send `{ "csv": "<file text>", "dry_run": true }`. Always offer the dry run first: the response lists every row (`line`, `airtel_code`, `action` = create | update | unchanged | error, `problems[]`) and counts. If `rejected` is above zero nothing was written; show the problems by line and let them fix the file. Then send again with `dry_run: false`. Allowed columns are in the endpoint description (`airtel_code` is required; on an existing site a blank cell leaves the value alone). At most 1,000 rows and about 800 KB. Administrators (site management) only. (059)

### Fuel (Coloplus staff only; `fuel.read` / `fuel.record`)
- [x] Fuel panel on the site screen: `GET /sites/:id/fuel` gives `estimate` and the latest `entries`. The level is always an **estimate**, never a meter reading: label it so, show `explanation`, and show `provisional` / `unreported_days` ("N days since the last entry have no approved report, so the real level may be lower"). Never show a level when `available` is false; show `reason` instead (usually: no agreed rate).
- [x] Record a delivery or a tank reading: `POST /sites/:id/fuel` `{ kind: "delivery" | "reading", litres, occurred_at, note?, client_key }`. Send a fresh `client_key` per form open so a retried tap does not record twice. A reading answers with `variance`; when `review_warning` is true show "differs from what we expected" with `estimated_before_litres` and `difference_litres`. It corrects nothing.
- [x] Void a mistaken entry: `POST /fuel/:id/void` `{ reason }` (confirm dialog). Entries are never edited.
- [x] On dashboard rows `fuel` is `{ litres, hours_remaining, low, empty, provisional }`; show a low or empty badge. Absent for people who may not see fuel. (050, 052) — `MySiteRowDTO` didn't carry `fuel` at all; added it (decision-free, additive DTO field, `backend/internal/services/dashboard.go`'s `MySites()` and `backend/internal/dto/reports.go`), then wired it into `MySiteCard.tsx` and `pages/ops/ReporterOverview.tsx`. `pages/ops/Sites.tsx`/`SupervisorSites.tsx`'s site list stays a lean directory (`AdminSite`, no live figures at all) by design, not a dashboard — fuel wasn't added there.

### Alert centre
- [x] Bell with an unread count: `GET /alerts?unread=true` returns `unread`. List, `POST /alerts/:id/acknowledge`, `POST /alerts/acknowledge-all`. Everyone has it; each person sees only their own. (056)
- [x] Report anomaly alerts arrive in the alert centre and by email as kind `report.anomaly` ("Worth a look"). Show them like any alert; do not turn them into badges on the report or imply a fault. (066)

### Weekly summary email setting
- [x] A toggle in the user's settings: `GET` / `PUT /digest/preference` `{ weekly: true|false }`. On by default; the email itself is switched on by the owner on the server. (060)

### Tickets (Airtel raise, Coloplus work them)
- [x] Raise from a site (`POST /sites/:id/tickets`: `category`, `subject`, `description`, `priority`, `contact_email?`, `client_key`), list (`GET /tickets`, filters in swagger), summary counts (`GET /tickets/summary`), detail with history (`GET /tickets/:id`), comment (`POST /tickets/:id/comments`, `internal: true` is Coloplus-only), transition (`POST /tickets/:id/transition`), link an incident (`POST /tickets/:id/incident`, Coloplus). Airtel never sees internal notes, staff ids or a linked incident until it is published. The priority is the requester's wish, not an incident severity. (046)

### Evidence files (attachments)
- [x] Three steps, never through the API: `POST /attachments` `{ owner_type: ticket|incident|dispute|acceptance, owner_id, filename, content_type, size_bytes, sha256, shared? }` returns an `upload` target (`url`, `method`, `headers`); `PUT` the file to that URL with **exactly** those headers; then `POST /attachments/:id/complete`. Compute the SHA-256 in the browser (`crypto.subtle`) first. Accepted: JPEG, PNG, WebP, HEIC, PDF, plain text, CSV, Word, Excel; at most 25 per owner. List with `GET /attachments?owner_type&owner_id` (for acceptance `owner_id` is the site id). Download is `GET /attachments/:id/download`, which returns a link good for about two minutes: open it at once, do not store it. `internal` files are Coloplus-only. `owner_type: report` attaches evidence to a daily report (a meter photo); it is always internal, open to reporters while the report is a draft, rejected or submitted, and closed once approved (409). Customers cannot add or list it. (066) A `503` means file storage is off; show a plain message. `shared: true` lets Airtel see a Coloplus file. (047) — `owner_type: report` wired into `DailyReportPanel.tsx` and `pages/ops/DailyReports.tsx`. `FileDropZone.tsx` now catches a 503 specifically and swaps itself for a sticky `AlertBanner` ("File storage is currently off"), instead of only a toast that fades.

### Dashboard
- [x] **Period, region and data quality:** `GET /dashboard/overview?from&to&region[&quality=good|fair|poor]` (or `?date=`), pooled by minutes. `quality` keeps only those site rows; the counts and portfolio still cover every site, so say so beside a filtered list. `regions` in the response fills the region filter. (043)
- [x] **Trend grain:** `GET /dashboard/trend?from&to&region&grain=day|week|month`. A bucket nobody reported is `null`, not zero. Each point carries `from`, `to` and `days` (a part bucket is plain), and `site_days` / `site_days_reported` to say how much rests on reports. (043)
- [x] **Drill-down:** `GET /dashboard/outages?from&to&region&site_id` lists the outage stretches behind the figures (at most 92 days), each with `report_id` and `report_revision`. `reason` is Coloplus-only. `GET /incidents/:id/outages` shows the outage reported during an incident and `total_minutes`. (055)
- [x] **Source hours:** `source_minutes` (`{ grid: 700, generator: 600 }`) on a summary. Sources can overlap, so do not make them add up to served time. (055)
- [x] **SLA badge:** when `sla` is present (it is absent until the owner sets a target) show `met` / `at_risk` / `missed` with `target_percent`, and always show `confirmed_meets_target` beside it: a site that meets the target only because nobody reported must look different. Do not infer a status yourself. (063)

### Planned maintenance
- [x] Request: `POST /sites/:id/maintenance` `{ starts_at, ends_at, reason }` (reporters, supervisors, administrators). The list response carries `rules` (`notice_hours`, `max_hours`, `monthly_cap_hours`): show them on the form and check them client-side as a courtesy; the API is the authority and its 400 names the broken rule. A window stays within one local day and cannot overlap another of the site (409). — `utils/maintenanceValidation.ts`'s `maintenanceRuleErrors()` now checks notice/max/same-day/overlap/monthly-cap against the site's other windows, not just the required-fields/end-after-start checks it had before.
- [x] Approvals queue: `GET /maintenance?status=requested`; `POST /maintenance/:id/approve|reject|cancel` `{ note }` (note required to reject and cancel). The person who asked can never approve their own (403): hide the button for them. Approval needs a confirm dialog because it changes the site's figures.
- [x] The customer can see windows (read only), but not who asked or decided. (062) — A new "Maintenance" tab in `SiteDetailPanel.tsx`.

### Closing months and stored runs (administrators / Coloplus staff)
- [x] Periods page: `GET /periods` lists closures (`locked` says whether it still holds). `POST /periods/:month/close` (`2026-10`) is allowed once five days after the month ends; the 400 says from when. `POST /periods/:month/reopen` `{ reason }` (at least 10 characters). Both need confirm dialogs (reopen is `warning` tone). A closed month answers **409** on creating, correcting or approving a report for any of its days: show "this month is closed" and who can reopen it. (064)
- [x] Runs: `POST /calculation-runs` `{ month: "2026-10", region?, label? }`, `GET /calculation-runs`, `GET /calculation-runs/:id` (every site's frozen figures), `GET /calculation-runs/:id/compare` (only the sites that differ, each with `then` and `now`, plus `reports_changed`, `formula_changed`). A stored run never changes; the comparison is how a later correction becomes visible. Closing a month already stores one. (061)

### Reports and exports
- [x] Download buttons (CSV): `GET /exports/daily-summaries.csv?from&to[&site_id&region]`, `/exports/incidents.csv?from&to`, `/exports/monthly-performance.csv?month=2026-10`, `/exports/weekly-exceptions.csv?week_start=` (a Monday), `/exports/commissioning-backlog.csv`. Needs export access. Fetch with the auth header and save the blob; a plain link will not carry the token. Months in the future and ranges over 92 days are refused with a 400. (037, 056) — Built on `pages/customer/Reports.tsx`; `pages/ops/IssuedExports.tsx` (renamed "Exports" in the nav) now has the same generation cards for admin/supervisor, who hold `exports.create` too.
- [x] **Issued exports register:** every download is recorded. The response headers `X-Export-Id` and `X-Content-SHA256` identify it. `GET /exports/issued` lists them (`stored` says whether the exact file was kept), `GET /exports/issued/:id/download` returns a short-lived link to the exact file. Add an "Issued exports" page; a person sees their own, auditors see all. (058)
- [x] The CSVs now include `assumed_outage_minutes`, `excluded_maintenance_minutes`, and (monthly) `sla_*` columns. Nothing to build; just do not parse them by position.

### Auditor role
- [x] A read-only role (`auditor`): sees everything in scope, can export and read the audit log, can change nothing. Hide every write control when the permission is missing; do not hide by role name. (045)

### Acceptance and disputes
- [x] Every acceptance step now has a `due_at` (a default per stage when none is set); show it and mark an overdue one. (067)
- [x] Overdue acceptances also email administrators after 7 days (`acceptance.escalated`) and a dispute waiting on Coloplus for over 72 hours alerts its managers (`dispute.overdue`); both arrive in the alert centre. Show them as ordinary alerts. (067)

## 2. Rules to keep in mind

- Authorise by permission (`/auth/me` returns the list), never by role name. A person whose role requires two-step sign-in and has not finished it holds **no permissions**; route them to MFA setup.
- Out-of-scope ids come back as 404 on purpose. Treat 404 as "not yours or not there".
- Nothing may show an invented zero. `null` means "not reported" or "not applicable" and is shown as `—`.
- Approved reports are immutable; a change is a correction that needs a reason and a second person. A closed month blocks even that until reopened.
- Anything with an effect on figures or on someone else (approve, publish, reopen a month, approve maintenance, void fuel) needs a confirm dialog.
- Use a stable `client_key` per form open for fuel and tickets so a double tap cannot create two.

## 3. Switches the owner controls on the server

These are settings, not code. You may see different behaviour depending on them:

- `SLA_TARGET_PERCENT`: 99.5 by default, so the `sla` block is present; it is absent only if the owner sets 0.
- `WEEKLY_DIGEST_ENABLED`: the weekly email is on by default; each user can opt out with `PUT /digest/preference`.
- `MIDNIGHT_CARRY_OVER`: `assumed_outage_minutes` is always 0 when it is off.
- `RETENTION_ENABLED`: on by default; emails older than 90 days lose their address and text. Off means they are only counted.
- File storage (`R2_*`): attachments and kept export copies answer 503 / `stored: false` without it.
