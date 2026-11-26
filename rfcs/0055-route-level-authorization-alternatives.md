# RFC 0055: Route-Level Authorization Alternatives

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-12
- Supersedes: RFC 0054 controller-operation policy selection and context-bound permission validation only; RFC 0039 authenticated Project-token selector exception only

## Context

RFC 0054 introduced named controller operations for routes whose accepted callers have different
contexts or permission requirements. Middleware admits the parent route, then a controller selects
an operation and calls `AuthorizeOperation`. This makes controllers aware of authorization policy
and leaves a route only partially authorized when the operation call is absent.

The authentication principal already contains the verified subject, caller, tenant, proof, and
effective permissions required for a static route decision. Route policy and that principal must
be sufficient for authorization. Controllers should consume the authenticated principal without
receiving policy or selecting an authorization branch.

## Decision

Replace `authorization: controller` and named `operations` with route-level `authorization: any_of`
alternatives. Each alternative is one complete actor, caller-service, selected-tenant,
Project-token, and authenticated-or-permission predicate. Middleware authorizes the route when any
one alternative matches the principal.

Context matching retains RFC 0054 semantics. Organization context requires a nonzero Organization
ID and accepts either an Organization or Project tenant. Project context requires both tenant IDs.
User context requires a User subject and does not constrain its tenant selection. A Project-context
alternative may require an Organization permission because RFC 0054 intentionally includes the
selected Organization's authorities in a User's Project-tenant principal. Organization-context
alternatives may require only Organization permissions. Project permissions always require Project
context.

The six current controller routes are replaced by the following uniform route contracts. These are
not a mechanical union of request-dependent operations:

| Route | Selected tenant | Complete route alternatives | Data contract |
| --- | --- | --- | --- |
| `VaultService/GetAllOrganizationCredential` | Project | `organization.provider_credentials.read_metadata` or `project.provider_credentials.read_metadata` | Return credentials only for the principal's exact Project. |
| `AccessManagementService/GetRole` | Organization or Project | Organization context with `organization.roles.read`; Project context with `project.roles.read` | An Organization-selected principal may read the Organization-owned catalog. A Project-selected principal may read only Project-scoped definitions owned by the selected Organization, including when its Organization authority admitted the route. |
| `AccessManagementService/ListPermissions` | Organization or Project | Same context and permission alternatives as `GetRole` | An Organization-selected principal may list the Organization catalog. A Project-selected principal may list only Project-scoped permissions. |
| `AccessManagementService/ListRoles` | Organization or Project | Same context and permission alternatives as `GetRole` | An Organization-selected principal may list the Organization-owned catalog. A Project-selected principal may list only Project-scoped definitions. |
| `AuthenticationService/ScopeAuthorize` | Organization or Project | Authenticated Organization credential in Organization context; authenticated runtime Project credential in Project context | The requested scope must describe the already selected principal tenant. |
| `ProjectService/RevokeProjectCredential` | Project | `organization.project_access_tokens.revoke` or `project.access_tokens.revoke` | The target credential must belong to the principal's exact Project. |

The authenticated-only Project-token role-selector exception is removed from `GetRole` and
`ListRoles`.
Project credential creation remains disabled under RFC 0053. If that workflow is enabled later, it
must use a distinct route with its own complete policy rather than weakening a general role-catalog
route. Organization-level revocation now requires selecting the target Project before calling the
route. This is an intentional compatibility change that prevents request data from selecting an
authorization branch after middleware admission.

The policy parser rejects an empty list, duplicate alternatives, invalid contexts or permissions,
alternatives that mix authenticated and permission predicates, and malformed Project-token or
caller-service constraints.

`AuthorizeOperation`, `OperationPolicy`, and `AuthorizationController` are removed. Middleware does
not inspect protobuf requests, HTTP bodies, query semantics, or persisted resources. A controller
retrieves the immutable principal from context, validates the typed request, and passes the
principal to its service. It does not receive route policy, select an authorization alternative,
evaluate permissions, or decide resource access. The service derives its Organization, Project,
subject, and actor inputs from the principal and owns persisted resource-to-tenant checks. A request
tenant field is only a redundant binding assertion and must equal the principal tenant before any
resource lookup or mutation.

## Contracts and Ownership

- HTTP and gRPC authentication middleware own complete route authorization from exactly two inputs:
  the route policy and authenticated principal.
- The versioned policy owns every route-level alternative and remains the authorization source of
  truth.
- Controllers consume the principal and own request-shape validation only. They have no policy,
  permission-evaluation, or authorization-operation dependency.
- Services accept the immutable principal instead of `types.Authentication`. They derive tenant and
  audit-actor facts from it, own persisted resource-to-tenant constraints, and do not perform a
  second static route-permission decision.
- Owned clients select the tenant required by the route. The Web UI Project-credential revocation
  client sends `x-project-id` equal to the protobuf request's Project ID; it does not rely on the
  request body to select authorization context.
- Sensitive mutation services retain RFC 0053's locked transactional rereads. Those checks decide
  whether an already admitted mutation may commit against current state; they do not create a
  controller-visible authorization path.
- The pure route authorizer evaluates simple policies and route-level `any_of` alternatives. It has
  no transport, database, service, assignment, or role-catalog dependency.
- Public protobuf and HTTP request contracts do not change.

## Alternatives Rejected

