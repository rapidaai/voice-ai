#!/usr/bin/env bash
set -Eeuo pipefail

script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/rapida-signal-lifecycle-test.XXXXXX")
runner_pid=''
fake_timeout_pid=''

cleanup_test_processes() {
  if [[ -n "$runner_pid" ]]; then
    kill -TERM "$runner_pid" 2>/dev/null || true
  fi
  if [[ -n "$fake_timeout_pid" ]]; then
    kill -TERM "$fake_timeout_pid" 2>/dev/null || true
  fi
  for _ in {1..20}; do
    if { [[ -z "$runner_pid" ]] || ! kill -0 "$runner_pid" 2>/dev/null; } &&
      { [[ -z "$fake_timeout_pid" ]] || ! kill -0 "$fake_timeout_pid" 2>/dev/null; }; then
      break
    fi
    sleep 0.1
  done
  if [[ -n "$runner_pid" ]]; then
    kill -KILL "$runner_pid" 2>/dev/null || true
    wait "$runner_pid" 2>/dev/null || true
  fi
  if [[ -n "$fake_timeout_pid" ]]; then
    kill -KILL "$fake_timeout_pid" 2>/dev/null || true
    wait "$fake_timeout_pid" 2>/dev/null || true
  fi
  rm -rf "$temporary_directory"
}
trap cleanup_test_processes EXIT
trap 'exit 1' HUP INT TERM
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
  'if [ "${FLOW_SIGNAL_IGNORE_TERM:-false}" = true ]; then' \
  '  trap '\''printf "child-ignored-term\\n" >> "$FLOW_SIGNAL_LOG"'\'' HUP INT TERM' \
  'else' \
  '  trap '\''printf "child-terminated\\n" >> "$FLOW_SIGNAL_LOG"; exit 143'\'' HUP INT TERM' \
  'fi' \
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
  fake_timeout_pid=$(pgrep -P "$runner_pid" | head -1 || true)
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
runner_pid=''
fake_timeout_pid=''
[[ "$runner_status" -ne 0 ]]
[[ $(sed -n '1p' "$event_log") == 'child-terminated' ]]
grep -Fq 'cleanup-assistant_db' "$event_log"
grep -Fq 'cleanup-web_db' "$event_log"

rm -f "$ready_file"
: > "$event_log"
PATH="$fake_bin:$PATH" \
  FLOW_SIGNAL_LOG="$event_log" \
  FLOW_SIGNAL_READY="$ready_file" \
  FLOW_SIGNAL_IGNORE_TERM=true \
  FLOW_PROCESS_TERMINATION_TIMEOUT_SECONDS=1 \
  FLOW_POSTGRES_HOST=unused \
  PGPASSWORD=unused \
  "$script_directory/assistant-lifecycle/nodejs/run.sh" >/dev/null 2>&1 &
runner_pid=$!
for _ in {1..50}; do
  fake_timeout_pid=$(pgrep -P "$runner_pid" | head -1 || true)
  [[ -f "$ready_file" ]] && break
  sleep 0.1
done
[[ -f "$ready_file" ]]
kill -TERM "$runner_pid"
for _ in {1..40}; do
  kill -0 "$runner_pid" 2>/dev/null || break
  sleep 0.1
done
if kill -0 "$runner_pid" 2>/dev/null; then
  printf 'runner did not bound termination of a child that ignored TERM\n' >&2
  exit 1
fi
set +e
wait "$runner_pid"
runner_status=$?
set -e
[[ "$runner_status" -ne 0 ]]
if [[ -n "$fake_timeout_pid" ]] && kill -0 "$fake_timeout_pid" 2>/dev/null; then
  printf 'fake timeout child remained after bounded termination\n' >&2
  exit 1
fi
runner_pid=''
fake_timeout_pid=''

deadline_script="$temporary_directory/deadline.sh"
deadline_log="$temporary_directory/deadline.log"
# shellcheck disable=SC2016 # The generated script exercises the sourced lifecycle owner.
printf '%s\n' \
  '#!/bin/sh' \
  'set -eu' \
  '. "$FLOW_PROCESS_LIFECYCLE"' \
  'cleanup() {' \
  '  trap '\'' '\'' HUP INT TERM' \
  '  while :; do :; done' \
  '}' \
  'register_flow_cleanup cleanup' \
  > "$deadline_script"
chmod +x "$deadline_script"
FLOW_PROCESS_LIFECYCLE="$script_directory/process-lifecycle.sh" \
  FLOW_CLEANUP_TIMEOUT_SECONDS=1 \
  timeout -k 1s 4s "$deadline_script" >"$deadline_log" 2>&1 || true
grep -Fq 'flow cleanup exceeded 1 seconds' "$deadline_log"

printf 'Flow signal lifecycle regression tests passed\n'
