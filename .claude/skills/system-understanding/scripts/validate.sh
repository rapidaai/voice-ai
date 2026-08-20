#!/usr/bin/env bash
set -euo pipefail

skill_dir="$(cd "$(dirname "$0")/.." && pwd)"
skill_name="$(basename "$skill_dir")"
repo_root="$(cd "$skill_dir/../../.." && pwd)"

required=("SKILL.md" "template.md" "examples/sample.md" "scripts/validate.sh")
for f in "${required[@]}"; do
  [[ -f "$skill_dir/$f" ]] || { echo "Missing required file: $f" >&2; exit 1; }
done

grep -q '^name:' "$skill_dir/SKILL.md" || { echo "SKILL.md missing name" >&2; exit 1; }
grep -q '^description:' "$skill_dir/SKILL.md" || { echo "SKILL.md missing description" >&2; exit 1; }

scan_files=("$skill_dir/SKILL.md" "$skill_dir/template.md" "$skill_dir/examples/sample.md")
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

    allowed_patterns=('^\.claude/skills/'"$skill_name"'/')
    disallowed_patterns=()

    case "$skill_name" in
      end-of-speech-integration)
        allowed_patterns+=(
          '^api/assistant-api/internal/end_of_speech/end_of_speech.go$'
          '^api/assistant-api/internal/type/end_of_speech.go$'
          '^api/assistant-api/internal/type/packet.go$'
          '^api/assistant-api/internal/adapters/internal/dispatch.go$'
          '^api/assistant-api/internal/end_of_speech/internal/'"$provider_re"'/'
          '^ui/src/providers/'"$provider_re"'/eos.json$'
          '^ui/src/providers/'"$provider_re"'/model-options.json$'
          '^ui/src/providers/config-loader.ts$'
          '^ui/src/app/components/providers/end-of-speech/'
        )
        disallowed_patterns+=(
          '^api/assistant-api/internal/vad/internal/'
        )
        ;;
      vad-integration)
        allowed_patterns+=(
          '^api/assistant-api/internal/vad/vad.go$'
          '^api/assistant-api/internal/type/vad.go$'
          '^api/assistant-api/internal/type/packet.go$'
          '^api/assistant-api/internal/adapters/internal/dispatch.go$'
          '^api/assistant-api/internal/vad/internal/'"$provider_re"'/'
          '^ui/src/providers/'"$provider_re"'/vad.json$'
          '^ui/src/providers/config-loader.ts$'
          '^ui/src/app/components/providers/vad/'
        )
        disallowed_patterns+=(
          '^api/assistant-api/internal/end_of_speech/internal/'
        )
        ;;
      telephony-integration)
        allowed_patterns+=(
          '^api/assistant-api/internal/channel/telephony/telephony.go$'
          '^api/assistant-api/internal/channel/telephony/inbound.go$'
          '^api/assistant-api/internal/channel/telephony/outbound.go$'
          '^api/assistant-api/internal/type/telephony.go$'
          '^api/assistant-api/internal/type/streamer.go$'
          '^api/assistant-api/internal/channel/telephony/internal/'"$provider_re"'/'
          '^ui/src/providers/provider\.(development|production)\.json$'
          '^ui/src/app/components/providers/telephony/'
        )
        ;;
      stt-integration)
        allowed_patterns+=(
          '^api/assistant-api/internal/transformer/transformer.go$'
          '^api/assistant-api/internal/type/stt_transformer.go$'
          '^api/assistant-api/internal/type/packet.go$'
          '^api/assistant-api/internal/transformer/stt_integration_test.go$'
          '^api/assistant-api/internal/transformer/'"$provider_re"'/'
          '^ui/src/providers/provider\.(development|production)\.json$'
          '^ui/src/providers/index.ts$'
          '^ui/src/providers/config-loader.ts$'
          '^ui/src/providers/'"$provider_re"'/stt.json$'
          '^ui/src/providers/'"$provider_re"'/speech-to-text-models.json$'
          '^ui/src/providers/'"$provider_re"'/speech-to-text-languages.json$'
          '^ui/src/providers/'"$provider_re"'/speech-to-text-language.json$'
          '^ui/src/providers/'"$provider_re"'/speech-to-text-model.json$'
          '^ui/src/providers/'"$provider_re"'/languages.json$'
          '^ui/src/app/components/providers/speech-to-text/'
        )
        ;;
      tts-integration)
        allowed_patterns+=(
          '^api/assistant-api/internal/transformer/transformer.go$'
          '^api/assistant-api/internal/type/tts_transformer.go$'
          '^api/assistant-api/internal/type/packet.go$'
          '^api/assistant-api/internal/transformer/tts_integration_test.go$'
          '^api/assistant-api/internal/transformer/'"$provider_re"'/'
          '^ui/src/providers/provider\.(development|production)\.json$'
          '^ui/src/providers/index.ts$'
          '^ui/src/providers/config-loader.ts$'
          '^ui/src/providers/'"$provider_re"'/tts.json$'
          '^ui/src/providers/'"$provider_re"'/voices.json$'
          '^ui/src/providers/'"$provider_re"'/text-to-speech-voices.json$'
          '^ui/src/providers/'"$provider_re"'/languages.json$'
          '^ui/src/providers/'"$provider_re"'/text-to-speech-models.json$'
          '^ui/src/providers/'"$provider_re"'/models.json$'
          '^ui/src/app/components/providers/text-to-speech/'
        )
        ;;
      llm-integration)
        allowed_patterns+=(
          '^api/integration-api/internal/caller/caller.go$'
          '^api/integration-api/internal/type/callers.go$'
          '^api/integration-api/api/unified_provider.go$'
          '^pkg/clients/integration/integration_client.go$'
          '^api/integration-api/internal/caller/'"$provider_re"'/'
          '^ui/src/providers/provider\.(development|production)\.json$'
          '^ui/src/providers/config-loader.ts$'
          '^ui/src/providers/'"$provider_re"'/text-models.json$'
          '^ui/src/providers/'"$provider_re"'/models.json$'
          '^ui/src/providers/'"$provider_re"'/text-embedding-models.json$'
          '^ui/src/app/components/providers/text/'
        )
        ;;
      telemetry-integration)
        allowed_patterns+=(
          '^api/integration-api/internal/caller/metrics/metrics_builder.go$'
          '^api/integration-api/internal/caller/metrics/metrics_builder_test.go$'
          '^api/integration-api/internal/entity/external_audit.go$'
          '^api/integration-api/internal/caller/'"$provider_re"'/'
          '^api/assistant-api/internal/transformer/'"$provider_re"'/'
          '^api/assistant-api/internal/type/packet.go$'
          '^api/assistant-api/internal/adapters/internal/dispatch.go$'
        )
        ;;
      system-understanding)
        allowed_patterns+=(
          '^CLAUDE.md$'
          '^\.claude/skills/README.md$'
          '^\.claude/skills/ENTERPRISE_POLICY.md$'
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

echo "Skill validation passed: $skill_dir"
