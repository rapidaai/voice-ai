#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
status=0

for client in nodejs react rest; do
  printf 'Running signin flow with %s client\n' "$client"
  if ! "$script_directory/$client/run.sh"; then
    status=1
  fi
done

if [ "$status" -ne 0 ]; then
  exit "$status"
fi

echo 'All signin flows passed'