- Keep controller-selected operations: rejected because missing calls leave authorization partial
  and couple controllers to policy structure.
- Inspect typed requests in middleware: rejected because generic middleware would depend on every
  protobuf and HTTP request contract and would acquire controller validation ownership.
- Flatten request-dependent operation alternatives: rejected because admission under one branch
  could be reused with request data belonging to another branch.
- Let services repeat static route permission checks: rejected because it creates a hidden second
  policy and can disagree with the versioned route policy. RFC 0053 transactional rereads remain a
  narrowly defined commit-time invariant for sensitive mutations.
- Chain credential-specific middleware: rejected because middleware order would become an identity
  selection input and multi-actor routes would not have one deterministic decision.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Internal policy schema changes from named operations to route-level alternatives. Public message shapes remain unchanged. Project-token role selection loses its authenticated-only exception, and Organization-level Project-credential revocation requires a selected target Project. Both are intentional fail-closed behavior changes. |
| Failure and recovery | Missing, empty, duplicate, or malformed alternatives fail policy startup. Unknown routes and unmatched alternatives deny before controller execution. |
| Security and privacy | Permission evaluation becomes mandatory in middleware. Services constrain every resource lookup and mutation to the principal tenant. No credential or permission set is logged. |
| Data and migration | No new persisted data or schema change. Existing RFC 0047, 0049, and 0053 enforcement prerequisites remain required. |
| Concurrency and resources | N/A. Authorization evaluates one immutable per-request principal and introduces no retained resource. |
| Operations and observability | Authentication failure, disabled route, policy denial, controller request rejection, and service tenant/resource rejection remain distinguishable without exposing secrets. |

## Rollout and Rollback

1. Add route-level alternative parsing and pure-authorizer tests with enforcement off.
2. Replace the six controller policies with the uniform contracts above, remove operation
   authorization APIs, and prove policy route inventory and alternative coverage.
3. Cut controllers and affected service interfaces from legacy `types.Authentication` to the
   immutable principal. Remove duplicate static service permission checks and verify every
   service-owned resource-to-tenant boundary.
4. Update the Web UI Project-credential revocation client to select the target Project in
   `x-project-id`, and prove the header and request Project IDs cannot diverge.
5. Complete the remaining RFC 0053 and 0054 enforcement gates, then atomically install HTTP, unary,
   and streaming middleware while removing the legacy authentication chain.

Stop on any uncovered route, controller or affected service still reading legacy authentication,
missing tenant constraint, shared-secret service assertion, migration prerequisite failure, or
unresolved Critical or Major finding. Before enforcement, rollback removes the unwired
implementation. After enforcement, use the retained RFC 0053 unique-key-only safety-floor artifact
and matching configuration.

## Verification

- All route-level alternatives accept only their declared actors, contexts, permissions, callers,
  and Project-token proof.
- Zero alternatives, duplicate alternatives, malformed alternatives, and unknown routes deny closed.
- The six former controller routes enforce the uniform contracts in the decision table without
  named operation selection or request-dependent authorization.
- No production controller references route policy or `AuthorizeOperation`.
- Every protected controller reads only the principal authentication object from context.
- A repository check proves no protected controller or affected service references
  `types.Authentication`, `types.Authorize`, `types.AuthorizeUser`, or `AuthorizeOperation`.
- Service tests cover same-tenant success, cross-tenant denial, and redundant request tenant
  mismatch before persistence.
- Role-catalog tests prove Organization and Project views cannot be exchanged, and prove the former
  authenticated-only Project-token selector is denied.
- Project credential revocation tests prove both permission alternatives require the exact selected
  Project.
- The Web UI credential client test proves Project revocation sends `x-project-id` equal to the
  request Project ID.
- Policy inventory exactly matches registered Web API HTTP and gRPC routes before enforcement.
- The middleware replaces, rather than supplements, the legacy authentication chain.

Exact verification commands:

```bash
go test ./pkg/authentication/... ./pkg/middlewares/authentication ./api/web-api/api/... ./api/web-api/internal/service/... ./cmd/web
go test -race ./pkg/authentication/... ./pkg/middlewares/authentication ./api/web-api/api/... ./api/web-api/internal/service/... ./cmd/web
go vet ./pkg/authentication/... ./pkg/middlewares/authentication ./api/web-api/api/... ./api/web-api/internal/service/... ./cmd/web
! rg -n 'types\.(Authentication|Authorize|AuthorizeUser)|AuthorizeOperation' api/web-api/api api/web-api/internal/service
yarn --cwd ui test --runInBand ui/src/clients/__tests__/credential.client.test.ts
just agent-finalize "rfcs/0055-route-level-authorization-alternatives.md,pkg/authentication/authorizer,pkg/authentication/policy,pkg/middlewares/authentication,api/web-api/api,api/web-api/internal/service,cmd/web,ui/src/clients/credential.client.ts,ui/src/clients/__tests__/credential.client.test.ts"
```

## Review and Approval

- Challenger: Independent Astra and legacy-Sol reviews
- Outcome: Approved
- Resolved findings: The first draft's unsafe mechanical flattening, permission-dependent Project
  revocation binding, omitted service-contract cutover, missing owned-client migration, and
  incomplete supersession lineage are corrected in this revision.
- Open questions: None
- Approval reference: Repository owner accepted this exact RFC for implementation in the Access
  Management cutover session on 2026-11-12
