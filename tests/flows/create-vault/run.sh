#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
status=0

for client in nodejs react; do
  printf 'Running create vault flow with %s\n' "$client"
  if ! "$script_directory/$client/run.sh"; then
    status=1
  fi
done

if [ "$status" -ne 0 ]; then
  exit "$status"
fi

echo 'Create vault flows passed'
