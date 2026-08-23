# RFC 0003: Native SIP Party Identity Resolution

- Status: Accepted
- Date: 2026-08-22
- Owners: Assistant API Native SIP, Assistant Authentication
- Reviewers: SIP, Telephony, UI, Security, and SRE owners

## Abstract

This RFC defines how native SIP resolves and propagates the two parties of a call.

SIP provides three different addressing concepts that MUST remain separate:

- Request-URI identifies where the request is routed.
- The `From` header identifies the logical initiator.
- The `To` header identifies the logical recipient.

Rapida MUST capture the `From` and `To` header addresses from the dialog-forming SIP request
and preserve them without phone-number validation, normalization, or DID inference.

For inbound calls:

- `client.phone` is the `From` header address.
- `client.assistant_phone` is the `To` header address.

For outbound calls:

- `client.phone` is the resolved remote destination.
- `client.assistant_phone` is the resolved local originating identity.

Request-URI is used only for routing. It MUST NOT populate either party identity.

This contract applies equally when the dialog-forming request represents a direct call, a new
call created after `REFER`, or a replacement/transfer INVITE. Each new SIP dialog resolves its
own parties from its own dialog-forming request.

This RFC changes only native SIP identity handling and assistant-authentication UI keys.
Other telephony providers, public APIs, protobuf contracts, and database schemas remain
compatible. Existing exported Go symbols receive deprecated adapters for at least one release.

## Status of This Memo

This document is a design proposal. Implementation MUST NOT begin while its status is
`Draft`.

Before implementation, the final RFC bytes MUST pass independent challenge and exact-digest
confirmation according to `DEVELOPMENT_PROCESS.md`. Any byte change after confirmation
requires another challenge and confirmation.

## Conformance Language

The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHOULD**, **SHOULD NOT**, and **MAY** in
this document describe normative requirements.

## Problem Statement

The native SIP implementation currently carries values named `FromURI` and `ToURI`, then
attempts to extract a DID or phone number from those values. A defensive reconstruction path
also derives the assistant-side value from `SessionInfo.LocalURI`.

That behavior conflates two independent concerns:

1. SIP request routing.
2. SIP party identity.

A SIP identity can be a telephone number, extension, assistant address, route-specific name,
or another valid SIP user. Direct calls, transferred calls, and referred calls do not
guarantee numeric identities.

The system therefore MUST NOT decide whether a SIP party identity looks like a phone number.
It must capture the identities supplied by SIP, map them according to call direction, and
preserve them for the lifetime of the call.

## Protocol Model

### Request-URI

The Request-URI identifies the user or service to which the current request is routed.

Rapida MAY use Request-URI to select an assistant, deployment, or route. Request-URI MUST NOT
populate `client.phone`, `client.assistant_phone`, `CallContext.CallerNumber`, or
`CallContext.FromNumber`.

### From Header

The `From` header identifies the logical initiator of the dialog-forming request.

Rapida MUST capture the parsed header address:

```go
request.From().Address.String()
```

The display name and header parameters, including the dialog tag, are not part of the party
identity stored by this RFC.

### To Header

The `To` header identifies the logical recipient of the dialog-forming request.

Rapida MUST capture the parsed header address:

```go
request.To().Address.String()
```

The display name and header parameters, including the dialog tag, are not part of the party
identity stored by this RFC.

### Identity Representation

The captured `From` and `To` addresses are opaque SIP party identities.

The implementation MUST NOT:

- require digits;
- require or remove a leading `+`;
- apply E.164 validation;
- strip `agent-`, `did-`, or any other prefix;
- remove a host, port, URI parameter, or URI header;
- convert a SIP URI to only its user component;
- infer a country code;
- reject an identity because it is non-numeric; or
- substitute a deployment phone number.

The implementation MUST use the parsed address object's `String()` representation and then
preserve that result unchanged through downstream call state.

Examples of valid opaque identities include:

```text
sip:+15551234567@carrier.example
sip:15551234567@carrier.example;user=phone
sip:agent-42@sip.rapida.ai
sip:alice@example.com
tel:+442079460123
```

This RFC does not assert that every value stored under a legacy `phone`-named field is a PSTN
number. For native SIP, those fields carry SIP party identities.

