#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
status=0

for flow in signin signup-organization-project create-assistant create-assistant-provider create-vault; do
  printf 'Running %s flow\n' "$flow"
  if ! "$script_directory/$flow/run.sh"; then
    status=1
  fi
done

if [ "$status" -ne 0 ]; then
  echo 'One or more user flows failed' >&2
  exit "$status"
fi

echo 'All user flows passed'
