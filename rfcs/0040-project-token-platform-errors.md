# RFC 0040: Project Credential Validation and Platform Errors

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-09
- Supersedes: RFC 0039 project-credential structural-validation ownership and public error mapping only

## Context

RFC 0039 made project-token role selection mandatory and defined the transaction that validates an
eligible Project and role before creating a project credential. Its public error contract used raw
HTTP values, `400` and `404`, as `Error.errorCode` values and kept public messages beside controller
branches. That differs from the established Web API convention: `response.code` carries the
HTTP-style status, while `Error.errorCode` carries an operation-specific `pkg/errors.ErrorCode` and
the associated `PlatformError` owns both machine and human messages.

The current creation path also divides structural request checks between the Web API controller and
project service. Request shape does not require database access, while Project and role eligibility
must stay with the service transaction to avoid a check-to-insert race. A durable decision is needed
because correcting the numeric application errors changes a public response contract and changes
which layer owns request validation.

This RFC supersedes only RFC 0039's decisions for `CreateProjectCredential` structural-validation
ownership and expected failure-envelope mapping. RFC 0039 remains authoritative for authentication,
caller authorization, Project and role eligibility, persistence, lock order, migration, catalog
reads, SDK and UI behavior, rollout prerequisites, credential disclosure, and every other decision.

This file, `rfcs/0040-project-token-platform-errors.md`, is the sole RFC decision and approval
record. Planning and execution evidence remain in the governed task or pull request. No RFC sidecar
receipt, digest file, or alternate RFC path is created.

## Decision

After authentication and Organization-context extraction succeed, the Web API controller first
validates that the authenticated Organization ID is in `1..math.MaxInt64`. A missing, zero, or
out-of-range ID returns no response envelope and the existing generic gRPC `PermissionDenied`
status with `project credential creation permission denied`. The controller does not call the
service. This is authenticated-context validation, not a request-envelope failure.

After the Organization ID check succeeds, the controller validates the three request fields in
protobuf field order:

1. `projectId` is in `1..math.MaxInt64`.
2. `name`, after one `strings.TrimSpace` operation, is not blank.
3. `roleDefinitionId` is in `1..math.MaxInt64`.

The first failed check returns its expected failure envelope with gRPC status `OK`. The controller
does not call the service after a structural failure. It passes the checked, trimmed name to the
service, and neither layer adds a credential-name length policy.

The repository `pkg/errors` package is the sole source of truth for these four operation-specific
`PlatformError` definitions. The `1026xxx` family was unused when this Draft was written and must be
rechecked against the integration head before acceptance and merge.

| Condition | HTTP-style `response.code` | `error.errorCode` | `error.errorMessage` | `error.humanMessage` |
| --- | ---: | ---: | --- | --- |
| Invalid `projectId` | 400 | 1026001 | `invalid project identifier` | `Select a valid project and try again.` |
| Blank trimmed credential name | 400 | 1026002 | `project credential name is required` | `Enter a project credential name and try again.` |
| Invalid `roleDefinitionId` | 400 | 1026003 | `invalid project-token role identifier` | `Select a project-token role and try again.` |
| Eligible Project or role not found | 404 | 1026004 | `eligible project or project-token role not found` | `The selected project or role is unavailable.` |

Every expected failure response has `success=false`, no credential data, and a nil Go error. The
controller copies `PlatformError.HTTPStatusCodeInt32()` to `response.code`, `PlatformError.Code` to
`error.errorCode`, `PlatformError.Error` to `error.errorMessage`, and
`PlatformError.ErrorMessage` to `error.humanMessage`. An endpoint-local response builder may perform
that mechanical mapping, but it accepts one `PlatformError`; it does not accept a raw numeric code
or independent message arguments. This decision adds no generic response abstraction.

The project service no longer owns public request-shape policy or public validation text. It retains
one defensive authenticated-context invariant for direct and internal callers: an Organization ID
outside `1..math.MaxInt64` returns the typed internal sentinel
`ErrCredentialOrganizationInvalid` before any SQL. This is not an eligible-selection failure and
does not receive a `PlatformError` envelope. If it reaches the controller despite the controller
guard, the controller returns no response and gRPC `Internal` with
`project credential creation failed`.

The service remains the sole owner of database-backed eligibility and creation atomicity. It
observes caller cancellation, creates the credential identity and key, locks the active
same-Organization Project, then locks the selected active eligible role. The role remains eligible
only when it is Project-scoped with `PROJECT_TOKEN` subject and is either a global built-in role or
a custom role owned by the same Organization. The service retains Project-then-role lock order,
credential and role-association insertion, rollback on every pre-commit failure, and return of the
credential and key only after commit.

The service exposes one semantic eligible-selection-not-found sentinel for controller mapping. Its
text is not public contract data. Unknown, foreign, archived, wrong-scope, and wrong-subject Projects
or roles all map to the same `1026004` response, preserving tenant and hidden-resource
indistinguishability.