## Scope

### In Scope

- Capturing `From` and `To` header addresses from native SIP dialog-forming requests.
- Carrying Request-URI separately for routing.
- Direction-aware mapping of the captured party identities into call context.
- Consistent behavior for direct, referred, transferred, and replacement calls.
- Removal of native SIP phone-number validation and DID extraction.
- Removal of native SIP identity reconstruction from local or remote session URIs.
- Preservation of existing outbound request behavior.
- Canonical `client.assistant_phone` authentication UI configuration.
- Backend and UI tests directly covering these changes.

### Out of Scope

- Twilio, Vonage, Vobiz, Exotel, Telnyx, generic SIP webhook, and Asterisk adapters.
- Changes to provider webhook parsing or payloads.
- Non-native-SIP call behavior.
- Changes to SIP route formats or route selection results.
- Changes to media, codecs, registration, or transport handling.
- Number validation or normalization.
- Inspection of `P-Asserted-Identity`, `P-Called-Party-ID`, `History-Info`, `Diversion`, or
  provider-specific headers.
- Database or protobuf migrations.
- Public REST or SDK contract changes.
- Compatibility support for `client.assistantPhone`.

## Terminology

### Dialog-Forming Request

The initial SIP request that creates a new dialog and native SIP call context. This is normally
an `INVITE`, including an INVITE generated as the result of referral or transfer processing.

### From Identity

The parsed address from the dialog-forming request's `From` header.

### To Identity

The parsed address from the dialog-forming request's `To` header.

### Route Identity

The Request-URI used to route the request. Route identity is not a party identity.

### Existing Dialog Request

A request, including a re-INVITE, that belongs to an already established dialog. It does not
create a new call identity snapshot.

## Protocol Invariants

1. Request-URI MUST be used only for routing.
2. Every native SIP call context MUST be based on one dialog-forming request.
3. The call's From identity MUST come from that request's parsed `From` header address.
4. The call's To identity MUST come from that request's parsed `To` header address.
5. Party identities MUST be treated as opaque values.
6. Party identities MUST NOT be validated as phone numbers.
7. Party identities MUST NOT be derived from Request-URI.
8. Party identities MUST NOT be reconstructed from `SessionInfo.LocalURI` or
   `SessionInfo.RemoteURI`.
9. Party identities MUST remain immutable after the call context is created.
10. Requests within an existing dialog MUST NOT replace the stored party identities.
11. A new dialog created by referral, transfer, or replacement MUST resolve identities from
    its own dialog-forming request.
12. Metadata writers MUST use call context only and MUST NOT inspect SIP state.

## Canonical Internal Contract

### SIP Request Identity

The native SIP inbound identity MUST contain three distinct fields:

```go
type inboundInviteIdentity struct {
    callID       string
    fromTag      string
    requestURI  string
    fromIdentity string
    toIdentity   string
}
```

The semantic source of each field is fixed:

| Field | Source |
| --- | --- |
| `requestURI` | `request.Recipient.String()` |
| `fromIdentity` | `request.From().Address.String()` |
| `toIdentity` | `request.To().Address.String()` |

Canonical exported boundary fields MUST use these names:

```go
RequestURI   string
FromIdentity string
ToIdentity   string
```

New internal code MUST use the canonical fields. Existing exported `FromURI` and `ToURI`
fields MUST remain as deprecated compatibility aliases for at least one released version.
Boundary adapters MUST populate them from `FromIdentity` and `ToIdentity`; routing and metadata
code MUST ignore the deprecated aliases. Their later removal requires a separately approved
breaking-change RFC.

The exported `ExtractDIDFromURI` function MUST remain as a deprecated compatibility wrapper for
downstream Go consumers. Native SIP implementation code MUST NOT call it.

Existing exported callback signatures MUST remain available. New identity-aware callbacks MUST
accept a value object rather than adjacent positional strings:

```go
type SIPRequestIdentity struct {
    RequestURI   string
    FromIdentity string
    ToIdentity   string
}
```

The existing callbacks receive the canonical From and To values through their historical two
string parameters. Native SIP application wiring MUST use the identity-aware callbacks so it
also receives RequestURI. Removing the deprecated callbacks requires a separately approved
breaking-change RFC.

