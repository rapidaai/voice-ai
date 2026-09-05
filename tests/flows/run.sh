#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/report.sh"

if [ -d /reports ] && [ -w /reports ]; then
  export FLOW_REPORT_FILE=/reports/flows.tsv
  flow_summary_file=/reports/flows.md
else
  export FLOW_REPORT_FILE="${TMPDIR:-/tmp}/rapida-flow-results-$$.tsv"
  flow_summary_file="${TMPDIR:-/tmp}/rapida-flow-summary-$$.md"
fi
: > "$FLOW_REPORT_FILE"
status=0
failed_flows=''

for flow in signin signup-organization-project create-assistant create-assistant-provider create-vault create-assistant-deployment-phone-call; do
  printf '\n##### Flow group: %s #####\n' "$flow"
  if ! "$script_directory/$flow/run.sh"; then
    status=1
    failed_flows="$failed_flows $flow"
  fi
done

write_flow_summary "$flow_summary_file"
cat "$flow_summary_file"

if [ "$status" -ne 0 ]; then
  printf 'Failed flow groups:%s\n' "$failed_flows" >&2
  if [ "${GITHUB_ACTIONS:-false}" = 'true' ]; then
    printf '::error title=User flow suite failed::Failed flow groups:%s\n' "$failed_flows"
  fi
  exit "$status"
fi

echo 'All user flows passed'