Authentication and caller denial continue to return no response message and use canonical non-`OK`
gRPC statuses. Cancellation, deadline, cast, insert, commit, and other unexpected failures retain
RFC 0039's non-`OK` status behavior and fixed public messages. They do not expose database details,
resource distinctions, or credential keys. Success remains `code=200`, `success=true`, credential
data present, and gRPC status `OK`.

## Contracts and Ownership

- `pkg/errors` owns the four application codes, HTTP-style statuses, machine messages, and human
  messages. No controller constant or service error string is a parallel public-message source.
- The Web API controller owns authentication sequencing, Organization-context extraction, the
  signed-BIGINT Organization-ID guard, deterministic structural validation, trimmed-name handoff,
  response-envelope construction, and service-error-to-public-response mapping.
- The project service owns the defensive signed-BIGINT Organization-ID precondition, context
  observation, generated credential identity and key, database-backed Project and role eligibility,
  tenant isolation, transaction and lock lifetime, Project-then-role lock order, insertion,
  rollback, and commit authority.
- The protobuf fields, response-envelope shape, generated clients, gRPC success-versus-error split,
  and caller authorization predicate do not change.
- Expected-envelope `error.errorCode` changes from raw `400` and `404` values to `1026001` through
  `1026004`. `response.code` remains `400` or `404`. The existing role-selection and hidden-resource
  human messages remain byte-for-byte unchanged.
- RFC 0039 remains authoritative outside the narrow decisions explicitly superseded here.

## Alternatives Rejected

- Keep raw `400` and `404` values in `error.errorCode`: rejected because it conflates transport-style
  status with the operation-specific application code and conflicts with repository conventions.
- Keep public message literals in controller branches or derive them from the service sentinel:
  rejected because either choice creates a second public-error source beside `pkg/errors`.
- Keep public request-shape checks in both controller and service: rejected because the sole
  production caller is the controller and duplicate policy obscures ownership without improving
  database safety. The service's authenticated Organization-ID precondition is retained separately
  so direct and internal callers cannot send an out-of-range value to SQL.
- Move Project or role eligibility to the controller: rejected because eligibility can change before
  insertion and must remain under the service-owned locks and transaction.
- Add a generic response mapper, validated request type, or service-layer request object: rejected
  because one endpoint and four direct mappings do not justify a new abstraction.
- Change protobuf, SDK, or UI contracts: rejected because the existing envelope already carries the
  required fields and the UI reads the preserved human message without interpreting the numeric
  application code.

## Consequences and Risks

The result has one public-error catalog, one request-shape owner, and one database transaction owner.
The main compatibility cost is deliberate: clients that incorrectly equate `error.errorCode` with
`response.code` will observe new operation-specific values.

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | `response.code`, envelope shape, gRPC status behavior, and existing role-selection and hidden-resource human messages remain stable. `error.errorCode` deliberately changes from `400` or `404` to `1026001..1026004`; consumers must treat it as an application code. The inspected UI does not branch on that number. Access Management must notify known SDK, UI, and direct gRPC consumer owners before deployment. |
| Failure and recovery | Organization-context failure returns the existing generic `PermissionDenied` result without a service call. The service also rejects an invalid Organization ID with a typed internal sentinel before SQL. Structural failures return before service invocation. Database selection, insertion, and rollback remain one service-owned transaction, so no new partial-success state is introduced. Cancellation, deadline, cast, insert, commit, and unexpected failures retain RFC 0039 behavior. |
| Security and privacy | The controller maps only semantic sentinel identity, never service error text. One generic `1026004` response covers missing, foreign, archived, wrong-scope, and wrong-subject resources, preventing a new existence oracle. Keys and database details remain absent from public failures. Any future endpoint log must exclude keys and hidden cross-tenant details. |
| Data and migration | N/A. This decision changes no schema, stored representation, migration, existing row, or credential association. |
| Concurrency and resources | Project-then-role lock order, transaction lifetime, and commit authority remain in the service. Moving only structural checks cannot introduce a validation-to-insert race. No goroutine, channel, timer, stream, connection, retry, or cleanup worker is added. |
| Operations and observability | `CreateProjectCredential` currently has no dedicated endpoint log, and this decision does not claim one. The rollout relies on existing request and gRPC telemetry plus explicit response probes. If implementation deliberately adds an endpoint failure log, it must exclude credential keys and hidden cross-tenant details. Release checks must stop on an error-code collision, incorrect envelope, or lost transaction invariant. |

## Rollout and Rollback

The correction ships with the Web API backend implementation after this RFC is independently
challenged and approved. Recheck that `1026001..1026004` remain unallocated before acceptance and
again against the merge candidate. Before backend deployment, Access Management inventories known
SDK, UI, and direct gRPC consumer owners and gives them advance notice of the deliberate
`error.errorCode` change. The notice states the four new application codes, unchanged
`response.code` and gRPC behavior, preserved messages, cutover timing, and rollback contact. Missing
notice to a known consumer blocks deployment.

After notice, deploy the error definitions, controller mapping, and service ownership adjustment as
one backend release; no client, SDK, UI, schema, or migration ordering is required. RFC 0039's
broader rollout prerequisites remain in force.

