# VAD Integration Template

## Request classification

- Change type (new provider / modify existing / retune):
- Target provider:
- Detector stack (onnx/native/sdk):
- False-positive tolerance:

## Inputs and defaults

- Explicit user constraints:
- Assumptions used:
- Baseline provider selected:

## Planned edit scope (strict)

- Provider implementation folder:
- Factory file edits:
- Contract/packet files (if any):
- UI config files:
- Explicitly out of scope (must not edit):

## Detection contract mapping

- Input audio assumptions:
- Speech onset interruption logic:
- Speech heartbeat logic:
- Silence frame logic:
- Cleanup/lifecycle behavior:

## VAD options mapping

- `microphone.vad.provider`:
- `microphone.vad.threshold`:
- `microphone.vad.min_speech_frame`:
- `microphone.vad.min_silence_frame`:

## Test plan and evidence

- Unit tests:
- Benchmarks:
- UI provider tests:
- Validation script command:

## Result summary

## Governed lifecycle evidence

- Approved plan artifact and attestation:
- Assigned implementation owner:
- Exact commands and results:
- Verification report:
- Independent reviewer and decision:

- Final behavior change:
- Risk notes and rollback:
