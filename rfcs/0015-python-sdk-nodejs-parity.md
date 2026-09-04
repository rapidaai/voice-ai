# RFC 0015: Python SDK AgentKit and Public Export Parity

- Status: Accepted
- Owner: Python SDK maintainers
- Created: 2026-09-03
- Updated: 2026-09-03
- Reviewers: Independent plan challenger, SDK reviewer

## Summary

Bring the Python SDK AgentKit runtime and AgentKit-related public exports to
behavioral parity with the Node.js SDK pinned at commit
`5faa0e000bf0dc9a6b4c14b4f2cbe095258ee91d`.

The change preserves the existing Python AgentKit API, adds the Node.js V2
conversation and routing model using Python conventions, and refreshes generated
Python protocol bindings from tracked internal-bridge revision
`820666fda038215717d9636e63411fcb8779d0c7` for `agentkit.proto` and
`observability-api.proto` only. The pinned Node.js generated bindings remain
the behavioral and wire-contract source of truth, and unrelated generated churn
is prohibited.

## Context

The repository pins:

- `sdks/nodejs` at `5faa0e000bf0dc9a6b4c14b4f2cbe095258ee91d`.
- `sdks/python` at `a58d3836c7e70184e07236829214443dd5304a87`.

Parent-tree evidence from `git ls-tree HEAD sdks/nodejs sdks/python`:

```text
160000 commit 5faa0e000bf0dc9a6b4c14b4f2cbe095258ee91d	sdks/nodejs
160000 commit a58d3836c7e70184e07236829214443dd5304a87	sdks/python
```

At the pinned Node.js revision, AgentKit is implemented by:

- `sdks/nodejs/src/agentkit/index.ts`: legacy response helpers, ordered
  middleware, TLS, gRPC health, optional HTTP health, and server lifecycle.
- `sdks/nodejs/src/agentkit/v2.ts`: per-conversation agents, routing, packet
  parsing and construction, control packets, observability, and lifecycle
  instrumentation.
- `sdks/nodejs/src/index.ts`: root-level AgentKit and related protocol exports.
- `sdks/nodejs/src/__tests__/agentkit/agentkit.test.ts` and
  `sdks/nodejs/src/__tests__/agentkit/v2.test.ts`: executable behavior contract.
- `sdks/nodejs/README.md`: documented public usage and operational contract.

The current Python SDK provides only the legacy `AgentKitAgent`,
`AgentKitServer`, `SSLConfig`, `AuthConfig`, and `AuthorizationInterceptor` in
`sdks/python/rapida/agentkit/__init__.py`. It does not provide the Node.js V2
`AgentConversation`, `Agent`, `AgentRunner`, or `V2Agent` model, generic
middleware, standard health services, control helpers, or AgentKit
observability helpers.

The generated protocol bindings are also behind the pinned Node.js contract.
The pinned Node.js SDK records protocol revision
`2ac388cd67cddf34141f89182e1f14ae258303d7`, while the Python SDK records
`d389a0d27efefd5a37821c4b3d79818dedb9465a`. The Python `TalkInput` lacks the
assistant, tool-call, and tool-call-result input variants, and `TalkOutput`
lacks user, observability, and control variants. Python also has no generated
`observability_api_pb2` module or `ConversationControl` message.

The required Node.js protocol object is not present in either currently checked
out nested protocol repository. Both configured internal-bridge remotes reject
direct fetch of `2ac388cd67cddf34141f89182e1f14ae258303d7` as not their ref, so
Python generation uses tracked internal-bridge revision
`820666fda038215717d9636e63411fcb8779d0c7` instead. The path
`sdks/python/rapida/artifacts/internal` appears in the Python SDK's
`.gitmodules` file but is not tracked in its current tree; it is not part of
this change unless a separately approved amendment restores it.

### Protocol Source Evidence

- Unavailable fetch evidence: both `lexatic/internal-bridge` and
  `rapidaai/internal-bridge` reject direct fetch of
  `2ac388cd67cddf34141f89182e1f14ae258303d7` as not their ref.
- Tracked fallback revision:
  `820666fda038215717d9636e63411fcb8779d0c7`.
- `agentkit.proto` blob at the fallback revision:
  `7bd94e1b70ef3a08f8a30e1cdc1b0269b0c39807`.
- `observability-api.proto` blob at the fallback revision:
  `ab06b72d70568e2f6d48afeb348e81055c793668`.
- Last relevant protocol changes at the fallback revision: `c97dee7` for
  AgentKit packet control and `a4b87e8` for observability telemetry detail.
- Wire check: the fallback `agentkit.proto` matches the pinned Node.js
  generated bindings for AgentKit oneof field numbers. `TalkInput` uses fields
  1 through 9, and `TalkOutput` uses fields 9 through 17, including
  `observability=16` and `control=17`.

Baseline Python tests were not executed successfully during planning. Direct
`python -m pytest` used Python 3.14 without pytest installed, and `pipenv run
pytest` created an external virtual environment but found no installed test
dependencies. This is an environment limitation, not passing baseline evidence.

## Goals

- Preserve every existing Python AgentKit import and legacy helper behavior.
- Add Pythonic AgentKit V2 APIs with one isolated agent instance per conversation.
- Match the pinned Node.js routing, packet, control, lifecycle, error, and
  observability behavior.
- Add ordered generic middleware and standard gRPC and optional HTTP health
  behavior to the existing Python server.
- Generate only the Python AgentKit and observability bindings from tracked
  internal-bridge revision `820666fda038215717d9636e63411fcb8779d0c7`, while
  treating the pinned Node.js generated bindings as the behavioral and
  wire-contract source of truth.
- Export the AgentKit runtime and required AgentKit protocol types from both
  `rapida.agentkit` and the package root where appropriate.
- Provide focused, full-suite, type, format, package, and repository validation.

## Non-Goals

- Modify `sdks/nodejs` or its nested generated-protocol checkout.
- Add unrelated Python client wrappers solely because Node.js exports them.
- Change server-side protocol definitions beyond the pinned Node.js generated
  contract.
- Redesign authentication or authorization.
- Implement application LLM, tool execution, or conversation business logic.
- Add deployment manifests or Kubernetes resources.
- Add retries, telemetry backends, routing predicates, or buffering policies
  absent from the pinned Node.js contract.
- Introduce an asynchronous Python server runtime in this release.
- Bump the Python package version unless the release owner separately requests it.

## Scope and Ownership

### Allowed Paths

- `sdks/python/rapida/clients/protos/artifacts` - protocol source gitlink owner;
  may move only to `820666fda038215717d9636e63411fcb8779d0c7`.
- `sdks/python/rapida/clients/protos/agentkit_pb2.py` - generated Python
  message output owner; no manual edits.
- `sdks/python/rapida/clients/protos/agentkit_pb2.pyi` - generated Python type
  output owner; no manual edits.
- `sdks/python/rapida/clients/protos/agentkit_pb2_grpc.py` - generated Python
  gRPC output owner; no manual edits.
- `sdks/python/rapida/clients/protos/observability_api_pb2.py` - generated
  Python message output owner; no manual edits.
- `sdks/python/rapida/clients/protos/observability_api_pb2.pyi` - generated
  Python type output owner; no manual edits.
- `sdks/python/rapida/clients/protos/observability_api_pb2_grpc.py` -
  generated Python gRPC output owner; no manual edits.
- `sdks/python/rapida/agentkit/__init__.py` - legacy AgentKit server,
  middleware, health, compatibility exports, and lifecycle owner.
- `sdks/python/rapida/agentkit/v2.py` - AgentKit V2 conversation, agent,
  routing, packet, and instrumentation owner.
- `sdks/python/rapida/__init__.py` - package root public export owner.
- `sdks/python/tests/agentkit/test_agentkit.py` - legacy AgentKit behavior owner.
- `sdks/python/tests/agentkit/test_v2.py` - AgentKit V2 behavior owner.
- `sdks/python/tests/test_proto_imports.py` - generated protocol import owner.
- `sdks/python/tests/test_public_exports.py` - package public export owner.
- `sdks/python/README.md` - Python AgentKit usage and operational documentation
  owner.
- `sdks/python/pyproject.toml` - package data owner only for shipping generated
  `*_pb2.pyi` files; version remains unchanged.

No other generated path or gitlink is in scope. The implementation owner must
reject churn outside the single tracked
`sdks/python/rapida/clients/protos/artifacts` gitlink, the three
`agentkit_pb2*` outputs, the three `observability_api_pb2*` outputs, and any
required package-data wiring for generated `.pyi` files.

### Out-of-Scope Paths

