#!/usr/bin/env bash
set -Eeuo pipefail

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)

run_case() {
  name=$1
  inspect_absent_at=$2
  cache_absent_at=$3
  expected_status=$4
  expected_attempts=$5
  expected_sleeps=$6

  sandbox=$(mktemp -d)
  mkdir -p "${sandbox}/repo/.github/actions/system-test" \
    "${sandbox}/repo/tests/system/bin" "${sandbox}/bin" "${sandbox}/state"
  cp "${repository_root}/.github/actions/system-test/cleanup.sh" \
    "${sandbox}/repo/.github/actions/system-test/cleanup.sh"
  printf 'test-cache-scope\n' > "${sandbox}/state/cache-scope"

  cat > "${sandbox}/bin/timeout" <<'SCRIPT'
#!/usr/bin/env bash
while [[ $# -gt 0 ]]; do
  case $1 in
    --signal=*|--kill-after=*) shift ;;
    *s|*m) shift; break ;;
    *) break ;;
  esac
done
exec "$@"
SCRIPT
  cat > "${sandbox}/bin/sleep" <<'SCRIPT'
#!/usr/bin/env bash
printf 'sleep\n' >> "${MOCK_STATE_DIR}/sleep-calls"
SCRIPT
  cat > "${sandbox}/bin/go" <<'SCRIPT'
#!/usr/bin/env bash
printf 'go %s\n' "$*" >> "${MOCK_STATE_DIR}/commands"
SCRIPT
  cat > "${sandbox}/repo/tests/system/bin/docker-compose" <<'SCRIPT'
#!/usr/bin/env bash
printf 'compose %s\n' "$*" >> "${MOCK_STATE_DIR}/commands"
SCRIPT
  cat > "${sandbox}/bin/docker" <<'SCRIPT'
#!/usr/bin/env bash
set -eu
printf 'docker %s\n' "$*" >> "${MOCK_STATE_DIR}/commands"
case "$1 $2" in
  'buildx rm')
    exit 0
    ;;
  'buildx inspect')
    count=$(($(cat "${MOCK_STATE_DIR}/inspect-count" 2>/dev/null || echo 0) + 1))
    printf '%s\n' "${count}" > "${MOCK_STATE_DIR}/inspect-count"
    [[ ${count} -lt ${MOCK_INSPECT_ABSENT_AT} ]]
    ;;
  'buildx du')
    count=$(($(cat "${MOCK_STATE_DIR}/du-count" 2>/dev/null || echo 0) + 1))
    printf '%s\n' "${count}" > "${MOCK_STATE_DIR}/du-count"
    [[ ${count} -lt ${MOCK_CACHE_ABSENT_AT} ]]
    ;;
esac
SCRIPT
  chmod +x "${sandbox}/bin/"* "${sandbox}/repo/tests/system/bin/docker-compose"

  set +e
  (
    cd "${sandbox}/repo"
    PATH="${sandbox}/bin:${PATH}" \
      MOCK_STATE_DIR="${sandbox}/state" \
      MOCK_INSPECT_ABSENT_AT="${inspect_absent_at}" \
      MOCK_CACHE_ABSENT_AT="${cache_absent_at}" \
      COMPOSE_PROJECT_NAME="cleanup-${name}" \
      SYSTEM_BUILDER_NAME="builder-${name}" \
      SYSTEM_STATE_DIR="${sandbox}/state" \
      .github/actions/system-test/cleanup.sh
  ) > "${sandbox}/stdout" 2> "${sandbox}/stderr"
  actual_status=$?
  set -e

  [[ ${actual_status} -eq ${expected_status} ]]
  [[ $(<"${sandbox}/state/inspect-count") -eq ${expected_attempts} ]]
  [[ $(<"${sandbox}/state/du-count") -eq ${expected_attempts} ]]
  actual_sleeps=$(wc -l < "${sandbox}/state/sleep-calls" 2>/dev/null || echo 0)
  [[ ${actual_sleeps} -eq ${expected_sleeps} ]]
  [[ $(grep -Fc "docker buildx rm builder-${name}" "${sandbox}/state/commands") -eq 1 ]]
  grep -Fq -- '--retries 10 --interval 1s' "${sandbox}/state/commands"
  if [[ ${expected_status} -eq 0 ]]; then
    [[ ! -s ${sandbox}/stderr ]]
  else
    grep -Fq "Buildx builder or cache remains after cleanup: builder-${name}" \
      "${sandbox}/stderr"
  fi
  rm -rf "${sandbox}"
}

run_case eventual-success 3 4 0 4 3
run_case exhausted 99 99 1 10 9
echo "cleanup retry regression tests passed"
