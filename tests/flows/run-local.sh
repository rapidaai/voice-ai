#!/usr/bin/env bash
set -Eeuo pipefail

readonly flow_client="${1:?usage: run-local.sh <nodejs|react> [journey]}"
case "$flow_client" in
  nodejs | react) ;;
  *)
    printf 'unsupported local flow client: %s\n' "$flow_client" >&2
    exit 2
    ;;
esac
shift

script_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
readonly script_directory
dependency_key=$(shasum -a 256 \
  "$script_directory/package.json" "$script_directory/package-lock.json" \
  | shasum -a 256 | awk '{print $1}')
readonly dependency_cache_root="${FLOW_DEPENDENCY_CACHE_DIR:-${TMPDIR:-/tmp}/rapida-flow-dependencies-cache}"
dependency_directory="$dependency_cache_root/$dependency_key"
readonly dependency_directory

if [[ ! -f "$dependency_directory/.complete" ]]; then
  mkdir -p "$dependency_directory"
  cp "$script_directory/package.json" "$script_directory/package-lock.json" "$dependency_directory/"
  npm ci --omit=dev --prefix "$dependency_directory"
  touch "$dependency_directory/.complete"
fi

export NODE_PATH="$dependency_directory/node_modules"
export FLOW_POSTGRES_HOST=${FLOW_POSTGRES_HOST:-localhost}
export PGPASSWORD=${PGPASSWORD:-rapida_db_password}
export FLOW_ARI_HOST=${FLOW_ARI_HOST:-localhost}

if [[ "$flow_client" == nodejs ]]; then
  export WEB_API_ENDPOINT=${WEB_API_ENDPOINT:-localhost:9001}
  export ASSISTANT_API_ENDPOINT=${ASSISTANT_API_ENDPOINT:-localhost:9007}
else
  export WEB_API_REACT_ENDPOINT=${WEB_API_REACT_ENDPOINT:-http://localhost:9001}
  export ASSISTANT_API_REACT_ENDPOINT=${ASSISTANT_API_REACT_ENDPOINT:-http://localhost:9007}
fi

"$script_directory/run.sh" "$flow_client" "$@"