- `sdks/nodejs/**`
- `sdks/python/rapida/artifacts/internal`, which is not tracked at the verified
  Python revision
- `api/**`
- `ui/**`
- `pkg/**`
- `cmd/**`
- Other SDK and example submodules
- RFC JSON artifacts, except when changed by their assigned lifecycle owner

## Proposed Design

### Protocol Source and Generated Bindings

1. Record the failed fetch evidence for
   `2ac388cd67cddf34141f89182e1f14ae258303d7` and verify tracked internal-bridge
   commit `820666fda038215717d9636e63411fcb8779d0c7` plus the referenced
   `agentkit.proto` and `observability-api.proto` blobs.
2. Update only `sdks/python/rapida/clients/protos/artifacts` to
   `820666fda038215717d9636e63411fcb8779d0c7`.
3. Run selective `grpc_tools.protoc` generation only for
   `agentkit.proto` and `observability-api.proto`, then apply the existing
   import-rewrite step only to those regenerated outputs. The broad
   `sdks/python/bin/artifacts-generate.sh` script is prohibited for this RFC
   because it deletes and regenerates unrelated bindings.
4. Confirm the generated AgentKit contract includes:
   - `TalkInput` variants for initialization, configuration, user,
     interruption, metadata, metric, tool call, tool-call result, and assistant.
   - `TalkOutput` variants for initialization, interruption, user, assistant,
     tool call, tool-call result, error, observability, and control.
   - `ConversationControl` actions and controlled packet types.
   - Observability log, event, metric, wrapper, and record-kind messages.
5. Review the generated diff and retain only changes in
   `agentkit_pb2.py`, `agentkit_pb2.pyi`, `agentkit_pb2_grpc.py`,
   `observability_api_pb2.py`, `observability_api_pb2.pyi`, and
   `observability_api_pb2_grpc.py`, plus any required `.pyi` packaging updates.

### Legacy AgentKit Server

Keep the current synchronous `grpc.Server`, thread-pool ownership, TLS support,
and Python method names. Extend the current server directly instead of adding a
second server abstraction.

The server will accept an ordered middleware sequence. Middleware may be a
callable or a `Middleware` instance and receives a request context containing
metadata, the full method path, peer, host when available, deadline when
available, a first-value metadata accessor, and a rejection operation. Returning
`False` rejects with `UNAUTHENTICATED`; an explicit rejection may set status,
details, and trailing metadata; middleware exceptions map to `INTERNAL` without
exposing exception details. Only the exact
`/grpc.health.v1.Health/Check` method bypasses application middleware.

`AuthConfig` and `AuthorizationInterceptor` remain public compatibility APIs.
Their behavior is implemented through or adapted to the generic middleware
pipeline so authentication has one enforcement path.

The server registers the standard gRPC health check by default. Its status is
`NOT_SERVING` before startup, becomes `SERVING` only after successful startup,
can be changed explicitly, and returns to `NOT_SERVING` during shutdown or
startup failure. Callers may disable SDK-owned gRPC health if they register a
different implementation.

An optional standard-library HTTP server exposes health on a separate host,
port, and path. It accepts `GET` and `HEAD`, returns JSON with status and HTTP
200 only while serving, returns 503 otherwise, returns 404 for other paths, and
returns 405 for other methods. If HTTP health binding fails after gRPC startup,
the server closes the gRPC server and leaves no partially running service.
`stop()` closes both servers and remains safe before start or when called again.

### AgentKit V2 Runtime

Add `sdks/python/rapida/agentkit/v2.py` as the sole owner of the V2 runtime.
Public payloads use typed dataclasses or equivalent explicit Python types and
retain the raw protobuf message for advanced consumers.

`AgentConversation` owns exactly one stream's mutable state, initialization,
latest configuration, active user message ID, output queue, closed state, and
shutdown signal. `AgentRunner` owns packet parsing, packet construction,
per-stream sequencing, routing, lifecycle dispatch, and automatic
instrumentation. Each stream owns one `AgentConversation` and one selected
`Agent`; state is never shared across streams.

The synchronous implementation uses one bounded, explicitly closed output path
per stream. Input processing preserves receive order, output writes preserve
send order, cancellation wakes blocked work, and close is idempotent. Queue or
worker ownership must remain inside the stream handler, and no background
thread may outlive its gRPC stream.

