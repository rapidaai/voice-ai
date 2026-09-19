# Transformer Tests

Run commands from the repository root. The default suite requires no provider
credentials. Tests that contact a provider are gated by the `integration` build
tag and an explicitly enabled configuration entry.

The [behavioral testing and reporting contract](BDD_TESTING.md) defines the
Ginkgo/Gomega pilot. It is additive: existing provider and integration tests
remain in place until migration parity is established.

## Layout

```text
transformer/
  transformer_test.go            Factory selection and invalid configuration
  <provider>/
    *_test.go                   Unit tests beside their implementation
  tests/
    README.md                   Test commands and configuration
    BDD_TESTING.md               Behavioral testing and reporting contract
    contracts/                  Offline Ginkgo/Gomega pilot and report checks
    run_transformer_integration_tests_test.go  Runner selection and exit behavior
    integration/
      tts_contract_test.go      Local HTTP, WebSocket, and synchronous fixtures
      tts_scenarios_test.go     Shared TTS assertions
      tts_integration_test.go   Shared live TTS scenarios
      stt_integration_test.go   Shared live STT scenarios
      <provider>/
        tts_integration_test.go Provider-specific live TTS scenarios
        stt_integration_test.go Provider-specific live STT scenarios
    testutil/                   Shared collectors, configuration, audio support
    testdata/
      hello_world.pcm
      integration_config.yaml.example
      integration_config.yaml  Local credentials, ignored by Git
```

Keep unit tests beside the package they exercise. Integration suites are external
test packages that import provider APIs rather than private implementation details.
Shared TTS scenarios accept a test constructor so local fixtures and live providers
execute the same assertions without modifying the production factory. The existing
test support is shared with unit tests; it is not part of the production runtime.

## Prerequisites

Use the Go version declared in [go.mod](../../../../../go.mod). The race detector
requires CGO. Tests importing the shared factory also link the assistant's native
dependencies, including the Azure Speech SDK and ONNX Runtime. Those libraries
must match the host architecture and be available to the compiler and loader,
even when the selected test uses a local fixture.

The repository's [native CI workflow](../../../../../.github/workflows/reusable-assistant-native-ci.yml)
and [native dependency lock](../../../../../docker/assistant-api/native-deps.lock)
describe the supported CI environment. Missing native headers or libraries are
build prerequisites, not provider failures. Do not disable tests to hide them.

## Default Suite

Run all transformer unit and local-transport tests:

```sh
go test -race -count=1 -timeout=180s ./api/assistant-api/internal/transformer/...
```

Run the shared offline TTS contracts, including interruption and recovery:

```sh
go test -race -run '^TestTTS.*Contract$' -count=3 -timeout=120s ./api/assistant-api/internal/transformer/tests/integration
```

Run one provider's default tests:

```sh
go test -race -count=1 ./api/assistant-api/internal/transformer/cartesia/...
```

The shared contracts cover buffered HTTP responses, WebSocket audio before Done,
late packets after interruption, same-instance connection recovery, message-scoped
metrics, and synchronous audio callbacks. They use local fixtures, not real
provider credentials. Provider-package tests cover additional protocol and error
paths. Passing these tests is not evidence of live service compatibility or call
playback behavior.

## Behavioral Contracts and Reports

The `contracts` package uses Ginkgo v2.33.0, Gomega v1.43.1, and the official
Allure Go model/writer v1.3.1. These are pinned in the root Go module. No separate
Ginkgo installation is needed. The suite covers factory selection, Cartesia and
Deepgram TTS recovery, custom TTS response ownership, AWS/Groq/NVIDIA HTTP STT,
and Deepgram/Speechmatics streaming STT. Tests use local endpoints or injected
transports and synthetic credentials, not live provider accounts.

Run the contracts plus their reporting regression tests:

```sh
go test -race -count=1 -timeout=120s ./api/assistant-api/internal/transformer/tests/contracts
```

Generate reports for the actual contract scenarios in a fresh directory:

```sh
report_dir="$(mktemp -d)"
TRANSFORMER_REPORT_DIR="$report_dir" \
  go test -race -count=1 -timeout=60s \
  ./api/assistant-api/internal/transformer/tests/contracts \
  -run '^TestTransformerContracts$' -args -ginkgo.label-filter='offline'
npx --yes allure@3.18.0 generate "$report_dir/allure-results" --output "$report_dir/html"
npx --yes allure@3.18.0 open "$report_dir/html"
```

Only HTML generation needs Node.js/npm. The Go tests and result files do not
depend on the Allure CLI. The `open` command starts a local report viewer.
Use `transformer-reports/<unique-run>/` for reports inside the repository; that
directory is ignored. Pass its absolute path because `go test` runs inside the
test package directory. Nonempty result directories are rejected to prevent stale
results being mixed into a new run. Use `-count=1` with report output.

- `ginkgo.json` is the authoritative per-scenario result for agents. Check both
  `SuiteSucceeded` and spec outcomes; an absent report is not a successful run.
