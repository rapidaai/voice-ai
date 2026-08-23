#!/usr/bin/env bash
set -Eeuo pipefail

: "${COMPOSE_PROJECT_NAME:?COMPOSE_PROJECT_NAME is required}"
: "${SYSTEM_BUILDER_NAME:?SYSTEM_BUILDER_NAME is required}"
: "${SYSTEM_STATE_DIR:?SYSTEM_STATE_DIR is required}"
: "${SYSTEM_DIAGNOSTICS_DIR:?SYSTEM_DIAGNOSTICS_DIR is required}"

diagnostic_work_dir="${SYSTEM_STATE_DIR}/diagnostics-work"
command_log="${SYSTEM_STATE_DIR}/diagnostics-command.log"
rm -rf "${SYSTEM_DIAGNOSTICS_DIR}"
mkdir -p "${diagnostic_work_dir}"
if [[ -f ${SYSTEM_STATE_DIR}/cache-scope ]]; then
  export SYSTEM_CACHE_SCOPE
  SYSTEM_CACHE_SCOPE=$(<"${SYSTEM_STATE_DIR}/cache-scope")
else
  export SYSTEM_CACHE_SCOPE=diagnostics
fi
diagnostics_safe=false

on_exit() {
  status=$?
  trap - EXIT HUP INT TERM
  if [[ ${diagnostics_safe} != true ]]; then
    rm -rf "${SYSTEM_DIAGNOSTICS_DIR}"
  fi
  rm -rf "${diagnostic_work_dir}"
  rm -f "${command_log}"
  exit "${status}"
}
trap on_exit EXIT HUP INT TERM

capture() {
  output=$1
  shift
  if "$@" > "${output}" 2>&1; then
    return
  else
    status=$?
    printf 'command unavailable or failed with exit code %s\n' "${status}" >> "${output}"
  fi
}

capture_pids=()
capture "${diagnostic_work_dir}/buildx-builder.txt" \
  timeout --signal=TERM --kill-after=5s 10s \
  docker buildx inspect "${SYSTEM_BUILDER_NAME}" &
capture_pids+=("$!")
capture "${diagnostic_work_dir}/buildx-du.txt" \
  timeout --signal=TERM --kill-after=5s 10s \
  docker buildx du --builder "${SYSTEM_BUILDER_NAME}" &
capture_pids+=("$!")
capture "${diagnostic_work_dir}/docker-system-df.txt" \
  timeout --signal=TERM --kill-after=5s 10s docker system df &
capture_pids+=("$!")
for capture_pid in "${capture_pids[@]}"; do
  wait "${capture_pid}"
done

if [[ -f ${SYSTEM_STATE_DIR}/cache-scope ]]; then
  printf '%s\n' "${SYSTEM_CACHE_SCOPE}" \
    > "${diagnostic_work_dir}/cache-scope.txt"
fi

if ! timeout --signal=TERM --kill-after=5s 30s \
  go run ./tests/system/cmd/systemcheck collect-diagnostics \
  --compose-project "${COMPOSE_PROJECT_NAME}" \
  --directory "${diagnostic_work_dir}" \
  > "${command_log}" 2>&1; then
  echo "Sanitized system diagnostic collection failed" >&2
  exit 1
fi

mkdir -p "${SYSTEM_DIAGNOSTICS_DIR}"
for artifact in diagnostics.json build-diagnostics.json; do
  install -m 600 \
    "${diagnostic_work_dir}/${artifact}" \
    "${SYSTEM_DIAGNOSTICS_DIR}/${artifact}"
done

if ! timeout --signal=TERM --kill-after=5s 15s \
  go run ./tests/system/cmd/systemcheck sanitize-artifacts \
  --directory "${SYSTEM_DIAGNOSTICS_DIR}" \
  > "${command_log}" 2>&1; then
  echo "Sanitized system diagnostic verification failed" >&2
  exit 1
fi
diagnostics_safe=true
