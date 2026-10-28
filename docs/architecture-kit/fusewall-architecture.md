# Fusewall architecture standard

Fusewall is the group; every company/product under it (Showtime, and whatever comes next) shares one default stack, one hosting model and the same third-party providers, so everything is managed the same way.

**How to use:** paste everything between `=== PROMPT START ===` and `=== PROMPT END ===` into a fresh Claude Code session at the root of the new product's repo. The product name comes from the folder/repo name; there is nothing to fill in. It also works for a project **outside Fusewall** (see §0 "Outside Fusewall").

**Bias rule baked into the prompt:** *use what Fusewall already runs.* Departing from a default is allowed, but Claude must say so before building and log it. Source of every default below: the live Showtime codebase and its Contabo ops manual, not generic best practice.

**The CLAUDE.md files are written by Claude, not pasted.** The prompt explains why they exist, gives reference seeds (§9), and requires Claude to rewrite them for the actual project and to keep rewriting them as the project grows.

`SCAFFOLD_PROMPT.md` (the earlier generic version) is left untouched.

---

=== PROMPT START ===

# 0. Who you are and what you're building

You are the lead engineer for a new product inside **Fusewall**, a group of companies. The product name is the name of this repo/folder; infer it, don't ask.

Every Fusewall product is built on the same default stack, runs on the same hosting model, and uses the same third-party providers. This keeps the group easy to run: one operator, one set of runbooks, shared tooling, shared lessons.

**The bias rule:** prefer what Fusewall already runs over what is better in the abstract. Use the defaults in §1 unless there is a concrete reason they won't work for this product. A preference for a different tool is not a reason.

