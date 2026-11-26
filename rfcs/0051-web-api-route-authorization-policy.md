# RFC 0051: Web API Route Authorization Policy

- Status: Draft
- Owner: Access Management
- Approved by: Pending
- Date: 2026-11-11
- Supersedes: RFC 0050 backend-handler authorization ownership only

## Context

Web API authorization is currently distributed across controller accessors, direct identity and
context checks, and capability-specific code. The web router registers native management services,
proxy services, and HTTP handlers, while the role catalog already defines stable Organization and
Project permission keys. There is no single auditable contract connecting a registered route to
its permitted actor types, required context, and minimum permissions.

RFC 0049 requires user authorization to use the effective union of active role permissions. RFC
0050 keeps backend authorization authoritative but leaves its enforcement in handlers. Moving the
static portion of authorization before controller invocation changes that ownership and requires a
durable decision. Request-dependent validation and domain rules remain controller and service
responsibilities.

## Decision

Add a repository-owned static policy at
`api/web-api/config/authorization-policy.yaml`. It covers every gRPC method and HTTP route
registered by `WebApiRoute` and `ProxyApiRoute`, plus HTTP health routes registered by
`HealthCheckRoutes`. gRPC keys use canonical full method names such as
`/web_api.OrganizationService/InviteUserToOrganization`. HTTP keys use the uppercase method and
registered Gin route template. Standard-library or framework routes are excluded unless the
application registers them explicitly.

The policy has a versioned top-level schema and one entry per registered route. Each entry declares
one of four explicit authorization states:

- `public`: no credential is required;
- `authenticated`: the declared actor and context are required but no catalog permission applies,
  as with user self-service;
- `permission`: the declared actor, context, and every listed static permission are required before
  controller invocation;
- `controller`: the declared actor and context are required before invocation, while a request
  property selects the permission or resource and the controller owns that explicit check;
- `disabled`: the registered method is unavailable and the controller is never invoked.

There is no implicit default state. Permission entries declare the complete set of minimum catalog
permission keys that every request to that route requires. Controller entries are permitted only
for operations whose permission cannot be selected without typed request data; they are not a
fallback for incomplete mapping.

```yaml
version: 1

grpc:
  /web_api.AuthenticationService/Authenticate:
    authorization: public

  /web_api.OrganizationService/InviteUserToOrganization:
    authorization: permission
    actors: [user]
    context: organization
    permissions:
      - organization.members.invite
      - organization.access.manage

http:
  "POST /v1/auth/authenticate/":
    authorization: public
```

Valid contexts are `user`, `organization`, and `project`. A `user` context permits an authenticated
onboarding user without an Organization or Project. An `organization` context requires a current
Organization. A `project` context requires a Project and its owning Organization. Empty
permissions are forbidden for `public`, `authenticated`, `controller`, and `disabled` entries.
They are required and nonempty for `permission` entries.

The file is immutable after startup and is not merged with deployment environment configuration.
The loader rejects unknown fields, YAML aliases and merge keys, duplicate route keys, invalid
authorization, actor, or context values, duplicate actors, blank or duplicate permissions,
permissions absent from the active catalog, permission scope mismatches, and invalid conditional
field combinations. Startup validation compares the policy with the registered gRPC and HTTP
inventory. Missing routes, stale routes, and duplicate canonical keys within one transport fail
startup. Corresponding HTTP and gRPC operations are independent entries. An unknown route never
falls back to public or authenticate-only access.

The complete route matrix is an approval-bound appendix at
`rfcs/0051-web-api-route-authorization-policy.yaml`. RFC acceptance covers that exact matrix. The
production policy is a mechanical copy of the accepted appendix. Changes to a route's actor,
context, authorization state, or permissions require review under the authorization lifecycle;
they are not ordinary deployment configuration.

Implementation is staged. The first phase adds the complete YAML policy, strict parser, catalog
validation, and route-coverage tests without changing request enforcement. The second phase wires
the validated policy into the authentication interceptors. Enforcement begins only after the
complete policy and effective-permission resolution pass verification together.

