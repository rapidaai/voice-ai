#!/bin/sh

flow_process_id=''
flow_cleanup_command=''
flow_signal_pending=false

run_flow_cleanup() {
  trap '' HUP INT TERM
  if [ -n "$flow_cleanup_command" ]; then
    "$flow_cleanup_command"
  fi
}

exit_flow_on_signal() {
  trap '' HUP INT TERM
  if [ -n "$flow_process_id" ]; then
    kill "$flow_process_id" 2>/dev/null || true
    wait "$flow_process_id" 2>/dev/null || true
    flow_process_id=''
  fi
  exit 1
}

defer_flow_signal() {
  flow_signal_pending=true
}

register_flow_cleanup() {
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
