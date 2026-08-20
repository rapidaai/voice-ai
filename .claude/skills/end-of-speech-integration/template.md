# End Of Speech Integration Template

## Request classification

- Change type (new provider / modify existing provider / tuning only):
- Target provider:
- EOS signal mode (transcript / audio-model / history-aware):
- Latency vs accuracy priority:

## Inputs and defaults

- Explicit user constraints:
- Assumptions used (if missing info):
- Why this baseline provider was selected:

## Planned edit scope (strict)

- Provider implementation folder:
- Factory file edits:
- Contract files (if any):
- UI config/component files:
- Explicitly out of scope (must not edit):

## Packet contract mapping

- Consumed packets:
- Interim output behavior:
- Finalization trigger behavior:
- Duplicate-finalization guard strategy:

## EOS options mapping

- `microphone.eos.provider`:
- `microphone.eos.timeout`:
- `microphone.eos.threshold`:
- `microphone.eos.quick_timeout`:
- `microphone.eos.silence_timeout`:

## Test plan and evidence

- EOS unit tests:
- Adapter/dispatcher integration tests:
- UI config-loader tests:
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
