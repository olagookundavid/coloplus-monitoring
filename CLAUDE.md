# Coloplus: engineering guide (Fusewall standard)

Coloplus is the Remote Monitoring Portal for Coloplus power services (customer: Airtel). It is a Fusewall product and follows the Fusewall stack and hosting model by default. These are standing rules; follow them without being asked. Detail: `backend/CLAUDE.md`. Reasoning and deviations: `docs/DECISIONS.md` (read it before changing anything architectural). The product spec is `docs/Coloplus_Remote_Monitoring_PRD.docx`; the standard is `docs/architecture-kit/fusewall-architecture.md`.

## Repo layout
- `backend/`: Go API (hexagonal). The current focus.
- `frontend/`: reserved, empty. The front-end engineer adds the React app and its `CLAUDE.md` (Fusewall conventions, section 9.3 of the standard).
- `pkg-common/`: shared Go module (PASETO tokens, logger, auth/recover/rate-limit middleware) that the backend builds against through `go.work`. Any other app added to this repo uses it too. **No product names in it** (cookie names etc. come from config), **only add to it, never change what an existing function does**.
- `docs/`: PRD, architecture standard, `DECISIONS.md`.

## Fusewall defaults: bias toward these
- Backend Go + Gin + pgx/pgxpool + goose + swag. Postgres 18 self-hosted on the shared Contabo VPS (one DB + least-privilege user per product).
- Frontend React 19 + Vite + TS strict + Tailwind v4 + TanStack Query/Table, hosted on Vercel.
- Email: **Resend**. Files and backups: **Cloudflare R2**. Edge: **Cloudflare + Caddy**. Hosting: **Contabo + Docker Compose**. CI/CD: **GitHub Actions to ghcr.io to SSH forced-command deploy**. Uptime: UptimeRobot. Ops alerts: david@fusewall.africa.
- No Paystack: Coloplus takes no payments (DECISIONS 015). No Redis. No Kubernetes. No managed DB.
- **If a default won't fit, say so before building** ("Default X; need Y because Z"), then log it in `docs/DECISIONS.md` as a Deviation. Never substitute a provider silently.

## Hard rules
1. **Hexagonal backend.** domain <- services <- transport; services use `ports` (interfaces); `adapters` implement ports; SQL only in `adapters/postgres`; vendor SDKs only in their adapter. Enforced by depguard and `tests/unit/architecture_test.go`.
2. **Constructor injection only.** No globals, `init()` wiring, lazy singletons. Only `cmd/*/main.go` knows concrete types.
3. **Pool everything, once.** One pgxpool, one HTTP client per provider, one bounded worker pool, all injected and env-sized. Shared Postgres means a per-product connection budget (`DB_MAX_CONNS`).
4. **Batch high-volume, low-criticality writes** (audit, analytics) with `platform.Batcher`. **Never batch money, stock or auth.** No N+1. Approvals, corrections and publications write their audit row in the same transaction (`AuditService.RecordInTx`).
5. **Context + deadline on every I/O call.** No third-party call while a DB transaction is open.
6. **Idempotent, locked writes** for anything that can be retried or raced.
7. **A third party down must not block a page.** singleflight + stale-while-revalidate + back-off.
8. **Errors:** sentinel (`internal/errors`) + wrapped, mapped to HTTP in one place (`transport/http/errors.go`), worded for the end user.
9. **Tests:** pure domain tests; service tests with fake ports; adapter tests on real Postgres; a bug fix gets a regression test.
10. **PRD trust rules:** a missing report is never shown as "power available"; every derived figure carries provenance; approval needs a second person; the audit log is append-only.
11. **Docs stay true:** a new standing rule goes into the right CLAUDE.md in the same change; a questionable choice gets a `docs/DECISIONS.md` entry.

## Commands
`make db/up` (local Postgres) · `make run` · `make test` (unit) · `make test/integration` · `make test/all` · `make lint` · `make migrate` · `make help`.

## Working agreements
- Plan briefly, walking skeleton before breadth, every step compiles and is tested.
- Ask before: breaking migrations, new infrastructure or dependencies, deleting data, anything outward-facing (deploy, email, firewall).
- Never commit or push unless asked. No AI/Claude attribution in commits or PRs.
- Definition of done: builds, lints, tests pass, migration included, docs updated, and what you did not verify is stated.
