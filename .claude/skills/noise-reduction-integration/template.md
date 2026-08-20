# Noise Reduction Integration Template

## Request classification

- Change type (new provider / modify existing):
- Target provider:
- Runtime type (native/sdk/http):
- Quality tradeoff target:

## Inputs and defaults

- Explicit user constraints:
- Assumptions used:
- Baseline provider selected:

## Planned edit scope (strict)

- Provider implementation folder:
- Factory file edits:
- Contract/packet files (if any):
- UI metadata files:
- Explicitly out of scope (must not edit):

## Denoiser packet contract

- Input handling (`DenoiseAudioPacket`):
- Output handling (`DenoisedAudioPacket`):
- Ordering/cadence guarantees:
- Error fallback behavior (`NoiseReduced=false`):
- Cleanup/lifecycle behavior:

## Option and metadata mapping

- Provider option key (`microphone.denoising.provider`):
- Provider JSON files updated:
- Default/fallback provider behavior:

## Test plan and evidence

- Denoiser unit tests:
- Factory/fallback tests:
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