Agent resolution order is:

1. A configured custom factory.
2. An exact assistant ID and assistant version route.
3. An assistant ID route with no version.
4. The default agent class.

Initialization creates the agent, stores conversation metadata, emits the
initialization acknowledgement, records lifecycle instrumentation, and then
invokes the application initialization hook. A non-initialization packet before
agent creation uses the default agent if present; otherwise it produces the
documented AgentKit error path rather than invoking a missing agent.

Optional lifecycle hooks cover initialization, configuration, user, assistant,
interruption, tool call, tool-call result, close, and error. Conversation and
agent convenience methods cover raw send, reply, assistant and user messages,
tool calls and results, interruption, control, block, unblock, transfer, end
conversation, errors, raw observability, logs, events, metrics, and close.

String replies and control helpers use the active user message ID unless an ID
is supplied. Message builders support text or audio and explicit completion.
Tool values are converted to protocol strings consistently, including lowercase
boolean values, and failed results add `success=false` only when the caller did
not supply that key. Transfer and end-conversation helpers use the reserved
names `transfer_conversation` and `end_conversation`, the matching action enums,
and a generated `agentkit-tool-<uuid>` tool ID when one is not supplied.

Instrumentation is enabled by default and can be disabled. It emits lifecycle
events, initialization and user-handler duration metrics, and error logs; manual
log, event, and metric helpers remain available. Records use UTC protobuf
timestamps, generated `agentkit-<uuid>` IDs, string attributes, the
`agentkit-sdk` event component default, and available platform correlation
fields. Instrumentation write failure must not fail or close an otherwise
healthy conversation.

### Public Exports

`rapida.agentkit` and the package root explicitly expose the supported legacy
and V2 runtime names. Existing imports remain valid. AgentKit-related generated
types include `TalkInput`, `TalkOutput`, `ConversationControl`, initialization,
configuration, user, assistant, interruption, tool-call, tool-call-result,
error, tool action, and observability messages and enums.

Python uses snake_case methods and attributes. It does not add duplicate
camelCase method aliases merely to reproduce TypeScript syntax.

## Contracts and Compatibility

- Existing `AgentKitAgent`, `AgentKitServer`, `SSLConfig`, `AuthConfig`, and
  `AuthorizationInterceptor` imports and callable behavior remain supported.
- Existing synchronous `AgentKitServer.start()`, `stop()`, and
  `wait_for_termination()` usage remains valid.
- New V2 hooks and send helpers are synchronous in the first release to match
  the established Python server execution model.
- Generated field numbers, oneof membership, enum values, and service paths must
  exactly match the pinned Node.js generated bindings, even though Python
  generation uses tracked revision `820666fda038215717d9636e63411fcb8779d0c7`.
- Public Python names follow Python conventions while providing the same
  behavior and protocol coverage as the pinned Node.js API.
- No existing root export may be removed or shadowed.
- No new runtime dependency is permitted unless implementation proves existing
  grpc, protobuf, and standard-library support insufficient and the RFC is
  amended and reconfirmed.

## Failure and Recovery

- Missing fallback protocol source or descriptor mismatch against the pinned
  Node.js generated bindings blocks generation and implementation. Generated
  files must never be reconstructed by hand.
- Invalid or unsupported inbound packets do not corrupt conversation state and
  do not reorder later valid packets.
- Missing initialization without a default route produces a deterministic error.
- Route construction or lifecycle-hook failure flows through `on_error` when an
  agent exists, otherwise it emits an internal AgentKit error; a secondary error
  closes the stream.
- Middleware rejection prevents the AgentKit handler from receiving stream data.
- Middleware exceptions fail closed with `INTERNAL` and generic details.
- TLS file or credential errors fail startup without insecure fallback.
- HTTP health startup failure shuts down any gRPC server started by that call.
- Cancellation and peer disconnect stop input work, wake output waits, invoke
  close at most once, and release stream-owned resources.
- `stop()` is idempotent and leaves both health status and server references in
  a stopped state.

## Security and Privacy

- Middleware bypass is limited to the exact standard gRPC health RPC path.
- Authentication compatibility uses one middleware enforcement path to prevent
  divergent policy.
- Middleware context exposes metadata to application code but SDK logs and error
  responses never include credentials, tokens, or complete arbitrary metadata.
- Application exception text is not returned automatically when it may contain
  secrets; public error details are explicit application-owned values.
