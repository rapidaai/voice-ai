#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/report.sh"
readonly flow_client="${1:-all}"

case "$flow_client" in
  all | nodejs | react | rest) ;;
  *)
    printf 'unsupported flow client: %s\n' "$flow_client" >&2
    exit 2
    ;;
esac
export FLOW_CLIENT="$flow_client"
if [ "$#" -gt 0 ]; then
  shift
fi
readonly flow_manifest="$script_directory/manifest.txt"
readonly shard_index="${FLOW_SHARD_INDEX:-1}"
readonly shard_total="${FLOW_SHARD_TOTAL:-1}"

case "$shard_index:$shard_total" in
  *[!0-9:]* | :* | *:) printf 'flow shard values must be positive integers\n' >&2; exit 2 ;;
esac
if [ "$shard_index" -lt 1 ] || [ "$shard_total" -lt 1 ] || [ "$shard_index" -gt "$shard_total" ]; then
  printf 'invalid flow shard %s of %s\n' "$shard_index" "$shard_total" >&2
  exit 2
fi

for requested_journey in "$@"; do
  if ! grep -Fqx "$requested_journey" "$flow_manifest"; then
    printf 'unknown flow journey: %s\n' "$requested_journey" >&2
    exit 2
  fi
done

journey_requested() {
  candidate=$1
  shift
  if [ "$#" -eq 0 ]; then
    return 0
  fi
  for requested_journey in "$@"; do
    if [ "$candidate" = "$requested_journey" ]; then
      return 0
    fi
  done
  return 1
}

report_directory=${FLOW_REPORT_DIRECTORY:-/reports}
if [ -d "$report_directory" ] && [ -w "$report_directory" ]; then
  export FLOW_REPORT_FILE="$report_directory/flows.tsv"
  flow_summary_file="$report_directory/flows.md"
else
  export FLOW_REPORT_FILE="${TMPDIR:-/tmp}/rapida-flow-results-$$.tsv"
  flow_summary_file="${TMPDIR:-/tmp}/rapida-flow-summary-$$.md"
  cleanup_reports() {
    rm -f "$FLOW_REPORT_FILE" "$flow_summary_file"
  }
  trap cleanup_reports EXIT HUP INT TERM
fi
: > "$FLOW_REPORT_FILE"
status=0
failed_flows=''
selected_journeys=0
executed_journeys=0

while IFS= read -r flow || [ -n "$flow" ]; do
  if ! journey_requested "$flow" "$@"; then
    continue
  fi
  selected_journeys=$((selected_journeys + 1))
  if [ $(((selected_journeys - 1) % shard_total + 1)) -ne "$shard_index" ]; then
    continue
  fi
  executed_journeys=$((executed_journeys + 1))
  printf '\n##### Flow group: %s #####\n' "$flow"
  if ! "$script_directory/$flow/run.sh"; then
    status=1
    failed_flows="$failed_flows $flow"
  fi
done < "$flow_manifest"

if [ "$executed_journeys" -eq 0 ]; then
  printf 'no flow journeys selected for shard %s of %s\n' "$shard_index" "$shard_total" >&2
  exit 2
fi

write_flow_summary "$flow_summary_file"
cat "$flow_summary_file"

if [ "$status" -ne 0 ]; then
  printf 'Failed flow groups:%s\n' "$failed_flows" >&2
  if [ "${GITHUB_ACTIONS:-false}" = 'true' ]; then
    printf '::error title=User flow suite failed::Failed flow groups:%s\n' "$failed_flows"
  fi
  exit "$status"
fi

printf 'All %s user flows passed for shard %s of %s\n' \
  "$flow_client" "$shard_index" "$shard_total"
