#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)

for client in nodejs react; do
  printf 'Running signup, organization, and project flow with %s client\n' "$client"
  "$script_directory/$client/run.sh"
done

echo 'All signup, organization, and project flows passed'