- Requested TLS never silently falls back to an insecure listener.
- HTTP health binds only to configured host and port, exposes status only, and
  does not expose AgentKit metadata or application state.
- Per-conversation state and correlation attributes are not shared between
  streams or tenants.

## Observability

AgentKit V2 emits protocol-level observability packets rather than adding a new
SDK telemetry backend. Automatic records cover conversation initialization,
configuration receipt, user receipt, user-handler duration, assistant receipt,
interruption, tool calls, tool-call results, conversation close, and errors in
parity with the pinned Node.js implementation.

Record construction includes IDs, kinds, timestamps, component, description,
values, string attributes, and available platform correlation fields.
Instrumentation is best effort and must not become the reason a conversation
fails. Server startup, secure or insecure binding, middleware enablement,
shutdown, and startup failure retain actionable Python logging without secrets.

## Data and Migration

None. The change adds SDK runtime behavior, public exports, and generated wire
bindings. It does not modify persistent schemas or stored data.

## Rollout

1. Record the unavailable-fetch evidence for
   `2ac388cd67cddf34141f89182e1f14ae258303d7` and verify fallback revision
   `820666fda038215717d9636e63411fcb8779d0c7`.
2. Update the single tracked Python protocol gitlink and regenerate only
   `agentkit_pb2.py`, `agentkit_pb2.pyi`, `agentkit_pb2_grpc.py`,
   `observability_api_pb2.py`, `observability_api_pb2.pyi`, and
   `observability_api_pb2_grpc.py`.
3. Review generated descriptors against the pinned Node.js AgentKit fields and
   enum values before handwritten work proceeds.
4. Implement and test legacy middleware and health behavior without removing
   compatibility APIs.
5. Implement and test the V2 runtime and per-stream resource lifecycle.
6. Add root and package export tests, then update the README.
7. Run focused tests, the full Python suite, static checks, package build, and
   repository finalization.
8. Obtain independent verification and code review before release.

Rollout stops if the fallback revision no longer reproduces the pinned Node.js
contract, any unrelated generated file changes appear, existing imports fail,
stream ordering or cleanup tests fail, middleware permits rejected calls,
health status is inaccurate, or any required validation fails.

## Rollback

Restore the Python protocol gitlink to
`d389a0d27efefd5a37821c4b3d79818dedb9465a` and revert the selectively
generated `agentkit_pb2*` and `observability_api_pb2*` files, then revert the
single Python SDK implementation commit containing AgentKit runtime, exports,
tests, documentation, and any `pyproject.toml` package-data change for
generated `.pyi` files. No server-side or persistent-data rollback is
required. If runtime disablement is needed before package rollback, consumers
can continue using the preserved legacy `AgentKitAgent` and omit V2,
middleware, and HTTP health options.

## Alternatives Considered

- Hand-edit Python generated protobuf files. Rejected because generated files
  are not authoritative and manual edits risk wire drift.
- Generate every Python binding from a nearby or current protocol revision.
  Rejected because parity remains pinned to the Node.js generated contract, and
  this amendment allows only the minimal tracked fallback needed for
  `agentkit.proto` and `observability-api.proto`.
- Replace the existing Python server with `grpc.aio`. Rejected because it would
  break the established synchronous lifecycle and add migration complexity not
  required for first parity.
- Put legacy and V2 behavior in one large module. Rejected because the pinned
  Node.js SDK already has a clear legacy and V2 ownership split.
- Add a new health-check dependency. Rejected unless existing grpc, protobuf,
  and standard-library facilities prove insufficient.
- Export every Node.js client and protocol symbol. Rejected because the goal is
  AgentKit and its required contract surface, not cross-language SDK expansion.
- Remove `AuthConfig` and `AuthorizationInterceptor`. Rejected because parity
  must not break existing Python consumers.

## Testing and Verification

Prepare the Python development environment:

- `cd sdks/python && pipenv sync --dev`

Regenerate and inspect protocol output:

- `cd sdks/python && python3 -m grpc.tools.protoc -I ./rapida/clients/protos/artifacts --pyi_out=./rapida/clients/protos --python_out=./rapida/clients/protos --grpc_python_out=./rapida/clients/protos ./rapida/clients/protos/artifacts/agentkit.proto ./rapida/clients/protos/artifacts/observability-api.proto`
- `cd sdks/python && sed -i.bak -E '/^import [a-zA-Z0-9_]+_pb2/ s|import ([a-zA-Z0-9_]+_pb2)|import rapida.clients.protos.\\1|' rapida/clients/protos/agentkit_pb2.py rapida/clients/protos/agentkit_pb2.pyi rapida/clients/protos/agentkit_pb2_grpc.py rapida/clients/protos/observability_api_pb2.py rapida/clients/protos/observability_api_pb2.pyi rapida/clients/protos/observability_api_pb2_grpc.py`
- `cd sdks/python && rm -f rapida/clients/protos/agentkit_pb2.py.bak rapida/clients/protos/agentkit_pb2.pyi.bak rapida/clients/protos/agentkit_pb2_grpc.py.bak rapida/clients/protos/observability_api_pb2.py.bak rapida/clients/protos/observability_api_pb2.pyi.bak rapida/clients/protos/observability_api_pb2_grpc.py.bak`
- `git -C sdks/python diff --stat`
- `git -C sdks/python diff -- rapida/clients/protos/agentkit_pb2.py rapida/clients/protos/agentkit_pb2.pyi rapida/clients/protos/agentkit_pb2_grpc.py rapida/clients/protos/observability_api_pb2.py rapida/clients/protos/observability_api_pb2.pyi rapida/clients/protos/observability_api_pb2_grpc.py`

Run focused behavior and export tests:

- `cd sdks/python && pipenv run pytest -q tests/agentkit/test_agentkit.py tests/agentkit/test_v2.py tests/test_proto_imports.py tests/test_public_exports.py`

Run full Python verification:

- `cd sdks/python && pipenv run pytest -q`
- `cd sdks/python && pipenv run mypy rapida --exclude=artifacts --ignore-missing-imports --cache-dir=/dev/null`
- `cd sdks/python && pipenv run black --check rapida tests`
- `cd sdks/python && python -m build`

Run repository checks from the repository root:

- `git diff --check`
- `just validate-rfc-layout`
- `just agent-finalize "sdks/python"`

Focused tests must cover:

- Existing legacy response and request helpers.
- Middleware order, callable and class middleware, default and explicit
  rejection, exception handling, binary metadata conversion, and exact health
  bypass.
- gRPC health default registration, status transitions, disablement, and
  explicit override.
- HTTP health GET, HEAD, 200, 503, 404, 405, dynamic bound address, bind
  failure cleanup, and stop behavior.
- V2 parsing and construction for every supported packet variant.
- Text and audio oneof behavior, completion flags, action enums, generated IDs,
  result failure encoding, and scalar conversion.
- One agent and isolated state per concurrent stream.
- Sequential lifecycle delivery and output ordering.
- Route precedence, default routing, custom factory creation, and missing-route
  failure.
- Active message ID correlation and block/unblock control helpers.
- Manual and automatic observability records, correlation fields, UTC
  timestamps, disablement, and isolated telemetry failure.
- Cancellation, peer error, close-once behavior, queue shutdown, and no leaked
  stream worker.
- Imports from both `rapida.agentkit` and `rapida`, including all preserved
  legacy names and required new AgentKit protocol names.
- Wheel and source distribution contents include the V2 module, generated
  protocol bindings, and generated `.pyi` files.
- Descriptor tests assert AgentKit field numbers against the pinned Node.js
  generated bindings and cover observability record kinds.

## Acceptance Criteria

- [ ] The Python artifact gitlink exactly matches
  `820666fda038215717d9636e63411fcb8779d0c7`, and only
  `agentkit_pb2.py`, `agentkit_pb2.pyi`, `agentkit_pb2_grpc.py`,
  `observability_api_pb2.py`, `observability_api_pb2.pyi`, and
  `observability_api_pb2_grpc.py` are regenerated from that source.
- [ ] Python `TalkInput`, `TalkOutput`, `ConversationControl`, and observability
  descriptors match the pinned Node.js field numbers, oneofs, enum values, and
  service definitions.
- [ ] Existing AgentKit public imports and legacy helper behavior remain valid.
- [ ] Generic middleware runs in order, fails closed, supports explicit
  rejection, and bypasses only the exact standard health RPC.
- [ ] gRPC health is enabled by default and reports accurate lifecycle state;
  optional HTTP health implements the documented method, path, status, and
  cleanup contract.
