#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
# shellcheck disable=SC1091
. "$script_directory/../report.sh"

run_flow_clients "$script_directory" 'Create assistant provider' nodejs react rest
