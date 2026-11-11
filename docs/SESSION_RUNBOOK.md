# Test-session runbook (David)

For the session on 4 November 2026. The test plan page is what the testers follow; this is what you do around it. Nothing here is a product rule, and nothing is committed to the server.

## Before the session

1. Server `.env` has `WEEKLY_DIGEST_ENABLED=true`, `SLA_TARGET_PERCENT=99.5`, `RETENTION_ENABLED=true`, a live `RESEND_API_KEY` and a verified `MAIL_FROM`; then `docker compose up -d` in `~/apps/coloplus`.
2. `curl -s https://api.remotems.com/healthcheck` answers `{"status":"ok"}`; the front end loads at https://www.remotems.com and you can sign in.
3. Load the ten `TEST-` sites (the CSV is on the plan's Test data tab): the site import screen, or `POST /sites/import` with `{"csv": "...", "dry_run": true}` first.
4. Invite the users (Users screen): two supervisors, two reporters, the customer reviewer (role `reviewer`), an optional `viewer`, a spare account for the lockout test, and **one more supervisor for yourself** (see below).
5. Sites: choose them when you invite (the Sites list in the invite form), or later from Users, open the person, Edit sites (Select all and Clear help with long lists). Site assignments also has a field officer and a supervisor picker per site. Supervisors get the sites on the Test data tab; the reviewer gets the eight sites that are not Kano or Port Harcourt.
6. Send one test invitation to yourself first and confirm the link opens the set-password page on www.remotems.com.
7. **Sample history (optional, once everyone is invited and assigned).** On the server, in `~/apps/coloplus`:

   ```bash
   docker compose run --rm --no-deps coloplus-backend ./server seed-demo \
     --reporter=<reporter email> \
     --supervisors=<your supervisor email>,<teammate supervisor email> \
     --reviewer=<reviewer email>
   ```

   (The same one-off form the deploy uses for `./server migrate`; run it after the deploy that contains this command has finished.) It stops with a plain message if an email is not invited, has the wrong role, or is not assigned to TEST-LAG-001, 002 and 003. Running it again changes nothing. It sends no email and leaves today empty. The audit log will name those people as the authors of the sample history (decision 073), so it is only for a database that gets wiped before real data goes in.

## Why you need a supervisor login of your own

Coloplus's side of disputes, support tickets and acceptance belongs to the **supervisor** role (`disputes.manage`, `tickets.manage`, `acceptance.manage`); the administrator cannot do them. The front end now has screens for all three (`pages/ops/SupervisorDisputes.tsx`, `SupervisorTickets.tsx`, and acceptance embedded in `SupervisorSites.tsx`) — the commands below are still useful for a quick command-line walkthrough, but the portal works too. Invite yourself as an extra supervisor with a second email address, finish two-step setup, and give that account the Lagos sites and TEST-ENU-001. Then use the commands below, or the portal, for tests C8, C9 and C10.

## Commands

Needs `curl` and `jq`. The cookie file holds your session, so keep it out of the repository.

```bash
API=https://api.remotems.com/api/v1
JAR=$HOME/.coloplus-session.jar
login() {
  read -r -p "Email: " E; read -rs -p "Password: " P; echo; read -r -p "Authenticator code: " C
  curl -sS -c "$JAR" -H 'Content-Type: application/json' \
    -d "{\"email\":\"$E\",\"password\":\"$P\",\"totp_code\":\"$C\"}" "$API/auth/login" | jq .
}
call() { curl -sS -b "$JAR" -c "$JAR" -H 'Content-Type: application/json' "$@"; }
```

Run `login` once; the session lasts about twelve hours. Every write needs the JSON header, which `call` adds.

### C8: a dispute, Coloplus's side

```bash
call "$API/disputes" | jq .                       # find the dispute id
D=<dispute id>
call -X POST "$API/disputes/$D/transition" -d '{"status":"under_review","note":"We are checking the report against the site log."}' | jq .
call -X POST "$API/disputes/$D/transition" -d '{"status":"awaiting_evidence","note":"Please send your NOC log extract."}' | jq .
# The reviewer comments in the portal. Then:
call -X POST "$API/disputes/$D/comments" -d '{"body":"Thanks, we have your log.","internal":false}' | jq .
call -X POST "$API/disputes/$D/transition" -d '{"status":"under_review","note":"Evidence received."}' | jq .
call -X POST "$API/disputes/$D/transition" -d '{"status":"resolved","outcome":"not_upheld","note":"The report matches the site log."}' | jq .
```

The reviewer then acknowledges and closes it in the portal. To show an upheld dispute: have Reporter A request a correction and Supervisor A approve it, then send `{"status":"agreed_correction","note":"...","resolution_report_id":"<the new report id>"}` before `resolved` with `"outcome":"upheld"`. A step the rules do not allow is refused with the reason.

### C9: a support ticket

```bash
call "$API/tickets" | jq .
T=<ticket id>
call -X POST "$API/tickets/$T/transition" -d '{"status":"in_progress","note":"Checking with the site team."}' | jq .
call -X POST "$API/tickets/$T/comments" -d '{"body":"The key holder is the Lekki site manager.","internal":false}' | jq .
call -X POST "$API/tickets/$T/comments" -d '{"body":"Internal: confirm with Ops.","internal":true}' | jq .   # the customer must never see this one
call -X POST "$API/tickets/$T/transition" -d '{"status":"resolved","note":"Answered."}' | jq .
```

### C10: acceptance of TEST-ENU-001

```bash
S=$(call "$API/sites" | jq -r '.. | objects | select(.customer_site_code? == "TEST-ENU-001") | .id' | head -1); echo "$S"
call -X POST "$API/sites/$S/acceptance" -d '{"next_action":"Finish the build","due_at":"2026-11-18T00:00:00Z"}' | jq .
call -X POST "$API/sites/$S/acceptance/transition" -d '{"stage":"built","note":"Build complete."}' | jq .
call -X POST "$API/sites/$S/acceptance/transition" -d '{"stage":"ready_for_inspection","note":"Dossier submitted."}' | jq .
```

Then the reviewer marks it inspected, then accepted, in the portal. A site that reaches accepted becomes active.

## Known gaps to say out loud

- The report form now sends the outage reason and the late reason (`docs/FRONTEND.md`'s "Do these first" section 0 is clear of both). Test R5 and the outage-day tests (C6, C7, E3, E5) no longer need the API stand-in.
- Everything else not in this release is listed on the plan's first tab.

## After the session

Copy each person's results (Sign-off tab) into one place, list the failures with owners, and fix the Blocked ones first. Wipe the database before real data goes in: drop it, `api migrate`, then `api create-admin`. The audit log cannot be edited, so do not delete rows one by one.
