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
    '. "$script_directory/../../process-lifecycle.sh"' \
    '. "$script_directory/../../project-fixture.sh"' \
    "export FLOW_FIXTURE_ID='$fixture_id'" \
    "export FLOW_EMAIL='$email'" \
    'cleanup() { :; }' \
    'register_flow_cleanup cleanup' \
    'seed_project_fixture' \
    'run_flow_process timeout 1s node "$script_directory/flow.js"' \
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
  printf '%s\n' '#!/bin/sh' > "$root/process-lifecycle.sh"
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
  sed -i.bak '/run_flow_process timeout 1s node/i\
verify_assistant_fixture' "$1/beta/nodejs/run.sh"
  rm "$1/beta/nodejs/run.sh.bak"
}

remove_react_coverage() {
  sed -i.bak 's/ nodejs react$/ nodejs/' "$1/alpha/run.sh"
  rm "$1/alpha/run.sh.bak"
}

remove_signal_exit() {
  sed -i.bak '/register_flow_cleanup/d' "$1/alpha/nodejs/run.sh"
  rm "$1/alpha/nodejs/run.sh.bak"
}

create_temp_before_cleanup() {
  # shellcheck disable=SC2016 # The inserted command must retain command substitution.
  sed -i.bak '/register_flow_cleanup/i\
temporary_directory=$(mktemp -d)' "$1/alpha/nodejs/run.sh"
  rm "$1/alpha/nodejs/run.sh.bak"
}

use_zero_sdk_timeout() {
  sed -i.bak 's/timeout 1s node/timeout 0s node/' "$1/alpha/nodejs/run.sh"
  rm "$1/alpha/nodejs/run.sh.bak"
}

add_unbounded_sdk_operation() {
  # shellcheck disable=SC2016 # The inserted command must retain the fixture variable.
  sed -i.bak '/run_flow_process timeout 1s node/a\
node "$script_directory/flow.js"' "$1/alpha/nodejs/run.sh"
  rm "$1/alpha/nodejs/run.sh.bak"
}

add_same_line_unbounded_sdk_operation() {
  # shellcheck disable=SC2016 # The inserted commands must retain the fixture variable.
  sed -i.bak 's|run_flow_process timeout 1s node "$script_directory/flow.js"|run_flow_process timeout 1s node "$script_directory/flow.js"; node "$script_directory/flow.js"|' \
    "$1/alpha/nodejs/run.sh"
  rm "$1/alpha/nodejs/run.sh.bak"
}

add_rest_client() {
  local root=$1
  local max_time=$2
  local extra_operation=${3:-}
  local client_directory="$root/alpha/rest"
  sed -i.bak 's/ nodejs react$/ nodejs react rest/' "$root/alpha/run.sh"
  rm "$root/alpha/run.sh.bak"
  mkdir -p "$client_directory"
  # shellcheck disable=SC2016 # These literals belong to the generated fixture.
  printf '%s\n' \
    '#!/bin/sh' \
    'set -eu' \
    '' \
    'script_directory=$(cd "$(dirname "$0")" && pwd)' \
    '. "$script_directory/../../process-lifecycle.sh"' \
    '. "$script_directory/../../project-fixture.sh"' \
    "export FLOW_FIXTURE_ID='9300001'" \
    "export FLOW_EMAIL='rest-alpha@example.invalid'" \
    'cleanup() { :; }' \
    'run_flow() {' \
    "  curl --silent --max-time $max_time http://example.invalid" \
    "$extra_operation" \
    '}' \
    'register_flow_cleanup cleanup' \
    'seed_project_fixture' \
    'run_flow' \
    'verify_assistant_fixture' \
    > "$client_directory/run.sh"
  chmod +x "$client_directory/run.sh"
}

use_zero_rest_timeout() {
  add_rest_client "$1" 0
}

add_unbounded_rest_operation() {
  add_rest_client "$1" 1 '  curl --silent http://example.invalid'
}

add_same_line_unbounded_rest_operation() {
  add_rest_client "$1" 1
  sed -i.bak 's|curl --silent --max-time 1 http://example.invalid|curl --silent --max-time 1 http://example.invalid; curl --silent http://example.invalid|' \
    "$1/alpha/rest/run.sh"
  rm "$1/alpha/rest/run.sh.bak"
}

add_continued_rest_operation() {
  add_rest_client "$1" 1
  awk '
    /curl --silent --max-time 1/ {
      print "  curl --silent \\"
      print "    --max-time 1 http://example.invalid"
      next
    }
    { print }
  ' "$1/alpha/rest/run.sh" > "$1/alpha/rest/run.sh.updated"
  mv "$1/alpha/rest/run.sh.updated" "$1/alpha/rest/run.sh"
  chmod +x "$1/alpha/rest/run.sh"
}

valid_root="$temporary_directory/valid"
write_fixture "$valid_root"
"$script_directory/check-contract.sh" "$valid_root" >/dev/null

continued_root="$temporary_directory/continued"
write_fixture "$continued_root"
add_continued_rest_operation "$continued_root"
"$script_directory/check-contract.sh" "$continued_root" >/dev/null

expect_failure invalid-javascript 'contains invalid JavaScript' break_javascript
expect_failure duplicate-id 'fixture identifier 9100001 is reused' reuse_fixture_id
expect_failure invalid-email 'must use a reserved .invalid email address' use_real_email_domain
expect_failure wrong-order 'must verify persistent state after exercising the public client' verify_before_operation
expect_failure missing-react 'must declare react coverage' remove_react_coverage
expect_failure missing-signal-exit 'must register cleanup with the process lifecycle' remove_signal_exit
expect_failure early-temporary-directory 'must register cleanup before creating temporary files' create_temp_before_cleanup
expect_failure zero-sdk-timeout 'must bound every SDK call with a positive timeout' use_zero_sdk_timeout
expect_failure unbounded-sdk-operation 'must bound every SDK call with a positive timeout' add_unbounded_sdk_operation
expect_failure same-line-sdk-operation 'must bound every SDK call with a positive timeout' add_same_line_unbounded_sdk_operation
expect_failure zero-rest-timeout 'must bound every REST call with a positive --max-time' use_zero_rest_timeout
expect_failure unbounded-rest-operation 'must bound every REST call with a positive --max-time' add_unbounded_rest_operation
expect_failure same-line-rest-operation 'must bound every REST call with a positive --max-time' add_same_line_unbounded_rest_operation

printf 'Flow contract regression tests passed\n'
