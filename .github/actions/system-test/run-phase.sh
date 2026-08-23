#!/usr/bin/env bash
set -Eeuo pipefail

phase=${1:?usage: run-phase.sh PHASE}
: "${SYSTEM_STATE_DIR:?SYSTEM_STATE_DIR is required}"
: "${SYSTEM_DIAGNOSTICS_DIR:?SYSTEM_DIAGNOSTICS_DIR is required}"

if [[ ${phase} == initialize ]]; then
  rm -rf "${SYSTEM_STATE_DIR}" "${SYSTEM_DIAGNOSTICS_DIR}"
  mkdir -p "${SYSTEM_STATE_DIR}" "${SYSTEM_DIAGNOSTICS_DIR}"
  date +%s > "${SYSTEM_STATE_DIR}/started-at"
  printf '%s\n' "$(( $(date +%s) + 1200 ))" > "${SYSTEM_STATE_DIR}/deadline"
  sha256sum \
    docker-compose.yml \
    docker-compose.ci.yml \
    docker/assistant-api/native-deps.lock \
    tests/system/service-images.lock \
    tests/system/tools.lock | sha256sum | cut -c1-20 > "${SYSTEM_STATE_DIR}/cache-key"
  printf 'system-%s\n' "$(cat "${SYSTEM_STATE_DIR}/cache-key")" \
    > "${SYSTEM_STATE_DIR}/cache-scope"
  if [[ -n ${GITHUB_ENV:-} ]]; then
    printf 'SYSTEM_CACHE_SCOPE=%s\n' "$(<"${SYSTEM_STATE_DIR}/cache-scope")" \
      >> "${GITHUB_ENV}"
  fi
  exit 0
fi

if [[ ! -f ${SYSTEM_STATE_DIR}/deadline ]]; then
  echo "System test deadline is not initialized" >&2
  exit 1
fi

deadline=$(<"${SYSTEM_STATE_DIR}/deadline")
remaining=$(( deadline - $(date +%s) ))
if (( remaining <= 0 )); then
  echo "The 20-minute system-test main budget expired before ${phase}" >&2
  exit 124
fi

on_exit() {
  status=$?
  trap - EXIT HUP INT TERM
  if [[ ${status} -ne 0 ]]; then
    printf '%s\n' "${phase}" > "${SYSTEM_STATE_DIR}/failure-phase"
  fi
  if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
    if [[ ${status} -eq 0 ]]; then
      printf -- "- \`%s\`: passed\n" "${phase}" >> "${GITHUB_STEP_SUMMARY}"
    else
      printf -- "- \`%s\`: failed with exit code \`%s\`\n" "${phase}" "${status}" \
        >> "${GITHUB_STEP_SUMMARY}"
    fi
  fi
  exit "${status}"
}
trap on_exit EXIT HUP INT TERM

timeout --signal=TERM --kill-after=30s "${remaining}s" \
  "${GITHUB_ACTION_PATH}/system-phase.sh" "${phase}"