### Direction-Aware Call Context Mapping

The mapping is:

| Direction | `CallContext.CallerNumber` | `CallContext.FromNumber` |
| --- | --- | --- |
| Inbound | `FromIdentity` | `ToIdentity` |
| Outbound | resolved remote destination | resolved local `fromNumber` |

For inbound native SIP:

```go
callContext.CallerNumber = stage.FromIdentity
callContext.FromNumber = stage.ToIdentity
```

No extraction helper or fallback is permitted.

For outbound native SIP, the existing channel pipeline remains authoritative for the remote
destination and local `fromNumber`. SIP response headers and session URIs MUST NOT overwrite
the outbound call context.

### Metadata Mapping

Metadata writers MUST apply this mapping:

| Call context field | Metadata key | Presence rule |
| --- | --- | --- |
| `CallerNumber` | `client.phone` | Emit when non-empty |
| `FromNumber` | `client.assistant_phone` | Emit when non-empty |

The existing metadata names are retained for compatibility even though native SIP values may
be non-numeric SIP identities.

### Authentication Key

The only supported assistant authentication source is:

```text
client.assistant_phone
```

The authentication UI MUST save canonical snake_case client variable names, including
`assistant_phone` and `provider_call_id`.

The runtime MUST NOT add `client.assistantPhone` or `client.providerCallId` metadata aliases.
When the authentication UI loads or saves an existing configuration, it MUST normalize
`assistantPhone` to `assistant_phone` and `providerCallId` to `provider_call_id`. If both legacy
and canonical keys are present, the canonical key wins. The saved configuration MUST contain
only canonical keys.

## Inbound Call Protocol

### Initial INVITE Capture

When native SIP accepts an initial INVITE, it MUST:

1. Preserve existing mandatory SIP request validation.
2. Capture Request-URI as `requestURI` for routing.
3. Capture `request.From().Address.String()` as `fromIdentity`.
4. Capture `request.To().Address.String()` as `toIdentity`.
5. Pass all three values to inbound middleware.

The captured identity strings MUST NOT be transformed after capture.

### Routing

Routing middleware MUST use `RequestURI` as its primary route input.

Routing middleware MUST NOT use `FromIdentity` as a route fallback.

If legacy routing still requires `ToIdentity`, that behavior MUST be documented and tested as
a routing-only compatibility path. It MUST NOT change either captured party identity.

Route parsing and assistant lookup MUST NOT rewrite `FromIdentity` or `ToIdentity`.

### Context Construction

After routing and authentication succeed, the session-established pipeline MUST carry:

```go
RequestURI   string
FromIdentity string
ToIdentity   string
```

Inbound call context construction MUST set:

```go
CallerNumber = FromIdentity
FromNumber   = ToIdentity
```

The same mapping MUST be used by normal construction and any defensive reconstruction path.

### Existing Dialog Requests

ACK, BYE, CANCEL, UPDATE, INFO, and re-INVITE handling MUST NOT change the party identities
stored for the call.

The original dialog-forming request remains authoritative for that call context.

## Direct Call Protocol

A direct call to a Rapida SIP address is handled like every other inbound initial INVITE.

Example:

```text
INVITE sip:agent-42@sip.rapida.ai SIP/2.0
From: <sip:alice@example.com>;tag=caller-tag
To: <sip:agent-42@sip.rapida.ai>
```

Result:

```text
RequestURI                   = sip:agent-42@sip.rapida.ai
FromIdentity                 = sip:alice@example.com
ToIdentity                   = sip:agent-42@sip.rapida.ai
CallContext.CallerNumber     = sip:alice@example.com
CallContext.FromNumber       = sip:agent-42@sip.rapida.ai
client.phone                 = sip:alice@example.com
client.assistant_phone       = sip:agent-42@sip.rapida.ai
```

The non-numeric assistant identity is valid and MUST NOT be omitted or rewritten.

## Referral and Transfer Protocol

### REFER Request

A `REFER` request instructs a recipient to contact the resource identified by `Refer-To`.

Receiving or sending REFER MUST NOT immediately mutate the current call context's
`CallerNumber` or `FromNumber`. The existing dialog retains its original party identities
until it terminates.

