#!/usr/bin/env bash
set -Eeuo pipefail

script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/rapida-flow-contract-test.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM

write_client() {
  local root=$1
  local journey=$2
  local client=$3
  local fixture_id=$4
  local email=$5
  local client_directory="$root/$journey/$client"
  mkdir -p "$client_directory"
  # shellcheck disable=SC2016 # These literals belong to the generated fixture.
  printf '%s\n' \
    '#!/bin/sh' \
    'set -eu' \
    '' \
    'script_directory=$(cd "$(dirname "$0")" && pwd)' \
    '. "$script_directory/../../project-fixture.sh"' \
    "export FLOW_FIXTURE_ID='$fixture_id'" \
    "export FLOW_EMAIL='$email'" \
    'cleanup() { :; }' \
    'trap cleanup EXIT HUP INT TERM' \
    'seed_project_fixture' \
    'timeout 1s node "$script_directory/flow.js"' \
    'verify_assistant_fixture' \
    > "$client_directory/run.sh"
  printf '%s\n' 'console.log("contract fixture");' > "$client_directory/flow.js"
  chmod +x "$client_directory/run.sh"
}

write_journey() {
  local root=$1
  local journey=$2
  local fixture_id=$3
  local email=$4
  mkdir -p "$root/$journey"
  # shellcheck disable=SC2016 # These literals belong to the generated fixture.
  printf '%s\n' \
    '#!/bin/sh' \
    'set -eu' \
    '' \
    'script_directory=$(cd "$(dirname "$0")" && pwd)' \
    '. "$script_directory/../report.sh"' \
    "run_flow_clients \"\$script_directory\" '$journey' nodejs react" \
    > "$root/$journey/run.sh"
  chmod +x "$root/$journey/run.sh"
  write_client "$root" "$journey" nodejs "$fixture_id" "$email"
  write_client "$root" "$journey" react "8${fixture_id:1}" "react-$email"
}

write_fixture() {
  local root=$1
  mkdir -p "$root"
  printf '%s\n' alpha beta > "$root/manifest.txt"
  printf '%s\n' '#!/bin/sh' 'run_flow_clients() { :; }' > "$root/report.sh"
  printf '%s\n' '#!/bin/sh' > "$root/project-fixture.sh"
  write_journey "$root" alpha 9100001 'alpha@example.invalid'
  write_journey "$root" beta 9100002 'beta@example.invalid'
}

expect_failure() {
  local name=$1
  local expected=$2
  local root="$temporary_directory/$name"
  shift 2
  write_fixture "$root"
  "$@" "$root"
  if "$script_directory/check-contract.sh" "$root" >"$root/output.log" 2>&1; then
    printf 'contract regression case unexpectedly passed: %s\n' "$name" >&2
    return 1
  fi
  grep -Fq "$expected" "$root/output.log" || {
    printf 'contract regression case returned the wrong failure: %s\n' "$name" >&2
    cat "$root/output.log" >&2
    return 1
  }
}

break_javascript() {
  printf '%s\n' 'const = invalid;' > "$1/alpha/nodejs/flow.js"
}

reuse_fixture_id() {
  sed -i.bak "s/9100002/9100001/" "$1/beta/nodejs/run.sh"
  rm "$1/beta/nodejs/run.sh.bak"
}

use_real_email_domain() {
  sed -i.bak 's/beta@example.invalid/beta@example.com/' "$1/beta/nodejs/run.sh"
  rm "$1/beta/nodejs/run.sh.bak"
}

verify_before_operation() {
  sed -i.bak '/verify_assistant_fixture/d' "$1/beta/nodejs/run.sh"
  sed -i.bak '/timeout 1s node/i\
verify_assistant_fixture' "$1/beta/nodejs/run.sh"
  rm "$1/beta/nodejs/run.sh.bak"
}

remove_react_coverage() {
  sed -i.bak 's/ nodejs react$/ nodejs/' "$1/alpha/run.sh"
  rm "$1/alpha/run.sh.bak"
}

valid_root="$temporary_directory/valid"
write_fixture "$valid_root"
"$script_directory/check-contract.sh" "$valid_root" >/dev/null

expect_failure invalid-javascript 'contains invalid JavaScript' break_javascript
expect_failure duplicate-id 'fixture identifier 9100001 is reused' reuse_fixture_id
expect_failure invalid-email 'must use a reserved .invalid email address' use_real_email_domain
expect_failure wrong-order 'must verify persistent state after exercising the public client' verify_before_operation
expect_failure missing-react 'must declare react coverage' remove_react_coverage

printf 'Flow contract regression tests passed\n'
