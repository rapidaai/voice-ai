# RFC 0002: Assistant Native CI and Linux Core System Tests

- Status: Accepted
- Date: 2026-08-22
- Owners: Platform, Assistant API, and Release Engineering teams
- Reviewers: Web API, Integration API, Endpoint API, UI, Data, Security, and Telephony owners

## Summary

Close the two verified assistant-api CI omissions with executable Linux-native
coverage and add one hermetic, stack-owning Linux system-test job.

This RFC deliberately separates the smallest required first merge from larger
reliability programs:

1. a dedicated assistant native-toolchain CI lane performs package load, vet, unit
   tests, race tests, and a production-equivalent binary build;
2. one Linux job owns one disposable Compose stack and sequentially performs fresh
   migrations, semantic health checks, UI/nginx checks, the existing assistant-api
   OpenAPI/Newman smoke flow, and cleanup assertions;
3. one compatibility job checks the existing assistant/talk OpenAPI artifacts and
   descriptors generated from tracked protobuf Go packages against the pull-request
   target branch.

Previous-release upgrades, FreeSWITCH voice lifecycles, provider-failure injection,
restart/drain behavior, OpenSearch, and generalized multi-service smoke orchestration
remain required roadmap items, but they are not silently coupled to this first CI
correction. They require follow-up RFCs with their own owners, runtime evidence, and
confirmation gates.

## Motivation

`.github/workflows/reusable-go-ci.yml` excludes `./api/assistant-api/...` and
`./cmd/assistant/...` from its Go package list. Its generic binary build matrix also
excludes assistant-api.

Adding assistant paths to the existing generic jobs is not valid. Assistant-api is
a native Linux application that requires CGO, ONNX Runtime, Azure Speech SDK,
RNNoise, Opus, libc++, and TEN VAD. A verified probe of the proposed generic command
failed:

```text
CGO_ENABLED=0 GOOS=linux go build ./cmd/assistant/assistant.go
```

The failure contains unresolved native symbols across Opus, ONNX, Azure Speech,
RNNoise, FireRed/Silero VAD, Pipecat, and TEN VAD. The production assistant image
already supplies the required native toolchain. CI must exercise that environment
instead of claiming parity through a CGO-disabled build.

The root developer Compose file is the authoritative service topology, but its
developer defaults persist home-directory state, publish fixed ports, and use fixed
container names. CI must exercise that same topology through a thin override that
neutralizes only host-specific behavior and adds test orchestration.

## Verified Current State

- `GO_CI_PACKAGES` omits assistant-api and `cmd/assistant`.
- The generic Go build matrix contains web, integration, and endpoint only.
- The package workflow produces standalone CGO-disabled tarballs and therefore
  cannot package a complete assistant runtime.
- `docker/assistant-api/Dockerfile` builds assistant-api with CGO and the required
  native libraries.
- `reusable-docker-ci.yml` already builds the assistant image, but it does not run
  package load, vet, tests, race tests, or system behavior checks.
- The four Go API commands run migrations on startup and panic when migration fails;
  migration failure is already fatal.
- The four APIs expose `/readiness/` and `/healthz/`.
- Each readiness handler always returns HTTP 200, so HTTP status alone is not a
  readiness assertion. Each handler keys its JSON `data` map by the PostgreSQL
  connector's `Name()`. In the CI topology, host `postgres` and port `5432` produce
  the exact key `PSQL psql://postgres:5432`.
- Existing checked-in REST contract artifacts are limited to
  `openapi/artifacts/assistant-api.yaml`, `talk-api.yaml`, and `common.yaml`.
- The existing assistant smoke collection is
  `openapi/postman/assistant-api/assistant-api.smoke.postman_collection.json`.
- The tracked generated protobuf Go packages are `protos/*.pb.go`; the source
  `protos/artifacts` submodule is not required for compatibility execution.
- Recursive checkout is not viable in GitHub Actions because unrelated/private
  sibling submodules are inaccessible to `GITHUB_TOKEN` and no repository submodule
  credential exists.
- Local networking suites cannot bind the required ports in the reported restricted
  sandbox. Linux CI is authoritative for system execution.

## Scope Decisions

### Deployable services

The core stack contains the services already present in the root Compose release
topology:

- PostgreSQL;
- Redis;
- web-api;
- integration-api;
- endpoint-api;
- assistant-api;
- UI;
- nginx;
- a system-test runner.

`api/document-api` remains outside this RFC because it is not in the root Compose,
generic package, or primary release workflow. Adding it requires a separate
cross-language release RFC.

### Assistant packaging

Assistant-api remains OCI-image-only in this RFC. It is not added to
`.github/workflows/package.yml` or the generic CGO-disabled packaging action.

A standalone assistant distribution requires a separately reviewed bundle contract
covering the binary, shared libraries, model files, architecture, licenses, loader
paths, checksums, and installation layout.

### Smoke execution

The existing assistant Newman collection is one explicit named step in the
stack-owning system job. A shared smoke matrix is not introduced until a second
service has a concrete collection and owner.

Future service smoke collections continue to use:

```text
openapi/postman/<service>/<service>.smoke.postman_collection.json
```

When a second smoke suite is proposed, its RFC must choose either sequential steps in
the same stack-owning job or a separate job that creates and cleans its own stack.
Jobs must never assume access to containers created on another GitHub-hosted runner.

### Compatibility baseline

For pull requests, the primary compatibility baseline is the pull request target
branch SHA provided by GitHub. CI checks out tracked repository content without
submodules and records the target SHA plus the SHA-256 values of the generated current
and baseline protobuf descriptor images.

For protected-branch pushes, the baseline is the first parent of the pushed commit.
Tag-build compatibility behavior is outside this narrowed RFC; the reusable
compatibility workflow is not invoked by tag-only workflows.

Previous-release database upgrades are not part of RFC 0002.

## Goals

1. Give assistant-api Linux-native package, vet, unit, race, and build coverage.
2. Keep generic CGO-disabled Go jobs valid for the services they currently own.
3. Exercise the authoritative root Compose stack through a hermetic CI override with
   no developer-state dependency.
4. Verify fresh migrations reach clean head versions for all four Go APIs.
5. Verify semantic readiness and liveness for all four Go APIs.
6. Verify UI static assets are served through nginx.
7. Execute the existing assistant OpenAPI smoke flow against the production image.
8. Detect breaking changes reachable from the existing assistant/talk OpenAPI roots
   and from deterministic descriptors generated from tracked protobuf Go packages.
9. Fail safely and publish sanitized, actionable diagnostics.
10. Establish measured runtime and flake thresholds before expanding required gates.

