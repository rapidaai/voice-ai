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
    rm -f "$install_lock/owner"
    rmdir "$install_lock" 2>/dev/null || true
  fi
}

release_install_lock() {
  if [[ "$owns_install_lock" == true ]]; then
    rm -f "$install_lock/owner"
    rmdir "$install_lock"
    owns_install_lock=false
  fi
}

lock_owner_is_active() {
  local lock_pid
  local lock_started_at
  local process_started_at
  [[ -f "$install_lock/owner" ]] || return 1
  lock_pid=$(sed -n '1p' "$install_lock/owner")
  lock_started_at=$(sed -n '2p' "$install_lock/owner")
  [[ "$lock_pid" =~ ^[0-9]+$ && -n "$lock_started_at" ]] || return 1
  process_started_at=$(ps -p "$lock_pid" -o lstart= 2>/dev/null) || return 1
  [[ "$process_started_at" == "$lock_started_at" ]]
}

discard_stale_lock() {
  local stale_lock="$install_lock.stale.$$.$RANDOM"
  if mv "$install_lock" "$stale_lock" 2>/dev/null; then
    rm -rf "$stale_lock"
  fi
}
trap cleanup_install EXIT
trap 'exit 1' HUP INT TERM

umask 077
mkdir -p "$dependency_cache_root"
validate_private_directory "$dependency_cache_root"

if [[ ! -f "$dependency_directory/.complete" ]]; then
  wait_attempt=0
  while [[ "$owns_install_lock" == false ]]; do
    if mkdir "$install_lock" 2>/dev/null; then
      owns_install_lock=true
      printf '%s\n%s\n' "$$" "$(ps -p $$ -o lstart=)" > "$install_lock/owner"
      break
    fi
    if [[ -f "$dependency_directory/.complete" ]]; then
      break
    fi
    if [[ ! -f "$install_lock/owner" ]]; then
      sleep 1
    fi
    if ! lock_owner_is_active; then
      discard_stale_lock
      continue
    fi
    wait_attempt=$((wait_attempt + 1))
    if [[ "$wait_attempt" -ge 300 ]]; then
      printf 'timed out waiting for flow dependency cache installation: %s\n' "$install_lock" >&2
      exit 1
    fi
    sleep 1
  done

  if [[ "$owns_install_lock" == true ]]; then
    if [[ ! -f "$dependency_directory/.complete" ]]; then
      install_directory=$(mktemp -d "$dependency_cache_root/$dependency_key.install.XXXXXX")
      cp "$script_directory/package.json" "$script_directory/package-lock.json" "$install_directory/"
      npm ci --omit=dev --prefix "$install_directory"
      touch "$install_directory/.complete"
      if [[ -e "$dependency_directory" ]]; then
        rm -rf "$dependency_directory"
      fi
      mv "$install_directory" "$dependency_directory"
      install_directory=''
    fi
    release_install_lock
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
