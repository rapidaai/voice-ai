#!/usr/bin/env bash
set -Eeuo pipefail

script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/rapida-run-local-test.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM
real_ln=$(command -v ln)
fake_bin="$temporary_directory/bin"
mkdir -p "$fake_bin"
# shellcheck disable=SC2016 # The generated executable reads this environment variable.
printf '%s\n' \
  '#!/bin/sh' \
  'printf "install\\n" >> "$FLOW_NPM_LOG"' \
  'sleep 0.2' \
  > "$fake_bin/npm"
chmod +x "$fake_bin/npm"

dependency_key=$(shasum -a 256 \
  "$script_directory/package.json" "$script_directory/package-lock.json" \
  | shasum -a 256 | awk '{print $1}')

abandoned_cache="$temporary_directory/abandoned-cache"
abandoned_log="$temporary_directory/abandoned-install.log"
mkdir -p "$abandoned_cache/$dependency_key.install.abandoned"
if PATH="$fake_bin:$PATH" FLOW_NPM_LOG="$abandoned_log" \
  FLOW_DEPENDENCY_CACHE_DIR="$abandoned_cache" \
  bash "$script_directory/run-local.sh" react missing-journey >/dev/null 2>&1; then
  printf 'unknown journey unexpectedly passed after abandoned install recovery\n' >&2
  exit 1
else
  status=$?
fi
[[ "$status" -eq 2 ]]
[[ -L "$abandoned_cache/$dependency_key" ]]
[[ -f "$abandoned_cache/$dependency_key/.complete" ]]
[[ $(wc -l < "$abandoned_log") -eq 1 ]]

concurrent_cache="$temporary_directory/concurrent-cache"
concurrent_log="$temporary_directory/concurrent-install.log"
PATH="$fake_bin:$PATH" FLOW_NPM_LOG="$concurrent_log" \
  FLOW_DEPENDENCY_CACHE_DIR="$concurrent_cache" \
  bash "$script_directory/run-local.sh" nodejs missing-journey >/dev/null 2>&1 &
node_pid=$!
PATH="$fake_bin:$PATH" FLOW_NPM_LOG="$concurrent_log" \
  FLOW_DEPENDENCY_CACHE_DIR="$concurrent_cache" \
  bash "$script_directory/run-local.sh" react missing-journey >/dev/null 2>&1 &
react_pid=$!
set +e
wait "$node_pid"
node_status=$?
wait "$react_pid"
react_status=$?
set -e
[[ "$node_status" -eq 2 && "$react_status" -eq 2 ]]
[[ -L "$concurrent_cache/$dependency_key" ]]
[[ -f "$concurrent_cache/$dependency_key/.complete" ]]
install_count=$(wc -l < "$concurrent_log")
[[ "$install_count" -ge 1 && "$install_count" -le 2 ]]
[[ $(find "$concurrent_cache" -maxdepth 1 -type d -name "$dependency_key.install.*" | wc -l) -eq 1 ]]

interrupt_cache="$temporary_directory/interrupt-cache"
interrupt_log="$temporary_directory/interrupt-install.log"
publish_bin="$temporary_directory/publish-bin"
mkdir -p "$publish_bin"
# shellcheck disable=SC2016 # The generated executable signals its invoking shell.
printf '%s\n' \
  '#!/bin/sh' \
  '"$FLOW_REAL_LN" "$@"' \
  'kill -TERM "$PPID"' \
  > "$publish_bin/ln"
chmod +x "$publish_bin/ln"
if PATH="$publish_bin:$fake_bin:$PATH" FLOW_REAL_LN="$real_ln" \
  FLOW_NPM_LOG="$interrupt_log" FLOW_DEPENDENCY_CACHE_DIR="$interrupt_cache" \
  bash "$script_directory/run-local.sh" react missing-journey >/dev/null 2>&1; then
  printf 'interrupted cache publication unexpectedly passed\n' >&2
  exit 1
fi
[[ -L "$interrupt_cache/$dependency_key" ]]
[[ -f "$interrupt_cache/$dependency_key/.complete" ]]
[[ $(find "$interrupt_cache" -maxdepth 1 -type d -name "$dependency_key.install.*" | wc -l) -eq 1 ]]
if PATH="$fake_bin:$PATH" FLOW_NPM_LOG="$interrupt_log" \
  FLOW_DEPENDENCY_CACHE_DIR="$interrupt_cache" \
  bash "$script_directory/run-local.sh" react missing-journey >/dev/null 2>&1; then
  printf 'unknown journey unexpectedly passed after interrupted publication\n' >&2
  exit 1
else
  status=$?
fi
[[ "$status" -eq 2 ]]
[[ $(wc -l < "$interrupt_log") -eq 1 ]]

unsafe_cache="$temporary_directory/unsafe-cache"
mkdir -p "$unsafe_cache"
chmod 0777 "$unsafe_cache"
if PATH="$fake_bin:$PATH" FLOW_NPM_LOG="$temporary_directory/unsafe-install.log" \
  FLOW_DEPENDENCY_CACHE_DIR="$unsafe_cache" \
  bash "$script_directory/run-local.sh" react missing-journey \
  > "$temporary_directory/unsafe-output.log" 2>&1; then
  printf 'unsafe cache unexpectedly passed\n' >&2
  exit 1
fi
grep -Fq 'flow dependency cache must not be writable by group or others' \
  "$temporary_directory/unsafe-output.log"

printf 'Local flow dependency cache regression tests passed\n'
