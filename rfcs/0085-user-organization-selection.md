# RFC 0085: User Organization Selection

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-14
- Supersedes: RFC 0076 User credential selection contract only

## Context

The authentication aggregate and `Principal.Tenant` support multiple Organization memberships,
but `ResolveUserCredentialRequest` carries only `selectedProjectId`. The User authentication
service compensates by rejecting a User whose active Organization role assignments span more than
one Organization. This makes the reusable aggregate unable to represent the multi-Organization
membership graph it owns and prevents a User from selecting an Organization that has no Project.

The correction must preserve the clean authentication cutover, keep role and permission derivation
inside the aggregate, and reject any Organization and Project combination that is not backed by the
active assignments loaded for the request.

## Decision

Add `selectedOrganizationId` as field 4 of `ResolveUserCredentialRequest`. Field 3 remains
`selectedProjectId`; no existing field is renumbered or reused. Add the optional
`x-organization-id` User-credential transport value beside `x-project-id` in Go, Python, and the
React SDK authentication boundaries.

User credential resolution selects a tenant from the active assignment graph loaded for that
request:

- a nonzero selected Organization must match an active User Organization role assignment;
- a nonzero selected Project must match an active User Project role assignment;
- when both are nonzero, the Project must belong to the selected Organization;
- when only a Project is supplied, its active assignment determines the Organization;
- when neither is supplied and the User's active assignments contain exactly one distinct
  Organization ID, that Organization is selected to preserve current behavior regardless of how
  many Roles are assigned there;
- when neither is supplied and the User belongs to zero or multiple Organizations, the Principal is
  tenantless while the Authentication aggregate still contains the complete verified membership
  graph.

Unknown, archived, mismatched, duplicate, malformed, and out-of-range selections fail closed. The
resolver response remains one canonical `Authentication`; the receiving aggregate constructor
continues to validate membership and derive authorities.

Password, social, registration, session creation, session refresh, and
`AuthenticationForPrincipal` use the same multi-Organization graph rules. Session issuance no
longer rejects a User merely because active assignments span multiple Organizations. A session
producer without an explicit tenant selects the sole Organization or returns a tenantless
Principal when zero or multiple Organizations exist.

The UI stores `selectedOrganizationID` separately from the immutable Authentication snapshot. A
valid Principal-selected Organization wins. Otherwise a still-valid prior user selection is
preserved, a sole Organization is selected, and multiple Organizations remain unselected. The UI
does not fall back to the first Organization. An Organization selector sets the explicit selection;
Project selection is then limited to that Organization. User request metadata includes both the
selected Organization and Project when present.

The authenticated application shell distinguishes verified Organization memberships from a
selected Organization. A User with memberships but no selection remains in the shell so the
Organization selector is reachable. Absence of a selected Project does not redirect the entire
shell to onboarding; Organization-scoped workspace and membership routes remain usable without a
Project.

## Contracts and Ownership

- `authentication.proto` owns the new request field and preserves protobuf wire compatibility.
- Authentication transport middleware owns extraction and validation of `x-organization-id`.
- `pkg/authentication/resolver` owns propagation of the selected Organization and verification that
  the returned Principal represents the requested or deterministically inferred tenant.
- Web Access Management owns active membership lookup and Organization/Project binding.
- Web command CORS owns browser admission for `x-organization-id`.
- The React SDK and Document API expose and forward the optional transport value.
- UI authentication state owns the current Organization selection and the global navigation owns
  its explicit selector. Neither may silently select the first of multiple Organizations.
- The protected application shell owns membership onboarding only. It must not treat an unselected
  tenant as absence of verified memberships or require a Project for Organization-scoped routes.
- Access Management and Workspace clients own Organization-scoped request metadata. Their callers
  pass the selected Organization explicitly; clients do not import or read global authentication
  state.
- The Authentication aggregate remains the sole owner of permission derivation. No authority list,
  role projection, or selected-tenant state is duplicated.

## Alternatives Rejected

- Infer the Organization only from `selectedProjectId`: rejected because an Organization without a
  Project cannot be selected and Organization-scoped requests remain ambiguous.
- Select the first Organization for a multi-Organization User: rejected because ordering is not an
  authorization decision and could grant the wrong tenant context.
- Require the new field immediately on every existing request: rejected because a selected Project
  already identifies its Organization and a sole Organization is unambiguous.
