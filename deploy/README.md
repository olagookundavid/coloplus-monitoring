# Deploying Coloplus to the Contabo box

Follows `docs/CONTABO_OPERATIONS_MANUAL.md` §6 ("adding a new container/app"),
adapted for a **different domain** (`remotems.com`, not the `showtimeflag.football`
zone the existing wildcard cert covers). Files here are the source-controlled
copies; the ones that run live on the server.

| Repo file | Server location |
| --- | --- |
| `deploy/docker-compose.yml` | `~/apps/coloplus/docker-compose.yml` |
| `deploy/.env.production.example` | `~/apps/coloplus/.env` (filled in, `chmod 600`) |
| `deploy/deploy.sh` | `~/scripts/coloplus-deploy.sh` (CI forced command) |

Values used below (change if you pick another hostname):
`API_HOST=api.remotems.com`, container `coloplus-backend`, internal port `8080`,
health route `/healthcheck`.

## 1. Before touching the server (outside the box)

1. **Cloudflare:** `remotems.com` must be a zone on the same Cloudflare account.
   Set SSL/TLS mode to **Full (strict)** *before* proxying.
2. **Origin Certificate for `remotems.com`:** Cloudflare, SSL/TLS, Origin Server,
   create for `remotems.com` and `*.remotems.com`. Save as
   `~/infra/caddy/certs/remotems-cert.pem` and `remotems-key.pem` on the server
   (`chmod 600` the key). The existing `origin-cert.pem` does not cover this domain.
3. **DNS:** `A` (and `AAAA`) record `api` -> server IP, **Proxied (orange)**.
   No firewall change: `DOCKER-USER` allows Cloudflare ranges on 80/443 for all hostnames.
4. **Resend:** verify the sending domain (DKIM + SPF) and create an API key.
   Production will not start with `MAIL_PROVIDER=log`.
5. **Front end origin:** needed for `CORS_ALLOW_ORIGINS` and `APP_BASE_URL`
   (https, exact origin). Production refuses to start without them.

## 2. Server

```bash
# Database (least privilege, shared Postgres). Hex password: URL-safe.
PW="$(openssl rand -hex 24)"
printf 'coloplus DB password:\n%s\n' "$PW" > ~/infra/postgres/coloplus-db-password.txt
chmod 600 ~/infra/postgres/coloplus-db-password.txt
docker exec -i postgres psql -U postgres <<SQL
CREATE USER coloplus WITH PASSWORD '${PW}';
CREATE DATABASE coloplus OWNER coloplus;
\c coloplus
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO coloplus;
SQL

# App directory: copy deploy/docker-compose.yml and fill in .env
mkdir -p ~/apps/coloplus && cd ~/apps/coloplus
nano .env && chmod 600 .env        # from deploy/.env.production.example
docker compose config >/dev/null && echo "YAML valid"
```

**Backups gap:** `~/scripts/pg-backup.sh` only dumps the `showtime` database.
Extend it to include `coloplus` **before go-live** (and add `coloplus-db-password.txt`
and `SECRETS_KEY` to wherever secrets are kept; losing `SECRETS_KEY` disables
two-step login for everyone).

## 3. First start (manual, before wiring CI)

```bash
# the image must exist first: push to main with DEPLOY_ENABLED=true builds it,
# or build/push by hand. ghcr package is private: `docker login ghcr.io` on the box.
cd ~/apps/coloplus
docker compose pull
docker compose run --rm --no-deps coloplus-backend ./server migrate
docker compose up -d && docker compose ps     # wait for "healthy"
```

## 4. Caddy route

```bash
cat >> ~/infra/caddy/Caddyfile <<'CADDY'

api.remotems.com {
	tls /etc/caddy/certs/remotems-cert.pem /etc/caddy/certs/remotems-key.pem
	reverse_proxy coloplus-backend:8080
}
CADDY
docker compose -f ~/infra/caddy/docker-compose.yml exec caddy caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
docker compose -f ~/infra/caddy/docker-compose.yml exec caddy caddy reload   --config /etc/caddy/Caddyfile --adapter caddyfile
curl -s https://api.remotems.com/healthcheck
```

## 5. CI/CD (own key, one job)

1. `cp deploy/deploy.sh ~/scripts/coloplus-deploy.sh && chmod +x ~/scripts/coloplus-deploy.sh`
2. New dedicated key, forced command in `~/.ssh/authorized_keys`:
   `command="/home/<user>/scripts/coloplus-deploy.sh",no-port-forwarding,no-agent-forwarding,no-X11-forwarding,no-pty ssh-ed25519 AAAA... coloplus-deploy`
   Do not reuse the Showtime key; it is locked to Showtime's script.
3. GitHub repo secrets: `DEPLOY_KEY`, `DEPLOY_KNOWN_HOSTS`, `DEPLOY_USER`, `DEPLOY_HOST`.
   Repo variable `DEPLOY_ENABLED=true` turns on build + deploy in `ci.yml`.
4. UptimeRobot: new monitor on `https://api.remotems.com/healthcheck` (GET or HEAD both answer).

## 6. Verify

- From a laptop, not the server: `curl -sk --connect-timeout 8 https://<origin-ip>/` must time out;
  `curl -s https://api.remotems.com/healthcheck` must answer.
- Sign in from the front end: cookie is `Secure` + `SameSite=None`, so https only.
- Logs: Dozzle has no login (DECISIONS 017). Keep customer data out of logs, or put it behind a login before go-live.
