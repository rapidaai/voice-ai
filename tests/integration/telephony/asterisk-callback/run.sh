#!/bin/sh
set -eu

script_directory=$(cd "$(dirname "$0")" && pwd)
exec "$script_directory/../callbacks/run.sh" asterisk
