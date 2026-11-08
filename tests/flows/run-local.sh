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
cache_user_id=$(id -u)
readonly cache_user_id
readonly dependency_cache_root="${FLOW_DEPENDENCY_CACHE_DIR:-${XDG_CACHE_HOME:-${TMPDIR:-/tmp}}/rapida-flow-dependencies-$cache_user_id}"
dependency_directory="$dependency_cache_root/$dependency_key"
readonly dependency_directory
install_lock="$dependency_cache_root/$dependency_key.lock"
readonly install_lock
install_directory=''
owns_install_lock=false

validate_private_directory() {
  local directory=$1
  local permissions
  [[ -d "$directory" && ! -L "$directory" && -O "$directory" ]] || {
    printf 'flow dependency cache must be a user-owned directory: %s\n' "$directory" >&2
    return 1
  }
  if [[ $(uname -s) == Darwin ]]; then
    permissions=$(stat -f '%Lp' "$directory")
  else
    permissions=$(stat -c '%a' "$directory")
  fi
  if (((8#$permissions & 8#022) != 0)); then
    printf 'flow dependency cache must not be writable by group or others: %s\n' "$directory" >&2
    return 1
  fi
}

cleanup_install() {
  if [[ -n "$install_directory" && -d "$install_directory" ]]; then
    rm -rf "$install_directory"
  fi
  if [[ "$owns_install_lock" == true ]]; then
    rmdir "$install_lock" 2>/dev/null || true
  fi
}
trap cleanup_install EXIT HUP INT TERM

umask 077
mkdir -p "$dependency_cache_root"
validate_private_directory "$dependency_cache_root"

if [[ ! -f "$dependency_directory/.complete" ]]; then
  wait_attempt=0
  while ! mkdir "$install_lock" 2>/dev/null; do
    if [[ -f "$dependency_directory/.complete" ]]; then
      break
    fi
    wait_attempt=$((wait_attempt + 1))
    if [[ "$wait_attempt" -ge 300 ]]; then
      printf 'timed out waiting for flow dependency cache installation: %s\n' "$install_lock" >&2
      exit 1
    fi
    sleep 1
  done

  if [[ -d "$install_lock" && ! -f "$dependency_directory/.complete" ]]; then
    owns_install_lock=true
    install_directory=$(mktemp -d "$dependency_cache_root/$dependency_key.install.XXXXXX")
    cp "$script_directory/package.json" "$script_directory/package-lock.json" "$install_directory/"
    npm ci --omit=dev --prefix "$install_directory"
    touch "$install_directory/.complete"
    if [[ -e "$dependency_directory" ]]; then
      rm -rf "$dependency_directory"
    fi
    mv "$install_directory" "$dependency_directory"
    install_directory=''
    rmdir "$install_lock"
    owns_install_lock=false
  fi
fi

validate_private_directory "$dependency_directory"

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
