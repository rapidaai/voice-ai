#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)

for flow in signin signup-organization-project create-assistant create-assistant-provider create-vault; do
  printf 'Running %s flow\n' "$flow"
  "$script_directory/$flow/run.sh"
done

echo 'All user flows passed'