`Refer-To` is a transfer target, not the From or To identity of the existing call. It MUST NOT
be written into the existing call context.

### New INVITE Caused by REFER

If REFER processing creates a new outbound or inbound INVITE, that INVITE creates a distinct
call leg and identity snapshot.

The new call leg MUST resolve its identities according to its own direction:

- new inbound leg: `FromIdentity` to `CallerNumber`, `ToIdentity` to `FromNumber`;
- new outbound leg: existing outbound destination and `fromNumber` resolution.

The new call leg MUST NOT inherit the old leg's `FromIdentity` or `ToIdentity` unless those
same values are explicitly present in the new dialog-forming request or outbound call request.

### INVITE with Replaces

An INVITE containing `Replaces` creates a new dialog intended to replace an existing dialog.

The replacement INVITE MUST create its own identity snapshot from its own `From` and `To`
headers. The referenced dialog's identities MUST NOT overwrite the replacement dialog's
identities.

### Blind and Attended Transfer

Blind and attended transfers MUST follow the same call-leg rule:

- each existing dialog keeps its original immutable party identities;
- each new dialog resolves its own identities once;
- correlation between call legs does not imply identity inheritance; and
- transfer targets remain routing inputs until they become part of a new dialog-forming
  request.

## Outbound Call Protocol

Outbound behavior remains unchanged because Rapida initiates the call and already owns the
two application-level identities.

The outbound pipeline MUST continue to:

1. Use the resolved destination as `CallContext.CallerNumber`.
2. Use the explicit `fromNumber`, or the existing outbound deployment fallback, as
   `CallContext.FromNumber`.
3. Use those values to construct outbound SIP signaling.
4. Preserve the call context values for authentication and metadata.

SIP responses, Contact values, dialog remote targets, and later in-dialog requests MUST NOT
overwrite the outbound call context identities.

## Failure Semantics

| Condition | Required behavior |
| --- | --- |
| Initial request lacks a required `From` or `To` header | Preserve existing malformed-request rejection |
| Parsed `From` address is empty | Preserve existing malformed-request behavior |
| Parsed `To` address is empty | Preserve existing malformed-request behavior |
| Identity is non-numeric | Accept and preserve it |
| Identity uses `sip`, `sips`, or `tel` | Accept the parsed address representation |
| Request-URI differs from `ToIdentity` | Route with Request-URI and preserve `ToIdentity` |
| Existing-dialog request has different displayed headers | Do not mutate stored identities |
| New transfer leg is created | Resolve a new identity snapshot for the new leg |
| Call context reconstruction is required | Use propagated From/To identities, never session URIs |

No identity format may cause a panic or an otherwise valid call failure.

## Message Flows

### Inbound Direct or PSTN-Originated SIP Call

```text
Initial INVITE
    -> capture RequestURI
    -> capture FromIdentity from From.Address.String()
    -> capture ToIdentity from To.Address.String()
    -> route using RequestURI
    -> preserve identities through middleware
    -> propagate identities through session establishment
    -> CallerNumber = FromIdentity
    -> FromNumber = ToIdentity
    -> emit client.phone and client.assistant_phone
```

### REFER-Based Transfer

```text
Existing dialog receives REFER
    -> keep existing call-context identities unchanged
    -> treat Refer-To as transfer routing target
    -> create a new dialog-forming INVITE when transfer proceeds
    -> resolve the new call leg from its own From/To headers or outbound inputs
```

### INVITE with Replaces

```text
Replacement INVITE arrives
    -> capture its RequestURI, FromIdentity, and ToIdentity
    -> create a new call-leg identity snapshot
    -> correlate with the replaced dialog
    -> do not copy party identities from the replaced dialog
```

## Service Impact and Required Updates

### Impact Matrix

