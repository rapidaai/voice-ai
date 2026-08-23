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
termination_requested=false

check_builder() {
  local timeout_seconds=$1
  builder_present=false
  cache_addressable=false
  timeout --signal=TERM --kill-after=1s "${timeout_seconds}s" \
    docker buildx inspect "${SYSTEM_BUILDER_NAME}" >/dev/null 2>&1 &
  inspect_pid=$!
  timeout --signal=TERM --kill-after=1s "${timeout_seconds}s" \
    docker buildx du --builder "${SYSTEM_BUILDER_NAME}" >/dev/null 2>&1 &
  du_pid=$!
  if wait "${inspect_pid}"; then
    builder_present=true
  fi
  if wait "${du_pid}"; then
    cache_addressable=true
  fi
}

remove_builder() {
  timeout --signal=TERM --kill-after=1s 5s \
    docker buildx rm "${SYSTEM_BUILDER_NAME}" >/dev/null 2>&1 || true

  for ((attempt = 1; attempt <= 10; attempt++)); do
    check_builder 1
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

remove_builder_emergency() {
  timeout --signal=TERM --kill-after=1s 3s \
    docker buildx rm "${SYSTEM_BUILDER_NAME}" >/dev/null 2>&1 || true
  check_builder 1
  if [[ ${builder_present} == true || ${cache_addressable} == true ]]; then
    echo "Buildx builder or cache remains after emergency cleanup: ${SYSTEM_BUILDER_NAME}" >&2
    cleanup_status=1
  fi
}

on_exit() {
  status=$?
  trap - EXIT HUP INT TERM
  if [[ ${status} -ne 0 && ${cleanup_status} -eq 0 ]]; then
    cleanup_status=${status}
  fi
  if [[ ${termination_requested} == true ]]; then
    remove_builder_emergency
  else
    remove_builder
  fi
  exit "${cleanup_status}"
}
on_signal() {
  termination_requested=true
  exit "$1"
}
trap on_exit EXIT
trap 'on_signal 129' HUP
trap 'on_signal 130' INT
trap 'on_signal 143' TERM

timeout --signal=TERM --kill-after=5s 30s \
  "${compose[@]}" down --volumes --remove-orphans --timeout 30 || cleanup_status=$?

timeout --signal=TERM --kill-after=5s 60s \
  go run ./tests/system/cmd/systemcheck cleanup \
  --compose-project "${COMPOSE_PROJECT_NAME}" \
  --retries 10 \
  --interval 1s || cleanup_status=$?
