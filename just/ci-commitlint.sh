#!/usr/bin/env bash
set -Eeuo pipefail

base_sha=${CI_BASE_SHA:-}
if [[ -z "$base_sha" ]] && git rev-parse --verify origin/main >/dev/null 2>&1; then
  base_sha=$(git merge-base origin/main HEAD)
fi
if [[ -z "$base_sha" ]] && git rev-parse --verify HEAD^ >/dev/null 2>&1; then
  base_sha=$(git rev-parse HEAD^)
fi

if [[ -z "$base_sha" || "$base_sha" == "$(git rev-parse HEAD)" ]]; then
  echo 'No commit range is available; skipping commit-message lint.'
  exit
fi

npm_prefix=${XDG_CACHE_HOME:-$HOME/.cache}/rapida-ci-tools/npm
mkdir -p "$npm_prefix"
npm install --global --prefix "$npm_prefix" \
  @commitlint/cli@21.2.2 @commitlint/config-conventional@21.2.2
NODE_PATH="$npm_prefix/lib/node_modules${NODE_PATH:+:$NODE_PATH}" \
  "$npm_prefix/bin/commitlint" --from "$base_sha" --to HEAD --verbose