For an enforced request, the existing project, organization, service, and user credential
authenticators establish one existing `Authentication` context. One final route-authorization
middleware runs after every credential authenticator for Gin, unary gRPC, and streaming gRPC. No
credential authenticator interprets route policy. The final middleware performs exactly one policy
lookup, validates the final actor and context, resolves permissions when required, and invokes the
controller only after success.

`actors` refers to the final `Authentication.AuthType` after service delegation. A delegated
service is evaluated as its delegated user or project actor; a non-delegated service remains a
service actor. User Organization permissions resolve through active Organization user assignments
and active `USER` roles. User Project permissions resolve through active Project user assignments
and active `USER` roles. Project actor permissions resolve through the active project credential
and its active `PROJECT_TOKEN` role. Every role, role-permission row, and permission definition must
be active and match the selected tenant, scope, and subject.

`user` context permits no catalog permissions. `organization` context accepts only Organization
permissions. `project` context accepts only Project permissions. Organization, non-delegated
service, and system actors cannot appear on a permission entry until a separate accepted binding
contract exists. Missing, inactive, mismatched, cancelled, or failed resolution denies the
request. The existing `types.Authentication` request contract does not change, and credentials,
raw tokens, role names, and feature permissions are not route-policy inputs.

The YAML owns only static route requirements. Controllers continue to own request validation,
resource and tenant matching, target-role eligibility and permission ceilings, owner protection,
and other decisions that depend on request data. Services continue to own transactions,
persistence, locking, and domain state transitions. A static route check never substitutes for a
request-dependent controller check.

## Contracts and Ownership

- The accepted RFC appendix is the reviewed source of truth for web-api route exposure, allowed
  actor types, required context, authorization state, and minimum permission keys. The production
  file must match it exactly.
- The role catalog remains the source of truth for permission existence, scope, role membership,
  and role status.
- User assignment tables and project credentials remain the source of truth for assigned roles.
- Credential middleware owns credential verification and construction of the existing request
  authentication. One final route-authorization middleware owns policy lookup,
  effective-permission resolution, and static authorization before controller invocation.
- Controllers own request validation and authorization rules that depend on request resources or
  selected role definitions.
- Services own persistence and domain operations and do not interpret route policy.
- The existing request `types.Authentication`, protobuf request and response contracts, and RPC
  names do not change.
- `controller` entries retain an explicit controller permission check selected from typed request
  data. Static legacy role-name checks are not valid controller authorization.
- Organization credentials and non-delegated service actors receive no implicit permission bypass.
  Permission entries cannot admit such actors until an accepted role-binding contract exists.

## Alternatives Rejected

- Keep permission checks only in controllers: rejected because route policy remains distributed,
  difficult to audit, and easy to omit when a handler is added.
- Store route policy in the database: rejected because it introduces mutable production policy,
  bootstrap dependencies, and operational configuration not required for the current catalog.
- Infer permissions from RPC or handler names: rejected because naming does not encode read,
  mutation, tenant, or resource semantics reliably.
- Treat missing policy as authenticate-only: rejected because a newly registered route could gain
  access without an explicit authorization decision.
- Represent every operation as a static permission entry: rejected because scope-selected role
  catalog reads and target-resource authorization require typed request data. Those exceptional
  routes use explicit `controller` entries.
- Encode request-dependent target-role rules in YAML: rejected because the static policy cannot
  safely express tenant ownership, selected role contents, assignment state, or transaction-time
  invariants.
- Keep legacy feature permissions as a fallback: rejected because that creates a second policy
  system unrelated to the role catalog.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Phase one changes no request behavior. Phase two may deny calls that currently reach controllers without the required actor, context, or permission, and explicitly disabled unsafe or unimplemented methods become unavailable. Public protobuf and HTTP shapes remain unchanged. |
