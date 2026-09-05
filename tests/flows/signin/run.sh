#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)

for client in nodejs react rest; do
  printf 'Running signin flow with %s client\n' "$client"
  "$script_directory/$client/run.sh"
done

echo 'All signin flows passed'
