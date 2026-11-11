# Coloplus quick test: 15 to 20 minutes

Portal: https://www.remotems.com · API health: https://api.remotems.com/healthcheck

Everything entered is test data and will be wiped before real use. Mark each step **P** (pass), **F** (fail) or **B** (blocked), and for every F write the screen, who you were signed in as, and the exact message on screen.

## Who is signed in

| Tab | Account | Role | Used for |
|---|---|---|---|
| 1 | David | admin | Users, sites, audit log |
| 2 | David (second email) | supervisor | Approve, incidents, disputes, tickets, acceptance |
| 3 | Teammate | supervisor | The second approver |
| 4 | Reporter | reporter | Submit reports, request maintenance |
| 5 | Reviewer | reviewer (customer) | Sees approved data only, raises disputes and tickets |

Use five separate browser profiles or private windows, because one browser holds one session.

## Before you start (once)

- [ ] Health check answers `{"status":"ok"}`.
- [ ] Sites are loaded: Sites screen, Import, choose `docs/test-data/test-sites.csv`. The preview must show 10 creates and 0 errors; then confirm.
- [ ] Everyone has accepted their invitation, set a password and enrolled two-step sign-in (the reporter's is optional).
- [ ] Site assignments: the reporter and both supervisors have TEST-LAG-001, 002 and 003, and the supervisors also TEST-ENU-001. The reviewer has every site except TEST-KAN-001 and TEST-PHC-001.
- [ ] Sample history is in place (`docs/SESSION_RUNBOOK.md`, step 7: `seed-demo`). It leaves **today** empty on purpose; the steps below fill it in.

## What the sample history already holds

Nothing in this list needs testing; it is there to look at and work with. Everything it created starts with "Sample:" or is dated before today.

- **TEST-LAG-001, 002, 003:** approved reports for the last three days. TEST-LAG-001 also has one day that was corrected (the original shows as superseded). TEST-LAG-002 has one report that was returned for a fix, waiting on the reporter's Corrections screen.
- **Incidents:** one closed and published (the reviewer can see it), one in progress, one just reported.
- **Disputes:** one open on TEST-LAG-001 (a supervisor can work it), one closed.
- **Tickets:** one open on TEST-LAG-001, one resolved.
- **Maintenance:** one window waiting for approval and one approved, both on TEST-LAG-001, a few days ahead.

## The run

### 1. Reporting (about 6 minutes)

| # | Who | Do | Expect | Result | Notes |
|---|---|---|---|---|---|
| 1 | Reporter | Submit today's normal day for TEST-LAG-001: grid on all day, generator 0 | Submitted, waiting for review | | |
| 2 | Reporter | Submit today's outage day for TEST-LAG-003, first with no outage reason | Refused with a plain message. With a reason, it is accepted | | |
| 3 | Reviewer | Open Sites and Reports | Earlier days show (the sample history). Today's two reports do not, because they are not approved yet. Time nobody reported reads "assumed", never "confirmed" | | |
| 4 | Supervisor (David) | Approve today's TEST-LAG-001 | Approved, and it can no longer be edited | | |
| 5 | Reporter | Try to approve your own report | No way to do it | | |
| 6 | Teammate | Approve today's TEST-LAG-003 | Approved | | |
| 7 | Reviewer | Refresh | Both of today's reports now visible. No internal notes, staff names or fuel | | |
| 8 | Reporter | Request a correction on today's TEST-LAG-001 (generator 4.5, with a reason) | The original stays until approved | | |
| 9 | Teammate | Approve the correction | Original marked superseded, new revision current | | |
| 10 | Reporter | Open Corrections, fix and resubmit the returned TEST-LAG-002 report; a supervisor approves it | It goes back for review and then shows as approved | | |

### 2. Platform (about 9 minutes)

| # | Who | Do | Expect | Result | Notes |
|---|---|---|---|---|---|
| 11 | Reporter | Log a high power-outage incident on TEST-LAG-003 | Appears as Reported | | |
| 12 | Supervisor | Acknowledge it, then publish it to the customer | Reviewer now sees it, with no internal notes. The closed sample incident was already visible | | |
| 13 | Reviewer | Raise a dispute on TEST-LAG-003 ("Duration looks wrong") | Dispute created, the figure shows as questioned | | |
| 14 | Supervisor | Take the **sample** open dispute on TEST-LAG-001: under review, ask for evidence, resolve as not upheld | Each step recorded. A step the rules refuse says why | | |
| 15 | Reviewer | Acknowledge and close that dispute | Closed | | |
| 16 | Reviewer | Raise a support ticket on TEST-LAG-002 | Ticket created | | |
| 17 | Supervisor | On the sample open ticket: reply publicly, add one internal comment, resolve | Reviewer sees the public reply and **not** the internal comment | | |
| 18 | Supervisor | Approve the sample maintenance window that is waiting | Approved. The reporter who asked can never approve their own | | |
| 19 | Reporter | Request a new maintenance window on TEST-LAG-002; a supervisor approves it | Requested, then approved | | |
| 20 | Supervisor (David) | Start acceptance on TEST-ENU-001: built, then ready for inspection | Stage moves each time | | |
| 21 | Reviewer | Mark it inspected, then accepted | Site becomes active | | |

### 3. Quick checks (about 3 minutes)

| # | Who | Do | Expect | Result | Notes |
|---|---|---|---|---|---|
| 22 | Admin | Open the audit log | Names shown, not ids. Nothing can be edited or deleted. The sample history appears under the invited people's names (this is expected; see decision 073) | | |
| 23 | Admin | Look through every admin screen | No raw ids anywhere. Write down each screen that still has one | | |
| 24 | Reviewer | Download one CSV export | Unreported days are blank, not 0 | | |
| 25 | Reviewer | Open the address of TEST-KAN-001 | Not found, no data | | |
| 26 | Spare account | Enter a wrong password 5 times | Locked for 15 minutes, with a plain message | | |

## Capture

Fill this in once at the end and send it to David.

```
Date/time of run:
Who ran it:
Passed (step numbers):
Failed (step number, screen, role, exact message):
Blocked (step number, why):
Anything confusing but not broken:
```

## After the run

Before real data goes in, the database is wiped (drop it, run migrate, create the admin, re-invite everyone), so testers should expect their accounts and the sample history to disappear.