## Non-Goals

- Standalone assistant binary packaging.
- Previous-release database upgrade testing.
- UI-to-web-to-integration/endpoint/assistant business-flow coverage.
- Inbound or outbound voice-call lifecycle tests.
- Provider timeout, malformed callback, dependency outage, or cancellation suites.
- Restart-during-call or graceful call-drain behavior changes.
- OpenSearch-backed behavior.
- Generalized multi-service smoke matrices.
- Changing readiness HTTP status semantics.
- Changing readiness response schema, connector naming, or API source behavior.
- Proving goroutine cleanup from outside a process.
- Adding document-api to the release topology.

The excluded reliability suites are tracked as follow-up RFC work, not treated as
implicitly passing because local execution is restricted.

## Acceptance Criteria

### Assistant native CI

1. A dedicated `assistant-native` job runs on `ubuntu-24.04` with a 25-minute hard
   timeout.
2. The job uses a repository-owned assistant CI image/target derived from the same
   native dependency contract as `docker/assistant-api/Dockerfile`.
3. The native environment runs, in order:
   - `go list -deps -test ./api/assistant-api/... ./cmd/assistant/...`;
   - `go vet ./api/assistant-api/... ./cmd/assistant/...`;
   - `go test -count=1 -covermode=atomic` for the same paths;
   - `go test -race -count=1` for the same paths;
   - pinned `govulncheck` for the same paths;
   - `go build -trimpath` for `./cmd/assistant/assistant.go`.
4. All commands run with `CGO_ENABLED=1`, `GOOS=linux`, `GOARCH=amd64`, and the
   explicit compiler, linker, and runtime library paths required by the assistant
   native dependency lock.
5. Packages with existing opt-in `integration`, `sipintegration`, or `freeswitch`
   build tags remain excluded from this default lane.
6. The generic `GO_CI_PACKAGES` and generic CGO-disabled build/package matrices are
   not modified to pretend assistant compatibility.
7. The production-equivalent assistant image remains covered by Docker CI.

### Native dependency lock

8. The pinned builder image digest is the atomic source of ONNX Runtime, Azure Speech
   SDK, RNNoise, Opus, compiler, and base operating-system packages. Native CI and
   the assistant production build use the same digest.
9. A base-image update requires rebuilding
   `docker/base/rapida-golang-bookworm.Dockerfile`, recording its resolved input
   versions/checksums in the base-image publication PR, publishing one immutable
   digest, and updating the assistant digest only after the native lane passes.
10. Repository-owned child dependency metadata records, at minimum:
   - builder and runtime image digests;
   - the exact Python `onnx` package version and hash;
   - apt snapshot/repository identity for child-stage packages;
   - TEN VAD library architecture and checksum;
   - LiveKit model revision and SHA-256 values;
   - Pipecat model immutable revision and SHA-256;
   - license/provenance references.
11. Docker and native CI consume the same child lock/checksum script. Mutable `main`
    downloads and unchecked child assets are prohibited.
12. A builder digest or child-lock change invalidates the shared assistant BuildKit
    cache scope.

### Core Linux system job

13. One `system-core` job owns stack creation, all assertions, diagnostics, and
    cleanup on a single runner. No other job depends on its live containers.
14. The job runs on `ubuntu-24.04` with a 30-minute hard timeout. Its main test driver
    has a 20-minute deadline, reserving up to five minutes for `if: always()`
    diagnostics and cleanup. The remaining time covers checkout, step transitions,
    and artifact upload without consuming the diagnostics and cleanup reserve.
15. Every stack lifecycle or test command applies
    `-f docker-compose.yml -f docker-compose.ci.yml`; the contract validator also
    renders `docker-compose.yml` alone for base-versus-merged comparison.
    `docker-compose.yml` and the existing `docker/**` Dockerfiles and runtime configs
    remain authoritative for PostgreSQL, Redis, the four Go APIs, UI, and nginx; the
    override adds only test orchestration and CI-specific behavioral overrides.
16. CI pins Docker Compose exactly to `v2.24.4`, the minimum version permitted by
    this RFC because the override contract uses both `!reset` and `!override` merge
    tags. The workflow verifies `docker compose version --short` equals `2.24.4`
    before parsing either file; newer or distribution-provided versions require an
    explicit lock update and the same rendered-model tests.
17. The rendered CI model uses a unique `COMPOSE_PROJECT_NAME` and has no fixed
    `container_name`, `${HOME}` data mount, host-network mode, privileged container,
    Docker socket mount, or fixed published port. For each inherited service,
    `docker-compose.ci.yml` sets `container_name: !reset null` and
    `ports: !reset []`. Volume handling is exact:
    - PostgreSQL replaces only `/var/lib/postgresql/data` with the project-scoped
      `postgres-data` volume and preserves the existing
      `docker/postgres/init.sql` mount;
    - Redis uses `volumes: !override []` and `tmpfs: [/data]`, so the rendered model
      cannot contain both a volume and tmpfs at `/data`;
    - nginx, web-api, integration-api, endpoint-api, and assistant-api replace only
      the `${HOME}`-backed `/app/rapida-data/assets` target with the project-scoped
      `system-assets` volume and preserve every other inherited mount;
    - UI has no data-volume override.
    No other base-service field is duplicated in the override.
18. Mutable data uses project-scoped named volumes or tmpfs and is removed by
    unconditional cleanup.
19. The system job builds each required image exactly once using a dedicated named
    Buildx builder and cache scope. Subsequent phases reuse those local image tags and
    run Compose with `--no-build`.

### Fresh migrations

20. A migration phase starts from empty service databases, runs each service's
    current migrations through an explicit migration command, and fails on any error
    or dirty state.
21. Long-running API containers start with `-skip-migration` after the migration
    phase, preserving one migration owner per database.
22. Before application teardown, the runner records each API's service name,
    migration version, expected repository-head version, and dirty boolean. It fails
    unless every dirty value is false and every version equals repository head for
    web, integration, endpoint, and assistant; the sanitized version-plus-dirty
    record remains available in failure diagnostics.

### Health semantics

23. Liveness passes only when HTTP is 200 and the JSON response satisfies:
    `code == 200`, `success == true`, and `data.healthy == true`.
24. Readiness passes only when HTTP is 200 and the JSON response satisfies:
    `code == 200`, `success == true`, and
    `data["PSQL psql://postgres:5432"] == true`. This is the explicit connector name
    produced by the unchanged CI PostgreSQL host and port; no arbitrary-true map
    entry or fallback key is accepted.