| Service or component | Impact | Required update |
| --- | --- | --- |
| `assistant-api`: SIP core | Required | Capture distinct RequestURI, FromIdentity, and ToIdentity values from the dialog-forming request. |
| `assistant-api`: SIP infra | Required | Preserve all three values across core/infra conversions and retain deprecated exported adapters for one release. |
| `assistant-api`: SIP middleware | Required | Route using RequestURI and preserve FromIdentity and ToIdentity unchanged. |
| `assistant-api`: SIP pipeline | Required | Map identities by direction and remove DID extraction or session-URI fallback. |
| `assistant-api`: SIP transfer | Verification required | Keep existing-leg identities immutable and resolve every new leg independently. |
| `assistant-api`: telephony metadata | Verification required | Continue emitting values only from call context. |
| `assistant-api`: authentication | Verification required | Read only canonical snake_case metadata keys; add no runtime aliases. |
| `ui`: authentication configuration | Required | Save canonical `assistant_phone` and `provider_call_id`; normalize legacy camelCase keys when editing existing configurations. |
| Non-SIP telephony providers | No implementation change | Preserve current behavior; do not modify provider adapters. |
| `web-api` | No impact | No route, schema, or authentication contract change. |
| `integration-api` | No impact | No caller or model-provider change. |
| `endpoint-api` | No impact | No endpoint runtime contract change. |
| `document-api` | No impact | No document contract change. |
| Database and migrations | No impact | Continue using existing conversation metadata storage. |
| Public Go, protobuf, REST, and SDK contracts | Compatibility update | Preserve existing exported Go fields, functions, and callbacks through deprecated adapters; preserve all wire fields and names. |

### Required File Boundaries

| Area | Allowed paths | Responsibility |
| --- | --- | --- |
| SIP core | `api/assistant-api/sip/internal/core/` | Capture and own the immutable identity snapshot. |
| SIP infra | `api/assistant-api/sip/infra/` | Preserve identity fields across package boundaries. |
| SIP middleware | `api/assistant-api/sip/middleware/` | Route independently from party identity. |
| SIP pipeline | `api/assistant-api/sip/pipeline/` | Apply direction-aware call-context mapping. |
| SIP transfer | Existing SIP transfer files and tests | Verify call-leg identity isolation. |
| Telephony metadata | Existing base/observability files only if tests expose a mismatch | Preserve call-context-only metadata emission. |
| Authentication UI | `ui/src/app/pages/assistant/actions/configure-assistant-authentication/` | Save the canonical key. |

The following are out of scope:

- non-SIP provider adapter directories;
- STT, TTS, VAD, end-of-speech, and noise-reduction internals;
- integration-api LLM caller code;
- database migrations;
- public protobuf definitions;
- global variable aliasing; and
- unrelated SIP media, codec, or registration changes.

If implementation requires another path, work MUST stop until the plan and RFC are revised and
reconfirmed.

## External and Wire Compatibility

This RFC introduces no SIP wire-format change and no immediate public API removal.

- Incoming SIP headers are consumed as already received.
- Outgoing SIP construction remains unchanged.
- The outbound public request field remains `fromNumber`.
- Protobuf field names and numbers remain unchanged.
- REST routes and JSON response shapes remain unchanged.
- No SDK regeneration is required.
- No non-SIP provider payload changes are required.
- Existing `SIPRequestContext.FromURI`, `SIPRequestContext.ToURI`, `ExtractDIDFromURI`, and
  two-string server callbacks remain callable but are deprecated.
- Identity-aware server callbacks use `SIPRequestIdentity` so RequestURI, FromIdentity, and
  ToIdentity cannot be swapped accidentally.

The intentional behavior change is internal representation:

- native SIP party identities are no longer reduced to numeric DID values;
- full parsed `From` and `To` addresses are preserved;
- Request-URI no longer supplies call metadata; and
- transfer-created call legs resolve independently.

## Persistence and Migration

No database migration is required.

Conversation metadata continues to use the existing `client.phone` and
`client.assistant_phone` keys. New native SIP calls may store full SIP or TEL identities where
older behavior stored a numeric user or another inferred value.

Historical metadata is not rewritten.

There is no compatibility alias or dual-write period for `client.assistantPhone`.

## Security Considerations

- Request-URI cannot directly set authentication identity metadata.
- Header identities are captured only after existing SIP parsing and request acceptance.
- Identity strings remain data and MUST NOT be executed, dialed, or used as URLs by
  authentication code.
- Authentication cannot access arbitrary SIP headers through this change.
- Existing tenant-scoped routing and authorization remain authoritative.
- Logs and metrics MUST NOT contain full party identities.

This RFC records SIP-provided logical identities. It does not claim that the headers
cryptographically verify a PSTN phone-number owner.

## Privacy Considerations

SIP party identities may contain phone numbers, usernames, hosts, or routing information.

- New logs and metrics MUST record only bounded outcome fields.
- Raw From, To, and Request-URI values MUST NOT be added to logs or metric labels.
- Existing metadata authorization remains unchanged.
- No new cross-service propagation is introduced.

## Observability

The capture boundary SHOULD record bounded state without identity values.

Recommended fields:

- `direction=inbound|outbound`;
- `provider=sip`;
- `identity_source=sip_headers|outbound_request`;
- `from_identity_present=true|false`;
- `to_identity_present=true|false`;
- `route_identity_present=true|false`; and
- `dialog_origin=direct|refer|replace|unknown` when already known by the call flow.

No label or attribute introduced by this RFC may contain an unbounded identity string.

## Implementation Plan

### Phase 1: Separate Identity Types

1. Replace ambiguous inbound `fromURI` and `toURI` fields with `requestURI`, `fromIdentity`,
   and `toIdentity`.
2. Populate them from `request.Recipient`, `request.From().Address`, and
   `request.To().Address` respectively.
3. Add equivalent exported fields to core/infra request context types.
4. Populate deprecated exported `FromURI` and `ToURI` aliases at compatibility boundaries only.
5. Retain `ExtractDIDFromURI` as a deprecated exported wrapper with no native SIP callers.
6. Preserve existing server callback signatures and add identity-aware callback variants using
   `SIPRequestIdentity`.

### Phase 2: Separate Routing

1. Route using `RequestURI`.
2. Remove `FromIdentity` from route fallback behavior.
3. Preserve any required legacy `ToIdentity` routing fallback as routing-only behavior.
4. Prove through tests that routing does not mutate party identities.

### Phase 3: Propagate Identity Snapshot

1. Add `RequestURI`, `FromIdentity`, and `ToIdentity` to core and infra
   `SessionEstablishedPipeline`.
2. Copy all fields through every conversion and construction site.
3. Keep the fields immutable after initial call construction.

### Phase 4: Map Call Context

1. Set inbound caller identity from `stage.FromIdentity`.
2. Set inbound assistant identity from `stage.ToIdentity`.
3. Update defensive reconstruction to use the same fields.
4. Remove inbound use of `ExtractDIDFromURI` and session URI fallbacks.
5. Preserve outbound behavior.

### Phase 5: Transfer Verification

1. Verify REFER does not mutate existing call context.
2. Verify a new REFER-created call leg resolves its own identity snapshot.
3. Verify an INVITE with Replaces resolves its own identity snapshot.
4. Verify in-dialog requests do not modify stored identities.

### Phase 6: Authentication UI

1. Save `assistant_phone` instead of `assistantPhone`.
2. Save `provider_call_id` instead of `providerCallId`.
3. Normalize both legacy camelCase keys while loading and before saving existing configurations.
4. Prefer canonical values and remove legacy keys when both forms exist.
5. Add UI form-submission and update-path regression tests.
6. Add no runtime compatibility aliases.

## Testing and Verification

### SIP Core Tests

Required cases:

- captures full parsed From address;
- captures full parsed To address;
- captures Request-URI separately;
- preserves numeric, non-numeric, SIP, SIPS, and TEL identities;
- preserves URI parameters and URI headers produced by the parser;
- excludes display names and From/To header parameters;
- rejects requests missing mandatory From or To headers using existing behavior; and
- performs no phone-number validation.

### SIP Middleware Tests

Required cases:

- routes from RequestURI;
- does not route from FromIdentity;
- preserves FromIdentity and ToIdentity;
- keeps any approved legacy To-based route fallback isolated from metadata; and
- rejects unknown routes using existing behavior.

### SIP Pipeline Tests

Required cases:

- inbound maps FromIdentity to `CallerNumber`;
- inbound maps ToIdentity to `FromNumber`;
- direct Rapida SIP identities are preserved;
- normal and reconstruction paths produce identical mappings;
- LocalURI and RemoteURI do not populate reconstructed call identities;
- outbound behavior remains unchanged; and
- initialization and conversation metadata agree.

### Transfer Tests

Required cases:

- REFER does not mutate existing call context;
- a new call leg created after REFER uses its own identity snapshot;
- INVITE with Replaces uses its own identity snapshot;
- the replaced call retains its identities until termination; and
- re-INVITE does not modify identities.

### UI Tests

Required cases:

- all client option values match canonical runtime metadata keys;
- Assistant Phone saves `assistant_phone` through the form submission path;
- Provider Call ID saves `provider_call_id` through the form submission path;
- existing `assistantPhone` and `providerCallId` values normalize during the update path;
- canonical values win if both legacy and canonical forms exist;
- labels remain unchanged; and
- no saved configuration contains `assistantPhone` or `providerCallId`.

### Required Commands

```bash
go test ./api/assistant-api/sip/internal/core/...
go test ./api/assistant-api/sip/infra/...
go test ./api/assistant-api/sip/middleware/...
go test ./api/assistant-api/sip/pipeline/...
go test ./api/assistant-api/internal/channel/telephony/internal/base/...
yarn --cwd ui test --watchAll=false --runTestsByPath \
  src/app/pages/assistant/actions/configure-assistant-authentication/__tests__/index.test.tsx
yarn --cwd ui checkTs
yarn --cwd ui eslint \
  src/app/pages/assistant/actions/configure-assistant-authentication/shared.ts \
  src/app/pages/assistant/actions/configure-assistant-authentication/__tests__/index.test.tsx
yarn --cwd ui prettier --check \
  src/app/pages/assistant/actions/configure-assistant-authentication/shared.ts \
  src/app/pages/assistant/actions/configure-assistant-authentication/__tests__/index.test.tsx
git diff --check
```

## Rollout

The implementation is code-only and requires no cross-service migration ordering. Existing
authentication configurations are migrated opportunistically when loaded and saved through the
UI; runtime metadata remains canonical-only.

Rollout checks:

1. Direct inbound SIP calls preserve full From and To identities.
2. PSTN-originated SIP calls preserve the addresses supplied in From and To.
3. Request-URI changes affect routing only.
4. Existing-dialog requests do not mutate call identity.
5. Transfer-created dialogs resolve independent identities.
6. Outbound calls preserve existing application-selected values.
7. The authentication UI saves only the canonical key.
8. Non-SIP providers remain unchanged.

## Rollback

No data rollback is required.

Rollback consists of reverting the native SIP and UI implementation. Public APIs, SIP wire
messages, and database schemas remain compatible.

Restoring DID extraction or session-URI fallback is not an approved long-term behavior.

## Alternatives Considered

### Validate Numeric DID Values

Rejected. SIP party identities can legitimately be non-numeric, especially for direct calls
and transfer-created dialogs.

### Use Only the URI User Component

Rejected. It discards the scheme, host, and URI parameters that distinguish SIP identities.

### Use Request-URI as the Assistant Identity

Rejected. Request-URI is a routing target and may change independently of the logical To
identity.

### Recompute Identity from Session Local and Remote URIs

Rejected. Session endpoints and routing state are not replacements for the dialog-forming
From and To identities.

### Mutate Identity During REFER or Re-INVITE

Rejected. Existing dialogs retain their original identity snapshot. New dialogs resolve new
snapshots.

### Support Runtime CamelCase Metadata Aliases

Rejected. Only the canonical snake_case runtime metadata contract is supported. Compatibility
is provided by normalizing legacy UI configuration keys during load/save, not by emitting or
reading duplicate runtime metadata.

### Update Non-SIP Providers

Rejected for this RFC. Their existing resolution contracts remain unchanged.

