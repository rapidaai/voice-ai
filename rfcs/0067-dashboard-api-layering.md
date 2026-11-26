# RFC 0067: Dashboard API Layering

- Status: Accepted
- Owner: Web API
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0066 for the Dashboard controller location and ownership boundary only

## Context

The native Web API Dashboard controller is separated from the other native Web API controllers in
a child package under `api/web-api/api`; the proxy controllers retain their separate proxy package.
Dashboard also owns request validation, downstream client construction, concurrent analytics and
usage loading, section composition, and transport error mapping. This differs from the established
native Web API boundary where controllers live in package `web_api`, validate transport input, and
call services under `api/web-api/internal/service` for domain and persistence work.

The Dashboard controller is a protected route covered by RFC 0066. Moving it therefore requires
the protected-controller inventory to move with it without changing the route policy, principal
contract, or registered gRPC behavior.

## Decision

Move the Dashboard gRPC controller into package `api/web-api/api`. The controller owns principal
extraction, project-tenant checks, protobuf timestamp checks, defaults, request limits, gRPC status
mapping, and response-envelope construction. Use `pkg/validator` for project and numeric range
validation.

Add a Dashboard service contract under `api/web-api/internal/service` and its implementation under
`api/web-api/internal/service/dashboard`. The contract is
`GetProjectDashboard(context.Context, authentication.Principal, DashboardQuery) (*protos.ProjectDashboard, error)`.
`DashboardQuery` contains `GeneratedAt`, `FromDate`, `ToDate`, `ComparisonFromDate`,
`ComparisonToDate`, `AssistantLimit`, and `RecentFailureLimit`. All fields are resolved and
validated by the controller before the call.

The internal service owns the analytics and product-usage dependencies, protobuf analytics-request
construction from the validated query, two concurrent section calls, timing logs, partial-result
rules, and `ProjectDashboard` composition including generated and period timestamps. Its result and
error contract is closed and evaluated in this order after both workers exit:

- caller context error wins and returns that `context.Canceled` or `context.DeadlineExceeded`;
- analytics cancellation wins over every usage result, with cancellation checked before deadline;
- analytics deadline wins over every remaining usage result;
- usage cancellation wins over every remaining outcome, with cancellation checked before
  deadline;
- usage deadline wins over every remaining outcome;
- analytics gRPC `InvalidArgument` wins over non-terminal usage success or failure and returns a
  centralized
  `pkg/errors.DashboardAnalyticsRequestInvalidError` that retains the downstream message and cause;
- both sections return non-terminal errors: return centralized
  `pkg/errors.DashboardUnavailableError` with both causes and no Dashboard;
- exactly one section returns a non-terminal error: return the composed partial Dashboard and nil
  error, with the failed section unavailable;
- both sections succeed: return the composed non-partial Dashboard;
- a nil section without an error preserves existing state composition and does not independently
  make the response partial.

The service passes the exact `authentication.Principal` argument unchanged to both dependency
calls. It does not derive, reduce, or replace tenant, subject, actor, proof, or authority facts.

The controller maps only that closed error set: context errors to their matching gRPC codes, the
analytics request error to `InvalidArgument` with the retained message, and Dashboard unavailable
or any unknown error to generic `Internal`.

Update only the router import and protected-controller matrix path. Keep the canonical gRPC route,
route policy, protobuf messages, authentication principal, status codes, defaults, limits, date
semantics, concurrent loading, and partial-result behavior unchanged.

## Contracts and Ownership

- Package `api/web-api/api` owns Dashboard transport validation and gRPC mapping.
- `internal_service.DashboardService` owns the internal Dashboard operation contract and validated
  query value.
- Package `api/web-api/internal/service/dashboard` owns downstream orchestration, concurrency,
  logging, and section composition.
- Package `pkg/errors` owns the transport-independent Dashboard analytics-request and unavailable
  error types.
- The generated Dashboard protobuf service and `pkg/authentication/policy/v1/web-api.yaml` remain
  unchanged.
- RFC 0066 remains authoritative for principal enforcement and protected-route inventory; this RFC
  supersedes only its Dashboard controller file location and ownership detail.