25. Web, integration, endpoint, and assistant each receive an independent 60-second
    readiness timeout and retry context, started when that service's probe begins;
    one slow service cannot consume another service's budget. The last sanitized
    response body is retained on failure.
26. Package tests for the response validator include a positive fixture and negative
    fixtures for non-200 HTTP status, malformed JSON, `code != 200`,
    `success != true`, missing/non-boolean/false
    `data["PSQL psql://postgres:5432"]`, a different true key, and an unrelated true
    map entry. Every negative fixture must fail closed without scanning for any
    arbitrary true value.
27. Readiness HTTP status behavior is not changed by this RFC. If a future RFC
    changes it, every affected API package requires tests. RFC 0002 changes no API
    readiness source; it adapts only the CI predicate to the verified existing
    payload.

The readiness key is intentionally topology-specific. If the PostgreSQL Compose
service name or port changes, the rendered-model contract and readiness fixtures must
change together through review; silently accepting some other true dependency would
turn a PostgreSQL readiness failure into a false pass.

### UI/nginx and assistant smoke

28. Through nginx, the UI service returns the SPA entry document at `/` and one
    content-hashed static asset referenced by that document.
29. The smallest shared production change removes the unused persistent `ui_build`
    mount and volume declaration from `docker-compose.yml` and changes the nginx
    catch-all `location /` in `docker/nginx/nginx.conf` to proxy the existing
    `ui:3000` service. Explicit locations preserve the existing non-gRPC web routes
    `/v1/`, `/oauth/`, `/readiness/`, and `/healthz/` by continuing to proxy them to
    `web-api:9001`; all existing specialized recording, assistant talk, WebSocket,
    and named API proxy locations remain unchanged. Runtime checks exercise those
    four web routes plus representative `/talk_api` and `/web_api` routes. No UI
    asset copy-up, persistent UI volume, or CI-only nginx config is created.
30. The system job runs
    `python3 openapi/scripts/generate_assistant_postman_collection.py --check`.
31. The job runs the checked-in assistant smoke collection with pinned Newman and
    `--folder "Smoke Flow" --bail` against `http://assistant-api:9007`.
32. One transient test-runner invocation seeds project-scoped authentication, writes
    `authToken`, `authId`, and `projectId` to a mode-0600 Newman environment JSON on
    the container's tmpfs, executes Newman using that file, and deletes it before
    exit. Generated credentials never enter host environment variables or command
    arguments.
33. All ten existing assistant/configuration/API-deployment smoke requests and their
    assertions pass.
34. The workflow runs the smoke invocation in its own GitHub Actions step named
    `Assistant smoke`; no migration, readiness, UI, diagnostics, or cleanup command
    shares that step. It remains in the stack-owning job and exposes distinct timing
    and outcome in the Actions UI.

### Contract compatibility

35. Contract scope is limited to:
    - `openapi/artifacts/assistant-api.yaml`;
    - `openapi/artifacts/talk-api.yaml`;
    - tracked `protos/*.pb.go` packages and their module inputs `go.mod` and `go.sum`.
    `common.yaml` is preserved in both directory trees so references resolve, but
    schemas not reachable from assistant-api or talk-api are not claimed as protected
    by RFC 0002.
36. The three YAML files are authoritative inputs; this RFC does not invent an
    OpenAPI generator. Pinned OpenAPI YAML parsing/linting and Postman drift checking
    run before comparison.
37. `oasdiff breaking` runs separately for assistant-api and talk-api using preserved
    baseline/current directory trees so relative `common.yaml` references resolve.
    Only root-reachable common schemas are covered; standalone common-schema coverage
    requires a follow-up contract RFC with an authoritative consumer or schema rule.
38. CI deterministically generates `FileDescriptorSet` images from the tracked current
    and target-base `protos/*.pb.go` packages, using the corresponding `go.mod` and
    `go.sum`, then runs pinned
    `buf breaking <current-image> --against <baseline-image>`. Descriptor generation
    and comparison fail closed. `buf lint` is not a gate because this lane detects
    compatibility from tracked generated packages rather than source-style debt.
39. `assistant-native`, `system-core`, and contract compatibility use
    `actions/checkout` without submodules. Compatibility uses `fetch-depth: 0` to
    materialize target-base tracked files.
40. Breaking-change allowlists require an owner, reason, migration link, and expiry.
    Compatibility jobs never use broad `continue-on-error`.

### Diagnostics and cleanup

41. The diagnostic collector writes only an allowlisted schema containing service
    name, timestamp, health summary, migration version, migration dirty state, exit
    code, and explicitly selected sanitized log lines.
42. Raw Compose environment, container inspect output, authorization headers,
    callback bodies, database passwords, Redis values, and full Newman environment
    are never staged as artifacts.
43. Before upload, a tested sanitizer scans the staging directory for configured
    secrets and token-shaped values. Any match deletes the staging directory and
    fails artifact publication closed.
    Tests inject sentinel database credentials, bearer/JWT-like tokens, and Newman
    values, capture stdout, stderr, `$GITHUB_OUTPUT`, and `$GITHUB_STEP_SUMMARY`, and
    fail if any sentinel or token-shaped value appears. They also assert the smoke
    environment file is never printed or staged and is removed by normal completion
    and the failure trap.
44. Before cleanup, including after a build failure, diagnostics capture sanitized
    `docker buildx inspect`, `docker buildx du`, `docker system df`, the exact cache
    scope, BuildKit metadata/image IDs, and the failing plain-progress BuildKit log.
    The metadata and log pass through the same fail-closed sanitizer before upload.
45. Cleanup steps use both a shell trap and workflow `if: always()` and run
    `docker compose down --volumes --remove-orphans` with a
    30-second timeout.
    The named Buildx builder is then removed directly, and cleanup fails if
    `docker buildx inspect <unique-builder>` still succeeds.
46. Cleanup fails if containers, networks, volumes, the named builder, or local
    BuildKit cache records owned by that disposable builder remain after ten
    one-second retries. The explicitly named reusable remote cache scope is retained
    by design and is invalidated by its dependency-lock key, not global pruning.
47. Package-local goroutine leak assertions are required only for production
    lifecycle code changed by this RFC. Container shutdown is not presented as proof
    of goroutine cleanup.

### Runtime graduation

48. Before `system-core` becomes required, it completes 20 consecutive protected-
    branch or explicitly dispatched runs with:
    - at least 19 successful runs;
    - no repeated infrastructure failure signature;
    - p95 runtime at or below 15 minutes;
    - maximum main-phase runtime below the 20-minute deadline.
49. `assistant-native` becomes required when it completes the same 20-run sample with
    p95 at or below 20 minutes and no more than one flaky failure.
