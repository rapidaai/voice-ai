# RFC 0003: Native SIP URI and Phone Resolution

- Status: Accepted
- Date: 2026-08-22
- Updated: 2026-09-03
- Owners: Assistant API Native SIP, Assistant Authentication
- Reviewers: SIP, Telephony, UI, Security, and SRE owners

## Summary

Native SIP MUST preserve exact SIP party addresses separately from phone values.

The existing `CallAddress` fields keep one explicit meaning:

```go
type CallAddress struct {
    From    string
    To      string
    FromURI string
    ToURI   string
    Headers map[string]string
}
```

- `FromURI` and `ToURI` are exact parsed SIP address strings.
- `From` and `To` are phone values only.
- `From` and `To` MUST NOT contain SIP URIs, route aliases, hosts, ports, parameters, or
  headers.
- `From` and `To` MUST NOT be renamed, marked deprecated, or replaced with additional phone
  fields.

For inbound calls, the caller phone comes from the `From` address user when it matches the
phone grammar. The assistant phone comes only from tenant-scoped route and active SIP deployment
data. Caller-controlled `To` headers and custom headers MUST NOT supply the assistant phone.

For outbound calls, the local and remote phone fields are populated only when their authoritative
outbound inputs match the phone grammar. SIP aliases remain valid signaling destinations, but
their phone field is empty.

Existing call-context and authentication metadata keys remain unchanged:

- `client.phone` receives the direction-aware remote phone value.
- `client.assistant_phone` receives the direction-aware assistant phone value.

## Status of This Memo

Implementation MUST NOT begin until the exact RFC bytes pass independent challenge and
exact-digest confirmation according to `DEVELOPMENT_PROCESS.md`. Any later RFC byte change
requires another challenge and confirmation.

## Conformance Language

The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHOULD**, **SHOULD NOT**, and **MAY** in
this document describe normative requirements.

## Context

Native SIP carries four related but distinct values:

1. The exact `From` header address.
2. The exact `To` header address.
3. The caller phone value used by conversation metadata and authentication.
4. The assistant phone value used by conversation metadata and authentication.

The current implementation captures both URI strings and URI users, but an inbound agent route
can copy `agent-<assistant-id>` into `CallAddress.To`. The pipeline then maps that alias through
`CallContext.FromNumber` to `client.assistant_phone`.

The route middleware already owns the validated relationship between Request-URI, tenant,
assistant, active SIP phone deployment, and configured deployment phone. It is therefore the
only native SIP component allowed to resolve the inbound assistant phone.

## Goals

- Preserve exact SIP `From` and `To` addresses for each dialog-forming request.
- Keep `CallAddress.From` and `CallAddress.To` as phone fields.
- Prevent route aliases and untrusted SIP headers from entering authentication phone metadata.
- Resolve inbound assistant phone values from validated route and deployment data.
- Preserve the existing call-context, metadata, protobuf, REST, SDK, and database contracts.
- Define deterministic behavior for inbound, outbound, transfer, referral, and replacement
  dialogs.
- Provide bounded observability without logging raw SIP addresses or phone values.

## Non-Goals

- Adding `FromPhone` or `ToPhone` fields.
- Renaming or marking `CallAddress.From` or `CallAddress.To` deprecated.
- Converting phone values to E.164.
- Inferring country codes.
- Accepting arbitrary display names, extensions, punctuation, or alphanumeric aliases as phone
  values.
- Trusting `P-Called-Party-ID`, `Diversion`, `History-Info`, or custom called-number headers.
- Changing non-SIP telephony providers.
- Adding database columns, protobuf fields, REST fields, SDK fields, or UI fields.

## Scope and Ownership

### Allowed Paths

- `api/assistant-api/sip/internal/core/` owns URI capture, phone parsing, accepted inbound
  call-address state, and outbound call-address construction.
- `api/assistant-api/sip/infra/` owns propagation between public middleware context and core
  middleware context.
