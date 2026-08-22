#!/usr/bin/env bash
set -Eeuo pipefail

: "${COMPOSE_PROJECT_NAME:?COMPOSE_PROJECT_NAME is required}"
: "${SYSTEM_BUILDER_NAME:?SYSTEM_BUILDER_NAME is required}"

if [[ -n ${SYSTEM_STATE_DIR:-} && -f ${SYSTEM_STATE_DIR}/cache-scope ]]; then
  export SYSTEM_CACHE_SCOPE
  SYSTEM_CACHE_SCOPE=$(<"${SYSTEM_STATE_DIR}/cache-scope")
else
  export SYSTEM_CACHE_SCOPE=cleanup
fi
compose=(./tests/system/bin/docker-compose -f docker-compose.yml -f docker-compose.ci.yml)
cleanup_status=0

remove_builder() {
  timeout 10s docker buildx rm "${SYSTEM_BUILDER_NAME}" >/dev/null 2>&1 || true

  for ((attempt = 1; attempt <= 10; attempt++)); do
    builder_present=false
    cache_addressable=false
    if timeout 3s docker buildx inspect "${SYSTEM_BUILDER_NAME}" >/dev/null 2>&1; then
      builder_present=true
    fi
    if timeout 3s docker buildx du --builder "${SYSTEM_BUILDER_NAME}" >/dev/null 2>&1; then
      cache_addressable=true
    fi
    if [[ ${builder_present} == false && ${cache_addressable} == false ]]; then
      return
    fi
    if [[ ${attempt} -lt 10 ]]; then
      sleep 1
    fi
  done

  echo "Buildx builder or cache remains after cleanup: ${SYSTEM_BUILDER_NAME}" >&2
  cleanup_status=1
}

on_exit() {
  status=$?
  trap - EXIT HUP INT TERM
  if [[ ${status} -ne 0 && ${cleanup_status} -eq 0 ]]; then
    cleanup_status=${status}
  fi
  remove_builder
  exit "${cleanup_status}"
}
trap on_exit EXIT HUP INT TERM

timeout --signal=TERM --kill-after=5s 30s \
  "${compose[@]}" down --volumes --remove-orphans --timeout 30 || cleanup_status=$?

timeout --signal=TERM --kill-after=5s 60s \
  go run ./tests/system/cmd/systemcheck cleanup \
  --compose-project "${COMPOSE_PROJECT_NAME}" \
  --retries 10 \
  --interval 1s || cleanup_status=$?
