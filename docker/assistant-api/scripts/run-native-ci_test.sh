#!/usr/bin/env bash
set -Eeuo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
sandbox=$(mktemp -d)
trap 'rm -rf "${sandbox}"' EXIT

mkdir -p "${sandbox}/bin"

cat > "${sandbox}/bin/go" <<'SCRIPT'
#!/bin/sh
printf 'go %s\n' "$*" >> "${TRACE}"

case " $* " in
  *' vet '*) exit "${GO_VET_EXIT:-0}" ;;
esac

exit 0
SCRIPT

cat > "${sandbox}/bin/govulncheck" <<'SCRIPT'
#!/bin/sh
printf 'govulncheck %s\n' "$*" >> "${TRACE}"
exit "${GOVULNCHECK_EXIT:-0}"
SCRIPT

chmod +x "${sandbox}/bin/go" "${sandbox}/bin/govulncheck"

expected_trace=$(cat <<'EOF'
go list -deps -test ./api/assistant-api/... ./cmd/assistant/...
go vet ./api/assistant-api/... ./cmd/assistant/...
go test -count=1 -covermode=atomic -coverprofile=/tmp/assistant-coverage.out ./api/assistant-api/... ./cmd/assistant/...
go test -race -count=1 ./api/assistant-api/... ./cmd/assistant/...
govulncheck ./api/assistant-api/... ./cmd/assistant/...
go build -trimpath -o /tmp/assistant-api ./cmd/assistant/assistant.go
EOF
)

run_scenario() {
  name=$1
  shift
  trace="${sandbox}/${name}.trace"
  stdout="${sandbox}/${name}.stdout"
  stderr="${sandbox}/${name}.stderr"

  set +e
  env PATH="${sandbox}/bin:${PATH}" TRACE="${trace}" "$@" \
    /bin/sh "${script_dir}/run-native-ci.sh" >"${stdout}" 2>"${stderr}"
  scenario_status=$?
  set -e
}

run_scenario vulnerability-failure GOVULNCHECK_EXIT=3
[[ ${scenario_status} -eq 3 ]]
[[ $(cat "${trace}") == "${expected_trace}" ]]
grep -Fqx -- '- govulncheck (exit 3)' "${stderr}"

run_scenario multiple-failures GO_VET_EXIT=2 GOVULNCHECK_EXIT=3
[[ ${scenario_status} -eq 2 ]]
[[ $(cat "${trace}") == "${expected_trace}" ]]
grep -Fqx -- '- go vet (exit 2)' "${stderr}"
grep -Fqx -- '- govulncheck (exit 3)' "${stderr}"

run_scenario success
[[ ${scenario_status} -eq 0 ]]
[[ $(cat "${trace}") == "${expected_trace}" ]]
[[ ! -s ${stderr} ]]

echo 'run-native-ci regression passed'