## Risks and Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Consumers assume metadata always contains digits | Authentication request may receive a SIP identity | Document opaque semantics and avoid validation in transport code. |
| Renaming fields misses a conversion boundary | Identity is lost | Add core/infra parity tests and compile-time updates at every construction site. |
| Routing fallback mutates To identity | Metadata no longer matches the request | Keep routing result separate and test identity immutability. |
| Transfer code reuses an old call context | New leg receives stale identities | Require a new snapshot for each dialog-forming request and add transfer tests. |
| Raw identities leak through observability | Sensitive SIP data exposure | Emit presence and origin enums only. |
| Existing camelCase UI mappings remain invalid | Existing authentication configuration may not resolve | Normalize legacy keys on UI load/save, prefer canonical values, and add no runtime alias. |
| Deprecated exported SIP symbols remain in use | Removal could later break downstream Go consumers | Mark adapters deprecated, keep them for at least one release, and require a separate breaking-change RFC before removal. |

## Acceptance Criteria

1. Native SIP captures RequestURI, FromIdentity, and ToIdentity as distinct values.
2. FromIdentity is exactly the parsed `From` header address string.
3. ToIdentity is exactly the parsed `To` header address string.
4. No phone-number validation, normalization, or DID extraction is performed.
5. RequestURI is used only for routing.
6. Inbound maps FromIdentity to `client.phone` through `CallerNumber`.
7. Inbound maps ToIdentity to `client.assistant_phone` through `FromNumber`.
8. Existing-dialog requests do not modify the identity snapshot.
9. New dialogs created by direct calls, REFER, transfer, or Replaces resolve independently.
10. Reconstruction paths never derive identities from LocalURI or RemoteURI.
11. Outbound behavior remains unchanged.
12. Non-SIP provider implementations remain unchanged.
13. The UI saves canonical `client.assistant_phone` and `client.provider_call_id` references.
14. Existing `assistantPhone` and `providerCallId` configuration keys normalize on UI load/save,
    with canonical keys taking precedence.
15. No camelCase runtime metadata alias is added.
16. Existing exported SIP fields, functions, and callbacks remain available through deprecated
    adapters for at least one release.
17. New identity-aware callbacks use `SIPRequestIdentity` rather than positional identity strings.
18. No database, protobuf, REST, SDK, provider webhook, or SIP wire-format migration is
    introduced.
19. Required backend and UI tests pass.
20. Independent review reports no unresolved critical or major findings.
21. Final RFC bytes receive exact-digest confirmation before implementation begins.

## References

- RFC 3261, *SIP: Session Initiation Protocol*.
- RFC 3515, *The Session Initiation Protocol Refer Method*.
- RFC 3891, *The Session Initiation Protocol Replaces Header*.
- RFC 5589, *Session Initiation Protocol Call Control - Transfer*.
- `api/assistant-api/sip/internal/core/inbound_call.go`
- `api/assistant-api/sip/internal/core/server.go`
- `api/assistant-api/sip/internal/core/pipeline.go`
- `api/assistant-api/sip/infra/server_type.go`
- `api/assistant-api/sip/infra/pipeline_type.go`
- `api/assistant-api/sip/pipeline/session.go`
- `api/assistant-api/sip/pipeline/runtime.go`
- `api/assistant-api/sip/pipeline/transfer.go`
- `api/assistant-api/internal/channel/telephony/internal/base/base.go`
- `ui/src/app/pages/assistant/actions/configure-assistant-authentication/shared.ts`

## Decision Log

| Date | Decision | Rationale |
| --- | --- | --- |
| 2026-08-22 | Preserve complete parsed SIP header addresses. | SIP identities are not limited to telephone numbers. |
| 2026-08-22 | Apply no number validation or normalization. | Direct, referred, and transferred calls may use arbitrary valid SIP identities. |
| 2026-08-22 | Keep Request-URI routing-only. | Request routing and logical party identity are independent. |
| 2026-08-22 | Snapshot From and To per dialog-forming request. | Existing dialogs remain stable while new transfer legs resolve independently. |
| 2026-08-22 | Keep `CallContext` and metadata field names. | Avoid public and persistence migrations while correcting native SIP semantics. |
| 2026-08-22 | Normalize legacy UI authentication keys without runtime aliases. | Existing configurations become valid when edited while runtime metadata remains singular. |
| 2026-08-22 | Retain deprecated exported SIP adapters for at least one release. | Preserve downstream Go compatibility while moving internal code to explicit identity contracts. |
| 2026-08-22 | Make no non-SIP provider changes. | This RFC addresses native SIP only. |
