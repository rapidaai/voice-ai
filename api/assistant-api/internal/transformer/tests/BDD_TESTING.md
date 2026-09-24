# Transformer Behavioral Tests and Reports

## Scope

This is a test-only rollout, not a production lifecycle change or a full test
migration. The `contracts/` suite covers provider selection, Cartesia and
Deepgram TTS recovery, custom TTS response ownership, HTTP STT providers,
streaming STT transports, and selected SDK configuration and lifecycle paths.
Retain the existing provider, shared-contract, and live-integration tests while
evaluating the new structure. No provider credentials or external calls are used.

Allowed changes are this test directory, the root Go dependency files, and the
report-output ignore rule. Do not change production code, live-provider selection,
or CI deployment workflows for this pilot.

## Test Contract

- Use Ginkgo v2 and Gomega for the pilot. Keep existing tests unchanged.
- Structure scenarios as contract, condition, and observable behavior. Keep
  nesting shallow and packet actions visible inside each `It`.
- Each scenario owns its transformer, fake server, connections, and cleanup.
- Coordinate asynchronous events through channels or bounded assertions. Do not
  use sleeps to try to reproduce ordering bugs.
- Exercise the real provider against local WebSocket endpoints or injected HTTP
  transports. Check message ownership, audio ordering, completion, failure,
  and recovery. Factory tests assert a nonnil implementation and its name.
- Keep playback receipts and session timeout policy outside transformer tests.

## Reporting Contract

Ginkgo owns test execution and outcomes. Generate its native JSON for agents and
JUnit XML for CI. Allure is a human-facing projection of the same results, not a
second source of pass/fail decisions.

The inspected community adapters are not used: `ramich2077/allure-ginkgo` at
`3ac6d01d327a` omits timeout status and swallows some write errors;
`Moon1706/ginkgo2allure` at `v0.3.0` infers status from failure metadata rather
than distinguishing skipped tests. The pilot uses the official Allure Go
`commons/model` and `commons/writer` in one suite-local report callback.
Do not add a generic reporting framework or change the production packet model.

- Reporting is opt-in through `TRANSFORMER_REPORT_DIR`, using a fresh directory
  for each run. Ordinary `go test` must not require an Allure installation.
- Emit `ginkgo.json`, `junit.xml`, and `allure-results/` beneath that directory.
- Record every selected scenario's outcome, timing, labels, and failure location.
  Timeouts and panics must never become passed or skipped results.
- Represent flat `By` steps and attach allowlisted packet metadata. Do not attach
  raw audio, transcript text, credentials, request headers, or whole packets.
- Preserve stable test identities across runs. Runtime seed and toolchain
  metadata must not alter those identities.
- Fail the test run on report-write errors. Preserve artifacts from failed tests.
- Empty selection fails. Skipped scenarios remain skipped and do not establish
  provider compatibility. Preserve every native spec outcome, including filtered
  specs and selected specs not attempted after another failure.
- Historical trend storage and CI publication are follow-up work. A local report
  must not imply that either has already been configured.

## Acceptance

1. Local synthesis emits audio before completion for the correct message.
2. Interruption suppresses late old-message output and admits the next message.
3. A connection failure emits an error, not successful completion, and the same
   transformer reconnects for the next message.
4. Report tests exercise pass, assertion failure, timeout, panic, skip,
   attachments, stable identities, empty selection, and a report-write failure.
   Preserve selected-but-unattempted specs and attribute failures to their actual
   step even when successful cleanup follows.
5. Existing transformer tests and live-suite selection remain unchanged.
6. Generate and inspect an Allure report from the actual pilot results.
7. AWS, Groq, and NVIDIA STT cover request payloads, transcript ownership,
   transport and response failures, cancellation, and pending-request shutdown.
8. Deepgram and Speechmatics STT exercise their streaming lifecycle locally.
   Deepgram opens its socket during construction, so its factory selection must
   run inside the local WebSocket fixture rather than the constructor matrix.
9. Deepgram TTS covers missing clear acknowledgements and cancellation; custom
   TTS covers configured response IDs and missing-ID fallback.

