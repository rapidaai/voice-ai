# RFC 0020: Project Dashboard Analytics

- Status: Accepted
- Owner: Voice AI platform team
- Approved by: Voice AI platform owner
- Date: 2026-10-17
- Supersedes: N/A

## Context

The project dashboard currently renders local sample data. Production assistant telemetry is
stored by assistant-api, while billable component usage is stored by web-api. The existing
`AssistantService.GetAssistantDashboard` RPC is assistant-scoped and cannot return a complete
project view without browser request fan-out. `ProductUsageService` stores project component
duration, but it has no session, reliability, health, assistant, or failure data.

The authoritative proto submodule now defines a separate dashboard contract in
`protos/artifacts/dashboard-api.proto`. It contains a public composite `DashboardService` and a
downstream `ProjectDashboardAnalyticsService`. Using these services keeps project dashboard
traffic outside `AssistantService` while preserving the current owners of analytics and usage
data.

The current mock screen includes component latency and channel distribution. The authoritative
contract instead provides component usage and recent failures. This decision aligns the screen to
the contract: component usage remains, recent failures replace channel distribution, and
component latency is not shown as a separate panel.

## Decision

Adopt `dashboard_api.DashboardService.GetProjectDashboard` as the only browser-facing project
dashboard RPC. Web-api owns this composite service and returns one response containing independent
analytics and usage sections.

Assistant-api implements
`dashboard_api.ProjectDashboardAnalyticsService.GetProjectDashboardAnalytics` as a separate API
and service package. It owns session, message, assistant, health, activity, and recent-failure
aggregation because it owns the source telemetry. The implementation does not add project methods
to `AssistantService`.

Web-api obtains analytics from assistant-api and aggregates component usage from its existing
`product_usages` data. It does not recompute assistant telemetry. The UI invokes the public
dashboard RPC once for each selected range and renders each section according to its state.

### Public and downstream RPC contracts

The source of truth is `protos/artifacts/dashboard-api.proto`. The relevant service and request
contracts are:

```proto
service DashboardService {
  rpc GetProjectDashboard(GetProjectDashboardRequest)
      returns (GetProjectDashboardResponse);
}

service ProjectDashboardAnalyticsService {
  rpc GetProjectDashboardAnalytics(GetProjectDashboardAnalyticsRequest)
      returns (GetProjectDashboardAnalyticsResponse);
}

message GetProjectDashboardRequest {
  google.protobuf.Timestamp fromDate = 1;
  google.protobuf.Timestamp toDate = 2;
  optional uint32 assistantLimit = 3;
  optional uint32 recentFailureLimit = 4;
}

message GetProjectDashboardAnalyticsRequest {
  google.protobuf.Timestamp fromDate = 1;
  google.protobuf.Timestamp toDate = 2;
  google.protobuf.Timestamp comparisonFromDate = 3;
  google.protobuf.Timestamp comparisonToDate = 4;
  uint32 assistantLimit = 5;
  uint32 recentFailureLimit = 6;
}
```

The fully qualified public method is
`/dashboard_api.DashboardService/GetProjectDashboard`. The downstream method is
`/dashboard_api.ProjectDashboardAnalyticsService/GetProjectDashboardAnalytics`. Organization and
project scope come only from authenticated metadata. Neither request can select a tenant.

The public response follows the committed proto contract:

- `generatedAt` identifies when web-api completed composition.
- `period` contains the resolved current and comparison ranges.
- `analytics` contains project state, KPI comparisons, activity, health, assistants, and recent
  failures.
- `usage` contains component duration totals.
- `partial` reports that one section is unavailable while the other section is usable.

Field numbers, optional presence, enums, and nested response messages remain exactly as declared
in the authoritative proto. Generated Go, Python, and SDK files are never edited manually.

### Request and range semantics

Web-api captures the current time once and resolves the public request in this order:

1. `toDate` is the supplied valid timestamp, or the captured current time when absent.
2. `fromDate` is the supplied valid timestamp, or 30 days before `toDate` when absent.
3. The current range is the half-open interval `[fromDate, toDate)`.
4. The comparison range has equal duration and ends at `fromDate`.
5. `assistantLimit` defaults to 3 and must be from 1 through 10 when present.
6. `recentFailureLimit` defaults to 5 and must be from 1 through 20 when present.

