#!/usr/bin/env bash
set -Eeuo pipefail

script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/rapida-project-lifecycle-fixture-test.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM
fake_bin="$temporary_directory/bin"
mutation_log="$temporary_directory/mutations"
mkdir -p "$fake_bin"

# shellcheck disable=SC2016 # The generated executable reads test environment variables.
printf '%s\n' \
  '#!/bin/sh' \
  'sql=$(cat)' \
  'case "$sql" in' \
  '  *"AS unexpected;"*) printf "%s\\n" "$FLOW_OWNERSHIP_COUNT" ;;' \
  '  *"Updated by the CI React project lifecycle flow"*) printf "1|1|1\\n" ;;' \
  '  *) printf "mutation\\n" >> "$FLOW_MUTATION_LOG" ;;' \
  'esac' \
  > "$fake_bin/psql"
printf '%s\n' '#!/bin/sh' 'exit 0' > "$fake_bin/timeout"
chmod +x "$fake_bin/psql" "$fake_bin/timeout"

if PATH="$fake_bin:$PATH" FLOW_OWNERSHIP_COUNT=1 FLOW_MUTATION_LOG="$mutation_log" \
  FLOW_TOKEN=fixture-token \
  "$script_directory/project-lifecycle/react/run.sh" >/dev/null 2>&1; then
  printf 'project lifecycle cleanup unexpectedly accepted an ownership collision\n' >&2
  exit 1
fi
[[ ! -s "$mutation_log" ]]

PATH="$fake_bin:$PATH" FLOW_OWNERSHIP_COUNT=0 FLOW_MUTATION_LOG="$mutation_log" \
  FLOW_TOKEN=fixture-token \
  "$script_directory/project-lifecycle/react/run.sh" >/dev/null
grep -Fq 'mutation' "$mutation_log"

printf 'Project lifecycle fixture regression tests passed\n'
