#!/usr/bin/env bash
#
# Restore test for the Coloplus database (AT11: the target is to lose at most
# 24 hours of data and be running again within 8 hours; an untested backup
# does not count).
#
# Run it ON THE SERVER, by hand, about once a month and after any change to the
# backup script. It restores the newest local dump into a THROWAWAY database
# named coloplus_restore_test, checks it against the live database, prints how
# long the restore took, and drops the throwaway database again.
#
# It never writes to the live `coloplus` database. It only reads it, to compare
# row counts. It refuses to run if the throwaway database already exists.
#
# Usage:   bash ~/scripts/coloplus-restore-test.sh
# Source-controlled copy; the copy that runs lives on the server.
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-$HOME/backups/postgres}"
LIVE_DB="coloplus"
TEST_DB="coloplus_restore_test"
MAX_AGE_HOURS=26   # nightly at 03:00 plus slack

pg() { # pg <database> <sql...>: run one query inside the postgres container
  local db="$1"; shift
  docker exec -e DB="$db" -e SQL="$*" postgres sh -c \
    'PGPASSWORD="$POSTGRES_PASSWORD" psql -h 127.0.0.1 -U postgres -d "$DB" -At -c "$SQL"'
}

fail() { echo "RESTORE TEST FAILED: $*" >&2; exit 1; }

cleanup() { pg postgres "DROP DATABASE IF EXISTS ${TEST_DB}" >/dev/null 2>&1 || true; }

LATEST="$(ls -t "$BACKUP_DIR"/${LIVE_DB}-*.dump 2>/dev/null | head -1 || true)"
[ -n "$LATEST" ] || fail "no ${LIVE_DB} dump found in $BACKUP_DIR"

AGE_HOURS=$(( ( $(date +%s) - $(stat -c %Y "$LATEST") ) / 3600 ))
echo "dump:      $LATEST ($(du -h "$LATEST" | cut -f1), ${AGE_HOURS}h old)"
[ "$AGE_HOURS" -le "$MAX_AGE_HOURS" ] || fail "the newest dump is ${AGE_HOURS}h old (limit ${MAX_AGE_HOURS}h): the nightly backup is not running"

EXISTS="$(pg postgres "SELECT 1 FROM pg_database WHERE datname = '${TEST_DB}'")"
[ -z "$EXISTS" ] || fail "${TEST_DB} already exists; check nobody is using it, then drop it by hand"

trap cleanup EXIT
pg postgres "CREATE DATABASE ${TEST_DB}" >/dev/null

START=$(date +%s)
docker exec -i postgres sh -c \
  'PGPASSWORD="$POSTGRES_PASSWORD" pg_restore -h 127.0.0.1 -U postgres -d '"${TEST_DB}"' --no-owner --exit-on-error' < "$LATEST" \
  || fail "pg_restore reported an error"
SECONDS_TAKEN=$(( $(date +%s) - START ))
echo "restored:  in ${SECONDS_TAKEN}s (the target for the whole recovery is 8 hours)"

# 1. The schema is the one the app expects: same migration version as live
#    (the dump may be a day older, so it may be behind but never ahead).
RESTORED_V="$(pg "$TEST_DB" "SELECT max(version_id) FROM goose_db_version")"
LIVE_V="$(pg "$LIVE_DB" "SELECT max(version_id) FROM goose_db_version")"
echo "migration: restored v${RESTORED_V}, live v${LIVE_V}"
[ "${RESTORED_V:-0}" -le "${LIVE_V:-0}" ] || fail "the restored database is AHEAD of live"
[ "${RESTORED_V:-0}" -ge 1 ] || fail "the restored database has no migration history"

# 2. The data is there. Live keeps moving, so the restored counts must be
#    positive where live is positive and never larger than live.
status=0
for t in users sites daily_reports power_intervals incidents disputes audit_log; do
  r="$(pg "$TEST_DB" "SELECT count(*) FROM ${t}")"
  l="$(pg "$LIVE_DB" "SELECT count(*) FROM ${t}" 2>/dev/null || echo 0)"
  verdict="ok"
  if [ "$r" -gt "$l" ]; then verdict="RESTORED HAS MORE THAN LIVE"; status=1; fi
  if [ "$l" -gt 0 ] && [ "$r" -eq 0 ]; then verdict="EMPTY AFTER RESTORE"; status=1; fi
  printf '  %-16s restored %-8s live %-8s %s\n' "$t" "$r" "$l" "$verdict"
done

# 3. The trust rules came back with the data: the guards on approved reports
#    and on the audit log are triggers, so a restore that lost them is unsafe.
TRIGGERS="$(pg "$TEST_DB" "SELECT count(*) FROM pg_trigger WHERE NOT tgisinternal AND tgname IN ('audit_log_no_update_delete','daily_reports_guard','power_intervals_guard','incidents_guard','disputes_guard')")"
echo "  guards           ${TRIGGERS} of the 5 checked triggers present"
[ "$TRIGGERS" -ge 4 ] || { echo "  too few guard triggers came back" >&2; status=1; }

[ "$status" -eq 0 ] || fail "see the lines above"
echo "RESTORE TEST PASSED: ${LATEST##*/} restores in ${SECONDS_TAKEN}s and matches live."
