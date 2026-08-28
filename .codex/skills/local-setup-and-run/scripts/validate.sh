#!/usr/bin/env bash
set -euo pipefail

skill_dir="$(cd "$(dirname "$0")/.." && pwd)"
repo_root="$(cd "$skill_dir/../../.." && pwd)"
skill_name="$(basename "$skill_dir")"

required=("SKILL.md" "references/checklist.md" "examples/sample.md" "scripts/validate.sh" "agents/openai.yaml")
for f in "${required[@]}"; do
  [[ -f "$skill_dir/$f" ]] || { echo "Missing required file: $f" >&2; exit 1; }
done

grep -q '^name:' "$skill_dir/SKILL.md" || { echo "SKILL.md missing name" >&2; exit 1; }
grep -q '^description:' "$skill_dir/SKILL.md" || { echo "SKILL.md missing description" >&2; exit 1; }
grep -q '^display_name:' "$skill_dir/agents/openai.yaml" || { echo "agents/openai.yaml missing display_name" >&2; exit 1; }
grep -q '^default_prompt:' "$skill_dir/agents/openai.yaml" || { echo "agents/openai.yaml missing default_prompt" >&2; exit 1; }

scan_files=("$skill_dir/SKILL.md" "$skill_dir/references/checklist.md" "$skill_dir/examples/sample.md" "$skill_dir/agents/openai.yaml")
if grep -E -i -n "(api[_-]?key|secret|token|password|bearer)\\s*[:=]\\s*[\\\"'][^\\\"']+[\\\"']" "${scan_files[@]}" >/dev/null; then
  echo "Potential hardcoded credential pattern found" >&2
  exit 1
fi

check_diff=0
if [[ "${1:-}" == "--check-diff" ]]; then
  check_diff=1
fi

if [[ $check_diff -eq 1 ]]; then
  cd "$repo_root"
  changed=()
  while IFS= read -r line; do
    [[ -n "$line" ]] && changed+=("$line")
  done < <(
    {
      git diff --name-only --diff-filter=ACMRTUXB HEAD -- . || true
      git ls-files --others --exclude-standard
    } | awk 'NF' | sort -u
  )

  allowed=(
    '^\.codex/skills/local-setup-and-run/'
    '^\.codex/skills/README.md$'
    '^\.codex/skills/SECURITY_GUIDELINES.md$'
    '^README.md$'
    '^justfile$'
    '^just/'
    '^docker-compose\.yml$'
    '^docker-compose\.knowledge\.yml$'
    '^SKILLS_QUICKSTART.md$'
  )

  viol=()
  for f in "${changed[@]}"; do
    ok=0
    for p in "${allowed[@]}"; do
      if [[ "$f" =~ $p ]]; then
        ok=1
        break
      fi
    done
    if [[ $ok -eq 0 ]]; then
      viol+=("$f")
    fi
  done

  if [[ ${#viol[@]} -gt 0 ]]; then
    echo "Files outside expected scope for $skill_name:" >&2
    printf ' - %s\n' "${viol[@]}" >&2
    exit 1
  fi
fi

echo "Codex skill validation passed: $skill_dir"
