#!/usr/bin/env bash
set -Eeuo pipefail

script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/rapida-flow-run-test.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM
framework_directory="$temporary_directory/framework"
report_directory="$temporary_directory/reports"
mkdir -p "$framework_directory" "$report_directory"
cp "$script_directory/run.sh" "$script_directory/report.sh" "$framework_directory/"
printf '%s\n' alpha beta gamma delta > "$framework_directory/manifest.txt"

write_journey() {
  local journey=$1
  local client=$2
  local display_name=$3
  mkdir -p "$framework_directory/$journey/$client"
  # shellcheck disable=SC2016 # These literals belong to the generated fixture.
  printf '%s\n' \
    '#!/bin/sh' \
    'set -eu' \
    'script_directory=$(cd "$(dirname "$0")" && pwd)' \
    '. "$script_directory/../report.sh"' \
    "run_flow_clients \"\$script_directory\" '$display_name' $client" \
    > "$framework_directory/$journey/run.sh"
  printf '%s\n' '#!/bin/sh' 'set -eu' > "$framework_directory/$journey/$client/run.sh"
  chmod +x "$framework_directory/$journey/run.sh" "$framework_directory/$journey/$client/run.sh"
}

write_journey alpha nodejs Alpha
write_journey beta react Beta
write_journey gamma nodejs Gamma
write_journey delta nodejs Delta

FLOW_REPORT_DIRECTORY="$report_directory" \
  "$framework_directory/run.sh" nodejs alpha >/dev/null
grep -Eq '^Alpha[[:space:]]+nodejs[[:space:]]+passed[[:space:]]+[0-9]+$' "$report_directory/flows.tsv"
[[ $(wc -l < "$report_directory/flows.tsv") -eq 1 ]]

FLOW_REPORT_DIRECTORY="$report_directory" \
  "$framework_directory/run.sh" nodejs beta >/dev/null
grep -Eq '^Beta[[:space:]]+nodejs[[:space:]]+skipped[[:space:]]+0$' "$report_directory/flows.tsv"
grep -Fq '| Beta | nodejs | ⏭️ Not supported | 0s |' "$report_directory/flows.md"

FLOW_REPORT_DIRECTORY="$report_directory" FLOW_SHARD_INDEX=1 FLOW_SHARD_TOTAL=2 \
  "$framework_directory/run.sh" nodejs >/dev/null
cut -f1 "$report_directory/flows.tsv" > "$temporary_directory/shard-one.txt"
printf '%s\n' Alpha Gamma > "$temporary_directory/expected-shard-one.txt"
diff -u "$temporary_directory/expected-shard-one.txt" "$temporary_directory/shard-one.txt"

FLOW_REPORT_DIRECTORY="$report_directory" FLOW_SHARD_INDEX=2 FLOW_SHARD_TOTAL=2 \
  "$framework_directory/run.sh" nodejs >/dev/null
cut -f1 "$report_directory/flows.tsv" > "$temporary_directory/shard-two.txt"
printf '%s\n' Beta Delta > "$temporary_directory/expected-shard-two.txt"
diff -u "$temporary_directory/expected-shard-two.txt" "$temporary_directory/shard-two.txt"

if FLOW_REPORT_DIRECTORY="$report_directory" \
  "$framework_directory/run.sh" nodejs unknown >"$temporary_directory/unknown.log" 2>&1; then
  printf 'unknown journey unexpectedly passed\n' >&2
  exit 1
fi
grep -Fq 'unknown flow journey: unknown' "$temporary_directory/unknown.log"

printf 'alpha\nbeta\ngamma\ndelta' > "$framework_directory/manifest.txt"
FLOW_REPORT_DIRECTORY="$report_directory" \
  "$framework_directory/run.sh" nodejs delta >/dev/null
grep -Eq '^Delta[[:space:]]+nodejs[[:space:]]+passed[[:space:]]+[0-9]+$' "$report_directory/flows.tsv"

printf 'Flow runner regression tests passed\n'
