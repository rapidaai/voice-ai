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
dependency_link="$dependency_cache_root/$dependency_key"
readonly dependency_link
dependency_directory=''
install_directory=''

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
}

select_dependency_directory() {
  local link_target
  local candidate
  if [[ -L "$dependency_link" ]]; then
    link_target=$(readlink "$dependency_link")
    case "$link_target" in
      "$dependency_key".install.*) ;;
      *)
        printf 'flow dependency cache link has an unexpected target: %s\n' "$dependency_link" >&2
        return 1
        ;;
    esac
    candidate="$dependency_cache_root/$link_target"
  elif [[ -d "$dependency_link" ]]; then
    candidate=$dependency_link
  else
    return 1
  fi
  validate_private_directory "$candidate"
  [[ -f "$candidate/.complete" ]] || {
    printf 'flow dependency cache is incomplete: %s\n' "$candidate" >&2
    return 1
  }
  dependency_directory=$candidate
}
trap cleanup_install EXIT
trap 'exit 1' HUP INT TERM

umask 077
mkdir -p "$dependency_cache_root"
validate_private_directory "$dependency_cache_root"

if [[ -e "$dependency_link" || -L "$dependency_link" ]]; then
  select_dependency_directory
else
  install_directory=$(mktemp -d "$dependency_cache_root/$dependency_key.install.XXXXXX")
  cp "$script_directory/package.json" "$script_directory/package-lock.json" "$install_directory/"
  npm ci --omit=dev --prefix "$install_directory"
  touch "$install_directory/.complete"
  if ln -s "$(basename "$install_directory")" "$dependency_link" 2>/dev/null; then
    dependency_directory=$install_directory
    install_directory=''
  else
    select_dependency_directory
    rm -rf "$install_directory"
    install_directory=''
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
