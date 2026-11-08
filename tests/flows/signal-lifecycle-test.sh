#!/usr/bin/env bash
set -Eeuo pipefail

script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/rapida-signal-lifecycle-test.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM
fake_bin="$temporary_directory/bin"
event_log="$temporary_directory/events"
ready_file="$temporary_directory/ready"
mkdir -p "$fake_bin"

# shellcheck disable=SC2016 # These generated executables read test environment variables.
printf '%s\n' \
  '#!/bin/sh' \
  'database=' \
  'while [ "$#" -gt 0 ]; do' \
  '  if [ "$1" = -d ]; then database=$2; shift 2; else shift; fi' \
  'done' \
  'sql=$(cat)' \
  'case "$sql" in *"AS unexpected;"*) printf "0\\n"; exit ;; esac' \
  'if [ -f "$FLOW_SIGNAL_READY" ]; then' \
  '  printf "cleanup-%s\\n" "$database" >> "$FLOW_SIGNAL_LOG"' \
  '  sleep 0.3' \
  'fi' \
  > "$fake_bin/psql"
# shellcheck disable=SC2016 # These generated executables read test environment variables.
printf '%s\n' \
  '#!/bin/sh' \
  'trap '\''printf "child-terminated\\n" >> "$FLOW_SIGNAL_LOG"; exit 143'\'' HUP INT TERM' \
  'touch "$FLOW_SIGNAL_READY"' \
  'while :; do sleep 1; done' \
  > "$fake_bin/timeout"
chmod +x "$fake_bin/psql" "$fake_bin/timeout"

PATH="$fake_bin:$PATH" \
  FLOW_SIGNAL_LOG="$event_log" \
  FLOW_SIGNAL_READY="$ready_file" \
  FLOW_POSTGRES_HOST=unused \
  PGPASSWORD=unused \
  "$script_directory/assistant-lifecycle/nodejs/run.sh" >/dev/null 2>&1 &
runner_pid=$!

for _ in {1..50}; do
  [[ -f "$ready_file" ]] && break
  sleep 0.1
done
[[ -f "$ready_file" ]]
: > "$event_log"
kill -TERM "$runner_pid"
for _ in {1..50}; do
  grep -Fq 'child-terminated' "$event_log" 2>/dev/null && break
  sleep 0.1
done
kill -TERM "$runner_pid" 2>/dev/null || true
set +e
wait "$runner_pid"
runner_status=$?
set -e
[[ "$runner_status" -ne 0 ]]
[[ $(sed -n '1p' "$event_log") == 'child-terminated' ]]
grep -Fq 'cleanup-assistant_db' "$event_log"
grep -Fq 'cleanup-web_db' "$event_log"

printf 'Flow signal lifecycle regression tests passed\n'
