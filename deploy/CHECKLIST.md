# Bring up api.remotems.com: tick as you go

Command blocks are in `deploy/README.md` (section numbers in brackets).
Order matters: later steps need earlier ones.

## A. Outside the server (start these first, some have waiting time)

- [ ] **A1.** Confirm the hostname is `api.remotems.com`.
- [ ] **A2.** Cloudflare: `remotems.com` is an active zone on your account.
- [ ] **A3.** Cloudflare SSL/TLS mode = **Full (strict)** (before any orange cloud).
- [ ] **A4.** Cloudflare, SSL/TLS, Origin Server: create an Origin Certificate for `remotems.com` and `*.remotems.com`. Keep the cert and the key text open for B2.
- [ ] **A5.** Resend: add `remotems.com`, add the DKIM/SPF DNS records it shows in Cloudflare (leave them DNS-only/grey), wait for "Verified" (can take minutes to hours).
- [ ] **A6.** Resend: create an API key. Note the sender, e.g. `alerts@remotems.com`.
- [ ] **A7.** Front end origin is `https://www.remotems.com` (decided). It is only config, so the front end need not exist yet. Keep it on `remotems.com` so the cookie stays same-site (`lax`).

## B. Server preparation

- [ ] **B1.** SSH in. Check `docker ps` shows `caddy` and `postgres`, and `docker network ls` shows `web` and `database`.
- [ ] **B2.** Save the A4 cert/key as `~/infra/caddy/certs/remotems-cert.pem` and `remotems-key.pem`; `chmod 600` the key. Confirm Caddy's compose mounts that folder at `/etc/caddy/certs`.
- [ ] **B3.** Create the `coloplus` database and user [README 2]. Password file saved, `chmod 600`.
- [ ] **B4.** `mkdir -p ~/apps/coloplus`; copy `deploy/docker-compose.yml` into it.
- [ ] **B5.** Create `~/apps/coloplus/.env` from `deploy/.env.production.example`; `chmod 600`.
  - [ ] `DATABASE_URL` uses the B3 password
  - [ ] `TOKEN_SYMMETRIC_KEY` and `SECRETS_KEY` (two different `openssl rand -hex 16`)
  - [ ] `SECRETS_KEY` copied to your password manager
  - [ ] `CORS_ALLOW_ORIGINS` and `APP_BASE_URL` = A7
  - [ ] `RESEND_API_KEY` and `MAIL_FROM` = A6
- [ ] **B6.** `cd ~/apps/coloplus && docker compose config` prints without error.
- [ ] **B7.** Extend `~/scripts/pg-backup.sh` to also dump `coloplus`; run it once and check the dump exists (local and R2).
- [ ] **B8.** `docker login ghcr.io` on the server with a token that has `read:packages` (needed because the image is private). Test later with `docker compose pull`.

## C. Deploy pipeline

- [ ] **C1.** Copy `deploy/deploy.sh` to `~/scripts/coloplus-deploy.sh`; `chmod +x`.
- [ ] **C2.** Confirm `~/scripts/alert.sh` takes `subject body` (the script assumes so).
- [ ] **C3.** Make a new SSH key pair for CI. Add the public half to `~/.ssh/authorized_keys` with the forced command pointing at `coloplus-deploy.sh` [README 5].
- [ ] **C4.** GitHub repo secrets: `DEPLOY_KEY` (private key), `DEPLOY_KNOWN_HOSTS` (`ssh-keyscan <host>`), `DEPLOY_USER`, `DEPLOY_HOST`.
- [ ] **C5.** Decide with Claude what to commit: `deploy/` and the large uncommitted backend work are not in git, and CI builds from `main`.
- [ ] **C6.** Set repo variable `DEPLOY_ENABLED=true`, then push to `main`. CI runs tests, builds, pushes to ghcr, and runs the deploy.
- [ ] **C7.** On the server: `docker ps` shows `coloplus-backend` as `healthy`; `tail ~/coloplus-deploy.log` shows migrations applied. If it failed, you got an alert email and the log has the reason.

## D. Make it public

- [ ] **D1.** Cloudflare DNS: `A` record `api` -> server IP, **Proxied (orange)**; add `AAAA` too if the box has IPv6.
- [ ] **D2.** Append the Caddy block, validate, then reload (not restart) [README 4].
- [ ] **D3.** `curl -s https://api.remotems.com/healthcheck` answers.

## E. Verify and watch

- [ ] **E1.** From your laptop (not the server): `curl -sk --connect-timeout 8 https://<origin-ip>/` times out (origin locked to Cloudflare).
- [ ] **E2.** Swagger is off: `https://api.remotems.com/swagger/index.html` does not load.
- [ ] **E3.** UptimeRobot monitor on `https://api.remotems.com/healthcheck`.
- [ ] **E4.** Test email: invite a user, confirm the mail arrives and the link points at the front end origin.
- [ ] **E5.** Dozzle: no customer data in the logs, or put Dozzle behind a login before real data (DECISIONS 017).
- [ ] **E6.** Add a restore test of the `coloplus` dump to the calendar.