| Failure and recovery | Invalid or incomplete policy fails startup before serving traffic. Authorization resolution failure denies the request through centralized errors. Rollback uses the prior binary and matching policy. |
| Security and privacy | Explicit complete coverage and fail-closed lookup reduce missing-check risk. Route policy contains permission keys only and no credentials, tokens, personal data, or secrets. |
| Data and migration | No schema or data migration. Policy validation reads the existing permission catalog after migrations are ready. |
| Concurrency and resources | Policy is parsed once and read immutably. Permission resolution must use request context cancellation and must not introduce background goroutines. |
| Operations and observability | Startup errors identify the invalid route or permission. Runtime denials identify route, actor type, context IDs, and required permission key without logging tokens, personal data, or the actor's full permission set. |

## Rollout and Rollback

1. Accept this RFC and its complete policy appendix. Add the production YAML as a mechanical copy,
   plus strict parsing, catalog validation, and registration coverage tests. Deploying this phase
   does not enforce the policy.
2. Complete RFC 0049 user-role assignment and effective-permission resolution. Verify the policy
   against migrated assignments and project-token roles.
3. After database migration and connection succeed, load and catalog-validate the policy. Construct
   credential and final authorization middleware, register native, proxy, and health routes,
   compare the complete inventories, seal the immutable policy, and only then open listeners.
4. Enable the final policy middleware and remove duplicate static authorization checks from
   controllers and services only after parity tests pass. Keep request-dependent validation,
   tenant, lifecycle, owner, and transaction-time checks.

Stop if policy coverage is incomplete, a catalog key is missing or has the wrong scope, an allowed
actor lacks a supported role-binding path, a controller entry lacks its explicit check, or parity
tests expose an unresolved behavior difference. No listener opens after startup validation fails.
The production policy is embedded or packaged atomically with the binary. Before enforcement,
rollback removes the policy loader with no request impact. After enforcement, roll back to the
prior binary-policy unit. No data rollback is required.

## Verification

Acceptance requires:

- every gRPC method registered by `WebApiRoute` or `ProxyApiRoute` and every HTTP route registered by
  `WebApiRoute` or `HealthCheckRoutes` has exactly one policy entry;
- strict YAML tests cover unknown fields, duplicate mapping keys, aliases and merge keys, invalid
  versions and enums, duplicate actors and permissions, and invalid field combinations for every
  authorization state;
- protected routes reject missing actors, missing context, unknown catalog keys, and scope
  mismatches;
- onboarding routes accept user context without Organization or Project access;
- resolver tests cover inactive assignments, roles, role permissions, and permission definitions;
  multiple-role union; duplicate keys; custom roles; wrong tenant, scope, and subject; project-token
  roles; delegated user and project services; unsupported actors; cancellation; and database
  failure;
- Gin, unary gRPC, and streaming gRPC tests cover public, authenticated, permission, controller, and
  disabled entries; missing credentials; wrong actors; missing context; missing permissions;
  unknown routes; controller-not-invoked behavior; and interceptor ordering;
- missing or stale route entries fail startup validation;
- request-dependent controller authorization and intentional custom-role behavior remain covered;
- static legacy owner/admin role checks are inventoried and removed when equivalent policy
  enforcement activates;
- the existing `types.Authentication` contract and public protobuf descriptors do not change.

Exact commands:

```text
go test ./api/web-api/config ./api/web-api/router ./pkg/middlewares ./cmd/web
go test ./api/web-api/...
git diff --check
just agent-finalize "api/web-api/config,api/web-api/router,api/web-api/internal/service/accessmanagement,pkg/middlewares,cmd/web"
```

## Review and Approval

- Challenger: Independent Codex task `/root/principal_contract`
- Outcome: Blocked after two challenge rounds
- Resolved findings: Corrected route inventory, cross-transport semantics, final middleware
  ownership, startup ordering, actor resolution, legacy service checks, feature-permission scope,
  strict schema, verification categories, and atomic packaging. Added an approval-bound complete
  policy appendix requirement.
- Open questions: Project-token routes require accepted audience and resource-binding policy;
  service-only Product Usage writes require retained caller-service provenance; controller-selected
  routes require an exact context and permission-selection contract.
- Approval reference: Pending repository owner approval