The service rejects invalid timestamps, equal or reversed bounds, and current ranges longer than
90 days. It passes concrete timestamps and limits to the downstream analytics service. All
timestamps identify UTC instants.

Activity uses 30-minute buckets for ranges up to 24 hours, 2-hour buckets for ranges up to 3 days,
4-hour buckets for ranges up to 7 days, and 24-hour buckets for longer ranges. Buckets are ordered,
half-open, cover the current range without gaps or overlap, and include zero-valued buckets. The
final bucket ends at `toDate`.

### Analytics semantics

Assistant-api applies these rules to both current and comparison ranges:

- A session is one tenant-scoped `assistant_conversations` row, assigned by conversation
  `created_date`.
- A failed session has canonical conversation status `FAILED` or `ERROR`. A missing or different
  status is not failed.
- Success rate is `(sessions - failed sessions) / sessions * 100`. It is absent when there are no
  sessions.
- Component latency is the arithmetic mean of recorded numeric STT, end-of-speech, TTS, and agent
  TTFT message metrics. Response latency is the arithmetic mean of the available non-zero
  component averages and is absent when every component is unavailable.
- Summary change fields are calculated by assistant-api. Count percentage change is absent when
  the comparison count is zero. Success-rate change uses percentage points. Latency change is
  current minus comparison and is absent when either value is unavailable.
- Failed-session share is failed sessions divided by sessions and is absent when there are no
  sessions.
- A slow session has response latency greater than or equal to 1,000 ms. A session without a
  recorded component latency is not slow.
- An assistant with no current-range sessions has `NO_DATA`. It is `HEALTHY` when success rate is
  at least 95 percent and response latency is below 1,000 ms. It `NEEDS_ATTENTION` when success rate
  is below 90 percent or response latency is at least 1,500 ms. Remaining assistants are `WATCH`.
- Assistants are ordered by session count descending and assistant identifier ascending. The
  response includes at most the resolved assistant limit.
- Recent failures are ordered by occurrence time descending and conversation identifier ascending.
  Known authentication, configuration, provider, network, cancellation, no-answer, and rejection
  causes map to the matching enum. All other causes map to `UNKNOWN`. The cause-to-enum mapping is
  owned by one assistant-api function with table-driven tests and is derived from the persisted
  conversation status and disconnect cause.
- `DashboardProjectState.assistantCount` counts tenant-scoped active assistants. `hasAssistants`
  and `hasActivity` are derived from the active-assistant and current-session counts.

Counts are non-negative. Rates are values from 0 through 100. Latency and conversation duration
values are integer milliseconds.

### Usage semantics

Web-api groups active `product_usages` rows in the current range by usage type and sums their
nanosecond values. It maps STT, TTS, LLM, VAD, EOS, and denoise duration to the committed
`DashboardUsageComponent` enum. Each total is converted to integer milliseconds after summation.

A ready usage section contains all six supported components in enum order, including zero totals.
The section is empty when the query succeeds but no supported usage row exists in the range.
Unsupported usage types are ignored and logged without their tenant values.

Usage duration is billable component consumption, not response latency. The UI labels the panel
`Component usage` and formats the duration without presenting it as latency.

### Section, error, and UI behavior

Section states have one meaning across the API and UI:

- `READY`: the query succeeded and the section contains current-range data.
- `EMPTY`: the query succeeded but the section has no current-range data.
- `UNAVAILABLE`: the section failed and its data is omitted.
- `UNSPECIFIED`: invalid in a successful response and treated as unavailable by the UI.

Successful responses set `code = 200`, `success = true`, and leave the `error` field unset. A
section failure is represented by section state and `partial`, never by the `error` field; RPC
failures use a gRPC status, so `error` is not populated. Web-api returns `code = 200`,
`success = true`, and `partial = false` when both sections are ready or empty. If one section is unavailable and the other is ready or empty, it returns a successful
response with `partial = true`. If both sections are unavailable, the RPC fails with gRPC
`INTERNAL` and returns no partial data.

Missing authentication returns `UNAUTHENTICATED`; unsupported scope or missing project context
returns `PERMISSION_DENIED`; invalid range or limit input returns `INVALID_ARGUMENT`; cancellation
and deadline expiry preserve their gRPC status. Internal messages do not expose query text or
tenant values.

