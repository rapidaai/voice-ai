---
name: telephony-integration
description: Add or modify telephony providers in assistant-api with transport-aware streamer/webhook implementations, factory wiring, and codec/resampling compatibility.
---

# Telephony Integration Skill

## Mission

Integrate telephony providers safely across webhook payloads, media streams, callbacks, and cleanup paths.

## Inputs expected from user

1. Transport type: WebSocket, SIP, AudioSocket, or mixed.
2. Inbound/outbound requirements.
3. Codec and sample-rate constraints.

If missing:
- Use nearest baseline (`twilio`, `exotel`, `vonage`, `asterisk`, `sip`).
- Preserve internal LINEAR16 16k and base resampler usage.

## Hard boundaries

In scope:
- `api/assistant-api/internal/channel/telephony/internal/<provider>/...`
- `api/assistant-api/internal/channel/telephony/telephony.go`
- optional: `inbound.go`, `outbound.go`, `type/telephony.go`, `type/streamer.go`
- UI telephony registry/component wiring

Out of scope:
- STT/TTS provider internals
- EOS/VAD internals
- integration-api LLM caller logic

## Runtime flow checklist

1. `ReceiveCall` maps webhook payload to `CallInfo`.
2. `InboundCall` returns provider connect handshake.
3. Streamer `Recv/Send` handles codec and buffer lifecycle.
4. `StatusCallback` maps provider events.
5. Disconnect and directive paths cleanly terminate sessions.

## Governed lifecycle

- Non-trivial work follows `understand -> plan -> challenge -> approve -> implement -> verify -> independent review -> ship` from `DEVELOPMENT_PROCESS.md`.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Implementation starts only from a coordinator-attested approved plan with explicit allowed paths, owners, tests, commands, and rollback.
- Return changed-file and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `go test ./api/assistant-api/internal/channel/telephony/...`
- `go test ./api/assistant-api/internal/adapters/internal/...`
- `cd ui && yarn test providers`
- `./skills/telephony-integration/scripts/validate.sh --check-diff --provider <provider>`

## References

- `references/checklist.md`
- `references/telephony-provider-checklist.md`
- `examples/sample.md`
