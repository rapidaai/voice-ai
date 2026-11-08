#!/bin/sh

flow_process_id=''
flow_cleanup_command=''
flow_signal_pending=false
flow_process_termination_timeout_seconds=${FLOW_PROCESS_TERMINATION_TIMEOUT_SECONDS:-5}
flow_cleanup_timeout_seconds=${FLOW_CLEANUP_TIMEOUT_SECONDS:-30}

flow_process_is_running() {
  kill -0 -- "-$1" 2>/dev/null || kill -0 "$1" 2>/dev/null
}

signal_flow_process_tree() {
  signal=$1
  process_id=$2
  kill "-$signal" -- "-$process_id" 2>/dev/null || kill "-$signal" "$process_id" 2>/dev/null || true
}

wait_for_flow_deadline() {
  deadline_ticks=$(($1 * 10))
  while [ "$deadline_ticks" -gt 0 ]; do
    sleep 0.1
    deadline_ticks=$((deadline_ticks - 1))
  done
}

terminate_flow_process() {
  process_id=$flow_process_id
  [ -n "$process_id" ] || return
  signal_flow_process_tree TERM "$process_id"
  (
    wait_for_flow_deadline "$flow_process_termination_timeout_seconds"
    if flow_process_is_running "$process_id"; then
      printf 'flow process exceeded %s seconds while terminating; sending KILL\n' \
        "$flow_process_termination_timeout_seconds" >&2
      signal_flow_process_tree KILL "$process_id"
    fi
  ) &
  termination_watchdog_id=$!
  wait "$process_id" 2>/dev/null || true
  if kill -0 -- "-$process_id" 2>/dev/null; then
    wait "$termination_watchdog_id" 2>/dev/null || true
  else
    signal_flow_process_tree KILL "$termination_watchdog_id"
    wait "$termination_watchdog_id" 2>/dev/null || true
  fi
  flow_process_id=''
}

run_flow_cleanup() {
  trap '' HUP INT TERM
  [ -n "$flow_cleanup_command" ] || return
  PGCONNECT_TIMEOUT=${PGCONNECT_TIMEOUT:-$flow_cleanup_timeout_seconds}
  PGOPTIONS="${PGOPTIONS:+$PGOPTIONS }-c lock_timeout=${flow_cleanup_timeout_seconds}s -c statement_timeout=${flow_cleanup_timeout_seconds}s"
  export PGCONNECT_TIMEOUT PGOPTIONS
  "$flow_cleanup_command" &
  cleanup_process_id=$!
  (
    wait_for_flow_deadline "$flow_cleanup_timeout_seconds"
    if flow_process_is_running "$cleanup_process_id"; then
      printf 'flow cleanup exceeded %s seconds; terminating cleanup\n' \
        "$flow_cleanup_timeout_seconds" >&2
      signal_flow_process_tree TERM "$cleanup_process_id"
      sleep 1
      signal_flow_process_tree KILL "$cleanup_process_id"
    fi
  ) &
  cleanup_watchdog_id=$!
  cleanup_status=0
  wait "$cleanup_process_id" || cleanup_status=$?
  signal_flow_process_tree KILL "$cleanup_watchdog_id"
  wait "$cleanup_watchdog_id" 2>/dev/null || true
  return "$cleanup_status"
}

exit_flow_on_signal() {
  trap '' HUP INT TERM
  terminate_flow_process
  exit 1
}

defer_flow_signal() {
  flow_signal_pending=true
}

register_flow_cleanup() {
  case "$flow_process_termination_timeout_seconds" in
    '' | 0 | *[!0-9]*) printf 'FLOW_PROCESS_TERMINATION_TIMEOUT_SECONDS must be a positive integer\n' >&2; return 2 ;;
  esac
  case "$flow_cleanup_timeout_seconds" in
    '' | 0 | *[!0-9]*) printf 'FLOW_CLEANUP_TIMEOUT_SECONDS must be a positive integer\n' >&2; return 2 ;;
  esac
  flow_cleanup_command=$1
  trap run_flow_cleanup EXIT
  trap exit_flow_on_signal HUP INT TERM
}

start_flow_process() {
  flow_signal_pending=false
  trap defer_flow_signal HUP INT TERM
  "$@" &
  flow_process_id=$!
  trap exit_flow_on_signal HUP INT TERM
  if [ "$flow_signal_pending" = true ]; then
    exit_flow_on_signal
  fi
}

run_flow_process() {
  flow_process_status=0
  start_flow_process "$@"
  wait "$flow_process_id" || flow_process_status=$?
  flow_process_id=''
  return "$flow_process_status"
}
