#!/bin/sh

run_flow_clients() {
  flow_directory=$1
  flow_name=$2
  shift 2
  declared_clients=" $* "
  failed_clients=''
  selected_clients=0

  for client in "$@"; do
    if [ "${FLOW_CLIENT:-all}" != 'all' ] && [ "$client" != "$FLOW_CLIENT" ]; then
      continue
    fi
    selected_clients=$((selected_clients + 1))
    label="$flow_name / $client"
    started_at=$(date +%s)
    printf '\n=== Running %s ===\n' "$label"
    if [ "${GITHUB_ACTIONS:-false}" = 'true' ]; then
      printf '::group::%s\n' "$label"
    fi

    if "$flow_directory/$client/run.sh"; then
      result='passed'
      printf 'PASS: %s\n' "$label"
    else
      result='failed'
      failed_clients="$failed_clients $client"
      printf 'FAIL: %s\n' "$label" >&2
      if [ "${GITHUB_ACTIONS:-false}" = 'true' ]; then
        printf '::error title=User flow failed::%s\n' "$label"
      fi
    fi
    duration_seconds=$(($(date +%s) - started_at))

    if [ -n "${FLOW_REPORT_FILE:-}" ]; then
      printf '%s\t%s\t%s\t%s\n' \
        "$flow_name" "$client" "$result" "$duration_seconds" >> "$FLOW_REPORT_FILE"
    fi
    if [ "${GITHUB_ACTIONS:-false}" = 'true' ]; then
      printf '::endgroup::\n'
    fi
  done

  if [ "$selected_clients" -eq 0 ]; then
    printf '%s is not available for %s\n' "$flow_name" "${FLOW_CLIENT:-all}"
    if [ -n "${FLOW_REPORT_FILE:-}" ]; then
      printf '%s\t%s\tskipped\t0\n' \
        "$flow_name" "${FLOW_CLIENT:-all}" >> "$FLOW_REPORT_FILE"
    fi
    return
  fi

  if [ "${FLOW_CLIENT:-all}" = 'all' ] && [ -n "${FLOW_REPORT_FILE:-}" ]; then
    for supported_client in nodejs react rest; do
      case "$declared_clients" in
        *" $supported_client "*) ;;
        *) printf '%s\t%s\tskipped\t0\n' \
          "$flow_name" "$supported_client" >> "$FLOW_REPORT_FILE" ;;
      esac
    done
  fi

  if [ -n "$failed_clients" ]; then
    printf 'Failed clients for %s:%s\n' "$flow_name" "$failed_clients" >&2
    return 1
  fi

  printf '%s passed for all clients\n' "$flow_name"
}

write_flow_summary() {
  summary_file=$1
  {
    printf '## User Flow Results\n\n'
    printf '| Flow | Client | Result | Duration |\n'
    printf '| --- | --- | --- | ---: |\n'
    while IFS="$(printf '\t')" read -r flow client result duration_seconds; do
      case "$result" in
        passed) result='✅ Passed' ;;
        failed) result='❌ Failed' ;;
        skipped) result='⏭️ Not supported' ;;
      esac
      printf '| %s | %s | %s | %ss |\n' \
        "$flow" "$client" "$result" "$duration_seconds"
    done < "$FLOW_REPORT_FILE"
  } > "$summary_file"
}