- `junit.xml` provides CI-compatible results.
- `allure-results/` contains Allure results, flat execution steps, environment
  metadata, and sanitized packet attachments. No audio or transcript payloads
  are attached.
- Native and Allure reports preserve skipped specs, including filtered specs
  and specs not attempted after an earlier failure. These are not provider passes.
- Test failures remain failures even when reports are generated. Report-write
  failures also return a failing test exit status. In CI, publish artifacts in an
  always-run step without overriding that exit status.

`TestContractReports` uses subprocess-only fixtures to verify passing, failing,
timed-out, panicked, skipped, and empty runs; stable identities; attachments;
cleanup failure attribution; selected-but-unattempted specs; and output errors.
Intentional failures never run as part of the normal contract suite.
Use labels such as `stt && http`, `tts && deepgram`, or `factory` for focused
runs. History persistence, automatic CI publication, and migration of every
existing provider test remain outside this rollout. See `BDD_TESTING.md` for
remaining coverage gaps, including the unimplemented RevAI STT constructor.

## Live Integration

Use the schema in [integration_config.yaml.example](testdata/integration_config.yaml.example)
to supply local credentials and account-specific options in
`testdata/integration_config.yaml`, or point `TRANSFORMER_TEST_CONFIG` at another
file. An existing config at the former `transformer/testdata` location can still be
used through `TRANSFORMER_TEST_CONFIG`. Keep every unrelated provider disabled.
Live tests can incur provider usage.
Never commit credentials or include them in test failure messages.

Compile all integration-tagged packages without executing tests or using credentials:

```sh
go test -tags=integration -run '^$' ./api/assistant-api/internal/transformer/tests/integration/...
```

Run the shared TTS suite for one enabled provider:

```sh
TRANSFORMER_TEST_CONFIG=/absolute/path/transformer-tests.yaml \
  go test -tags=integration -count=1 -v -timeout=5m \
  -run '^TestTTSIntegration/cartesia/' ./api/assistant-api/internal/transformer/tests/integration
```

Run a provider-specific live suite:

```sh
go test -tags=integration -count=1 -v -timeout=10m \
  -run '^TestCartesiaTTS' ./api/assistant-api/internal/transformer/tests/integration/cartesia
```

Use `TestCartesiaSTT` for that provider's live STT suite. Shared STT entry points
remain `TestSTTBasic`, `TestSTTInterimAndFinal`, and the other `TestSTT...` cases in
`stt_integration_test.go`.

The runner selects both shared and provider-specific suites for the requested
providers. It always shows Go's verbose results, including skipped tests:

```sh
bin/run-transformer-integration-tests.sh --tts-only deepgram
bin/run-transformer-integration-tests.sh --stt-only azure google
```

No provider arguments selects all supported providers. Configuration still controls
which providers are enabled; successful command exit alone is not a live-provider pass.

Configuration keys match factory identifiers, including `azure-speech-service`,
`google-speech-service`, and `sarvamai`. RevAI TTS is unsupported; its default tests
check rejection rather than attempting synthesis. Custom TTS needs the endpoint
and protocol configuration required by its adapter.

### Interpreting Results

- `-run '^$'` checks compilation only. It does not validate a provider.
- A missing configuration file skips live tests. A missing or disabled provider
  entry skips that provider. Read verbose output before claiming a live pass.
- With `-count=1`, an executed result is not reused from the Go test cache.
- Default tests and local contract tests do not validate real audio quality,
  provider quotas, network conditions, idle prompts, or telephony playback.
- `TestTTSIntegration/<provider>/new-session` creates a fresh transformer. It is
  distinct from reconnecting a failed connection within the same transformer.

## Adding or Changing Tests

Use existing package fixtures first. Keep private implementation assertions in
provider unit tests and shared packet behavior in the common integration scenarios.

For a TTS change, cover success plus its relevant error, cancellation, interruption,
and shutdown paths. Verify message IDs and terminal packets, not just audio counts.
Generation completion is not a playback receipt. Retain regressions for old-message
callbacks and false success after provider errors.

Use channels or `testing/synctest` for ordering-sensitive local tests. Background
goroutines return errors to the test goroutine; they must not call `require` or
`FailNow`. Every test-created producer and transport must unblock and be joined on
failure as well as success. Do not run tests that override a default HTTP client or
WebSocket dialer in parallel; restore globals through cleanup.

Run `gofmt`, targeted tests, and the repository's explicit finalizer before review:

```sh
just agent-finalize "comma,separated,changed,paths"
```

For a changed-file list containing live-only provider packages, include the build
tag and use an empty configuration to avoid provider calls:

```sh
GOFLAGS=-tags=integration TRANSFORMER_TEST_CONFIG=/dev/null \
  just agent-finalize "comma,separated,changed,paths"
```

This runs local tests and compiles the live suites, whose provider cases are skipped.
Run approved live tests separately and report skipped or untested providers explicitly.
