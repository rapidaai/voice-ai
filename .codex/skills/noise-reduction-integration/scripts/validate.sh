#!/usr/bin/env bash
set -euo pipefail

skill_dir="$(cd "$(dirname "$0")/.." && pwd)"
skill_name="$(basename "$skill_dir")"
repo_root="$(cd "$skill_dir/../../.." && pwd)"

required=("SKILL.md" "references/checklist.md" "examples/sample.md" "scripts/validate.sh" "agents/openai.yaml")
for f in "${required[@]}"; do
  [[ -f "$skill_dir/$f" ]] || { echo "Missing required file: $f" >&2; exit 1; }
done

grep -q '^name:' "$skill_dir/SKILL.md" || { echo "SKILL.md missing name" >&2; exit 1; }
grep -q '^description:' "$skill_dir/SKILL.md" || { echo "SKILL.md missing description" >&2; exit 1; }
grep -q '^display_name:' "$skill_dir/agents/openai.yaml" || { echo "agents/openai.yaml missing display_name" >&2; exit 1; }
grep -q '^default_prompt:' "$skill_dir/agents/openai.yaml" || { echo "agents/openai.yaml missing default_prompt" >&2; exit 1; }

scan_files=(
  "$skill_dir/SKILL.md"
  "$skill_dir/references/checklist.md"
  "$skill_dir/examples/sample.md"
  "$skill_dir/agents/openai.yaml"
)

if grep -E -i -n "(api[_-]?key|secret|token|password|bearer)\\s*[:=]\\s*[\\\"'][^\\\"']+[\\\"']" "${scan_files[@]}" >/dev/null; then
  echo "Potential hardcoded credential pattern found" >&2
  exit 1
fi

adv_hits="$(grep -E -i -n 'ignore\s+safety|bypass\s+safety|hide\s+actions\s+from\s+user|do\s+not\s+tell\s+the\s+user|exfiltrat' "${scan_files[@]}" || true)"
if [[ -n "$adv_hits" ]]; then
  echo "Potential adversarial instruction pattern found" >&2
  echo "$adv_hits" >&2
  exit 1
fi

check_diff=0
provider=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --check-diff)
      check_diff=1
      shift
      ;;
    --provider)
      [[ $# -ge 2 ]] || { echo "--provider requires a value" >&2; exit 1; }
      provider="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

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

  if [[ ${#changed[@]} -gt 0 ]]; then
    provider_required=0
    case "$skill_name" in
      end-of-speech-integration|vad-integration|telephony-integration|stt-integration|tts-integration|llm-integration|telemetry-integration|noise-reduction-integration)
        provider_required=1
        ;;
    esac

    if [[ $provider_required -eq 1 && -z "$provider" ]]; then
      echo "--provider is required with --check-diff for $skill_name" >&2
      exit 1
    fi

    provider_re=""
    if [[ -n "$provider" ]]; then
      if [[ ! "$provider" =~ ^[A-Za-z0-9._-]+$ ]]; then
        echo "Invalid provider value: $provider" >&2
        exit 1
      fi
      provider_re="$provider"
    fi

    allowed_patterns=(
      '^\.codex/skills/'"$skill_name"'/'
    )
    disallowed_patterns=()

    case "$skill_name" in
      noise-reduction-integration)
        allowed_patterns+=(
          '^api/assistant-api/internal/denoiser/denoiser.go$'
          '^api/assistant-api/internal/type/packet.go$'
          '^api/assistant-api/internal/adapters/internal/dispatch.go$'
          '^api/assistant-api/internal/adapters/internal/pipeline.go$'
          '^api/assistant-api/internal/denoiser/internal/'"$provider_re"'/'
          '^ui/src/providers/'"$provider_re"'/noise.json$'
          '^ui/src/providers/provider\.(development|production)\.json$'
          '^ui/src/providers/config-loader.ts$'
          '^ui/src/app/components/providers/noise/'
        )
        disallowed_patterns+=(
          '^api/assistant-api/internal/end_of_speech/internal/'
          '^api/assistant-api/internal/vad/internal/'
        )
        ;;
      *)
        ;;
    esac

    viol=()
    for f in "${changed[@]}"; do
      blocked=0
      for dp in "${disallowed_patterns[@]}"; do
        if [[ "$f" =~ $dp ]]; then
          blocked=1
          break
        fi
      done
      if [[ $blocked -eq 1 ]]; then
        viol+=("$f")
        continue
      fi

      ok=0
      for p in "${allowed_patterns[@]}"; do
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
fi

echo "Codex skill validation passed: $skill_dir"
