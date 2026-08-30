#!/usr/bin/env bash
set -Eeuo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
requirements="$repo_root/just/requirements-ci.txt"
requirements_hash=$(python3 -c 'import hashlib, pathlib, sys; print(hashlib.sha256(pathlib.Path(sys.argv[1]).read_bytes()).hexdigest()[:16])' "$requirements")
venv_root=${XDG_CACHE_HOME:-$HOME/.cache}/rapida-ci-tools/python
venv="$venv_root/$requirements_hash"
install_marker="$venv/.requirements-installed"

if [[ ! -x "$venv/bin/python3" || ! -f "$install_marker" ]]; then
  mkdir -p "$venv_root"
  rm -rf "$venv"
  python3 -m venv "$venv"
  "$venv/bin/python3" -m pip install --disable-pip-version-check --requirement "$requirements"
  touch "$install_marker"
fi

if [[ "${1:-}" == "--bin-dir" ]]; then
  printf '%s\n' "$venv/bin"
  exit
fi

exec "$venv/bin/python3" "$@"
