#!/usr/bin/env bash
set -Eeuo pipefail

errors=0
docker_ready=0

echo 'Running Docker preflight checks...'
if ! command -v docker >/dev/null 2>&1; then
  echo 'Docker CLI not found. Install Docker Desktop or Docker Engine.' >&2
  errors=$((errors + 1))
elif ! python3 just/run-with-timeout.py 15 docker info >/dev/null 2>&1; then
  echo 'Docker daemon is not reachable. Start Docker and retry.' >&2
  errors=$((errors + 1))
else
  echo 'Docker daemon is reachable.'
  docker_ready=1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo 'Docker Compose plugin not found.' >&2
  errors=$((errors + 1))
else
  echo 'Docker Compose is available.'
fi

free_kb=$(df -Pk "$HOME" | awk 'NR == 2 {print $4}')
minimum_kb=$((20 * 1024 * 1024))
if [[ -z "$free_kb" || "$free_kb" -lt "$minimum_kb" ]]; then
  free_gb=$(awk "BEGIN {printf \"%.1f\", ${free_kb:-0}/1024/1024}")
  echo "Low disk space: ${free_gb}GB free, at least 20GB required for a first Docker build." >&2
  errors=$((errors + 1))
else
  free_gb=$(awk "BEGIN {printf \"%.1f\", $free_kb/1024/1024}")
  echo "Disk space looks good (${free_gb}GB free)."
fi

if [[ ${DOCTOR_SKIP_CACHE_CHECK:-0} != 1 && $docker_ready == 1 ]]; then
  build_cache_mb=$(python3 just/run-with-timeout.py 30 docker system df --format '{{.Type}} {{.Size}}' 2>/dev/null | awk '$1 == "Build" && $2 == "Cache" {size=$3} END {
    if (size ~ /kB$/) {sub(/kB$/, "", size); mb=size/1024}
    else if (size ~ /MB$/) {sub(/MB$/, "", size); mb=size}
    else if (size ~ /GB$/) {sub(/GB$/, "", size); mb=size*1024}
    else if (size ~ /TB$/) {sub(/TB$/, "", size); mb=size*1024*1024}
    else if (size ~ /B$/) {sub(/B$/, "", size); mb=size/(1024*1024)}
    else {mb=-1}
    if (mb >= 0) printf "%.0f", mb
  }')
  if [[ -n "$build_cache_mb" && "$build_cache_mb" -gt 12288 ]]; then
    build_cache_gb=$(awk "BEGIN {printf \"%.1f\", $build_cache_mb/1024}")
    echo "Docker build cache is large (${build_cache_gb}GB). Consider: docker builder prune -af"
  fi
fi

for directory in "$HOME/rapida-data/assets" "$HOME/rapida-data/assets/db" "$HOME/rapida-data/assets/redis"; do
  mkdir -p "$directory"
  if [[ ! -w "$directory" ]]; then
    echo "Directory is not writable: $directory" >&2
    errors=$((errors + 1))
  fi
done

if [[ ${DOCTOR_SKIP_PORT_CHECK:-0} != 1 ]] && command -v lsof >/dev/null 2>&1; then
  for port in 3000 8080 9004 9005 9007 4573; do
    owner=$(lsof -nP -iTCP:"$port" -sTCP:LISTEN 2>/dev/null | awk 'NR == 2 {print $1}' || true)
    if [[ -n "$owner" && ! "$owner" =~ ^(com\.docke|docker-proxy|docker)$ ]]; then
      echo "TCP port $port is already in use by $owner." >&2
      errors=$((errors + 1))
    fi
  done
fi

if ((errors > 0)); then
  echo "Preflight failed with $errors issue(s)." >&2
  exit 1
fi

echo 'Preflight checks passed.'
