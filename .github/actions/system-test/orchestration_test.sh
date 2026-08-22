#!/usr/bin/env bash
set -Eeuo pipefail

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
sandbox=$(mktemp -d)
trap 'rm -rf "${sandbox}"' EXIT

mkdir -p "${sandbox}/cache/docker/assistant-api" "${sandbox}/cache/tests/system"
printf 'services: {}\n' > "${sandbox}/cache/docker-compose.yml"
printf 'services: {}\n' > "${sandbox}/cache/docker-compose.ci.yml"
printf 'native-a\n' > "${sandbox}/cache/docker/assistant-api/native-deps.lock"
printf 'services\n' > "${sandbox}/cache/tests/system/service-images.lock"
printf 'tools\n' > "${sandbox}/cache/tests/system/tools.lock"

cache_scope() {
  state=$1
  (
    cd "${sandbox}/cache"
    SYSTEM_STATE_DIR="${state}" \
      SYSTEM_DIAGNOSTICS_DIR="${state}/diagnostics" \
      GITHUB_ACTION_PATH="${repository_root}/.github/actions/system-test" \
      "${repository_root}/.github/actions/system-test/run-phase.sh" initialize
  )
  cat "${state}/cache-scope"
}

first_scope=$(cache_scope "${sandbox}/state-a")
printf 'native-b\n' > "${sandbox}/cache/docker/assistant-api/native-deps.lock"
second_scope=$(cache_scope "${sandbox}/state-b")
[[ ${first_scope} != "${second_scope}" ]]

mkdir -p \
  "${sandbox}/repo/.github/actions/system-test" \
  "${sandbox}/repo/bin" \
  "${sandbox}/repo/docker/assistant-api/scripts" \
  "${sandbox}/repo/openapi/scripts" \
  "${sandbox}/repo/scripts/contracts" \
  "${sandbox}/repo/tests/system/bin" \
  "${sandbox}/bin" \
  "${sandbox}/state" \
  "${sandbox}/diagnostics"
cp "${repository_root}/.github/actions/system-test/system-phase.sh" \
  "${sandbox}/repo/.github/actions/system-test/system-phase.sh"
cp "${repository_root}/.github/actions/system-test/collect-diagnostics.sh" \
  "${sandbox}/repo/.github/actions/system-test/collect-diagnostics.sh"
printf 'system-test-scope\n' > "${sandbox}/state/cache-scope"
printf 'lock\n' > "${sandbox}/repo/docker/assistant-api/native-deps.lock"

cat > "${sandbox}/repo/tests/system/bin/docker-compose" <<'SCRIPT'
#!/usr/bin/env bash
printf 'compose %s\n' "$*" >> "${TRACE}"
if [[ $* == '-f docker-compose.yml -f docker-compose.ci.yml config --format json' ]]; then
  cat <<'JSON'
{"services":{"test-runner":{"image":"rapida-system-test-runner:ci"},"ui":{"build":{}},"assistant-api":{"build":{}},"endpoint-api":{"build":{}},"integration-api":{"build":{}},"web-api":{"build":{}}},"name":"synthetic"}
JSON
fi
SCRIPT
cat > "${sandbox}/repo/tests/system/bin/verify-service-image-digests" <<'SCRIPT'
#!/usr/bin/env bash
printf 'image-digests %s\n' "$*" >> "${TRACE}"
SCRIPT
cat > "${sandbox}/repo/scripts/contracts/materialize-baseline.sh" <<'SCRIPT'
#!/usr/bin/env bash
mkdir -p "$1"
printf 'services: {}\n' > "$1/docker-compose.yml"
printf 'baseline %s\n' "$*" >> "${TRACE}"
SCRIPT
cat > "${sandbox}/repo/bin/check-go-version-consistency" <<'SCRIPT'
#!/usr/bin/env bash
printf 'go-version %s\n' "$*" >> "${TRACE}"
SCRIPT
cat > "${sandbox}/repo/docker/assistant-api/scripts/verify-native-deps.sh" <<'SCRIPT'
#!/usr/bin/env bash
printf 'native-deps %s\n' "$*" >> "${TRACE}"
SCRIPT
cat > "${sandbox}/repo/.github/actions/system-test/cleanup_test.sh" <<'SCRIPT'
#!/usr/bin/env bash
printf 'cleanup-test %s\n' "$*" >> "${TRACE}"
SCRIPT
cat > "${sandbox}/repo/.github/actions/system-test/orchestration_test.sh" <<'SCRIPT'
#!/usr/bin/env bash
printf 'orchestration-test %s\n' "$*" >> "${TRACE}"
SCRIPT
cat > "${sandbox}/repo/openapi/scripts/generate_assistant_postman_collection.py" <<'SCRIPT'
#!/usr/bin/env python3
SCRIPT
cat > "${sandbox}/bin/docker" <<'SCRIPT'
#!/usr/bin/env bash
printf 'docker %s\n' "$*" >> "${TRACE}"
case "$*" in
  'buildx inspect synthetic-builder')
    printf 'Name: synthetic-builder\nDriver: docker-container\nignored: credential=raw-secret\n'
    ;;
  'buildx du --builder synthetic-builder')
    printf 'Total: 12.3MB\nignored-token=raw-secret\n'
    ;;
  'system df')
    printf 'Images 6 6 1GB 0B\nignored-token=raw-secret\n'
    ;;
