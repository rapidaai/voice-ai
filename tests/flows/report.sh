#!/bin/sh

run_flow_clients() {
  flow_directory=$1
  flow_name=$2
  shift 2
  failed_clients=''

  for client in "$@"; do
    label="$flow_name / $client"
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

    if [ -n "${FLOW_REPORT_FILE:-}" ]; then
      printf '%s\t%s\t%s\n' "$flow_name" "$client" "$result" >> "$FLOW_REPORT_FILE"
    fi
    if [ "${GITHUB_ACTIONS:-false}" = 'true' ]; then
      printf '::endgroup::\n'
    fi
  done

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
    printf '| Flow | Client | Result |\n'
    printf '| --- | --- | --- |\n'
    while IFS="$(printf '\t')" read -r flow client result; do
      if [ "$result" = 'passed' ]; then
        result='✅ Passed'
      else
        result='❌ Failed'
      fi
      printf '| %s | %s | %s |\n' "$flow" "$client" "$result"
    done < "$FLOW_REPORT_FILE"
  } > "$summary_file"
}
