#!/usr/bin/env bash
set -Eeuo pipefail

script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/rapida-run-local-test.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM
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

stale_cache="$temporary_directory/stale-cache"
stale_log="$temporary_directory/stale-install.log"
mkdir -p "$stale_cache/$dependency_key.lock"
printf '%s\n%s\n' '999999' 'Mon Jan  1 00:00:00 2001' \
  > "$stale_cache/$dependency_key.lock/owner"
if PATH="$fake_bin:$PATH" FLOW_NPM_LOG="$stale_log" \
  FLOW_DEPENDENCY_CACHE_DIR="$stale_cache" \
  bash "$script_directory/run-local.sh" react missing-journey >/dev/null 2>&1; then
  printf 'unknown journey unexpectedly passed after stale lock recovery\n' >&2
  exit 1
else
  status=$?
fi
[[ "$status" -eq 2 ]]
[[ -f "$stale_cache/$dependency_key/.complete" ]]
[[ ! -d "$stale_cache/$dependency_key.lock" ]]
[[ $(wc -l < "$stale_log") -eq 1 ]]

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
[[ -f "$concurrent_cache/$dependency_key/.complete" ]]
[[ ! -d "$concurrent_cache/$dependency_key.lock" ]]
[[ $(wc -l < "$concurrent_log") -eq 1 ]]

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