esac
SCRIPT
cat > "${sandbox}/bin/go" <<'SCRIPT'
#!/usr/bin/env bash
printf 'go %s\n' "$*" >> "${TRACE}"
if [[ $* == *'systemcheck build-metadata'* ]]; then
  while [[ $# -gt 0 ]]; do
    if [[ $1 == --compose-images ]]; then
      cp "$2" "${TRACE}.compose-images"
      stat -f '%Lp' "$(dirname "$2")" > "${TRACE}.diagnostics-work-mode" 2>/dev/null ||
        stat -c '%a' "$(dirname "$2")" > "${TRACE}.diagnostics-work-mode"
      break
    fi
    shift
  done
fi
if [[ $* == *'systemcheck collect-diagnostics'* ]]; then
  while [[ $# -gt 0 ]]; do
    if [[ $1 == --directory ]]; then
      work_dir=$2
      printf '{"services":[]}\n' > "${work_dir}/diagnostics.json"
      printf '{"builder":"synthetic-builder"}\n' > "${work_dir}/build-diagnostics.json"
      rm -f \
        "${work_dir}/buildx-builder.txt" \
        "${work_dir}/buildx-du.txt" \
        "${work_dir}/docker-system-df.txt" \
        "${work_dir}/cache-scope.txt" \
        "${work_dir}/buildkit.log" \
        "${work_dir}/compose-images.json"
      exit 0
    fi
    shift
  done
fi
while [[ $# -gt 0 ]]; do
  if [[ $1 == --output ]]; then
    mkdir -p "$(dirname "$2")"
    printf '{"images":[]}\n' > "$2"
    break
  fi
  shift
done
SCRIPT
cat > "${sandbox}/bin/python3" <<'SCRIPT'
#!/usr/bin/env bash
printf 'python %s\n' "$*" >> "${TRACE}"
SCRIPT
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
chmod +x \
  "${sandbox}/repo/tests/system/bin/"* \
  "${sandbox}/repo/scripts/contracts/materialize-baseline.sh" \
  "${sandbox}/repo/bin/check-go-version-consistency" \
  "${sandbox}/repo/docker/assistant-api/scripts/verify-native-deps.sh" \
  "${sandbox}/repo/.github/actions/system-test/"*.sh \
  "${sandbox}/bin/"*

run_phase() {
  (
    cd "${sandbox}/repo"
    PATH="${sandbox}/bin:${PATH}" \
      TRACE="${sandbox}/trace" \
      COMPOSE_PROJECT_NAME=synthetic \
      SYSTEM_BUILDER_NAME=synthetic-builder \
      SYSTEM_BASELINE_DIR="${sandbox}/baseline" \
      SYSTEM_STATE_DIR="${sandbox}/state" \
      SYSTEM_DIAGNOSTICS_DIR="${sandbox}/diagnostics" \
      .github/actions/system-test/system-phase.sh "$1"
  )
}

run_phase validate-native-locks
run_phase validate-system-support
run_phase build-images
run_phase migrations
run_phase health
run_phase ui-nginx
run_phase assistant-smoke

(
  cd "${sandbox}/repo"
  PATH="${sandbox}/bin:${PATH}" \
    TRACE="${sandbox}/trace" \
    COMPOSE_PROJECT_NAME=synthetic \
    SYSTEM_BUILDER_NAME=synthetic-builder \
    SYSTEM_STATE_DIR="${sandbox}/state" \
    SYSTEM_DIAGNOSTICS_DIR="${sandbox}/diagnostics" \
    .github/actions/system-test/collect-diagnostics.sh
) > "${sandbox}/diagnostics-stdout" 2> "${sandbox}/diagnostics-stderr"

grep -Fxq 'go-version ' "${sandbox}/trace"
grep -Fxq 'native-deps docker/assistant-api/native-deps.lock' "${sandbox}/trace"
grep -Fxq 'go test -count=1 ./tests/system/...' "${sandbox}/trace"
grep -Fxq 'cleanup-test ' "${sandbox}/trace"
grep -Fxq 'orchestration-test ' "${sandbox}/trace"
grep -Fxq 'compose -f docker-compose.yml -f docker-compose.ci.yml config --format json' "${sandbox}/trace"
grep -Eq 'go run ./tests/system/cmd/systemcheck build-metadata --compose-images .*/state/diagnostics-work/compose-images.json --output .*/state/diagnostics-work/buildkit-metadata.json' "${sandbox}/trace"
if grep -Fq ' images --format json' "${sandbox}/trace"; then
  echo "build metadata invoked docker compose images before service creation" >&2
  exit 1
fi
expected_compose_images='{"assistant-api":"synthetic-assistant-api","endpoint-api":"synthetic-endpoint-api","integration-api":"synthetic-integration-api","test-runner":"rapida-system-test-runner:ci","ui":"synthetic-ui","web-api":"synthetic-web-api"}'
[[ $(<"${sandbox}/trace.compose-images") == "${expected_compose_images}" ]]
[[ $(<"${sandbox}/trace.diagnostics-work-mode") == 700 ]]
config_line=$(grep -nFm1 'compose -f docker-compose.yml -f docker-compose.ci.yml config --format json' "${sandbox}/trace" | cut -d: -f1)
container_line=$(grep -nFm1 'compose -f docker-compose.yml -f docker-compose.ci.yml run --rm migrate-web up' "${sandbox}/trace" | cut -d: -f1)
(( config_line < container_line ))
grep -Fxq 'compose -f docker-compose.yml -f docker-compose.ci.yml run --rm test-runner systemcheck migrations --require-clean --require-head --report /reports/migrations.json' "${sandbox}/trace"
grep -Fxq "compose -f docker-compose.yml -f docker-compose.ci.yml run --rm test-runner systemcheck health --timeout-per-service 60s --interval 1s --readiness-key PSQL psql://postgres:5432 --reject-arbitrary-true-fallback" "${sandbox}/trace"
grep -Fxq 'compose -f docker-compose.yml -f docker-compose.ci.yml run --rm test-runner systemcheck ui-nginx --base-url http://nginx:8080 --require-spa-root --require-hashed-asset --proxy-route /talk_api.TalkService/GetAllAssistantConversation=assistant-api:9007 --proxy-route /web_api.AuthenticationService/ForgotPassword=web-api:9001' "${sandbox}/trace"
grep -Fxq 'compose -f docker-compose.yml -f docker-compose.ci.yml run --rm test-runner systemcheck assistant-smoke --collection openapi/postman/assistant-api/assistant-api.smoke.postman_collection.json --base-url http://assistant-api:9007 --tmpfs /run/secrets' "${sandbox}/trace"
grep -Fq 'go run ./tests/system/cmd/systemcheck collect-diagnostics --compose-project synthetic --directory' "${sandbox}/trace"
grep -Fq 'go run ./tests/system/cmd/systemcheck sanitize-artifacts --directory' "${sandbox}/trace"
test -f "${sandbox}/diagnostics/diagnostics.json"
test -f "${sandbox}/diagnostics/build-diagnostics.json"
[[ $(find "${sandbox}/diagnostics" -type f | wc -l) -eq 2 ]]
test ! -e "${sandbox}/diagnostics/buildkit.log"
test ! -e "${sandbox}/diagnostics/compose-images.json"
test ! -e "${sandbox}/diagnostics/buildkit-metadata.json"
test ! -e "${sandbox}/state/diagnostics-work"
test ! -s "${sandbox}/diagnostics-stdout"
test ! -s "${sandbox}/diagnostics-stderr"
if grep -R -Fq 'raw-secret' \
  "${sandbox}/diagnostics" \
  "${sandbox}/diagnostics-stdout" \
  "${sandbox}/diagnostics-stderr"; then
  echo "raw diagnostic secret escaped sanitized staging" >&2
  exit 1
fi
grep -Fq 'timeout --signal=TERM --kill-after=5s 60s' \
  "${repository_root}/.github/actions/system-test/cleanup.sh"
grep -Fq 'timeout --signal=TERM --kill-after=10s 190s .github/actions/system-test/cleanup.sh' \
  "${repository_root}/.github/workflows/reusable-system-ci.yml"
grep -Fq 'timeout --signal=TERM --kill-after=10s 80s .github/actions/system-test/collect-diagnostics.sh' \
  "${repository_root}/.github/workflows/reusable-system-ci.yml"

echo "system orchestration regression tests passed"
