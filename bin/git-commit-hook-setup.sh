#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$PROJECT_ROOT"

echo "> setting up git hooks ..."

git config --local commit.template "$PROJECT_ROOT/githooks/gitmessage.txt"
git config --local commit.cleanup strip
git config --local core.hooksPath githooks

for hook in pre-commit commit-msg pre-push; do
  if [[ ! -x "$PROJECT_ROOT/githooks/$hook" ]]; then
    echo "error: githooks/$hook is missing or not executable." >&2
    exit 1
  fi
done

echo "> git hooks installed successfully"