50. During stabilization, failures are visible and owned but non-required for no more
    than 14 calendar days from first merge. On day 14 the owner either promotes the
    lane because thresholds passed or removes the workflow invocation. Extending
    stabilization requires revised accepted RFC bytes and a new confirmation gate.
51. Graduation is independent per lane:
    - if both pass, both join `ci-success` and graduated `system-core` replaces the
      primary-CI invocation of `reusable-docker-ci.yml`;
    - if only `assistant-native` passes, it joins `ci-success`, `system-core` is
      removed on day 14, and Docker CI remains;
    - if only `system-core` passes, it joins `ci-success`, replaces Docker CI, and
      `assistant-native` is removed on day 14;
    - if neither passes, both invocations are removed on day 14 and existing Docker
      CI remains.
    One lane's failure does not block the other's threshold-backed promotion.

## Design

## 1. Assistant Native Toolchain Lane

Add a dedicated reusable workflow or dedicated jobs in the existing reusable Go
workflow. The lane must not reuse the generic `GO_CI_PACKAGES` execution environment.
It checks out tracked repository content without submodules.

The implementation adds a named `ci` target to
`docker/assistant-api/Dockerfile`; it does not create a parallel assistant
Dockerfile. The target consumes the same pinned native dependency lock,
installs/downloads native assets once, then runs the five required Go commands plus
pinned `govulncheck`, for six commands total.

The production Docker build and native CI target share download/checksum logic. They
must not maintain independent version lists.

Assistant release packaging remains the Docker image produced by the release image
workflow. The generic tarball package workflow remains unchanged.

## 2. Stack-Owning Core System Job

Add `.github/workflows/reusable-system-ci.yml` with one `system-core` job. The job
performs these named steps on the same runner:

1. checkout tracked repository content without submodules;
2. install exact Docker Compose `v2.24.4` and pinned test tooling, then reject any
   other Compose version;
3. validate native-builder and service-image digest locks with their separate
   commands;
4. build required images once through pinned Docker Compose `v2.24.4` using the
   named builder while recording image metadata, cache scope, and plain-progress
   BuildKit logs;
5. render and statically validate the merged
   `docker-compose.yml` plus `docker-compose.ci.yml` model;
6. create empty databases and run migrations;
7. record version and dirty state for all four APIs;
8. start the APIs, UI, and nginx with `--no-build`;
9. assert semantic liveness and readiness with an independent timeout per service;
10. assert UI/nginx static serving;
11. run assistant OpenAPI collection drift check;
12. run an Actions step named `Assistant smoke` containing only the isolated smoke
   command that seeds authentication, creates a
   tmpfs Newman environment, executes Newman, and destroys the environment;
13. collect sanitized service and BuildKit diagnostics when needed;
14. unconditionally tear down and assert project, volume, cache, and builder removal.

The job exposes one independently reported result eligible to become required after
graduation. Step summaries identify which phase failed.

## 3. CI Compose Contract

`docker-compose.yml` is the single authoritative service topology. CI always applies
`docker-compose.ci.yml` as a thin second file with
`docker compose -f docker-compose.yml -f docker-compose.ci.yml`; the override must not
copy the base service definitions, build contexts, Dockerfile selections, runtime
configuration mounts, dependency graph, networks, or healthchecks.

The merge and build contract uses Docker Compose `v2.24.4` exactly. `!reset` clears inherited
scalar/list values and `!override` replaces a list without normal append/unique-key
merging. The repository pins the binary and checksum in `tests/system/tools.lock`;
the system workflow does not rely on the runner's preinstalled Compose plugin.
Pinned Compose is the only component that parses the two Compose files for model
validation or image builds.

The override is limited to:

- applying `container_name: !reset null` and `ports: !reset []` to PostgreSQL,
  Redis, nginx, UI, web-api, integration-api, endpoint-api, and assistant-api;
- replacing the PostgreSQL `/var/lib/postgresql/data` target with
  `postgres-data:/var/lib/postgresql/data` while retaining
  `./docker/postgres/init.sql:/docker-entrypoint-initdb.d/init.sql`;
- applying `volumes: !override []` to Redis and `tmpfs: [/data]`, never both storage
  types at `/data`;
- replacing only the `/app/rapida-data/assets` target on nginx and the four APIs with
  `system-assets:/app/rapida-data/assets`, while retaining nginx config/SSL/UI and
  API config/go-module mounts;
- overriding API commands or migration behavior only as required to establish one
  explicit migration owner per database and start long-running APIs with
  `-skip-migration`;
- adding only `build.cache_from` and `build.cache_to` entries for inherited build
  services, using the lock-derived `SYSTEM_CACHE_SCOPE` without repeating or changing
  their context, Dockerfile, or build args;
- adding transient migration and test-runner orchestration used only by CI.

The override may also add only project-scoped `postgres-data`, `system-assets`, and
`system-reports` volumes needed by those rules. Migration version/dirty reports are
written to `system-reports` before teardown; no report volume survives cleanup.

The merged model reuses `docker/postgres/init.sql` and
`docker/nginx/nginx.conf`. It does not introduce CI copies of those files or parallel
service Dockerfiles/configuration under `docker/ci/**`. CI-only runner Dockerfiles,
tool locks, and executable wrappers live under `tests/system/**` or
`.github/actions/system-test/**`.

The shared production UI wiring is intentionally small and independently owned:
`docker-compose.yml` removes the unused `ui_build` mount from nginx and removes the
top-level `ui_build` volume declaration, while `docker/nginx/nginx.conf` changes the
catch-all `location /` to `proxy_pass http://ui:3000` and adds explicit locations for
`/v1/`, `/oauth/`, `/readiness/`, and `/healthz/` that continue proxying to
`web-api:9001`. Existing recording-asset, assistant talk, WebSocket, and other named
API proxy locations remain unchanged. The nginx test must fetch `/`, discover a
content-hashed asset reference in the returned entry document, fetch that asset
through nginx, and prove `/v1/`, `/oauth/`, `/readiness/`, `/healthz/`, representative
`/talk_api`, and `/web_api` routes still reach their existing upstreams.

The Compose contract validator renders base and merged JSON and permits only these
CI deltas: removal of `container_name` and `ports`, the exact storage substitutions
listed above, approved API command/migration overrides, lock-derived build cache
entries, project-scoped volume/tmpfs declarations, and addition of
migration/test-runner services. For every inherited service it fails unless build
context, Dockerfile, non-cache build args, image repository/tag or digest, read-only
repository config mounts, dependency edges, healthchecks, and network membership
equal the authoritative base model. It also fails if the override duplicates any
other base-service key, if any rendered path references `docker/ci`, or if a
`docker/ci` directory exists.