- `api/assistant-api/sip/middleware/` owns route validation and inbound assistant phone
  resolution.
- `api/assistant-api/sip/pipeline/` owns direction-aware call-context mapping and transfer
  propagation.
- `api/assistant-api/internal/channel/telephony/internal/base/` owns client metadata emission.
- `api/assistant-api/internal/adapters/internal/` owns authentication variable resolution tests.
- `rfcs/0003-native-sip-assistant-phone-resolution.md` and its `jsons/` directory own governance
  evidence.

### Out-of-Scope Paths

- `api/assistant-api/migrations/`
- `protos/`
- `openapi/`
- `sdks/`
- `ui/`
- non-SIP telephony provider packages

If implementation requires an out-of-scope path, work MUST stop and return to RFC review.

## Canonical Contract

### CallAddress

`CallAddress` remains the single source of truth for SIP party URIs and phone values:

| Field | Contract |
| --- | --- |
| `FromURI` | Exact `request.From().Address.String()` from the dialog-forming request. |
| `ToURI` | Exact `request.To().Address.String()` from the dialog-forming request. |
| `From` | Resolved initiator phone value, or empty when unavailable. |
| `To` | Resolved recipient phone value, or empty when unavailable. |
| `Headers` | Existing non-credential header snapshot. Headers do not resolve phone values. |

The URI fields MUST remain unchanged after capture. Middleware MAY enrich only `To` for an
inbound route. No middleware may change `FromURI`, `ToURI`, or `From`.

### Phone Grammar

After trimming surrounding ASCII whitespace, a phone value MUST match:

```text
^\+?[0-9]+$
```

The accepted value MUST be preserved exactly after trimming. This preserves national numbers
such as `07249994778`, international values such as `+447249994778`, and leading zeroes.

The parser MUST reject:

- an empty value;
- `+` without digits;
- multiple `+` characters;
- embedded whitespace;
- punctuation such as `-`, `(`, or `)`;
- URI parameters or headers;
- ASCII letters;
- Unicode digits; and
- route aliases such as `agent-42` or `did-+15551234567`.

Trusted route and deployment phone values are validated when configured and are trimmed before
entering `CallAddress.To`.

### Request-URI

Request-URI remains the route input. The route parser MAY remove the established `did-` prefix
to obtain a DID route value. The resulting value may populate `CallAddress.To` only after the
route lookup proves that it matches an active SIP phone deployment in the current tenant.

Request-URI MUST NOT replace `FromURI` or `ToURI`.

## Inbound Resolution

### Initial Capture

For each dialog-forming inbound INVITE, SIP core MUST:

1. Apply existing mandatory SIP request validation.
2. Set `FromURI` to `request.From().Address.String()`.
3. Set `ToURI` to `request.To().Address.String()`.
4. Parse `request.From().Address.User` with the phone grammar and set `From` to the accepted
   value or empty.
5. Initialize `To` as empty.
6. Pass Request-URI and the complete `CallAddress` to middleware.

The `To` header user MUST NOT populate inbound `CallAddress.To` because it is caller-controlled
and is not the validated route source.

### DID Route

For a DID route, middleware MUST:

1. Resolve the route value from Request-URI.
2. Query all active SIP phone deployment records whose `phone` option exactly matches that route
   value.
3. Reject zero matches using the existing route-not-found behavior.
4. Reject more than one match as ambiguous configuration before selecting an assistant, project,
   organization, or authentication context.
5. Resolve the assistant and tenant from that same unique deployment record.
6. Set `CallAddress.To` to the trimmed matched phone value.

No second deployment lookup or fallback is permitted after a DID match.

Duplicate DID matches MUST fail closed even when the matching records belong to the same
assistant, different assistants in one tenant, or assistants in different tenants. Database row
order MUST NOT influence route selection.

### Agent Route

For an `agent-<assistant-id>` route, middleware MUST resolve the assistant using the existing
assistant service with phone deployment injection, matching the dependency flow used by the
vault middleware.

The selection contract is:

- An injected `phone` option: trim the value and set `CallAddress.To`.
- No injected phone deployment or no phone option: accept the agent route and leave
  `CallAddress.To` empty.

Middleware MUST NOT use the `To` header user or a custom SIP header as a fallback.

### Middleware Propagation

Route enrichment MUST survive every boundary between middleware and session establishment.

1. Infra builds the middleware context with the captured `CallAddress`.
2. Middleware may set only `context.CallAddress.To` after successful route validation.
3. Infra copies only the middleware-owned `CallAddress.To` value back to the core middleware
   result. The captured `From`, `FromURI`, `ToURI`, and `Headers` remain owned by core.
4. Core stores the enriched `CallAddress` in the accepted inbound configuration and transfers it
   to the accepted inbound identity. Session state MUST NOT duplicate the call address.
5. Application-ready, invite, session-established, and pipeline callbacks receive the enriched
   `CallAddress` from the dialog identity.
6. Pipeline call-context construction reads the enriched `From` and `To` values.

Tests MUST prove both phone enrichment propagation and URI immutability across the infra and core
boundaries.

## Outbound Resolution

The outbound call pipeline remains authoritative for the local origin input and remote
destination input.

When SIP core constructs the dialog-forming INVITE:

- `FromURI` is the exact generated From address.
- `ToURI` is the exact generated destination address.
- `From` is the trimmed local origin input when it matches the phone grammar, otherwise empty.
- `To` is the trimmed remote destination input when it matches the phone grammar, otherwise
  empty.

A valid SIP alias may still be used for signaling. A non-phone alias MUST produce an empty phone
field rather than a signaling failure or fabricated number.

SIP response headers, session local or remote URIs, and in-dialog requests MUST NOT replace the
four values captured for the dialog.

## Direction-Aware Mapping

The call-context mapping is:

| Direction | `CallContext.CallerNumber` | `CallContext.FromNumber` |
| --- | --- | --- |
| Inbound | `CallAddress.From` | `CallAddress.To` |
| Outbound | `CallAddress.To` | `CallAddress.From` |

Empty phone values remain empty and are omitted by the existing metadata writer.

The metadata mapping remains:

| Metadata key | Source |
| --- | --- |
| `client.phone` | `CallContext.CallerNumber` when non-empty. |
| `client.assistant_phone` | `CallContext.FromNumber` when non-empty. |

URI values MUST NOT be added to authentication metadata by this RFC.

## Referral, Transfer, and Replacement

Each new SIP dialog MUST create a new four-field snapshot.

- An inbound dialog created after `REFER` follows the inbound capture, route resolution, and
  propagation rules.
- An outbound transfer leg follows the outbound rules.
- An INVITE with `Replaces` follows the rules for its new inbound dialog.
- Existing in-dialog requests MUST NOT mutate the stored snapshot.
- A new leg MUST NOT inherit `From`, `To`, `FromURI`, or `ToURI` from the previous leg.

## Authentication Configuration

The supported authentication variables remain:

```text
client.phone
client.assistant_phone
```

The UI and runtime continue using canonical snake_case keys. Runtime camelCase aliases MUST NOT
be added.

For the example that motivated this RFC, an inbound agent route produces:

```json
{
  "client.assistant_phone": "<active SIP deployment phone>",
  "client.phone": "07249994778"
}
```

The authentication payload MUST NOT receive `agent-<assistant-id>` through either phone
variable.

## Failure and Recovery

| Condition | Required behavior |
| --- | --- |
| Missing mandatory From or To header | Preserve existing SIP rejection behavior. |
| From user fails phone grammar | Preserve `FromURI`; leave `From` empty. |
| DID route is not found | Reject using existing route-not-found behavior. |
| DID route matches multiple active deployments | Reject as ambiguous configuration before selecting tenant or assistant. |
| Agent route has no active SIP phone deployment | Accept route; leave `To` empty. |
| Agent route has no phone option | Accept route; leave `To` empty. |
| Outbound local input fails grammar | Continue valid SIP signaling; leave `From` empty. |
| Outbound destination input fails grammar | Continue valid SIP signaling; leave `To` empty. |
| Propagation attempts to modify a URI field | Treat as an implementation defect and fail the affected test or invariant check. |

There are no retries for route or phone resolution. A new SIP request may be attempted after
configuration is corrected.

## Security and Privacy

- Only tenant-scoped validated route and deployment data may populate inbound
  `CallAddress.To`.
- Caller-controlled `To` headers and custom headers MUST NOT populate the assistant phone.
- Authorization, proxy authorization, and credentials MUST remain excluded from captured
  headers.
- Raw SIP URIs and phone values MUST NOT appear in metric labels or structured resolution logs.
- Phone resolution MUST occur after tenant and assistant scope is established.
- Ambiguous deployment state MUST fail closed rather than selecting an arbitrary record.

## Observability

Phone resolution diagnostics, when emitted, MUST use bounded attributes only.

Allowed `phone_source` values:

- `from_header_user`
- `did_route`
- `agent_deployment`
- `outbound_local_input`
- `outbound_remote_input`
- `unavailable`

Allowed `phone_result` values:

- `resolved`
- `missing`
- `invalid`
- `ambiguous`
- `not_phone`

Logs and metrics MAY include direction, route kind, provider, assistant identifier, and the
bounded values above. They MUST NOT include raw Request-URI, `FromURI`, `ToURI`, `From`, `To`,
or deployment phone values.

Tests MUST assert that resolution records use only the bounded values and omit raw identities and
phone values.

## Data and Compatibility

No persistent-data migration is required.

- `CallAddress` already contains all required fields.
- `CallContext.CallerNumber` and `CallContext.FromNumber` remain phone fields.
- Existing database column widths remain sufficient because URI values are not stored there.
- Protobuf, REST, SDK, UI, and non-SIP provider contracts remain unchanged.
- Existing exported `CallAddress.From` and `CallAddress.To` fields remain supported phone fields.

This is an intentional behavior correction for native SIP authentication payloads. Consumers
that depended on receiving SIP aliases through phone metadata must switch to native SIP URI state
before rollout.

## Implementation Plan

### Phase 1: Core Capture and Phone Parser

1. Define one phone parser beside `NewCallAddress` using the exact grammar and expose it through
   the existing infra facade for route validation.
2. Capture exact `FromURI` and `ToURI` values.
3. Populate inbound `From` only from a valid From address user.
4. Initialize inbound `To` as empty.
5. Update core tests for positive and negative grammar cases.

### Phase 2: Route Resolution

1. Replace the single-row DID lookup with a bounded query that detects zero, one, or multiple
   active deployment matches before selecting tenant or assistant.
2. Set `CallAddress.To` only from the single validated DID match.
3. Read the agent route phone from the assistant service result already requested with phone
   deployment injection.
4. Remove any inbound `To` header phone fallback.
5. Add missing agent phone and duplicate DID tests.

### Phase 3: Boundary Propagation

1. Return only the middleware-enriched `CallAddress.To` through the infra adapter.
2. Keep captured URI, caller phone, and header ownership in core without a second session copy.
3. Store the enriched value in core inbound configuration and transfer it to the accepted dialog
   identity.
4. Pass the identity snapshot unchanged to application and pipeline callbacks.
5. Add infra and core parity tests proving `To` propagation and captured-field isolation.

### Phase 4: Pipeline and Metadata

1. Keep inbound call-context mapping from `CallAddress.From` and `CallAddress.To`.
2. Keep outbound direction mapping from `CallAddress.To` and `CallAddress.From`.
3. Verify metadata omits empty values.
4. Verify authentication templates receive phone values only.

### Phase 5: Outbound and Transfers

1. Populate outbound phone fields only when authoritative inputs match the grammar.
2. Preserve exact generated URI fields.
3. Apply the same behavior to outbound transfer legs.
4. Verify new inbound transfer and replacement dialogs resolve independently.

### Phase 6: Observability

1. Emit bounded phone source and result attributes at the resolution owner.
2. Add tests proving raw URI and phone values are absent.

## Testing and Verification

Required tests include:

- exact `FromURI` and `ToURI` capture;
- accepted phone forms including leading zero and leading `+`;
- rejection of letters, punctuation, internal whitespace, Unicode digits, plus-only, and
  multiple-plus values;
- inbound `To` initialized empty before routing;
- DID route phone enrichment;
- duplicate DID matches across assistants in one tenant failing closed;
- duplicate DID matches across tenants failing closed before authentication context creation;
- agent route phone enrichment;
- missing, invalid, and ambiguous deployment behavior;
- middleware enrichment propagation through infra and core;
- URI immutability across propagation;
- inbound and outbound call-context direction mapping;
- outbound alias signaling with empty phone metadata;
- transfer, referral, and replacement dialog isolation;
- authentication payload phone values;
- bounded observability without raw values; and
- unchanged non-SIP behavior.

Required commands:

```bash
env GOCACHE=/private/tmp/voice-ai-gocache go test ./api/assistant-api/sip/internal/core
env GOCACHE=/private/tmp/voice-ai-gocache go test ./api/assistant-api/sip/infra
env GOCACHE=/private/tmp/voice-ai-gocache go test ./api/assistant-api/sip/middleware
env GOCACHE=/private/tmp/voice-ai-gocache go test ./api/assistant-api/sip/pipeline
env GOCACHE=/private/tmp/voice-ai-gocache go test ./api/assistant-api/internal/channel/telephony/internal/base
env GOCACHE=/private/tmp/voice-ai-gocache go test ./api/assistant-api/internal/adapters/internal
git diff --check
just agent-finalize "api/assistant-api/sip/internal/core,api/assistant-api/sip/infra,api/assistant-api/sip/middleware,api/assistant-api/sip/pipeline,api/assistant-api/internal/channel/telephony/internal/base,api/assistant-api/internal/adapters/internal"
```

## Rollout

1. Merge only after exact-digest confirmation, implementation verification, and independent
   code review.
2. Deploy the assistant API normally because no schema or service ordering change exists.
3. Validate one DID-routed inbound call, one agent-routed inbound call with a deployment phone,
   one agent route without a deployment phone, and one outbound SIP alias call.
4. Stop rollout if phone resolution produces aliases, raw SIP URIs, ambiguous deployment
   selection, or authentication regression.

## Rollback

Revert the complete implementation as one coherent change, including core parsing, route
resolution, infra and core propagation, outbound construction, transfer behavior, observability,
and tests. No database rollback or data repair is required because only new transient call state
and new conversation metadata values are affected.

## Alternatives Considered

### Add FromPhone and ToPhone

Rejected. Existing `From` and `To` fields already serve the phone role and adding parallel fields
would duplicate state and expand the change.

### Store SIP URIs in Phone Metadata

Rejected. Authentication integrations expect phone values, and route aliases such as
`agent-<assistant-id>` are not phone values.

### Use the To Header User as Assistant Phone

Rejected. The header is caller-controlled and is not the validated route or deployment source.

### Use Custom Called-Number Headers

Rejected for this RFC. A future change may define trusted ingress peers and explicit header
precedence, but arbitrary headers cannot safely drive authentication metadata.

### Add Call-Context URI Columns

Rejected. URI values remain in native SIP session and pipeline state, and no current consumer
requires persisted URI fields.

## Acceptance Criteria

1. `CallAddress` retains `From`, `To`, `FromURI`, `ToURI`, and `Headers` without replacement phone
   fields.
