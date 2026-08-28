#!/usr/bin/env bash
set -Eeuo pipefail

bash just/require-docker.sh

export COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-rapida-smoke-local}
export SMOKE_CACHE_SCOPE=${SMOKE_CACHE_SCOPE:-local}
export SMOKE_AUTH_TOKEN=${SMOKE_AUTH_TOKEN:-local-smoke-token}
export SMOKE_DIAGNOSTICS_DIR=${SMOKE_DIAGNOSTICS_DIR:-$PWD/smoke-diagnostics}
export SMOKE_REPORTS_DIR=${SMOKE_REPORTS_DIR:-$PWD/smoke-reports}

compose=(docker compose -f docker-compose.yml -f docker-compose.ci.yml)

cleanup() {
  status=$?
  if ((status != 0)); then
    mkdir -p "$SMOKE_DIAGNOSTICS_DIR"
    "${compose[@]}" ps --all > "$SMOKE_DIAGNOSTICS_DIR/compose-ps.txt" 2>&1 || true
    "${compose[@]}" logs --no-color > "$SMOKE_DIAGNOSTICS_DIR/compose.log" 2>&1 || true
    sed -i.bak "s/${SMOKE_AUTH_TOKEN}/[REDACTED]/g" "$SMOKE_DIAGNOSTICS_DIR"/* 2>/dev/null || true
    rm -f "$SMOKE_DIAGNOSTICS_DIR"/*.bak
  fi
  "${compose[@]}" down --volumes --remove-orphans --timeout 30 || true
  exit "$status"
}
trap cleanup EXIT

mkdir -p "$SMOKE_DIAGNOSTICS_DIR" "$SMOKE_REPORTS_DIR"
chmod 0777 "$SMOKE_REPORTS_DIR"

"${compose[@]}" config --quiet
bash -n tests/smoke/run.sh
shellcheck tests/smoke/run.sh
./bin/check-go-version-consistency
./docker/assistant-api/scripts/verify-native-deps.sh docker/assistant-api/native-deps.lock
python3 openapi/scripts/generate_assistant_postman_collection.py --check

"${compose[@]}" --progress plain build 2>&1 | tee "$SMOKE_DIAGNOSTICS_DIR/build.log"
"${compose[@]}" up -d --wait --wait-timeout 60 postgres redis
"${compose[@]}" run --rm migrate-web up
"${compose[@]}" run --rm migrate-integration up
"${compose[@]}" run --rm migrate-endpoint up
"${compose[@]}" run --rm migrate-assistant up
"${compose[@]}" up -d --no-build --wait --wait-timeout 180 \
  web-api integration-api endpoint-api assistant-api ui nginx
"${compose[@]}" run --rm smoke-runner