Services communicate by Compose DNS. All network-dependent migration assertions,
health checks, UI checks, fixture seeding, and Newman execution run inside the
Compose `test-runner` container. Host-side systemcheck commands are limited to Docker
project diagnostics, artifact sanitation, and cleanup. No host port is required. The
Linux CI runner remains authoritative even though the no-published-port design also
reduces local collisions.

The stack initially excludes OpenSearch, FreeSWITCH, and generalized provider mocks.
Those services enter only through follow-up profile RFCs with measured runner costs.

## 4. Migration Ownership

CI uses a pinned `golang-migrate` runner image against the repository SQL directories
and per-service PostgreSQL DSNs before starting long-running containers. It must not
start four competing migration owners.

Production command changes are not authorized. If the pinned migration-runner design
is shown incompatible with an existing migration directory, implementation returns
to RFC discussion rather than adding flags to four service commands.

## 5. Tool and Supply-Chain Pinning

The implementation pins:

- GitHub actions by full commit SHA;
- assistant builder/runtime images by digest;
- PostgreSQL, Redis, nginx, and test-runner images by digest, updating the
  authoritative `docker-compose.yml` references where the shared services are
  defined;
- Newman in a repository lockfile, invoked without `npx --yes`; the lock may use
  explicit overrides for vulnerable transitive packages to patched versions without
  changing the pinned Newman major/version contract. The currently required override
  is `handlebars=4.7.9`, and dependency-review must pass;
- Buf and oasdiff versions and checksums;
- native models/source archives through the native dependency lock.

The builder image digest is the authority for native libraries installed in
`docker/base/rapida-golang-bookworm.Dockerfile`; the child lock is authoritative only
for assets installed by the assistant Dockerfile.

Native builder validation and Compose service-image validation are separate
contracts. `bin/check-go-version-consistency` validates the Go version plus the
digest-qualified assistant builder reference, and the assistant native lock verifier
validates its child dependencies. A separate service-image lock validates the
PostgreSQL, Redis, nginx, migration-runner, and test-runner image repository,
declared tag/major track, digest, and platform used by the authoritative Compose
model. A digest refresh may not incidentally change `postgres:15`, `redis:7`, the
existing nginx Alpine track, or any other image's declared major/base track; such a
change is out of scope and requires an explicit dependency change with owner and
rollback evidence.

Tool versions are updated by explicit dependency changes, not downloaded from a
mutable channel during every run.

The workflow does not download or independently pin a second Buildx binary and does
not use `docker buildx bake` to parse Compose. The Docker installation's existing
`docker buildx` command is used only to create, select, inspect, measure, and remove
the named builder; pinned Docker Compose `v2.24.4` owns both merged-model rendering
and the single image build.

## 6. Compatibility Checks

OpenAPI comparison uses the assistant-api and talk-api roots plus their checked-in
common dependency tree. Web, integration, endpoint, and standalone common-schema
compatibility are not invented or claimed in this RFC.

For a pull request, CI materializes the complete target-branch `openapi/artifacts`
tree into a temporary directory using the exact target SHA. It runs separate
assistant-api and talk-api comparisons so relative `common.yaml` references resolve.
For protobuf, CI materializes target-base `go.mod`, `go.sum`, and tracked
`protos/*.go` beside the corresponding current files, then deterministically generates
current and baseline `FileDescriptorSet` images from those Go packages. Pinned
`buf breaking <current-image> --against <baseline-image>` compares the images and
fails closed if either descriptor generation or comparison fails.

The protobuf risk is explicitly bounded: RFC 0002 pins Buf and runs `buf breaking`
against deterministic descriptor images generated from tracked Go packages. It does
not require source-module checkout or claim source lint coverage; source lint cleanup
remains separately owned follow-up work.

The target SHA and SHA-256 values of both current and baseline protobuf descriptor
images are written to the job summary.

## Ownership and Allowed Paths

### Native CI owner

Writable paths:

- `.github/workflows/reusable-go-ci.yml` or a new
  `.github/workflows/reusable-assistant-native-ci.yml`;
- `docker/assistant-api/Dockerfile`;
- `docker/assistant-api/native-deps.lock`;
- `docker/assistant-api/scripts/**`;
- `bin/check-go-version-consistency` for the shared builder digest/version contract.

The owner does not edit `.github/workflows/package.yml`.

### System workflow owner

Writable paths:

- `.github/workflows/ci.yml`;
- `.github/workflows/reusable-system-ci.yml`;
- `.github/actions/system-test/**`.

This owner writes all workflow steps, including compatibility invocation. Contract
owners provide commands and review but do not edit workflow files.

### Shared stack and service-image owner

Writable paths:

- `docker-compose.yml`;
- `docker/nginx/nginx.conf`;
- `tests/system/service-images.lock`;
- `tests/system/bin/verify-service-image-digests`.

This owner makes only the removal of the unused root `ui_build` mount/volume, the
nginx catch-all proxy to the existing UI service, the explicit web-api locations for
`/v1/`, `/oauth/`, `/readiness/`, and `/healthz/`, and digest-qualified service-image
changes specified in this RFC. It preserves every other named API and WebSocket proxy
location and forbids incidental image major/base-track upgrades. Native builder
digest/version validation remains exclusively with the Native CI owner.

### CI Compose owner

Writable paths:

- `docker-compose.ci.yml`;
- `tests/system/test-runner/**`;
- `tests/system/bin/buf`;
- `tests/system/bin/oasdiff`;
- `tests/system/tools.lock`.

This owner must preserve the existing `docker/**` Dockerfiles and configuration as
the runtime authority, including `docker/postgres/init.sql` and
`docker/nginx/nginx.conf`; it does not edit those shared files or create
`docker/ci/**`. The owner also owns the pinned test-runner package lock and named
Buildx builder lifecycle. If workflow-local wrappers are preferred instead, the
system workflow owner places them under `.github/actions/system-test/**` and the
CI Compose owner does not create duplicate copies.

### System assertion owner

Writable paths:

- `tests/system/**`, excluding `tests/system/fixtures/assistant/**`,
  `tests/system/test-runner/**`, `tests/system/bin/**`, and
  `tests/system/tools.lock` and `tests/system/service-images.lock`.

This owner writes migration, health, UI/nginx, sanitizer, diagnostics, and cleanup
assertions. It does not edit `api/assistant-api/sip/**`.

### Assistant smoke owner

Writable paths:

- `openapi/postman/assistant-api/**`;
- `openapi/scripts/generate_assistant_postman_collection.py`;
- assistant-specific seed fixtures under `tests/system/fixtures/assistant/**`.

### Contract owner

Writable paths:

- `openapi/artifacts/assistant-api.yaml`;
- `openapi/artifacts/talk-api.yaml`;
- `openapi/artifacts/common.yaml`;
- `buf.yaml`;
- contract comparison scripts under `scripts/contracts/**`.

Baseline materialization reads tracked `go.mod`, `go.sum`, and `protos/*.go` but does
not authorize changes to them solely to satisfy CI. No production `cmd/**`, `api/**`,
or `pkg/**` paths are authorized by this RFC.

## Required Commands

System support code is implemented as a Go package and CLI under `tests/system/**`.
The implementation provides these executable command contracts:

```bash
# Workflow and Compose version contract
actionlint
test "$(docker compose version --short)" = "2.24.4"

# Native builder digest/version contract
./bin/check-go-version-consistency
./docker/assistant-api/scripts/verify-native-deps.sh docker/assistant-api/native-deps.lock

# Compose service-image digest contract
./tests/system/bin/verify-service-image-digests \
  --compose docker-compose.yml \
  --lock tests/system/service-images.lock \
  --baseline /tmp/baseline/docker-compose.yml \
  --forbid-major-change

# Rendered Compose allowlist and system-support tests
docker compose -f docker-compose.yml \
  config --format json > /tmp/base-compose.json
docker compose -f docker-compose.yml -f docker-compose.ci.yml \
  config --format json > /tmp/system-compose.json
go run ./tests/system/cmd/systemcheck compose-contract \
  --base-rendered /tmp/base-compose.json --override docker-compose.ci.yml \
  --merged-rendered /tmp/system-compose.json --compose-version 2.24.4 \
  --forbid-path docker/ci
go test -count=1 ./tests/system/...
go test -count=1 ./tests/system/... \
  -run 'Test(ReadinessValidator|SecretNonEmission|ComposeContract)'

# Assistant native lane, inside the pinned native image
go list -deps -test ./api/assistant-api/... ./cmd/assistant/... >/dev/null
go vet ./api/assistant-api/... ./cmd/assistant/...
go test -count=1 -covermode=atomic ./api/assistant-api/... ./cmd/assistant/...
go test -race -count=1 ./api/assistant-api/... ./cmd/assistant/...
govulncheck ./api/assistant-api/... ./cmd/assistant/...
go build -trimpath -o /out/assistant-api ./cmd/assistant/assistant.go

# Existing assistant smoke generation
python3 openapi/scripts/generate_assistant_postman_collection.py --check

# Contracts
go run ./tests/system/cmd/systemcheck openapi-parse openapi/artifacts
scripts/contracts/materialize-baseline.sh /tmp/baseline
scripts/contracts/generate-protobuf-descriptor.sh --module-root . --output /tmp/current-protos.binpb
scripts/contracts/generate-protobuf-descriptor.sh --module-root /tmp/baseline --output /tmp/baseline-protos.binpb
./tests/system/bin/buf breaking /tmp/current-protos.binpb --against /tmp/baseline-protos.binpb
sha256sum /tmp/current-protos.binpb /tmp/baseline-protos.binpb
./tests/system/bin/oasdiff breaking /tmp/baseline/openapi/artifacts/assistant-api.yaml openapi/artifacts/assistant-api.yaml
./tests/system/bin/oasdiff breaking /tmp/baseline/openapi/artifacts/talk-api.yaml openapi/artifacts/talk-api.yaml

# One stack-owning system run
docker buildx create --name <unique-builder> --driver docker-container --use
docker buildx inspect <unique-builder> --bootstrap
set -o pipefail
COMPOSE_PROJECT_NAME=<unique> SYSTEM_CACHE_SCOPE=<cache-scope> \
  BUILDX_BUILDER=<unique-builder> \
  docker compose -f docker-compose.yml -f docker-compose.ci.yml \
  build --progress plain \
  2>&1 | tee <staging-directory>/buildkit.log
COMPOSE_PROJECT_NAME=<unique> docker compose -f docker-compose.yml -f docker-compose.ci.yml \
  images --format json > <staging-directory>/compose-images.json
go run ./tests/system/cmd/systemcheck build-metadata \
  --compose-images <staging-directory>/compose-images.json \
  --output <staging-directory>/buildkit-metadata.json
COMPOSE_PROJECT_NAME=<unique> docker compose -f docker-compose.yml -f docker-compose.ci.yml run --rm migrate-web up
COMPOSE_PROJECT_NAME=<unique> docker compose -f docker-compose.yml -f docker-compose.ci.yml run --rm migrate-integration up
COMPOSE_PROJECT_NAME=<unique> docker compose -f docker-compose.yml -f docker-compose.ci.yml run --rm migrate-endpoint up
COMPOSE_PROJECT_NAME=<unique> docker compose -f docker-compose.yml -f docker-compose.ci.yml run --rm migrate-assistant up
COMPOSE_PROJECT_NAME=<unique> docker compose -f docker-compose.yml -f docker-compose.ci.yml run --rm test-runner \
  systemcheck migrations --require-clean --require-head \
  --report /reports/migrations.json
COMPOSE_PROJECT_NAME=<unique> docker compose -f docker-compose.yml -f docker-compose.ci.yml up \
  -d --no-build postgres redis web-api integration-api endpoint-api assistant-api ui nginx
COMPOSE_PROJECT_NAME=<unique> docker compose -f docker-compose.yml -f docker-compose.ci.yml run --rm test-runner \
  systemcheck health --timeout-per-service 60s --interval 1s \
  --readiness-key 'PSQL psql://postgres:5432' \
  --reject-arbitrary-true-fallback
COMPOSE_PROJECT_NAME=<unique> docker compose -f docker-compose.yml -f docker-compose.ci.yml run --rm test-runner \
  systemcheck ui-nginx --base-url http://nginx:8080 \
  --require-spa-root --require-hashed-asset \
  --http-route /v1/__systemcheck__=web-api:9001 \
  --http-route /oauth/__systemcheck__=web-api:9001 \
  --http-route /readiness/=web-api:9001 \
  --http-route /healthz/=web-api:9001 \
  --proxy-route /talk_api.TalkService/GetAllAssistantConversation=assistant-api:9007 \
  --proxy-route /web_api.AuthenticationService/ForgotPassword=web-api:9001
# GitHub Actions step name: Assistant smoke
COMPOSE_PROJECT_NAME=<unique> docker compose -f docker-compose.yml -f docker-compose.ci.yml run --rm test-runner \
  systemcheck assistant-smoke \
  --collection openapi/postman/assistant-api/assistant-api.smoke.postman_collection.json \
  --base-url http://assistant-api:9007 --tmpfs /run/secrets
# Workflow `if: always()` diagnostics and cleanup
docker buildx inspect <unique-builder> > <staging-directory>/buildx-inspect.txt
docker buildx du --builder <unique-builder> > <staging-directory>/buildx-du.txt
docker system df > <staging-directory>/docker-system-df.txt
go run ./tests/system/cmd/systemcheck collect-diagnostics --compose-project <unique> --directory <staging-directory>
go run ./tests/system/cmd/systemcheck sanitize-artifacts --directory <staging-directory>
COMPOSE_PROJECT_NAME=<unique> docker compose -f docker-compose.yml -f docker-compose.ci.yml down \
  --volumes --remove-orphans --timeout 30
go run ./tests/system/cmd/systemcheck cleanup --compose-project <unique> --retries 10 --interval 1s
docker buildx rm <unique-builder>
! docker buildx inspect <unique-builder>
! docker buildx du --builder <unique-builder>
```