- Store the selected Organization in a server-side session: rejected because it introduces mutable
  cross-request authorization state and weakens per-request tenant selection.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Adding field 4 is protobuf wire-compatible. Existing callers keep working through Project inference or the sole-Organization rule. The persisted UI store version increases and discards the old implicit selection state. Callers can opt into explicit Organization selection with the new header. |
| Failure and recovery | Invalid or cross-Organization selections return the existing authentication credential denial. Rollback removes field 4 and the optional header while retaining existing field numbers. |
| Security and privacy | Every selected tenant is verified against the active assignments loaded for the request. No caller-provided Organization or Project ID is trusted independently. Inconsistent loaded records fail closed. The response disclosure is unchanged. |
| Data and migration | No server data or schema migration is required. The browser authentication store version increases so stale implicit tenant selection is not rehydrated. |
| Concurrency and resources | N/A. Resolution remains request-scoped and adds no cache, goroutine, or retained resource. |
| Operations and observability | Existing authentication rejection logs remain credential-free. Deployments may roll forward atomically with generated clients; old wire callers remain compatible. |

## Rollout and Rollback

Regenerate every authentication protobuf artifact, then deploy Web and remote resolver consumers as
one candidate. Existing callers may omit field 4. Stop on generated-source drift, acceptance of a
mismatched Organization/Project pair, rejection of a valid multi-Organization snapshot, or any
unresolved Critical or Major review finding.

Rollback restores the previous resolver interface, middleware transport boundary, and generated
artifacts. It also restores the prior UI store shape. Rollback intentionally reintroduces denial for
multi-Organization Users. Session rows issued before rollback may remain until normal expiry, but
the old resolver will reject those Users and no row requires data conversion.

## Verification

Acceptance requires:

- exact descriptor coverage for `selectedOrganizationId = 4` with `JS_STRING` behavior;
- User resolver propagation and returned-tenant validation;
- unary, stream, and HTTP middleware coverage for valid, absent, duplicate, blank, malformed, and
  out-of-range Organization values;
- service coverage for explicit Organization selection, Project-derived Organization selection,
  the sole-Organization default, tenantless multi-Organization authentication, and mismatched or
  inactive assignments;
- password/session creation and refresh coverage for multi-Organization Users;
- receiving resolver rejection of malicious explicit, inferred, sole, and tenantless response
  mismatches;
- credential-family conflict coverage when `x-organization-id` is presented alone or with a
  Project, Organization, or Service credential;
- header-only HTTP extraction with duplicate, blank, malformed, and out-of-range rejection;
- Web CORS preflight coverage and UI tests proving multiple Organizations remain unselected until
  the User selects one, after which Organization and Project metadata stay consistent;
- protected-route coverage proving a tenantless multi-Organization User can reach the selector and
  an Organization without a Project can reach Organization-scoped pages;
- matching root and React protobuf sources and regenerated Go, Python, JavaScript, and TypeScript
  outputs.

```bash
cmp protos/artifacts/authentication.proto sdks/react/src/clients/protos/artifacts/authentication.proto
bash bin/artifacts-generate.sh
(cd sdks/react && bash bin/artifacts-generate.sh)
go test -count=1 ./pkg/authentication/... ./pkg/middlewares/authentication ./pkg/clients/web ./protos ./api/web-api/api ./api/web-api/internal/service/user
go test -race -count=1 ./pkg/authentication/... ./pkg/middlewares/authentication ./pkg/clients/web ./api/web-api/internal/service/user
python3 -m pytest api/document-api/tests/bridges/test_auth_bridge.py api/document-api/tests/middlewares/test_principal_authentication_middleware.py
(cd sdks/react && npm test -- --runInBand)
(cd ui && yarn test --watchAll=false --runInBand)
(cd ui && yarn checkTs)
just ui-browser-test
git diff --check
just agent-finalize "$(paste -sd, plans/0085-finalize-paths.txt)"
```

## Review and Approval

- Challenger: Independent Codex plan and contract review
- Outcome: Approved
- Resolved findings: Expanded the boundary to every session producer, explicit UI Organization
  selection and metadata, Web CORS, exact receiver validation, credential conflicts, and rollback
  behavior. Defined the sole-Organization rule over distinct IDs and made the HTTP value
  header-only.
- Open questions: None
- Approval reference: Repository owner directed correction of explicit User Organization selection
  and explicitly selected UI-first propagation on 2026-11-14.