2. `FromURI` and `ToURI` preserve exact parsed SIP addresses for each dialog-forming request.
3. `From` and `To` contain only values matching `^\+?[0-9]+$` or are empty.
4. Inbound `From` comes only from the valid From address user.
5. Inbound `To` comes only from exactly one validated DID route match or a deterministic active
   SIP deployment.
6. Inbound `To` never comes from the `To` header user or custom headers.
7. Zero DID matches return route not found, while multiple active DID matches fail closed before
   tenant or assistant selection.
8. Duplicate DID tests cover different assistants in one tenant and assistants in different
   tenants.
9. Agent routes with missing deployment phone data leave `To` empty.
10. Agent routes use the phone deployment already loaded by the assistant service.
11. Route-enriched `To` reaches session establishment and call-context construction.
12. URI fields remain unchanged across middleware, infra, core, and pipeline boundaries.
13. Outbound aliases remain valid signaling values but produce empty phone fields.
14. `client.phone` and `client.assistant_phone` receive direction-aware phone values only.
15. New transfer, referral, and replacement dialogs resolve independent four-field snapshots.
16. Resolution observability uses bounded source and result values without raw identities.
17. No database, protobuf, REST, SDK, UI, or non-SIP provider change is introduced.
18. Required tests and `just agent-finalize` pass.
19. Independent review reports no unresolved critical or major findings.
20. Final RFC bytes receive exact-digest confirmation before implementation begins.

## Open Questions

None.

## Challenge Resolution

Amendment review round one returned `REVISE` with five major and three moderate findings. This
revision resolves them by replacing the conflicting legacy contract, removing the caller-controlled
`To` fallback, specifying route enrichment propagation and deterministic deployment selection,
defining exact phone grammar and outbound behavior, and completing observability and rollback
requirements.

Amendment review round two returned `BLOCK` because DID routing did not reject duplicate active
deployment matches. The owner approved a fresh governed run. This revision requires exactly one
DID match and rejects duplicates before tenant, assistant, or authentication selection.

## Artifact Index

| Artifact | Purpose | Status |
| --- | --- | --- |
| `rfcs/0003-native-sip-assistant-phone-resolution/jsons/amendment-01-plan.json` | Approved-plan candidate for the amended contract. | Revised |
| `rfcs/0003-native-sip-assistant-phone-resolution/jsons/amendment-01-challenge-round-01.json` | First independent challenge and revision findings. | Resolved |
| `rfcs/0003-native-sip-assistant-phone-resolution/jsons/amendment-01-challenge-round-02.json` | Final blocked challenge from the first governed run. | Escalated |
| `rfcs/0003-native-sip-assistant-phone-resolution/jsons/amendment-02-plan.json` | Fresh-run plan for exact-one DID routing. | Accepted candidate |
| `rfcs/0003-native-sip-assistant-phone-resolution/jsons/amendment-02-challenge.json` | Independent exact-byte challenge receipt. | Pending |
| `rfcs/0003-native-sip-assistant-phone-resolution/jsons/amendment-02-confirmation.json` | Exact-digest confirmation receipt. | Pending |

## Decision Log

| Date | Decision | Rationale |
| --- | --- | --- |
| 2026-08-22 | Preserve exact parsed SIP header addresses. | Signaling and transfer code require exact party addresses. |
| 2026-08-31 | Keep `From` and `To` as phone fields. | Reuse the existing contract and avoid duplicate state. |
| 2026-08-31 | Resolve inbound assistant phone from validated route and deployment data only. | Prevent caller-controlled values from entering authentication metadata. |
| 2026-08-31 | Use `^\+?[0-9]+$` as the phone grammar. | Provide one deterministic rule without country inference. |
| 2026-08-31 | Leave outbound phone fields empty for SIP aliases. | Preserve valid signaling while keeping metadata truthful. |
| 2026-08-31 | Require explicit middleware enrichment propagation. | Prevent validated assistant phone data from being discarded before pipeline setup. |
| 2026-08-31 | Require exactly one active DID deployment match. | Prevent database order from routing a duplicate DID to the wrong assistant or tenant. |