The rendered-model validator compares the base and merged JSON models against the
normative allowlist in the CI Compose Contract. It rejects ports, fixed
`container_name`, host-persistent mounts, Redis `/data` volume/tmpfs conflicts,
unapproved inherited-field changes, duplicated topology, any `docker/ci` reference,
or an on-disk `docker/ci` directory. It also verifies the inherited PostgreSQL and
nginx mounts still resolve to `docker/postgres/init.sql` and
`docker/nginx/nginx.conf` and executes the containerized health probe against
Compose DNS.

`systemcheck assistant-smoke` creates `/run/secrets/smoke.postman_environment.json`
with mode `0600`, invokes the lockfile-installed Newman binary with `--environment`
and `--bail`, and removes the file through both normal return and a deferred failure
handler. The file lives only on the transient container tmpfs and is never copied to
the host or artifact staging directory.

## Diagnostics

Diagnostics are allowlisted JSON and text records generated by
`tests/system/diagnostics`. The collector has unit tests covering:

- known database-password redaction;
- bearer/JWT-like token redaction;
- Newman environment redaction;
- non-emission of sentinels through captured stdout, stderr, `$GITHUB_OUTPUT`, and
  `$GITHUB_STEP_SUMMARY`;
- migration version and dirty-state reporting for all four APIs;
- sanitized BuildKit inspect, disk-usage, cache-scope, metadata, and failure-log
  records;
- refusal to stage raw inspect/environment payloads;
- fail-closed deletion when a secret scanner matches.

On failure, retained fields are limited to service, phase, timestamp, exit code,
migration version, migration dirty state, semantic health booleans, image ID,
builder/cache identifiers, disk-usage totals, and selected redacted log messages.

## Runtime, Cost, and Flake Control

- Docker images are built once per stack-owning job.
- The job uses a dedicated named Buildx builder and a lock-derived remote cache scope
  covering the assistant native lock and service-image lock. Pinned Compose performs
  the build and emits plain-progress BuildKit logs; Compose image output plus Docker
  image inspection supplies the image metadata. The job records builder inspect,
  BuildKit disk usage, cache scope, image metadata, failure logs, and
  `docker system df`; it removes the disposable builder directly and verifies its
  local cache disappears. It does not run global `docker system prune`, delete the
  reusable remote cache, download a separate Buildx binary, or ask Buildx to parse
  the Compose files.
- Core and native lanes have explicit hard timeouts and p95 graduation thresholds.
- Expanded suites do not become required through this RFC.
- A failure signature repeated twice during stabilization has one named owner and a
  48-hour investigation deadline.
- The 30-minute hard runner timeout may forfeit diagnostics if GitHub terminates the
  host. The 20-minute main deadline and up-to-five-minute diagnostics and cleanup
  reserve minimize this, while the remaining time covers checkout, step transitions,
  and artifact upload; the RFC does not claim cleanup after forced runner destruction.

## Rollback

Changes are delivered as independently revertible commits:

1. child native dependency lock, native CI target, and native builder
   version/digest checker;
2. assistant native workflow wiring;
3. service-image digest lock and digest-only root Compose image references;
4. removal of the unused `ui_build` volume plus nginx proxying to the existing UI
   service, limited to `docker-compose.yml` and `docker/nginx/nginx.conf`;
5. thin CI Compose override and system assertions;
6. system workflow wiring;
7. compatibility scripts and workflow wiring.

Disabling or reverting `system-core` also reverts commits 3 and 4 unless their shared
developer behavior has been separately promoted by an accepted change. The root
Compose/nginx commit is atomic and independently revertible: reverting it restores
the previous root web-api catch-all and `ui_build` nginx mount/volume without touching
the native lane, CI override, contract checks, or compatibility workflow. Reverting
service-image pinning restores the prior image references without changing Compose
topology; it must never substitute an unreviewed major image upgrade.

Platform on-call may temporarily disable a newly required native or core system gate
only for a confirmed CI infrastructure incident. The disablement must link an issue,
name an owner, expire within 48 hours, and state the restoration condition. Product
test failures are not infrastructure incidents.

No assistant drain/admission or service-command behavior is changed, so no runtime
feature flag is required by this RFC.

## Follow-Up RFCs

After RFC 0002 is stable, the coordinator reserves separate RFCs for:

1. previous-release migration upgrades with deterministic stable-tag resolution;
2. UI/web proxy paths through integration-api, endpoint-api, and assistant-api;
3. mocked provider timeout, outage, malformed callback, and cancellation behavior;
4. FreeSWITCH inbound/outbound voice lifecycles;
5. active-call restart and graceful draining;
6. optional OpenSearch system coverage;
7. multi-service smoke orchestration once a second collection exists;
8. protobuf lint-debt cleanup, followed by a separate decision to make pinned
   `buf lint` required only after the current module passes cleanly.

Local restricted-sandbox failures do not waive these follow-ups. Their authoritative
execution environment remains Linux CI.

## Challenge Resolution Record

The first independent challenge blocked draft SHA-256
`122bb964b884dc9d73f88a0569a9305a67fdf97367b162a8f06b0afa4e0bb46f`.

The second independent challenge required revision of draft SHA-256
`8d52fd65892bc49221ae31c3cfc3cd0086d61fbabec067bfd15158b3dc44eeca`.

The third independent challenge required revision of draft SHA-256
`818e4a26d64eff21b16acb22e7c52eefbb23ee6142334716face19b19fea35a0`.