Stop rollout if any new code is already allocated, any expected branch returns a non-`OK` gRPC
status, hidden resource classes produce distinguishable responses, structural failure reaches the
service, an invalid authenticated Organization ID reaches SQL, required consumer notice is missing,
or transaction and no-row-on-failure checks regress.

Before clients depend on the `1026xxx` codes, rollback is a code revert to the prior controller and
service behavior. There are no data effects to undo. After a client depends on the new application
codes, prefer a forward correction; an emergency rollback requires notice to affected consumers
because it restores raw `400` and `404` application codes. The database transaction and persisted
credential state remain unchanged in either direction.

## Verification

Acceptance requires observable evidence that:

- all four `PlatformError` values have the exact status, code, machine text, human text, and decimal
  code representation specified above, and no `1026xxx` allocation collides;
- controller tests cover each structural failure, protobuf-field validation order, no service call
  after an invalid Organization context or any structural failure, the exact generic
  `PermissionDenied` result, one-time name trimming and handoff, every envelope field, generic
  hidden selection, success, cancellation, deadline, cast failure, and unexpected failure;
- service tests prove the signed-BIGINT Organization-ID precondition returns
  `ErrCredentialOrganizationInvalid` without SQL and retain built-in and same-Organization
  custom-role creation, tenant isolation, inactive and ineligible selection rejection, fixed
  lock-backed selection, rollback, and no credential row or key on failure without presenting
  request-shape checks as service contract;
- authentication, caller denial, success, protobuf, SDK, UI, schema, and RFC 0039's other decisions
  are unchanged;
- rollout evidence records advance notice to known SDK, UI, and direct gRPC consumer owners; and
- an independent challenger reports no unresolved critical or major finding and a repository owner
  approves the exact RFC candidate before its status changes to `Accepted`.

Run these exact gates for the implementation candidate:

```bash
go test ./pkg/errors -run '^TestCreateProjectCredential'
go test ./api/web-api/api -run '^TestCreateProjectCredential'
go test ./api/web-api/internal/service/project -run '^TestProjectCredential'
go test ./pkg/errors ./api/web-api/api ./api/web-api/internal/service/project
go test ./api/web-api/...
python3 - <<'PY'
from pathlib import Path
import re

source = "\n".join(path.read_text() for path in Path("pkg/errors").glob("*.go"))
for code in range(1026001, 1026005):
    count = len(re.findall(rf"\bErrorCode\s*=\s*{code}\b", source))
    if count != 1:
        raise SystemExit(f"expected ErrorCode {code} exactly once, found {count}")
print("project credential ErrorCode declarations are unique")
PY
just validate-rfc-layout
just agent-style-check
git diff --check
```

The implementation task or pull request owns formatter commands, changed-path finalization, exact
file scope, execution results, and code-review evidence.

## Review and Approval

- Challenger: Codex plan challenger `term_e46c176017e49ccd1c289c547cfc93c0`, task
  `task_ccff43d29bf8`, dispatch `ctx_708c1e1c9140`
- Initial challenger: Codex challenger, task `task_7248c4653886`, dispatch
  `ctx_ed2d54c185a3`
- Reviewed digest: SHA-256 `f5715d79b2bb9c2b649473022d06edf378b2e86f06cfaf065e912819ade12b62`
- Reviewed at: 2026-11-09T07:45:51Z
- Outcome: Approved
- Revision round: Approved by targeted re-review after revision 1 of 2
- Resolved findings: The RFC author addressed all four Valid Major findings below. The approval
  challenger reviewed the exact revised digest and found no remaining Critical or Major issue.

| Finding | Disposition | Resolution in revision 1 |
| --- | --- | --- |
| Lifecycle evidence and allocation proof used stale paths and a non-failing search | Valid Major | Names `rfcs/0040-project-token-platform-errors.md` as the sole RFC record, prohibits RFC sidecars, and replaces the search with an executable check that fails unless each code is declared exactly once. |
| Authenticated Organization IDs could exceed the signed database range after moving structural checks | Valid Major | Keeps the controller's `1..math.MaxInt64` guard and generic `PermissionDenied` mapping, plus a typed service precondition that rejects invalid internal input before SQL and maps any escaped sentinel to `Internal`. |
| Observability claimed endpoint logging that does not exist | Valid Major | States that the endpoint has no dedicated log, relies on existing request and gRPC telemetry plus response probes, and constrains any deliberately added failure log. |
| The rollout did not notify consumers before changing public application codes | Valid Major | Requires advance notice to known SDK, UI, and direct gRPC consumer owners with the codes, stable fields, cutover timing, and rollback contact; missing notice blocks deployment. |

- Open questions: None
- Approval reference: Repository owner, current Codex coordination request, 2026-11-09. The owner
  explicitly directed that project-credential validation remain in the Web API controller and that
  public failures use the established `pkg/errors` catalog. That instruction initiated this
  superseding RFC and approves the exact resolved direction reviewed above.

This RFC is Accepted. The independent challenger approved the exact revised decision candidate,
the repository owner approved the resolved direction, no open question remains, and implementation
is authorized within this RFC's scope.
