#!/usr/bin/env bash
#
# Server-side deploy script for the Coloplus API.
#
# This is the SOURCE-CONTROLLED COPY. The copy that RUNS lives on the server at
# ~/scripts/coloplus-deploy.sh and is the SSH forced command of the CI deploy
# key (see .github/workflows/ci.yml, deploy job). The two are not synced: when
# you edit this file, copy it to the server too.
#
# Order: pull -> migrate (new image, one-off) -> start -> wait for the Docker
# HEALTHCHECK -> on failure roll back to the previous image and email logs.
# Migrations run before the new container starts and the running one is not
# touched if they fail (docs/DECISIONS.md, `api migrate`).
set -uo pipefail
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

exec > >(tee -a "$HOME/coloplus-deploy.log") 2>&1

APP="coloplus-backend"
IMAGE="ghcr.io/olagookundavid/coloplus-monitoring:latest"

cd "$HOME/apps/coloplus"

alert() { "$HOME/scripts/alert.sh" "$1" "$2" || echo "[deploy] WARNING: alert email FAILED TO SEND."; }

echo "[deploy] $(date -Is) starting..."

# Rollback target, captured before anything new is pulled.
PREVIOUS_IMAGE_ID="$(docker compose images -q "$APP" 2>/dev/null || true)"

echo "[deploy] pulling latest image..."
if ! docker compose pull; then
  echo "[deploy] FAILED: could not pull the image. Nothing changed."
  alert "[Coloplus] DEPLOY FAILED on $(hostname)" "Stage: image pull. docker compose pull failed; the running deployment is untouched."
  exit 1
fi

echo "[deploy] applying migrations..."
if ! MIGRATE_OUT="$(docker compose run --rm --no-deps "$APP" ./server migrate 2>&1)"; then
  echo "$MIGRATE_OUT"
  echo "[deploy] FAILED: migrations did not apply. The running container is untouched."
  alert "[Coloplus] DEPLOY FAILED (migration) on $(hostname)" "Stage: migrate. The running deployment is untouched.

${MIGRATE_OUT: -3000}"
  exit 1
fi
echo "$MIGRATE_OUT"

echo "[deploy] recreating container..."
docker compose up -d

echo "[deploy] waiting for the new container to report healthy..."
ATTEMPTS=0
MAX_ATTEMPTS=20   # ~100s, matching the Dockerfile's start period and retries
STATUS="starting"
while [ "$ATTEMPTS" -lt "$MAX_ATTEMPTS" ]; do
  STATUS="$(docker inspect -f '{{.State.Health.Status}}' "$APP" 2>/dev/null || echo "unknown")"
  [ "$STATUS" = "healthy" ] || [ "$STATUS" = "unhealthy" ] && break
  sleep 5
  ATTEMPTS=$((ATTEMPTS + 1))
done

if [ "$STATUS" != "healthy" ]; then
  echo "[deploy] FAILED: new container never became healthy (status: $STATUS)"
  # The failing container's logs, captured before rollback replaces it.
  CRASH_LOGS="$(docker logs --tail 50 "$APP" 2>&1)"

  ACTION="No previous image captured: manual rollback required"
  [ -n "$PREVIOUS_IMAGE_ID" ] && ACTION="Rolling back to previous image (${PREVIOUS_IMAGE_ID:0:12})"
  alert "[Coloplus] DEPLOY FAILED on $(hostname)" "Status: ${STATUS}
Action: ${ACTION}

--- Container logs (last 50 lines) ---
${CRASH_LOGS}"

  if [ -n "$PREVIOUS_IMAGE_ID" ]; then
    echo "[deploy] rolling back to $PREVIOUS_IMAGE_ID..."
    docker tag "$PREVIOUS_IMAGE_ID" "$IMAGE"
    docker compose up -d
    echo "[deploy] rollback applied. Investigate before deploying again."
  fi
  exit 1
fi

echo "[deploy] ${APP} is healthy"

# CI stamps the commit and build time on every image as OCI labels.
COMMIT_SHA="$(docker inspect -f '{{index .Config.Labels "org.opencontainers.image.revision"}}' "$APP" 2>/dev/null)"
BUILD_TIME="$(docker inspect -f '{{index .Config.Labels "org.opencontainers.image.created"}}' "$APP" 2>/dev/null)"
[ -z "$COMMIT_SHA" ] && COMMIT_SHA="unknown"
[ -z "$BUILD_TIME" ] && BUILD_TIME="unknown"

alert "[Coloplus] Deploy succeeded on $(hostname)" "Deployed At: $(date '+%Y-%m-%d %H:%M:%S %Z')
Commit:      ${COMMIT_SHA:0:7}
Image Built: ${BUILD_TIME}"

echo "[deploy] pruning old images..."
docker image prune -f

echo "[deploy] done."
