---
name: telephony-integration
description: Add or modify telephony providers in assistant-api with transport-aware streamers/webhooks, factory wiring, and codec/resampling compatibility.
---

# Telephony Integration Skill

## Mission

Integrate telephony providers safely across webhook signaling, media streaming, context routing, and disconnect handling.

## Inputs expected from user

1. Transport: WebSocket media stream, SIP RTP/session, AudioSocket TCP, or mixed.
2. Inbound and outbound call requirements.
3. Codec/sample-rate constraints from provider.

If user does not answer:
- Use nearest baseline provider (`twilio`, `exotel`, `vonage`, `asterisk`, `sip`).
- Preserve internal LINEAR16 16kHz core format and use base resampler paths.

## Hard boundaries

In scope:
- `api/assistant-api/internal/channel/telephony/internal/<provider>/...`
- `api/assistant-api/internal/channel/telephony/telephony.go`
- only if needed: `inbound.go`, `outbound.go`, `api/assistant-api/internal/type/telephony.go`, `streamer.go`
- UI telephony provider registration and component files

Out of scope:
- STT/TTS provider internals
- EOS/VAD internals
- integration-api LLM callers

## Runtime flow checklist

1. `ReceiveCall` parses provider webhook payload into `CallInfo`.
2. `InboundCall` returns provider-specific connect response.
3. `NewStreamer` path handles media transport and codec conversion.
4. `StatusCallback` and catch-all callback map provider events.
5. Interruption and end-conversation directives close sessions cleanly.

## Transport mapping baselines

- WebSocket media: `twilio`, `vonage`, `exotel`, `asterisk/websocket`
- SIP native session: `sip`
- TCP AudioSocket: `asterisk/audiosocket`

## Governed lifecycle

- Non-trivial work follows `understand -> plan -> challenge -> approve -> implement -> verify -> independent review -> ship` from `DEVELOPMENT_PROCESS.md`.
- This skill operates only in its assigned phase and path ownership; it may not approve its own plan or code review.
- Implementation starts only from a coordinator-attested approved plan with explicit allowed paths, owners, tests, commands, and rollback.
- Return changed-file and verification evidence to the coordinator, then route the complete verified diff to the read-only `code-reviewer`.

## Validation commands

- `go test ./api/assistant-api/internal/channel/telephony/...`
- `go test ./api/assistant-api/internal/adapters/internal/...`
- `cd ui && yarn test providers`
- `./.claude/skills/telephony-integration/scripts/validate.sh --check-diff --provider <provider>`
