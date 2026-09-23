#!/usr/bin/env bash
# Run live transformer suites only for the requested providers and speech direction.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."
INTEGRATION_PKG="./api/assistant-api/internal/transformer/tests/integration"
TIMEOUT="${INTEGRATION_TEST_TIMEOUT:-300s}"
PROVIDERS=()
MODE="(STT|TTS)"

for arg in "$@"; do
  case "$arg" in
    -v|--verbose) ;; # Go's verbose output preserves individual skipped-test results.
    --tts-only) MODE="TTS" ;;
    --stt-only) MODE="STT" ;;
    -h|--help)
      echo "Usage: $0 [-v] [--tts-only|--stt-only] [provider ...]"
      echo "Providers: deepgram google sarvam elevenlabs cartesia assemblyai azure rime"
      echo "           resemble neuphonic minimax nvidia groq speechmatics aws smallest revai custom-tts"
      echo "TRANSFORMER_TEST_CONFIG: YAML config (default: transformer/tests/testdata/integration_config.yaml)"
      echo "INTEGRATION_TEST_TIMEOUT: Go test timeout (default: 300s)"
      echo "Live tests require explicitly enabled providers and credentials. Skipped tests do not validate a provider."
      exit 0
      ;;
    -*) echo "Unknown option: $arg" >&2; exit 2 ;;
    *) PROVIDERS+=("$arg") ;;
  esac
done

if [ ${#PROVIDERS[@]} -eq 0 ]; then
  PROVIDERS=(deepgram google sarvam elevenlabs cartesia assemblyai azure rime
    resemble neuphonic minimax nvidia groq speechmatics aws smallest revai custom-tts)
fi

PACKAGES=("$INTEGRATION_PKG")
PROVIDER_PATTERN=""
TEST_PREFIX_PATTERN=""
for provider in "${PROVIDERS[@]}"; do
  provider_key="$provider"
  provider_directory="$provider"
  case "$provider" in
    deepgram) test_prefix="Deepgram" ;;
    google|google-speech-service)
      provider_key="google-speech-service"
      provider_directory="google"
      test_prefix="Google"
      ;;
    azure|azure-speech-service)
      provider_key="azure-speech-service"
      provider_directory="azure"
      test_prefix="Azure"
      ;;
    sarvam|sarvamai)
      provider_key="sarvamai"
      provider_directory="sarvam"
      test_prefix="Sarvam"
      ;;
    assemblyai|assembly-ai)
      provider_key="assemblyai"
      provider_directory="assembly-ai"
      test_prefix="Assemblyai"
      ;;
    resemble|resembleai)
      provider_key="resembleai"
      provider_directory="resembleai"
      test_prefix="ResembleAI"
      ;;
    elevenlabs) test_prefix="ElevenLabs" ;;
    cartesia) test_prefix="Cartesia" ;;
    rime) test_prefix="Rime" ;;
    neuphonic) test_prefix="Neuphonic" ;;
    minimax) test_prefix="Minimax" ;;
    nvidia) test_prefix="Nvidia" ;;
    groq) test_prefix="Groq" ;;
    speechmatics) test_prefix="Speechmatics" ;;
    aws) test_prefix="AWS" ;;
    smallest) test_prefix="Smallest" ;;
    revai) test_prefix="RevAI" ;;
    custom-tts) test_prefix="Custom" ;;
    *) echo "Unknown provider: $provider" >&2; exit 2 ;;
  esac

  case "$provider_key:$MODE" in
    assemblyai:TTS|revai:TTS|elevenlabs:STT|rime:STT|resembleai:STT|neuphonic:STT|minimax:STT|custom-tts:STT)
      echo "SKIP: $provider does not support $MODE"
      continue
      ;;
  esac

  if [ -d "$INTEGRATION_PKG/$provider_directory" ]; then
    PACKAGES+=("$INTEGRATION_PKG/$provider_directory")
  fi
  PROVIDER_PATTERN="${PROVIDER_PATTERN:+$PROVIDER_PATTERN|}$provider_key"
  TEST_PREFIX_PATTERN="${TEST_PREFIX_PATTERN:+$TEST_PREFIX_PATTERN|}$test_prefix"
done

if [ -z "$PROVIDER_PATTERN" ]; then
  exit 0
fi

case "$MODE" in
  TTS) SHARED_PATTERN="TestTTSIntegration" ;;
  STT) SHARED_PATTERN="TestSTT.*" ;;
  *) SHARED_PATTERN="Test(TTSIntegration|STT.*)" ;;
esac

# The second filter component restricts provider subtests in the shared suites.
exec go test -tags=integration -v -count=1 -timeout "$TIMEOUT" \
  -run "^(Test($TEST_PREFIX_PATTERN)$MODE.*|$SHARED_PATTERN)$/^($PROVIDER_PATTERN)$" \
  "${PACKAGES[@]}"