## Alternatives Rejected

- Move the file but retain orchestration in the controller: rejected because it preserves the
  ownership inconsistency the change is intended to remove.
- Let the internal service validate the protobuf request: rejected because transport validation
  belongs to controllers and would couple the service to optional protobuf input semantics.
- Split analytics and usage into additional Dashboard abstractions: rejected because the existing
  client and product-usage service contracts already provide the required dependency boundaries.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | No public contract changes. The same route, defaults, limits, status codes, and response data remain observable. |
| Failure and recovery | Cancellation, deadline, invalid analytics request, both-section failure, and partial-section behavior remain unchanged. Rollback restores the old controller package and router import. |
| Security and privacy | Principal and project-tenant checks stay in the controller before any service call. No credential or authorization facts are added. |
| Data and migration | N/A. No schema, persisted data, or query behavior changes. |
| Concurrency and resources | The internal service becomes the owner of the existing two buffered result channels and two request-scoped goroutines. It waits for both workers before returning, including terminal error paths; deterministic barrier tests prove both start concurrently and complete. |
| Operations and observability | Existing per-section timing and failure logs move with orchestration into the internal service; log messages and severity remain unchanged. |

## Rollout and Rollback

Move the controller, add the internal service contract and implementation, update router wiring and
the controller inventory, then run focused package and race tests before repository finalization.
Stop on any changed gRPC status, Dashboard response, policy route, principal guard, goroutine leak,
test failure, or unresolved Critical or Major review finding. Rollback restores the prior Dashboard
package, router import, and matrix path together. There are no irreversible effects.

## Verification

```bash
go test -count=1 ./api/web-api/api ./api/web-api/internal/service ./api/web-api/internal/service/dashboard ./api/web-api/router ./pkg/errors
go test -race -count=1 ./api/web-api/api ./api/web-api/internal/service ./api/web-api/internal/service/dashboard ./api/web-api/router ./pkg/errors
go vet ./api/web-api/api ./api/web-api/internal/service ./api/web-api/internal/service/dashboard ./api/web-api/router ./pkg/errors
go test -count=1 ./api/web-api/api -run TestWebRPCAuthenticationContract
test ! -d api/web-api/api/dashboard
git diff --check
just agent-finalize "rfcs/0067-dashboard-api-layering.md,api/web-api/api/dashboard.go,api/web-api/api/dashboard_test.go,api/web-api/api/testdata/authentication-policy-matrix.json,api/web-api/internal/service/dashboard.service.go,api/web-api/internal/service/dashboard.service_test.go,api/web-api/internal/service/dashboard/service.go,api/web-api/internal/service/dashboard/service_test.go,api/web-api/router/web.go,api/web-api/router/web_test.go,pkg/errors/dashboard.go,pkg/errors/dashboard_test.go"
```

Acceptance requires unchanged registered Dashboard behavior, validator-owned transport checks,
no Dashboard child API package, no service call after invalid input, exact protected-controller
matrix parity, and deterministic tests for full success, each partial result, both-section failure,
analytics `InvalidArgument`, caller cancellation, analytics-side and usage-side cancellation and
deadline, analytics `InvalidArgument` with a simultaneous non-terminal usage failure, conflicting
analytics cancellation and usage deadline precedence, unchanged principal forwarding, both calls
starting before either completes, and both workers exiting before return. No unresolved Critical
or Major finding may remain.

## Review and Approval

- Challenger: Independent GPT-6 Astra and GPT-5.6 Sol reviews with combined evidence panel
- Outcome: Approved
- Resolved findings: The final contract defines the exact query and result owners, centralized
  errors, unchanged principal forwarding, current mixed-failure precedence, and deterministic
  concurrency and worker-lifecycle coverage.
- Open questions: None
- Approval reference: Repository owner requested Dashboard API layering alignment on 2026-11-13.
  Astra, Sol, and the combined panel approved draft digest
  `0f62da4cfe57655cee14bbef5383787048f1278b427e58dc7b65ee001cb6b9c3`.
