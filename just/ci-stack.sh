#!/usr/bin/env bash
set -Eeuo pipefail

bash just/require-docker.sh

mode=${1:?usage: ci-stack.sh <build|integration|smoke>}
case "$mode" in
  build | integration | smoke) ;;
  *)
    printf 'unsupported CI stack mode: %s\n' "$mode" >&2
    exit 2
    ;;
esac

export COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-rapida-ci-local}
export CI_STACK_CACHE_SCOPE=${CI_STACK_CACHE_SCOPE:-local}
export CI_STACK_AUTH_TOKEN=${CI_STACK_AUTH_TOKEN:-local-ci-token}
export CI_STACK_PROJECT_API_KEY=${CI_STACK_PROJECT_API_KEY:-local-ci-project-api-key}
export CI_STACK_DIAGNOSTICS_DIR=${CI_STACK_DIAGNOSTICS_DIR:-$PWD/ci-stack-diagnostics}
export CI_STACK_REPORTS_DIR=${CI_STACK_REPORTS_DIR:-$PWD/ci-stack-reports}

compose=(docker compose -f docker-compose.yml -f docker-compose.ci.yml)

cleanup() {
  status=$?
  if ((status != 0)); then
    mkdir -p "$CI_STACK_DIAGNOSTICS_DIR"
    "${compose[@]}" ps --all > "$CI_STACK_DIAGNOSTICS_DIR/compose-ps.txt" 2>&1 || true
    "${compose[@]}" logs --no-color > "$CI_STACK_DIAGNOSTICS_DIR/compose.log" 2>&1 || true
    sed -i.bak "s/${CI_STACK_AUTH_TOKEN}/[REDACTED]/g" "$CI_STACK_DIAGNOSTICS_DIR"/* 2>/dev/null || true
    sed -i.bak "s/${CI_STACK_PROJECT_API_KEY}/[REDACTED]/g" "$CI_STACK_DIAGNOSTICS_DIR"/* 2>/dev/null || true
    rm -f "$CI_STACK_DIAGNOSTICS_DIR"/*.bak
  fi
  "${compose[@]}" down --volumes --remove-orphans --timeout 30 || true
  exit "$status"
}
trap cleanup EXIT

mkdir -p "$CI_STACK_DIAGNOSTICS_DIR" "$CI_STACK_REPORTS_DIR"
chmod 0777 "$CI_STACK_REPORTS_DIR"

"${compose[@]}" config --quiet
bash -n tests/smoke/run.sh
bash just/shellcheck.sh tests/smoke/run.sh
./bin/check-go-version-consistency
./docker/assistant-api/scripts/verify-native-deps.sh docker/assistant-api/native-deps.lock
bash just/ci-python.sh openapi/scripts/generate_assistant_postman_collection.py --check

"${compose[@]}" --progress plain build 2>&1 | tee "$CI_STACK_DIAGNOSTICS_DIR/build.log"
if [[ "$mode" == build ]]; then
  exit
fi

"${compose[@]}" up -d --wait --wait-timeout 60 postgres redis
"${compose[@]}" run --rm migrate-web up
"${compose[@]}" run --rm migrate-integration up
"${compose[@]}" run --rm migrate-endpoint up
"${compose[@]}" run --rm migrate-assistant up
"${compose[@]}" up -d --no-build --wait --wait-timeout 180 \
  web-api integration-api endpoint-api assistant-api ui nginx
"${compose[@]}" run --rm test-runner "$mode"
