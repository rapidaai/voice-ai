#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)

for client in nodejs react; do
  printf 'Running create vault flow with %s\n' "$client"
  "$script_directory/$client/run.sh"
done

echo 'Create vault flows passed'
