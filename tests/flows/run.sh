#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)

for flow in signin signup-organization-project; do
  printf 'Running %s flow\n' "$flow"
  "$script_directory/$flow/run.sh"
done

echo 'All user flows passed'
