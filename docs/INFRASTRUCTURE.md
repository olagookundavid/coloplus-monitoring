# Infrastructure Runbook

How the Showtime backend is hosted, deployed, backed up, and monitored.

> **No secrets in this file.** Real values live in the server's `.env` files
> (`chmod 600`) and in each provider's dashboard. Nothing here is confidential.

---

## Architecture

```
              Internet
                 |
        +--------v---------+
        |  Cloudflare edge  |  Proxied (orange cloud). Public TLS cert,
        |  (proxied)        |  DDoS/WAF absorption, hides origin IP.
        +--------+----------+
                 | only Cloudflare's published IP ranges may pass (see below)
        +--------v---------+
        |      Caddy       |  TLS to Cloudflare via Origin Certificate
        |  (only public    |  routes by hostname
        |   container)     |
        +--------+---------+
                 | docker network: web
        +--------v---------+
        | showtime-backend |  Go API, listens :8080
        +--------+---------+
                 | docker network: database
        +--------v---------+
        |    postgres:18   |  NOT exposed to the internet
        |   (volume pgdata)|
        +------------------+
```

**Two TLS hops, two different certificates** — this trips people up, so: the
browser validates a normal publicly-trusted cert against **Cloudflare's edge**
(hop 1, Cloudflare manages this automatically). Cloudflare then validates a
**Cloudflare Origin Certificate** against **Caddy** (hop 2, SSL mode
`Full (strict)`) — a cert valid until 2041, trusted only by Cloudflare, never
by browsers directly (a direct `curl`/browser hit to the origin will correctly
show "untrusted certificate" — that's by design, not a bug).

- **Frontend**: Vercel (static Vite/React build). Talks to the API via
  `VITE_API_URL` (see `frontend/.env.production`). Baked at **build time** —
  changing it requires a rebuild/redeploy.
- **API**: `https://api.showtimeflag.football` -> Cloudflare -> Caddy -> backend.
- **DNS**: Cloudflare, `api` A + AAAA records -> server IP, **proxied (orange
  cloud)**. SSL/TLS mode is **Full (strict)**.

### Cloudflare orange-cloud setup (origin cert + firewall lock)

Going from grey (DNS-only) to orange (proxied) needed three things together —
skipping any one leaves a gap:

1. **Cloudflare Origin Certificate** installed in Caddy
   (`~/infra/caddy/certs/origin-cert.pem` + `origin-key.pem`, referenced via a
   `tls` directive in the Caddyfile instead of Caddy's automatic Let's Encrypt
   management). Valid until 2041 — no ACME renewal to think about for this
   domain. Caddy logs `skipping automatic certificate management because one
   or more matching certificates are already loaded` to confirm it's using
   the manual cert, not ACME.
2. **Cloudflare SSL/TLS mode = Full (strict)** — must be set *before* flipping
   DNS to proxied, or you can get an SSL error loop.
3. **Firewall locked to Cloudflare's IP ranges.** Flipping DNS alone does
   **not** hide the origin — anyone who already has the server's real IP can
   still hit it directly, bypassing Cloudflare's DDoS/WAF entirely. The actual
   enforcement lives in the `DOCKER-USER` iptables chain (see below), not UFW.

#### Why `DOCKER-USER`, not UFW, and not `INPUT`

Caddy's ports are published via standard Docker bridge networking
(`ports: ["80:80", "443:443"]`), **not** `network_mode: host`. For this setup:

- **UFW is not sufficient on its own.** Docker inserts its own iptables rules
  for published ports *ahead of* UFW's chain — a UFW rule can say "allow only
  Cloudflare" while Docker's own rules let everything else through regardless.
  This is a well-known Docker+UFW interaction, not a misconfiguration.
- **The correct enforcement point is the `DOCKER-USER` chain** (part of
  `FORWARD`), which Docker guarantees it will always consult for forwarded
  container traffic and will never silently overwrite. Rules there:
  ```bash
  # 1. ! -i eth0 -j RETURN   -> container egress skips the filter (never remove this)
  # 2. ACCEPT Cloudflare ranges on 80/443, from eth0
  # 3. ACCEPT RELATED,ESTABLISHED
  # 4. DROP 80/443 from eth0
  sudo iptables -L DOCKER-USER -n -v --line-numbers   # -v shows the "in" column
  ```
  Build or refresh the chain with the script in
  `CONTABO_OPERATIONS_MANUAL.md` §5.10. It rebuilds everything, saves it, and
  has a verify step for both directions.
- **Rule 1 matters.** Without it, the catch-all `DROP` also blocks the
  backend's own outbound HTTPS (email, R2, Paystack, YouTube), because
  `DOCKER-USER` sees container egress too.
- **Save after every change: `sudo netfilter-persistent save`.** At boot the
  server restores the last *saved* rules, not the last applied ones. In
  October 2026 an unsaved fix was lost on a provider reboot and all container
  egress broke again (manual Phases 20 and 22). On a new server, reboot once
  and re-verify before going live.

- **Cloudflare's IP ranges can change.** There is no auto-refresh cron for
  this yet (deliberately deferred) — re-run the allow-list loops periodically
  by hand, or set up a scheduled refresh.

#### The self-test gotcha (read this before "debugging" a false alarm)

Testing firewall changes **from the server against its own public IP**
(`bash -c "cat < /dev/null > /dev/tcp/<origin-ip>/443"` run *on* the server)
can show `TCP CONNECTED` even when the block is working correctly — this is a
"hairpin" connection (host talking to its own external address), which can
take a different kernel path than genuine external traffic (e.g. straight to a
`docker-proxy` userland-proxy socket bound to `0.0.0.0:443`, bypassing
`FORWARD`/`DOCKER-USER` entirely because it never leaves the host).
**The test that actually matters is run from a genuinely external machine**
(a laptop, not the server) hitting the origin IP directly. That's the one
that should show blocked/timeout.

### Docker networks

| Network    | Members                      | Purpose                                  |
| ---------- | ---------------------------- | ---------------------------------------- |
| `web`      | caddy, showtime-backend      | Public-facing tier; Caddy reaches apps    |
| `database` | postgres, showtime-backend   | Private DB tier; Postgres is never on `web` |

Both are **external** networks (`docker network create web` / `database`) so
multiple compose projects can join them.

**Key security property:** Postgres publishes only to `127.0.0.1:5432`
(loopback), never `0.0.0.0`. It is unreachable from the internet. Only
containers on the `database` network can talk to it.

---

## Server layout

| Path                    | What                                              |
| ----------------------- | ------------------------------------------------- |
| `~/infra/postgres/`     | Postgres compose + `.env` (superuser password)     |
| `~/infra/caddy/`        | Caddy compose + `Caddyfile`                       |
| `~/apps/showtime/`      | Backend compose + `.env` (all app secrets)        |
| `~/scripts/`            | `deploy.sh`, `pg-backup.sh`, `disk-check.sh`, `alert.sh` |
| `~/backups/postgres/`   | Local nightly dumps (14-day retention)            |
| `~/deploy.log`          | Deploy history                                     |
| `~/backups/backup.log`  | Backup history                                     |
| `~/backups/monitor.log` | Disk-check history                                 |

Convention: **shared infrastructure** under `~/infra/`, **apps** under `~/apps/`.

---

## Server hardening (already applied)

- SSH: **key-only**, root login disabled, password auth disabled.
  Enforced via `/etc/ssh/sshd_config.d/00-hardening.conf` (the `00-` prefix
  makes it win — sshd takes the **first** value it reads, and cloud-init's
  `50-cloud-init.conf` would otherwise re-enable passwords).
- `ufw`: default deny inbound; allows only **22, 80, 443**.
- `fail2ban`: bans brute-force SSH sources.
- `unattended-upgrades`: automatic security patches (auto-reboot left **off**
  deliberately, so DB restarts stay manual/observed).
- 2 GB swap file with `vm.swappiness=10` (OOM safety net; avoids the OOM killer
  abruptly killing Postgres).

> **Docker + UFW caveat:** Docker writes iptables rules *ahead* of UFW, so a
> published container port is reachable regardless of UFW. Our defense is
> structural: only Caddy publishes public ports; Postgres binds to loopback only.

---

## Deploying

Deploys are automatic: **push to `main` -> image builds -> server deploys.**

```
git push origin main
   -> .github/workflows/deploy-backend.yml
      -> build-and-push : builds image, pushes to ghcr.io
      -> deploy         : SSHes to the server, runs ~/scripts/deploy.sh
```

The workflow only triggers on backend-relevant paths (`backend/**`,
`pkg-common/**`, `Dockerfile`, `go.work*`). Doc-only commits don't rebuild.

### Other ways to deploy

| Need                                       | Command                                   |
| ------------------------------------------ | ----------------------------------------- |
| Rebuild + redeploy current `main`          | `gh workflow run deploy-backend.yml`      |
| Redeploy the existing image (no rebuild)   | on the server: `~/scripts/deploy.sh`      |
| Apply `.env` changes                       | `cd ~/apps/showtime && docker compose up -d --force-recreate` |

### How the deploy is secured

CI authenticates with a **dedicated SSH key** whose `~/.ssh/authorized_keys`
entry has a **forced command**:

```
command="/home/<user>/scripts/deploy.sh",no-port-forwarding,no-agent-forwarding,no-X11-forwarding,no-pty ssh-ed25519 AAAA... github-actions-deploy
```

That key **cannot do anything except run the deploy script** — no shell, no
port forwarding. The deploy logic lives on the server, not in CI. GitHub
secrets used: `DEPLOY_SSH_KEY`, `DEPLOY_HOST`, `DEPLOY_USER`.

> `scripts/deploy.sh` in this repo is the **reference copy**. The copy that
> runs is `~/scripts/deploy.sh` on the server. They are **not** auto-synced —
> update both when changing deploy logic.

---

## Logs

| What                | Where                                                        |
| ------------------- | ------------------------------------------------------------ |
| Deploy output (CI)  | Actions -> the run -> `deploy` job                            |
| Deploy history      | `tail -f ~/deploy.log`                                        |
| Backend app logs    | `docker compose -f ~/apps/showtime/docker-compose.yml logs -f backend` |
| Postgres logs       | `docker compose -f ~/infra/postgres/docker-compose.yml logs -f` |
| Caddy / TLS logs    | `docker compose -f ~/infra/caddy/docker-compose.yml logs -f`  |
| Backup history      | `tail ~/backups/backup.log`                                   |

Docker container logs are capped via `/etc/docker/daemon.json`
(`max-size: 10m`, `max-file: 3`). Script logs rotate via
`/etc/logrotate.d/showtime`. Log caps apply at container **creation**, so a
recreate is needed for them to take effect on existing containers.

---

## Backups

- **What**: `pg_dump -Fc` (compressed custom format) of the `showtime` database.
- **When**: nightly 03:00 via cron (`~/scripts/pg-backup.sh`).
- **Where**: locally in `~/backups/postgres/` (14-day retention) **and** offsite
  to Cloudflare R2 bucket `showtime-db-backups` via `rclone` remote `r2backup`.
- **Alerting**: the script has an `ERR` trap — any failure emails ops via
  `~/scripts/alert.sh`. A silently-failing backup is worse than none.

The R2 backup bucket is **private** (no custom domain, no public access) and
uses its own **bucket-scoped API token**, separate from the app's R2 keys — so
an app compromise cannot reach or delete backups.

> `rclone` is configured with `no_check_bucket=true` because a bucket-scoped
> token can't perform account-level operations like `ListBuckets`. Listing all
> buckets (`rclone lsd r2backup:`) returning 403 is **expected and correct**;
> use `rclone ls r2backup:showtime-db-backups` instead.

### Restore

```bash
# fetch a dump
rclone copy r2backup:showtime-db-backups/showtime-YYYYMMDD-HHMMSS.dump ~/

# restore into the DB (as the app user, so it owns the objects)
SHOWTIME_PW="$(tail -n1 ~/infra/postgres/showtime-db-password.txt)"
docker exec -i postgres sh -c \
  "PGPASSWORD='$SHOWTIME_PW' pg_restore -h 127.0.0.1 -U showtime -d showtime --clean --if-exists" \
  < ~/showtime-YYYYMMDD-HHMMSS.dump
```

For a full reset-and-restore, stop the backend first, then
`DROP DATABASE ... WITH (FORCE)` / `CREATE DATABASE showtime OWNER showtime`
before restoring.

**An untested backup is not a backup** — periodically restore into a throwaway
database (`CREATE DATABASE showtime_restore_test;`) to prove it works.

---

## Monitoring & alerting

| Layer          | Tool                                                          |
| -------------- | ------------------------------------------------------------- |
| Uptime         | UptimeRobot -> `https://api.showtimeflag.football/api/v1/healthcheck` (5 min) |
| Disk space     | `~/scripts/disk-check.sh` via cron every 6h; alerts at **80%** |
| Backup failure | `ERR` trap in `~/scripts/pg-backup.sh`                        |
| Alert channel  | `~/scripts/alert.sh` -> email via Resend                       |

The uptime monitor is **external on purpose** — a monitor on the server dies
with the server.

> The healthcheck answers both **GET and HEAD** ([routes.go](../backend/internal/routes/routes.go)).
> UptimeRobot's free tier probes with **HEAD**, and Gin does not auto-handle
> HEAD on GET-only routes — it would 404. Keep the HEAD route registered.

---

## Database access (Beekeeper / GUI)

Postgres is not exposed publicly. Connect through an **SSH tunnel**:

- **DB fields**: host `127.0.0.1`, port `5432`, user `showtime`, database
  `showtime`, password from `~/infra/postgres/showtime-db-password.txt`
- **SSH tunnel**: your server host/port 22, your user, key file `~/.ssh/id_ed25519`

Or manually: `ssh -L 5432:localhost:5432 <alias>` then connect to
`localhost:5432` with the tunnel toggle off.

Log in as `showtime` (least privilege), not the `postgres` superuser.

---

## Adding another app

Each app is its own container routed by **hostname**, not port. Apps don't
publish host ports, so several apps can all listen on `:8080` internally.

1. DNS: `app2.showtimeflag.football` A record -> server IP (grey cloud).
2. Give the app its own `~/apps/app2/` compose; join networks `web` (for Caddy)
   and `database` (only if it needs Postgres).
3. Its own database + least-privilege user inside the shared Postgres:
   ```sql
   CREATE USER app2 WITH PASSWORD '...';
   CREATE DATABASE app2 OWNER app2;
   \c app2
   REVOKE ALL ON SCHEMA public FROM PUBLIC;
   GRANT ALL ON SCHEMA public TO app2;
   ```
4. Add a Caddyfile block and reload Caddy:
   ```
   app2.showtimeflag.football {
       reverse_proxy app2-backend:8080
   }
   ```

> Trade-off in the current design: all app backends share the `database`
> network, so they can technically reach each other. For stricter isolation,
> give each app its own network and attach the Postgres container to each one
> individually (a container can be on many networks; the apps then can't see
> each other).

---

## Where secrets live

Nothing secret is in this repo.

| Secret                              | Location                                    |
| ----------------------------------- | ------------------------------------------- |
| App secrets (`TOKEN_KEY`, Paystack, Resend, R2, `DB_URL`) | `~/apps/showtime/.env` (chmod 600) |
| Postgres superuser password         | `~/infra/postgres/.env`                     |
| `showtime` DB user password         | `~/infra/postgres/showtime-db-password.txt` |
| Backup R2 token                     | `~/.config/rclone/rclone.conf` (chmod 600)  |
| CI deploy key                       | GitHub secret `DEPLOY_SSH_KEY`; public half in server `authorized_keys` |

After editing `~/apps/showtime/.env`:
`cd ~/apps/showtime && docker compose up -d --force-recreate`

> Rotating `TOKEN_KEY` invalidates every existing session (all users logged
> out). Do it during low traffic.

---

## Gotchas & lessons learned

- **Pin the Postgres major version** (`postgres:18`, never `latest`). Major
  versions can't read the previous version's data files — an accidental bump
  means the DB won't start. `postgres:18` still gets safe minor/patch updates.
- **URL-safe DB passwords.** `DB_URL` is a URL: a password containing `/`, `+`,
  or `=` (as `openssl rand -base64` produces) breaks parsing. Use
  `openssl rand -hex 24`.
- **No `uuid-ossp` dependency.** Migrations use the built-in `gen_random_uuid()`.
  Managed providers often pre-install `uuid-ossp`; a vanilla Postgres does not,
  so `uuid_generate_v4()` would fail on a fresh DB.
- **Swagger docs are generated, not committed** (`backend/.gitignore` has
  `/docs`). Anything building the backend must run `swag init` before
  `go build` — both the Dockerfile and `ci.yml` do.
- **Auth cookies are `Secure` + `SameSite=None`**, so login only works over
  HTTPS. `COOKIE_DOMAIN` is intentionally **unset** -> host-only cookie scoped
  to the API's host, which is what we want.
- **`sslmode=disable` in `DB_URL` is correct here** — backend-to-Postgres
  traffic never leaves the private Docker network on one host.
- **Firewall changes aren't permanent until saved.** `iptables` edits apply
  instantly but only live in memory; run `sudo netfilter-persistent save` or
  the next reboot reverts them. Check rules with `iptables -L … -n -v`: without
  `-v` the interface column is hidden, and a broken rule looks fine.
- **The logger's `env:"dev"` label is hardcoded** in `main.go` and unrelated to
  `IS_PROD`. Ignore it; real prod mode is driven by `IS_PROD=true`.

---

## Troubleshooting

| Symptom                     | Check                                                        |
| --------------------------- | ------------------------------------------------------------ |
| Site down                   | `docker compose ps` in each dir; backend logs; UptimeRobot incident |
| TLS / cert errors           | Caddy logs. DNS must be **grey cloud**; port 80 must be reachable for renewal |
| Backend can't reach DB      | Is it on the `database` network? Is `DB_URL` host `postgres` (not an IP)? |
| Deploy failed               | Actions `deploy` job log; `~/deploy.log`; `which docker` path in the forced command |
| Disk filling                | `docker image prune -a`; old dumps in `~/backups/postgres`; log rotation applied? |
| Migration failed on boot    | Backend logs. goose runs in a transaction and has no "dirty" state — fix the cause and restart; it retries. |
| Postgres won't start after image change | Did the major version change? Restore from backup / revert the tag. |
| Backend outbound calls time out (email, uploads, payments, YouTube), often after a reboot | `docker exec showtime-backend wget -T 10 -O /dev/null https://www.google.com`. If it hangs: `DOCKER-USER` rule 1 (`! -i eth0 -j RETURN`) is missing. Re-add it, then `sudo netfilter-persistent save` (manual §5.10) |

---

## Maintenance calendar

- **Automatic**: TLS renewal (Caddy, ~60d), OS security patches, nightly
  backups, log rotation.
- **Occasional**: verify a test restore; review disk usage; Postgres **major**
  upgrades are deliberate (dump + restore, never in-place).
- **After any reboot** (yours or the provider's): run the manual's §5.10
  verify step: outbound from the container, and the origin still blocked
  from outside.