### Remaining STT Rollout

This test-only phase is limited to `tests/contracts/*_stt_test.go`,
`tests/contracts/factory_test.go`, and this directory's documentation. Keep
production providers, dependencies, live configuration, and reporting logic
unchanged. Retain existing tests until migration parity is established.

Validation also repairs an existing assertion in
`tests/integration/tts_contract_test.go`: Cartesia interruption can either send
cancel on an idle socket or close and reconnect when a write is in flight.
Accept only those paired connection/cancel counts; retain message-output checks.

- Add local transport contracts for Sarvam, Smallest, Cartesia, and custom STT.
- Check interim and final transcripts, PCM framing, turn ownership, provider
  failures, and shutdown using the existing public provider APIs.
- Exercise Azure and Google SDK lifecycle paths only where existing endpoints
  or client configuration permit offline execution. Record unsupported paths.
- Verify missing-credential rejection and pre-initialization cleanup without
  network access. Do not call AssemblyAI `Initialize`: it uses a private dialer
  with a hard-coded external endpoint and has no public transport injection.
- Run STT-focused race tests, randomized full contracts, full transformer tests,
  reporting regression tests, and `just agent-finalize` before completion.

## Remaining Gaps

RevAI STT is registered but its constructor currently returns `(nil, nil)`.
It is not counted as a working factory selection or covered runtime. Fixing
that production contract is separate from this test-only change.

Groq and NVIDIA STT share a latency start time across concurrent requests. The
reversed-response scenario reproduces two correctly owned transcripts but only
one latency metric. The native report records the observed metric contexts as
`overlapping-request-latency`; the test does not declare missing metrics valid.
Per-request latency ownership remains a production follow-up.

Sarvam and Smallest STT can initialize a connection after their session context
has been canceled. Local-server diagnostics reproduced successful WebSocket
upgrades and a nil initialization error after cancellation. The streaming
contracts cover explicit shutdown, not rejection of canceled initialization;
adding that expectation requires a production cancellation fix.

Azure STT ignores the result channels returned by native asynchronous start and
stop calls. The local handshake-rejection scenario observed one startup and one
shutdown goroutine blocked on result delivery after Close. A bounded subprocess
contains those SDK resources; waiter counts are diagnostic, not accepted leak
behavior. Consuming the SDK results remains a production follow-up. Subprocess
execution is not merged into the parent Go coverage profile.

These scenarios do not replace live protocol validation or cover every STT SDK
shutdown and reconnection branch. In particular:

| Provider | New Offline Boundary | Not Established |
| --- | --- | --- |
| AssemblyAI | Credentials and pre-initialization cleanup | Streaming, parsing, provider errors; endpoint and dialer are not injectable |
| Google | Recognition configuration and credential validation | Real streaming lifecycle; public constructor cannot accept a local client |
| Azure | Native configuration, pre-initialization cleanup, local handshake rejection | Successful recognition and complete native resource cleanup |

Existing Google stream-recreation tests remain in the provider package. They are
not replaced by configuration assertions in the new suite.

Primary test risks are leaked fake-server connections, process-global transport
overrides, and assertions that finish before provider callbacks. Transport specs
run serially, own cleanup, and use bounded callback or request barriers.

## Verification

Run from the repository root:

```sh
go test -race -count=1 -timeout=120s ./api/assistant-api/internal/transformer/tests/contracts
TRANSFORMER_REPORT_DIR=/tmp/transformer-contracts-run \
  go test -race -count=1 -timeout=60s \
  ./api/assistant-api/internal/transformer/tests/contracts -run '^TestTransformerContracts$'
go test -race -count=1 -timeout=180s ./api/assistant-api/internal/transformer/...
go test -tags=integration -run '^$' ./api/assistant-api/internal/transformer/tests/integration/...
```

Use a new report directory for each invocation. Run the repository's
`just agent-finalize` with the changed test paths before completion. Report
generation and exact pinned dependency versions are documented in `README.md`
after verification. Remove the pilot directory and its dependencies to roll back;
the existing test coverage remains available.