- [ ] TLS startup fails safely and never downgrades a requested secure listener.
- [ ] Each stream owns one conversation, one routed agent, isolated mutable
  state, ordered input processing, ordered output, and deterministic cleanup.
- [ ] Route selection follows custom factory, exact assistant/version,
  assistant-only, then default precedence.
- [ ] All supported input packets reach the corresponding lifecycle hook and all
  output helpers produce the expected protocol oneof and field values.
- [ ] Control, transfer, end-conversation, error, and active-message correlation
  behavior matches the pinned Node.js tests.
- [ ] Automatic instrumentation is enabled by default, can be disabled, and
  cannot fail the conversation when telemetry writes fail.
- [ ] AgentKit runtime and required protocol names are publicly importable from
  their documented modules without removing existing exports.
- [ ] Focused tests, full tests, type checks, formatting, package build,
  repository finalization, independent verification, and independent code
  review pass with no critical or major findings.

## Open Questions

None. The plan fixes Pythonic naming, synchronous execution, optional HTTP
health inclusion, AgentKit-only export scope, compatibility preservation, and
no version bump unless separately requested.

## Challenge Resolution

Independent challenge in `jsons/challenge.json` identified the missing protocol
source as a blocker. Accepted amendment `jsons/amendment-01-plan.json` resolves
that blocker by recording failed fetch evidence for
`2ac388cd67cddf34141f89182e1f14ae258303d7`, selecting tracked fallback revision
`820666fda038215717d9636e63411fcb8779d0c7`, and constraining regeneration to
`agentkit.proto` and `observability-api.proto` only. This candidate also keeps
the pinned Node.js generated bindings as the behavioral and wire-contract
source of truth, narrows generated scope to six Python outputs plus any
required `.pyi` packaging change, and prohibits unrelated generated churn.

## Artifact Index

- `rfcs/0015-python-sdk-nodejs-parity/jsons/plan.json` - Initial verified
  implementation plan. Accepted candidate.
- `rfcs/0015-python-sdk-nodejs-parity/jsons/challenge.json` - Independent
  challenge findings requiring this revision.
- `rfcs/0015-python-sdk-nodejs-parity/jsons/amendment-01-plan.json` -
  Accepted amendment replacing the unavailable protocol source with tracked
  fallback revision `820666fda038215717d9636e63411fcb8779d0c7` and selective
  AgentKit plus observability generation.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-09-03 | Use Governed lifecycle because this changes public API and generated protocol contracts. | Coordinator | `jsons/plan.json`, `DEVELOPMENT_PROCESS.md` |
| 2026-09-03 | Keep Node.js commit `5faa0e000bf0dc9a6b4c14b4f2cbe095258ee91d` and its generated bindings as the parity source of truth, but generate Python AgentKit and observability bindings from tracked internal-bridge revision `820666fda038215717d9636e63411fcb8779d0c7` because protocol revision `2ac388cd67cddf34141f89182e1f14ae258303d7` is unavailable from configured remotes. | Python SDK maintainers | `git ls-tree HEAD sdks/nodejs sdks/python`, failed fetch evidence, `jsons/amendment-01-plan.json` |
| 2026-09-03 | Preserve the synchronous Python server and expose Pythonic V2 methods. | Python SDK maintainers | `jsons/plan.json` |
| 2026-09-03 | Preserve legacy auth exports and adapt them to one generic middleware path. | Python SDK maintainers | `jsons/plan.json` |
| 2026-09-03 | Exclude the untracked `rapida/artifacts/internal` path from implementation scope. | RFC author | Python SDK tree and `.gitmodules` inspection |
| 2026-09-03 | Limit generated protocol scope to one tracked gitlink, the three `agentkit_pb2*` outputs, the three `observability_api_pb2*` outputs, and any required `.pyi` package-data wiring. | RFC author | `jsons/challenge.json`, `jsons/plan.json`, `jsons/amendment-01-plan.json` |
| 2026-09-03 | Allow `sdks/python/pyproject.toml` changes only to package generated `.pyi` files and never to bump the version for this RFC. | RFC author | `jsons/challenge.json`, `jsons/plan.json` |
| 2026-09-03 | Require selective `grpc_tools.protoc` generation only for `agentkit.proto` and `observability-api.proto`, and reject unrelated generated churn. | RFC author | `jsons/amendment-01-plan.json` |
