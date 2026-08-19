#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$PROJECT_ROOT"

echo "> setting up git hooks ..."

git config --local commit.template "$PROJECT_ROOT/githooks/gitmessage.txt"
git config --local commit.cleanup strip

if [ -x "$PROJECT_ROOT/bin/pre-commit" ]; then
  PRE_COMMIT="$PROJECT_ROOT/bin/pre-commit"
elif command -v pre-commit >/dev/null 2>&1; then
  PRE_COMMIT="pre-commit"
else
  echo "error: pre-commit was not found." >&2
  echo "Use the checked-in runner at bin/pre-commit or install pre-commit locally." >&2
  exit 1
fi

"$PRE_COMMIT" install --install-hooks --hook-type pre-commit --hook-type commit-msg --hook-type pre-push --overwrite

echo "> git hooks installed successfully"