The UI client lives outside `assistant.client.ts`, constructs `GetProjectDashboardRequest`, and
calls the generated `GetProjectDashboard` function with standard API metadata. Initial loading
uses layout-matched Carbon skeletons. A total failure shows retry. A partial response renders the
available section and an unavailable state for the failed section. Range changes ignore stale
responses. Refresh failures retain the last successful response.

The dynamic screen contains KPI comparisons, conversation activity, health summary, component
usage, recent failures, and top assistants. Channel distribution and component-latency panels are
removed because they are not part of the selected contract.

## Contracts and Ownership

- `protos/artifacts/dashboard-api.proto` owns both dashboard RPCs, all wire messages, field numbers,
  enums, and compatibility guarantees.
- Web-api owns `DashboardService`, range resolution, comparison-range construction, section
  composition, partial-response policy, component-usage aggregation, and public error mapping.
- Assistant-api owns `ProjectDashboardAnalyticsService`, telemetry aggregation, metric formulas,
  health classification, failure classification, deterministic ordering, and query cancellation.
- The existing `ProductUsageService` continues to own product-usage writes and raw usage reads. Its
  public contract does not change.
- `AssistantService.GetAssistantDashboard` remains assistant-scoped and unchanged.
- The typed downstream client for `ProjectDashboardAnalyticsService` lives in `pkg/clients/workflow`
  with the existing assistant clients; web-api owns its call site, parameters, and contract.
- The UI dashboard client owns public request construction. The page owns formatting, loading,
  empty, unavailable, retry, stale-response, and refresh presentation.
- No database writes, schema changes, backfill, or historical data conversion are included.
- Endpoint and assistant-detail dashboards retain their existing contracts.

## Alternatives Rejected

- Add `GetProjectDashboard` to `AssistantService`: rejected because a committed dedicated dashboard
  service exists and the project response composes data owned by two services.
- Use `ProductUsageService` as the browser dashboard API: rejected because it is a billing ledger
  without assistant telemetry and would give one service two unrelated contracts.
- Call assistant and usage APIs independently from the browser: rejected because it exposes
  composition policy, creates inconsistent period resolution, and cannot represent one coherent
  partial response.
- List assistants and call `GetAssistantDashboard` for each one: rejected because it creates
  request fan-out, cannot guarantee complete pagination, and duplicates aggregation policy.
- Keep channel and component-latency panels: rejected for this version because the committed
  dashboard contract deliberately supplies component usage and recent failures instead.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Both dashboard services are additive. Existing assistant, endpoint, observability, and product-usage methods remain unchanged. Servers deploy before the UI uses the public method. |
| Failure and recovery | Section states and `partial` permit one source to fail without discarding the other. Complete failure uses a gRPC error. The UI supports retry and retains prior data only on refresh failure. |
| Security and privacy | Both services derive organization and project from authenticated context. Requests have no tenant selector. Recent failures expose only project-visible identifiers, assistant name, category, channel, duration, and time. |
| Data and migration | The change is read-only and needs no migration. Existing indexes are used first. Any new index requires production-like query-plan evidence and a separate schema change. |
| Concurrency and resources | Web-api owns the lifecycle of both section operations and cancels downstream work with the caller context. The 90-day range and response limits bound work and payload size. |
| Operations and observability | Each section records duration and failure category without metric payloads or tenant values. Rollout stops on tenant-scope failure, aggregate mismatch, unacceptable query latency, or database load. |

## Rollout and Rollback

1. Accept this RFC and retain the reviewed dashboard proto contract.
2. Generate Go and Python artifacts from the proto submodule (`bash bin/artifacts-generate.sh`) and
   publish the compatible React SDK from `rapidaai/rapida-react` using the SDK commands in
   Verification.
3. Deploy assistant-api with `ProjectDashboardAnalyticsService` while it has no public caller.
4. Deploy web-api with `DashboardService` and the downstream analytics client.
5. Verify isolation, aggregate reconciliation, section failure behavior, query plans, and latency in
   a production-like environment.
6. Deploy the UI using the new SDK, remove static snapshots, and monitor dashboard failures.

If rollout fails, roll back the UI first so it stops calling the new method. Web-api and
assistant-api can then be rolled back independently. The additive proto services and messages may
remain published. No data recovery or reverse migration is required.