The fourth independent challenge required revision of draft SHA-256
`d2a060f54b982e287e9d3254b492cfb898ad6c5a5284c0af1f4f7b10b461c5ac`.

This revision addresses the challenge by:

- replacing impossible CGO-disabled assistant jobs with a dedicated native lane;
- retaining OCI-only assistant packaging;
- using one stack-owning job instead of cross-runner Compose sharing;
- choosing the target branch as the primary compatibility baseline;
- removing normative open questions;
- correcting migration-failure behavior;
- limiting OpenAPI scope to existing artifacts;
- requiring pinned tools, images, native sources, and checksums;
- asserting readiness JSON rather than HTTP status alone;
- assigning disjoint file ownership;
- adding runtime/flake budgets and image reuse;
- defining allowlisted, tested, fail-closed diagnostics;
- making cleanup predicates and deadlines explicit;
- defining rollback authority and expiry;
- replacing the premature smoke matrix with one explicit assistant step.

The second revision additionally:

- treats the pinned builder digest as the atomic authority for base native libraries;
- limits the child lock to assistant-stage dependencies and checksums;
- replaces nonexistent OpenAPI generation with authoritative YAML parsing, Postman
  drift checking, and per-root oasdiff commands;
- removes duplicate Docker CI invocation after measured graduation;
- removes ownership overlap for assistant seed fixtures;
- defines automatic rollback when stabilization misses day-14 thresholds;
- removes undefined tag compatibility behavior;
- reserves cleanup time and uses a dedicated removable Buildx builder;
- specifies executable system-support, migration, health, smoke, sanitizer, and
  cleanup commands;
- makes the pinned migration-runner design mandatory;
- adds assistant `govulncheck` to the native lane.

The third revision additionally:

- limits OpenAPI compatibility claims to schemas reachable from assistant-api and
  talk-api roots;
- runs every network-dependent assertion inside the no-host-port Compose network;
- defines independent native/core graduation and Docker-CI replacement outcomes;
- corrects the native lane command count after adding `govulncheck`.

The fourth revision additionally makes credential seeding and Newman execution one
transient test-runner operation using a mode-0600 tmpfs environment file with
normal-path and failure-path deletion.

This fifth revision additionally:

- makes `docker-compose.yml` and the existing `docker/**` Dockerfiles and configs the
  authoritative topology exercised by CI;
- limits `docker-compose.ci.yml` to a thin override that removes host-specific state,
  adjusts migration ownership, and adds test orchestration;
- reuses `docker/postgres/init.sql` and `docker/nginx/nginx.conf` instead of creating
  CI copies;
- places CI-only runners, tool locks, and wrappers under `tests/system/**` or
  `.github/actions/system-test/**`, with no `docker/ci/**` ownership;
- authorizes compatible `docker-compose.yml` and
  `bin/check-go-version-consistency` updates required by shared setup and digest
  pinning;
- preserves the native lane, contract checks, deferred scope, stabilization policy,
  and system-test assertions while returning the RFC to Draft for renewed challenge
  and confirmation.

The sixth independent challenge required revision of draft SHA-256
`4ec808cd1f4d62f3183c8e3d43c24a0a2020ced7359c7c7191af68d3c34c7764`.

This sixth revision additionally:

- pins Docker Compose `v2.24.4` and defines exact `!reset`, `!override`, and
  same-target volume behavior for every inherited service;
- authorizes the smallest shared root Compose and nginx changes needed to serve the
  built UI while preserving named API proxy routes;
- adds a fail-closed base-versus-merged allowlist validator, including absence of
  duplicated topology and `docker/ci`;
- separates native builder and Compose service-image digest ownership and forbids
  incidental major/base-track upgrades;
- adds strict readiness positive/negative tests, independent service timeouts, a
  distinct Actions smoke step, migration version/dirty evidence, and secret
  non-emission coverage for every workflow output channel;
- records BuildKit metadata, cache, disk usage, failure logs, and named-builder
  cleanup; and
- makes service-image and shared root Compose/nginx changes independently
  revertible from the system lane.

The seventh independent challenge required revision of draft SHA-256
`3db276fea544f44b4ae2361b3458e7a2b89bc171516d5e19e31216c9a05710cd`.

This seventh revision additionally:

- replaces persistent `ui_build` copy-up with nginx proxying the existing UI service
  for both the SPA entry document and its content-hashed assets while preserving API
  and WebSocket locations; and
- makes pinned Docker Compose `v2.24.4` the sole parser for model rendering and image
  builds, retaining only named-builder lifecycle and diagnostics through the
  existing Docker Buildx command and requiring no separately downloaded Buildx
  binary.

The eighth revision returns accepted SHA-256
`e4b13f009ba20085c08998e74bf54fe37ede82e6c9007d3796551b4353ac7391` to Draft and
additionally:

- matches the unchanged readiness payload's exact CI connector key,
  `data["PSQL psql://postgres:5432"]`, with positive and fail-closed negative tests
  and no arbitrary-true fallback; and
- retains pinned `buf breaking` compatibility while deferring unrelated protobuf
  source lint work to explicitly tracked follow-up debt.

The ninth revision returns accepted SHA-256
`6fb2cb9511c0908193be6f2e94e370dc34b6321f25ab5d7fcb221a3e4f20bbed` to Draft and
additionally:

- raises the `system-core` hard job timeout from 25 to 30 minutes while retaining the
  20-minute main deadline and up-to-five-minute diagnostics and cleanup reserve, with
  remaining time for checkout, step transitions, and artifact upload;
- permits explicit nginx locations preserving `/v1/`, `/oauth/`, `/readiness/`, and
  `/healthz/` routing to `web-api:9001` after the catch-all switches to UI, and
  requires route-level contract coverage; and
- corrects the migration report path to `/reports/migrations.json`.

The tenth revision corrects only the Required Commands `ui-nginx` invocation to use
the implemented HTTP route flag and probe paths and the executable gRPC method paths.

The eleventh revision removes inaccessible recursive submodule checkout, generates
and compares deterministic protobuf descriptor images from tracked current and
target-base Go packages, and permits the pinned Newman lock's required patched
`handlebars=4.7.9` transitive override subject to dependency-review.

## RFC Lifecycle State

An independent challenger reviews these exact Draft bytes. After a
READY-FOR-ACCEPTANCE result, the RFC author changes only the sole status metadata line
to `- Status: Accepted`; the same independent challenge role reviews those final
exact bytes before the coordinator creates the exact SHA-256 confirmation gate.
Implementation begins only after that gate resolves to `approved`.
