#!/usr/bin/env bash
#
# Source-controlled copy of ~/scripts/pg-backup.sh on the shared Contabo box.
# The copy that RUNS is on the server (cron, nightly 03:00); the two are not
# synced, so update both. It covers every product's database on the box.
set -euo pipefail

# Alert on ANY failure so a broken backup can't fail silently.
trap '"$HOME/scripts/alert.sh" "[Backups] BACKUP FAILED on $(hostname)" "The Postgres backup failed at $(date -Is). Check ~/backups/backup.log."' ERR

BACKUP_DIR="$HOME/backups/postgres"
RETENTION_DAYS=14
REMOTE="r2backup:showtime-db-backups"
TS="$(date +%Y%m%d-%H%M%S)"
# One dump per database. Add a name here when a product gets its own database.
DATABASES=(showtime coloplus)

mkdir -p "$BACKUP_DIR"

for DB in "${DATABASES[@]}"; do
  FILE="$BACKUP_DIR/${DB}-${TS}.dump"

  docker exec -e DB="$DB" postgres sh -c \
    'PGPASSWORD="$POSTGRES_PASSWORD" pg_dump -h 127.0.0.1 -U postgres -d "$DB" -Fc' > "$FILE"

  if [ "$(stat -c%s "$FILE")" -lt 1024 ]; then
    echo "ERROR: $DB dump suspiciously small, aborting" >&2
    exit 1
  fi

  rclone copy "$FILE" "$REMOTE/"
  find "$BACKUP_DIR" -name "${DB}-*.dump" -mtime +"$RETENTION_DAYS" -delete

  echo "$(date -Is) backup OK: $FILE ($(du -h "$FILE" | cut -f1))"
done