## Verification

Acceptance criteria:

- The UI makes one public dashboard request per selected range and does not call
  `AssistantService.GetAssistantDashboard` or `ProductUsageService` directly.
- Current and comparison summaries match direct tenant-scoped database aggregates for 24-hour,
  7-day, and 30-day ranges.
- Activity totals reconcile with the current summary, including empty buckets.
- Component usage totals match `product_usages` sums after nanosecond-to-millisecond conversion.
- Assistant and recent-failure limits, ordering, optional values, and enum mappings are stable.
- Full, empty, analytics-unavailable, usage-unavailable, and complete-failure states follow the
  section contract.
- Foreign organizations and projects are excluded from every query and response.
- Initial loading, retry, range change, stale response, partial response, and refresh failure have
  focused UI coverage.
- Light and dark screenshots and serious accessibility checks pass for the dashboard route.

The Go test packages named below are created by this change; they do not exist before
implementation.

Repository verification commands:

```bash
bash bin/artifacts-generate.sh
go test ./scripts/contracts/protobuf-descriptor
go test ./api/assistant-api/api/dashboard \
  ./api/assistant-api/internal/services/dashboard \
  ./pkg/clients/workflow \
  ./api/web-api/api/dashboard \
  ./api/web-api/internal/service/productusage
cd ui && yarn test --runInBand --watchAll=false --runTestsByPath \
  src/clients/__tests__/dashboard.client.test.ts \
  src/app/pages/main/web-dashboard/__tests__/home-page.test.tsx
cd ui && yarn eslint \
  src/clients/dashboard.client.ts \
  src/clients/__tests__/dashboard.client.test.ts \
  src/app/pages/main/web-dashboard/index.tsx \
  src/app/pages/main/web-dashboard/__tests__/home-page.test.tsx
cd ui && yarn checkTs
cd ui && yarn playwright test e2e/screenshot-journey.spec.js e2e/a11y.spec.js \
  --config=playwright.config.js --grep dashboard.home --workers=1
just ui-browser-test
just agent-finalize "api/assistant-api/api/dashboard,api/assistant-api/internal/services/dashboard,api/web-api/api/dashboard,api/web-api/internal/service/productusage,pkg/clients/workflow,protos,api/document-api/app/bridges/artifacts/protos,ui/src/clients/dashboard.client.ts,ui/src/clients/__tests__/dashboard.client.test.ts,ui/src/app/pages/main/web-dashboard,ui/e2e"
```

React SDK generation, compatibility, and publication run in the `rapidaai/rapida-react` repository
(`sdks/react` submodule, package `@rapidaai/react`), not in this repository. The nested
`src/clients/protos/artifacts` submodule must be pinned to the same internal-bridge commit as
`protos/artifacts` (`9d80daf`).

```bash
# In rapidaai/rapida-react (sdks/react):
git submodule update --init src/clients/protos/artifacts
bash bin/artifacts-generate.sh   # protoc + protoc-gen-ts + protoc-gen-grpc-web -> src/clients/protos
npm ci && npm test               # Jest compatibility suite (CI matrix: Node 18 and 20)
npm run build                    # tsup bundle; also runs through prepublishOnly
# Publication is automated on push to main by .github/workflows/npm-publish.yml:
#   npm test && npm run build && npm publish --access public --provenance
#   followed by a v<package.json version> Git tag and GitHub release.
```

After publication, the UI updates its pinned `@rapidaai/react` dependency from `1.1.88` to the
published version (currently `1.1.90`).

## Review and Approval

- Challenger: OpenCode independent challenge review (feature/project-dashboard)
- Outcome: Approved
- Resolved findings: Proto contract verified against `protos/artifacts` 9d80daf; project dashboard traffic uses the committed DashboardService, not AssistantService; the dynamic screen uses component usage and recent failures instead of unsupported component latency and channel panels; `pkg/clients/workflow` client ownership, error and `code` semantics, post-implementation test paths, and failure-cause ownership recorded; React SDK generation, compatibility, and publication commands recorded from `rapidaai/rapida-react`.
- Open questions: None
- Approval reference: Branch feature/project-dashboard with protos/artifacts 9d80daf and sdks/react 304f44e (@rapidaai/react 1.1.90)