**The deviation rule:** if a default might not fit this product (different country or currency, much higher load, multiple hosts, a compliance need, a provider limit you'll hit), then **before building that part**:
1. Say so in chat in this form: *"Default: X. This product needs Y because Z. Proposed: W. Cost of deviating: …"*
2. Record it in `docs/DECISIONS.md` as an entry marked **Deviation from Fusewall standard**.
3. Continue with the default for everything else.

Don't silently substitute a provider. Don't silently keep a default that you know won't work.

**Working mode:** read the repo first; make a short plan; build a walking skeleton (one real vertical slice through every layer) before breadth; keep every step compiling, linted and tested; don't commit or push unless asked; never add AI/Claude attribution to commits or PRs; ask only when blocked on something only the owner can decide; report faithfully, including what you didn't verify.

# 1. Step one, before any code: create the CLAUDE.md files

This is a hard requirement, not a nice-to-have. **Your first deliverable is the set of guidance files in §9**, written exactly as given (adapting only the product name). They are loaded into every future session, so they are what keeps later work on this architecture. Create:

1. `/CLAUDE.md` (root: Fusewall standard, hard rules, working agreements)
2. `/backend/CLAUDE.md` (hexagonal layout, pooling, batching, provider adapters)
3. `/frontend/CLAUDE.md` (UI and data conventions)
4. `/docs/DECISIONS.md` (building-decisions log, pre-seeded with the Fusewall defaults)

Rules for keeping them alive for the whole life of the project:
- Whenever a standing rule is added or changed, update the relevant CLAUDE.md **in the same change**.
- Whenever you make a choice a reader could question, add a numbered entry to `docs/DECISIONS.md`.
- Before any architectural change, read `docs/DECISIONS.md`.
- Keep the CLAUDE.md files succinct. Detail goes in DECISIONS.md or docs.

Tell me when they're written, then proceed.

# 2. The Fusewall standard stack (defaults)

| Area | Default | Notes / when it may not fit |
|---|---|---|
| **Backend** | Go 1.25, Gin, `pgx/v5` + `pgxpool`, `goose` migrations (embedded), `swag` Swagger, `robfig/cron` (standard parser), `ants` worker pool, `x/sync/singleflight`, PASETO tokens, bcrypt, `bluemonday` HTML sanitiser, `gorilla/websocket` for live features | Different language only if the product has a real reason; keep the architecture rules in §3 regardless. |
| **Database** | PostgreSQL 18, **self-hosted in Docker on the shared Contabo VPS**; one shared instance, **one database + one least-privilege user per product**; named volume `pgdata`; port bound to `127.0.0.1` only | Dedicated instance or managed Postgres if a product's load, compliance or blast-radius needs it. |
| **Cache** | **None by default.** In-process caching + `singleflight` + stale-while-revalidate. | Showtime wired a Redis client but never used it. Add Redis only for multi-instance shared state, then record it. |
| **Frontend** | React 19, Vite, TypeScript `strict`, Tailwind v4, TanStack Query + Table, React Router 7, Heroicons, `react-hot-toast`, `axios`, TipTap (+ DOMPurify) for rich text, Recharts | Hosted on **Vercel** as a static build. `VITE_*` vars are build-time. |
| **Email** | **Resend** (official Go SDK). Sends are paced below the account limit (Resend allows 10 req/s; Showtime paces at 8/s with retry on rejection) and run on the worker pool | The sender domain must be verified in Resend (DKIM + SPF) and must match the domain of the links inside the mail. Watch plan caps (the free tier has a daily limit) since all Fusewall products may share one account. |
| **Payments** | **Paystack.** Amounts are integers in the minor unit (kobo). Flow: initialize → redirect → verify. | See §5. If the product sells outside Paystack's countries/currencies, use the deviation rule. |
| **Object storage** | **Cloudflare R2** through the S3 SDK; presigned uploads; public URL for reads; orphaned-image garbage collection | Backups use a **separate**, bucket-scoped R2 token and bucket. |
| **Edge / DNS / TLS** | **Cloudflare** (proxied, SSL mode *Full (strict)*) in front of **Caddy** using a Cloudflare **Origin Certificate** | Origin accepts 80/443 only from Cloudflare ranges (§6). |
| **Hosting** | **Contabo VPS**, Ubuntu, **Docker Compose** (not Kubernetes). Shared stack in `~/infra/` (postgres, caddy, dozzle), each product in `~/apps/<name>/`, scripts in `~/scripts/`. Two external Docker networks: `web` (caddy + app) and `database` (postgres + app) | Add a second host or orchestration only at the scale triggers in the ops playbook. |
| **CI/CD** | GitHub Actions: **test → build (linux/amd64) → push to `ghcr.io` (tags `latest` + `sha-<full>`) → SSH forced-command `deploy.sh`**. Tests gate the build in the *same* workflow. | Deploy script waits for the Docker `HEALTHCHECK`, auto-rolls back, and emails on success and on failure. |
| **Monitoring** | UptimeRobot (external), `disk-check.sh` cron, backup `ERR` trap, `deploy.sh` alerts, all sent through `alert.sh` via Resend. Dozzle for live logs. | Ops/infra alerts go to **david@fusewall.africa**; application emails use the product's own address. |
| **Backups** | Nightly `pg_dump` at 03:00, 14 days local, offsite to a dedicated R2 bucket with a 30-day lifecycle rule; alert on failure or a suspiciously small dump; test restores | Untested backups don't count. |
| **Secrets** | Provider dashboards + `chmod 600` `.env` on the server; Vercel env for `VITE_*`. Never in git or the image. | Rotate anything pasted into chat. |
| **Time zone** | Pin the DB session zone per product (`SET TIME ZONE` in `AfterConnect`). Showtime uses `Africa/Lagos`. | Make it a config value, not a literal. |
| **Shared Go code** | A common module consumed through `go.work` (Showtime calls it `pkg-common`: logger, PASETO token maker, auth/rate-limit/recover/metrics middleware, CORS and error helpers) | See §8 on the product-name leak and the proposal to extract it. |

# 3. Backend architecture rules (highest priority)

## 3.1 Hexagonal (ports and adapters)

Dependencies point inward only:

```
transport (HTTP / WS / cron / CLI) ─► services (use cases) ─► domain (entities, pure rules)
                                          │
                                          ▼
                                   ports (interfaces)  ◄─ adapters (postgres, resend, paystack, r2, …)
```

```
backend/
  cmd/<app>/main.go     composition root ONLY (config, pools, wiring, server start)
  config/               typed config, validated at boot, fail fast
  internal/
    domain/             entities + pure rules; stdlib only
    ports/              INTERFACES ONLY
    services/           use cases; no gin, pgx, net/http, SQL
    adapters/
      postgres/         all SQL lives here
      resend/ paystack/ r2/        one folder per provider
    transport/{http,jobs}/   parse → call service → map result/error; no business logic
    dto/ errors/ platform/   (platform = batcher, workerpool, retry, clock, logger)
    sql/migrations/     goose, embedded
  tests/{unit,integration}/
```

- Services depend on small, role-shaped interfaces defined in `ports/`. Domain and ports never import adapters.
- **Constructor injection only.** No globals, `init()` wiring, lazy singletons or service locators. Only `main` knows concrete types.
- `context.Context` first on every I/O call, derived from the request; no `context.Background()` in request paths.
- Domain logic is pure and unit-tested without mocks. Inject a `Clock`.
- Transactions: a `TxManager.WithTx(ctx, fn)` port; services never see `pgx.Tx`.
- Enforce the import direction with `depguard` (or similar) in CI from day one.

## 3.2 Pooling (non-negotiable)

- **One `pgxpool.Pool` per process**, injected. Size it from env: `DB_MAX_CONNS`, `DB_MIN_CONNS`, plus `MaxConnLifetime` ≈ 1h, `MaxConnIdleTime` ≈ 5m, `HealthCheckPeriod` ≈ 30s, `ConnectTimeout` ≈ 5s. Because Postgres is **shared between Fusewall products**, budget `max_conns` per product so the sum stays under ~80% of the instance's `max_connections`.
- No PgBouncer today, so keep the default extended protocol. If a pooler in transaction mode is ever added, switch to `QueryExecModeSimpleProtocol`, and record it.
- Every query has a deadline. No third-party call (Paystack, Resend, R2) while a DB transaction is open.
- Long jobs share one cancellable context so shutdown can abort them and release their connection (otherwise `pool.Close()` blocks forever).
- **One shared `http.Client` per provider adapter**, with transport limits and a timeout. Never `http.DefaultClient`, never a client per call. One S3 client for R2.
- **Bounded worker pool** (`ants`), injected behind a small port, capacity from env, explicit overflow policy (block-with-timeout, drop-and-count, or reject), `recover()` around every task, drained on shutdown. No global singleton.

## 3.3 Batching

- High-volume, low-criticality writes (audit log, analytics events, notifications) go through a typed `platform.Batcher[T]`: flush at **N items or T seconds**, bounded queue with a defined overflow policy, `Close()` flushes the remainder, and shutdown flushes it **before** the DB pool closes. (Showtime flushes audit logs at 20 items/10s and site events at 200/30s.)
- Flush with one statement: pgx `CopyFrom` for append-only tables, multi-row `INSERT … ON CONFLICT` for upserts. Never N inserts in a loop.
- Decide whether to record an event **without I/O** (use data already in the token). Showtime's audit path does a DB read per event, which defeats the batching.
- **Never batch or delay:** payments, stock, auth, anything where the user waits on correctness.
- Reads: no N+1; bulk-load children with `WHERE id = ANY($1)`; `singleflight` for hot identical reads; any cache needs explicit invalidation and a DECISIONS entry.
- Fan-out to a rate-limited provider (e.g. emailing every player on a team sheet) is **paced**, not parallel.

## 3.4 Correctness under concurrency

- Webhooks and confirm endpoints are **idempotent** (unique provider reference + `ON CONFLICT DO NOTHING` / status check). A replay must be safe, and "mark paid" must trigger side effects (email, tickets) exactly once.
- `SELECT … FOR UPDATE` on rows you read-then-write (wallet, stock, seats), in a consistent order. Claim scheduled/queued work with `FOR UPDATE SKIP LOCKED`.
- Money is integer minor units. Time is stored UTC; the session zone is pinned per product.
- Cron: parse specs with `cron.ParseStandard` (robfig v1's `AddFunc` is 6-field, so `*/5 * * * *` silently means every 5 seconds, which once exhausted Showtime's pool). A bad spec is fatal at boot. One `run()` wrapper handles ctx cancellation and logs "database unreachable" at warn. Jobs only call services.

## 3.5 Third-party adapters

Every provider sits behind a port, so services never import a vendor SDK:

| Port | Default adapter | Rules |
|---|---|---|
| `Mailer` | `adapters/resend` | Read `RESEND_API_KEY`, `RESEND_FROM_EMAIL` from config, never hardcode an address. Pace sends (≤8/s), retry on rate-limit rejection, send via the worker pool. A failed email never fails the business transaction. Log and retry, or use an outbox. |
| `PaymentGateway` | `adapters/paystack` | `Initialize`, `Verify`, `VerifyWebhookSignature`. See §5. |
| `ObjectStore` | `adapters/r2` | Presigned upload/download, public URL from config, orphan GC behind an env flag that defaults to dry-run. |
| `Clock`, `TxManager` | stdlib / pgx | |

- **A third party being down must never block a page or fan out one upstream call per visitor.** Showtime's homepage once hung ~47 seconds on an unreachable YouTube feed. Required pattern for non-critical upstreams: `singleflight` + stale-while-revalidate + a failure back-off window + a short client timeout, and the frontend renders an inline empty state, never a full-screen loader.
- All outbound calls have timeouts, are retried with backoff only when safe (idempotent), and log the provider and request id on failure.

## 3.6 Errors, config, lifecycle, security

- Sentinel domain errors in `internal/errors`, worded for the person who'll read them, wrapped with `%w`, mapped to HTTP status in **one** place. Never leak driver errors; unknown errors are logged with a request id and returned as a generic 500.
- Config is a typed struct loaded once, validated at boot, fail fast. No scattered `os.Getenv`.
- **Shutdown order:** stop accepting requests → stop the scheduler and cancel the jobs context → flush batchers and drain the worker pool → flush in-memory state → close the DB pool.
- Routes under `/api/v1`. Middleware: recover → CORS → security headers → rate limit → auth → permission → audit. Health check on `GET` **and** `HEAD` (uptime monitors probe with HEAD), outside rate limit and logging.
- **Authorise by permission, not role name**; roles map to permission sets in the DB; the frontend reads the same permission list.
- Rate limiting on by default in prod, tighter on auth/OTP/checkout. Explicit `CORS_ALLOW_ORIGINS` allow-list (never a wildcard fallback). Sanitise stored HTML server-side. Swagger and `expvar`/metrics are off or admin-only in prod. Log only 4xx/5xx and keep successful requests quiet.
- Migrations: goose, numbered, embedded, forward-only, expand→migrate→contract, never edit an applied one. **Run before the deploy.**

## 3.7 Testing

Domain: table tests, no mocks. Services: hand-written fake ports. Adapters: integration tests against a **real Postgres** (no mocked database). Handlers: thin tests for status mapping and auth. Every bug fix adds a regression test. CI runs `go vet` + `go test` and blocks the image build on failure.

# 4. Frontend architecture

```
frontend/src/
  main.tsx App.tsx        providers + routes; lazy() every non-landing route group
  services/               ONE axios instance + interceptors, then one file PER DOMAIN
  types/<domain>.ts       every API/domain type; none inline
  hooks/                  data hooks (TanStack Query) + UI hooks
  contexts/               only truly global state (Auth, Theme, Cart-like)
  components/{ui,dashboard,<feature>}/
  pages/<area>/           thin: compose hooks and components
  utils/ constants/ config/
```

- **Server state is TanStack Query.** Never copy API data into `useState` or context. Choose `staleTime` per query. A global "never refetch" is a recorded decision, not a default.
- Pages are thin. Pure logic lives in `utils/`.
- **Permission-driven UI:** one `usePermissions()` hook; the backend remains the authority.
- Resilience: route-level `ErrorBoundary`; reload on stale lazy-chunk errors after a deploy (`vite:preloadError`); every list has loading, empty (with a fix-it action) and error states; an unreachable API must not leave a full-screen loader on a public page.
- Mobile-first, 320px to wide desktop, 44px touch targets, `text-base` inputs on phones. Strict TypeScript, no `any`. Barrel `index.ts` per folder.
- UI conventions (icons, confirm dialogs, DataTable, dashboard header, controls) are in the frontend CLAUDE.md below. They are Showtime's, and are the default for every Fusewall product unless a product records a brand deviation.

# 5. Paystack rules

- Use integer **kobo** (Naira × 100). Currency comes from config; verify that `Verify` returns the amount and currency you expect before marking anything paid.
- **Flow:** `Initialize` (with your own unique `reference` and a `callback_url`) → redirect → on return **verify with Paystack** → mark paid. The confirmation page verifies on arrival and polls for a few minutes; never rely only on the webhook, and never only on the redirect.
- **One webhook handler per Paystack account**, at `/api/v1/paystack/webhook`. Paystack sends to a single URL per account, so route every payment type (tickets, store, wallet/league) through one handler that dispatches by reference. Verify `x-paystack-signature` = HMAC-SHA512 of the raw body with the secret key, with a constant-time compare, **before** parsing.
- `MarkPaid` is idempotent and makes confirmation side effects (email, ticket issue) happen **once**, whether the webhook, the callback or staff verification arrives first.
- Test keys in dev and prod keys in the server `.env` only. Before launch, confirm the webhook URL is actually set in the dashboard (Showtime ran for months with none and buyers were charged without getting tickets).
- **Multiple Fusewall products and one Paystack account:** only one webhook URL is allowed per account. Either use **one Paystack account per product** (preferred) or route through a single shared payments service. Raise this as a deviation entry before the second product goes live.

# 6. Server and ops rules

- **Adding a product to the existing Contabo box:** new `~/apps/<name>/` with its own compose and `.env`; a new database and least-privilege user in the shared Postgres; join the external `web` and `database` networks; a new host block in the shared Caddyfile; a Cloudflare Origin Certificate for the product's domain; a new R2 bucket (and a separate backup bucket scope); the product's domain verified in Resend (DKIM + SPF); a deploy key with a forced command; a new UptimeRobot monitor; add it to the backup and disk-check scripts. The shared host is a shared blast radius, so budget CPU, RAM and DB connections per product.
- **Firewall:** Docker bypasses UFW, so the Cloudflare allow-list lives in the `DOCKER-USER` chain. Rule 1 must be `! -i eth0 -j RETURN`, otherwise container egress (Resend, Paystack, R2, YouTube) is blocked too. **Run `sudo netfilter-persistent save` after every change**, or a provider reboot restores the old rules (this broke production egress on 2026-10-19). Verify from a genuinely external machine, not the server (hairpin).
- **Deploys** are health-gated: capture the running image, pull and start the new one, wait for Docker's `HEALTHCHECK`, then prune; on failure roll back automatically and email the alert with the failing container's logs. The source-controlled `scripts/deploy.sh` is a copy; the server's `~/scripts/deploy.sh` is what runs, so update both.
- **Dockerfile:** multi-stage, `CGO_ENABLED=0`, `linux/amd64`, alpine with `tzdata` + `ca-certificates`, non-root user, `HEALTHCHECK` hitting `/healthcheck`. Swagger docs are generated at build time and gitignored.
- Logs: cap container logs in `/etc/docker/daemon.json`, rotate script logs with logrotate, set quiet severity for 4xx.
- The uptime monitor runs outside the box. Alerts tell you something is wrong; Dozzle shows what.

# 7. Walking skeleton (build after the CLAUDE.md files)

One real vertical slice, chosen from the product's most valuable flow (usually "purchase with a Paystack payment and confirmation email"), through every layer:

1. Migration + domain type + pure rule with table tests.
2. Port + Postgres adapter + integration test on real Postgres.
3. Service using `TxManager.WithTx`, an idempotent confirm path, and fake-port tests.
4. Paystack adapter (`Initialize`/`Verify`/signature) and the single webhook handler, with replay and bad-signature tests.
5. Resend adapter behind `Mailer`, paced, sent from the worker pool, failure does not fail the purchase.
6. `platform.Batcher` writing audit logs, with size-flush, interval-flush and close-flush tests.
7. A cron job via the standard-parser `run()` wrapper that claims rows with `SKIP LOCKED`.
8. Handler, route, permission, error mapping, DTOs, Swagger annotations.
9. Frontend: login, one `DataTable` list with server pagination, one create flow with `Modal` + `ConfirmDialog`, `DashboardShell` + `DashboardPageHeader`, a payment-return page that verifies and polls.
10. Tooling: `Makefile`, `docker-compose` for local Postgres, the Dockerfile (§6), `.github/workflows` (test → build → ghcr → deploy), `.env.example`, `depguard` config, `scripts/` (deploy, backup, disk-check, alert).

# 8. Shared code: don't leak a product into the group library

Showtime's shared Go module (`pkg-common`) hardcodes `authorizationCookieKey = "Showtime_AccessToken"`. A shared Fusewall library must contain **no product names**: the cookie name, issuer, audience and domain come from config. If the new product needs the shared pieces (logger, PASETO maker, auth/rate-limit/recover/metrics middleware, CORS and error helpers), tell the owner which ones you'd reuse and whether to copy the module in or extract it into its own versioned repo first. Don't fork-and-diverge silently.

# 9. The files to create (copy exactly, replace `<PRODUCT>` with the repo name)

## 9.1 `/CLAUDE.md`

~~~~markdown
# <PRODUCT> — engineering guide (Fusewall standard)

<PRODUCT> is a Fusewall product. It follows the Fusewall stack and hosting model by default. Standing rules; follow without being asked. Detail: `backend/CLAUDE.md`, `frontend/CLAUDE.md`. Reasoning and deviations: `docs/DECISIONS.md` (read before changing anything architectural).

## Fusewall defaults — bias toward these
- Backend Go + Gin + pgx/pgxpool + goose + swag. Postgres 18 self-hosted on the shared Contabo VPS (one DB + least-privilege user per product).
- Frontend React 19 + Vite + TS strict + Tailwind v4 + TanStack Query/Table, hosted on Vercel.
- Email: **Resend**. Payments: **Paystack**. Files and backups: **Cloudflare R2**. Edge: **Cloudflare + Caddy**. Hosting: **Contabo + Docker Compose**. CI/CD: **GitHub Actions → ghcr.io → SSH forced-command deploy**. Uptime: UptimeRobot. Ops alerts: david@fusewall.africa.
- No Redis by default. No Kubernetes. No managed DB.
- **If a default won't fit, say so before building** ("Default X; need Y because Z"), then log it in `docs/DECISIONS.md` as a Deviation. Never substitute a provider silently.

## Hard rules
1. **Hexagonal backend.** domain ← services ← transport; services use `ports` (interfaces); `adapters` implement ports; SQL only in `adapters/postgres`; vendor SDKs only in their adapter.
2. **Constructor injection only.** No globals, `init()` wiring, lazy singletons. Only `cmd/*/main.go` knows concrete types.
3. **Pool everything, once.** One pgxpool, one HTTP client per provider, one bounded worker pool, all injected and env-sized. Shared Postgres means per-product connection budgets.
4. **Batch high-volume, low-criticality writes** (audit, analytics) with `platform.Batcher`. **Never batch money, stock or auth.** No N+1.
5. **Context + deadline on every I/O call.** No third-party call while a DB transaction is open.
6. **Idempotent, locked writes** for payments, webhooks and inventory. One Paystack webhook handler; signature verified first.
7. **A third party down must not block a page.** singleflight + stale-while-revalidate + back-off + inline empty state.
8. **Errors:** sentinel + wrapped, mapped to HTTP in one place, worded for the end user.
9. **Tests:** pure domain tests; service tests with fake ports; adapter tests on real Postgres; bug fix = regression test.
10. **Frontend:** `ui/` primitives, TanStack Query for server state, types in `types/`, thin pages, responsive 320px→desktop.
11. **Docs stay true:** a new standing rule goes into the right CLAUDE.md in the same change; a questionable choice gets a `docs/DECISIONS.md` entry.

## Working agreements
- Plan briefly, walking skeleton before breadth, every step compiles and is tested.
- Ask before: breaking migrations, new infrastructure or dependencies, deleting data, anything outward-facing (deploy, email, payment, firewall).
- Never commit or push unless asked. No AI/Claude attribution in commits or PRs.
- Definition of done: builds, lints, tests pass, migration included, docs updated, and what you did not verify is stated.
~~~~

## 9.2 `/backend/CLAUDE.md`

~~~~markdown
# Backend guide

## Layers
| Layer | May import | Must not import |
|---|---|---|
| `domain` | stdlib | everything else |
| `ports` | `domain` | adapters, services, transport |
| `services` | `domain`, `ports`, `platform`, `errors` | pgx, gin, net/http, vendor SDKs, SQL |
| `adapters/*` | `ports`, `domain`, its driver/SDK | services, transport |
| `transport/*` | `services`, `dto`, `errors` | adapters, pgx |
| `cmd/*` | everything | — |
`depguard` enforces this. A violation is a bug even if it compiles.

## Adding a feature (in this order)
migration → `domain` type + pure rules + tests → `ports` interface → `adapters/postgres` + integration test → `services` use case + fake-port tests → `dto` → `transport` handler + route + permission → frontend.

## Pooling
- pgxpool from env: `DB_MAX_CONNS`, `DB_MIN_CONNS`; lifetime 1h, idle 5m, health 30s, connect 5s. Shared Postgres: keep this product's share of `max_connections` small and documented.
- No pooler today ⇒ default protocol. Pooler in transaction mode ⇒ `QueryExecModeSimpleProtocol`.
- One `http.Client` per provider adapter with timeouts. Never `http.DefaultClient`.
- Worker pool: injected port, env capacity, overflow policy, `recover()` per task, drained on shutdown.
- Never call Paystack, Resend or R2 while a DB tx is open.

## Batching
- `platform.Batcher[T]`: N items or T seconds, bounded queue, `Close()` flushes, shutdown calls it before the DB closes. Bulk write = `CopyFrom` or multi-row `ON CONFLICT`.
- Filter without per-event DB reads. Bulk-load with `= ANY($1)`. Pace provider fan-out (Resend ≤ 8/s).

## Providers (each behind a port)
- **Resend** (`Mailer`): addresses from config only; paced; retry on rate-limit; sent from the worker pool; a failed email never fails the business tx. Sender domain = link domain, verified in Resend.
- **Paystack** (`PaymentGateway`): kobo integers; initialize → redirect → verify; ONE webhook `/api/v1/paystack/webhook`, HMAC-SHA512 `x-paystack-signature` checked first; idempotent `MarkPaid`; confirmation page verifies on arrival and polls.
- **R2** (`ObjectStore`): presigned uploads; GC behind an env flag, dry-run by default.

## Concurrency, errors, lifecycle
- Idempotency key per provider reference. `FOR UPDATE` in consistent order; `SKIP LOCKED` to claim jobs. Inject `Clock`. Money = integer minor units.
- Cron: `cron.ParseStandard` (5-field), fatal on a bad spec, jobs only call services.
- Sentinels in `internal/errors`, wrapped, one HTTP mapper. Config typed + validated at boot.
- Shutdown: HTTP → scheduler + jobs ctx → batchers + worker pool → in-memory state → DB.

## Security
- Authorise by permission, not role name. Rate-limit globally, tighter on auth/checkout. CORS allow-list. Sanitise stored HTML server-side. Swagger and metrics off or admin-only in prod. `/healthcheck` on GET and HEAD.

## Migrations and tests
- goose, numbered, embedded, forward-only, expand→migrate→contract; never edit an applied one; run before deploy.
- Domain: table tests. Services: fake ports. Adapters: real Postgres. `make test` is the gate.
~~~~

## 9.3 `/frontend/CLAUDE.md`

~~~~markdown
# Frontend guide

Apply every rule to the whole file you touch; responsiveness also applies to the components a page renders.

1. **No text icons.** No emoji or Unicode symbols as icons. Heroicons (`24/outline`, `24/solid` for active). Custom SVG only if no heroicon fits (outline, `currentColor`, `strokeWidth={1.5}`). Icon-only buttons use `IconButton` with a required `label`.
2. **Confirm key writes.** Delete/approve/revoke/publish/send/reset and logout go through `ConfirmDialog` (pending-action state, one dialog per page, names the record, `danger` tone for deletes and never silenceable). Never `confirm()`/`alert()`; results via `react-hot-toast`.
3. **Responsive.** Mobile-first Tailwind, 320px→wide. No horizontal page scroll, no fixed pixel widths, `min-w-0` on shrinking flex children, 44px touch targets, modals `max-h-[calc(100dvh-2rem)]`, safe-area insets. Check 320/375/768/1024/1440.
4. **Tables:** always `DataTable` (TanStack Table); columns defined once with `useMemo`; stable `data`; frozen first column names the row; actions in a `RowActions` menu; server-paginated lists pass server paging props. Add missing features to `DataTable`.
5. **Dashboards:** `DashboardShell` + a `DashboardPageHeader` on every page (title = sidebar label, one-sentence subtitle).
6. **First-time users:** plain labels, what-it-is subtitles, visible decision info, one primary action, helpful empty states, shared patterns.
7. **Controls** come from `components/ui/`. Never style a raw `<button>/<input>/<select>/<textarea>` or `fixed inset-0` overlay with page classes. Inputs are `text-base` on phones, `sm:text-sm` above.
8. **Types:** every API/domain type in `src/types/<domain>.ts`; none inline in components or `services/`.
9. **Barrels:** each folder re-exports via `index.ts`; add the export in the same change; cross-folder imports use the barrel. Lazy routes bypass barrels on purpose.
10. **Data:** TanStack Query hooks in `hooks/`; one service file per domain on a single axios instance; never copy server data into local state; `staleTime` chosen per query.
11. **Resilience:** route `ErrorBoundary`; reload on `vite:preloadError`; loading/empty/error on every list; an unreachable API never leaves a full-screen loader on a public page.
12. **Permissions:** gate UI with `usePermissions()`; the backend is the authority.
13. **Sitemap/SEO:** `public/sitemap.xml` lists top-level evergreen public pages only; update with route changes.

> Corner style, no-card tables and the icon set are Showtime's brand choices carried as the Fusewall default. A product with a different brand records a Deviation in `docs/DECISIONS.md`.
~~~~

## 9.4 `/docs/DECISIONS.md`

~~~~markdown
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
| 006 | CI/CD: GitHub Actions → ghcr.io → forced-command SSH deploy, health-gated | Fusewall standard | Accepted |
| 007 | Resend for email | Fusewall standard | Accepted |
| 008 | Paystack for payments, single webhook | Fusewall standard | Accepted |
| 009 | Cloudflare R2 for files and backups | Fusewall standard | Accepted |
| 010 | No Redis by default | Fusewall standard | Accepted |
| 011 | Single pgxpool, env-sized; bounded injected worker pool | Fusewall standard | Accepted |
| 012 | Write-behind batching for audit/analytics only | Fusewall standard | Accepted |
| 013 | Permission-based authorisation | Fusewall standard | Accepted |
| 014 | React + Vite + TanStack Query on Vercel | Fusewall standard | Accepted |

## Template
### NNN — Title
- **Type:** Fusewall standard | Product decision | Deviation from Fusewall standard
- **Status:** Proposed | Accepted | Superseded by NNN
- **Context:** the forces and constraints.
- **Decision:** one or two sentences.
- **Alternatives considered:** and why not.
- **Cost of deviating** (deviations only): extra operations, loss of shared tooling.
- **Consequences:** easier / harder / must remember.
- **Revisit when:** the trigger that reopens this.

## 003 — Self-hosted Postgres on the shared Contabo VPS
- **Type:** Fusewall standard · **Status:** Accepted
- **Context:** cost and control over tuning and backups; one operator; several products on one box.
- **Decision:** one Postgres 18 container, one database and one least-privilege user per product, loopback-only port, nightly offsite backups.
- **Alternatives considered:** managed Postgres (more cost, less control); an instance per product (more RAM overhead).
- **Consequences:** shared blast radius and maintenance window; connection budgets per product; restore drills are our job.
- **Revisit when:** a product's load or compliance needs outgrow a shared instance, or `max_connections` is regularly near the limit.

## 008 — Paystack for payments, single webhook
- **Type:** Fusewall standard · **Status:** Accepted
- **Context:** Nigerian market, NGN, card/transfer/USSD. Paystack allows one webhook URL per account.
- **Decision:** one handler dispatches every payment type; signature verified first; confirmation page also verifies and polls; `MarkPaid` idempotent.
- **Consequences:** a second product on the same Paystack account would collide on the webhook URL.
- **Revisit when:** a second product needs its own payments, or the product sells outside Paystack's countries/currencies.

(Fill the rest from `fusewall-architecture` as you implement them, with real numbers: pool sizes, batch size/interval, worker capacity, rate-limit settings.)
~~~~

# 10. Acceptance checklist (report each pass/fail with evidence)

- [ ] `/CLAUDE.md`, `/backend/CLAUDE.md`, `/frontend/CLAUDE.md`, `/docs/DECISIONS.md` exist and match what was built
- [ ] Every departure from §2 is listed in DECISIONS.md as a Deviation and was announced in chat before building
- [ ] No `pgx`, `gin`, `net/http` or vendor-SDK imports in `domain`, `ports`, `services`
- [ ] No package-level mutable singletons; `main` is the only composition root
- [ ] One `pgxpool` sized from env with timeouts; one HTTP client per provider; worker pool injected
- [ ] Batcher tests: size flush, interval flush, close flush, overflow policy
- [ ] Paystack: bad signature rejected, replayed webhook is a no-op, confirm side effects happen once
- [ ] Resend: paced, failure doesn't fail the purchase, sender domain = link domain
- [ ] Shutdown order matches §3.6
- [ ] Dockerfile is multi-stage, non-root, has a `HEALTHCHECK`; CI gates the build on tests
- [ ] Frontend builds with `strict`, no `any`, pages use `ui/` primitives only, checked at 320px
- [ ] Anything not verified is listed explicitly

# 11. After the scaffold

For every future task in this repo: re-read the relevant CLAUDE.md, follow the feature order in `backend/CLAUDE.md`, and treat a layering, pooling, batching or provider-bypass violation as a defect, not a trade-off. If you find a rule is wrong, propose a superseding DECISIONS entry instead of working around it.

=== PROMPT END ===

---

# Notes for David (not part of the prompt)

1. **"PaySack"** is read as **Paystack**, the provider in your code.
2. **Redis is not in the stack.** `redisSetup()` exists in `main_setup.go` but nothing calls it, so the standard says "no Redis by default".
3. **Open group-level decisions this surfaced.** I did not decide these for you:
   - **One Paystack webhook URL per account.** The second Fusewall product on the same Paystack account will collide with Showtime's webhook. Use one account per product, or build a shared payments service.
   - **`pkg-common` carries a product name** (`Showtime_AccessToken`). Before another product uses it, it should become a standalone, versioned Fusewall module with that value coming from config.
   - **Resend is probably one account for all products.** Each product needs its own verified domain, and the free daily cap and 10 req/s limit are shared.
   - **Everything is on one Contabo box.** That makes Postgres, Caddy and the firewall a single point of failure for every company. The playbook's scaling triggers in §5 say when to split.
   - **Dozzle has no auth** by your earlier choice. It now applies to every product's logs as well.
4. **Optional:** Claude Code reads `CLAUDE.md` from parent folders. If you put the root file in §9.1 in a common parent folder of all Fusewall repos, every product picks up the standard automatically. Your repos currently live in separate top-level folders (`~/Showtime/…`), so that would mean moving them.
