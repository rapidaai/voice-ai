#!/usr/bin/env bash
set -Eeuo pipefail

if ! command -v docker >/dev/null 2>&1; then
  echo 'Docker CLI not found. Install Docker Desktop or Docker Engine.' >&2
  exit 1
fi

if ! python3 just/run-with-timeout.py 15 docker info >/dev/null 2>&1; then
  echo 'Docker daemon is not reachable. Start Docker and retry.' >&2
  exit 1
fi
