# RFC 0052: Versioned Service Authorization Policies

- Status: Draft
- Owner: Access Management
- Approved by: Repository owner direction on 2026-11-11
- Date: 2026-11-11
- Supersedes: RFC 0051 draft and RFC 0050 backend-handler authorization ownership only

## Context

Web API authorization is distributed across credential middleware, controller accessors, legacy
role-name checks, and service code. The role catalog already owns stable Organization and Project
permission keys, but no reviewed contract maps each registered ingress route to its allowed actor,
required context, permission, project-token constraints, or trusted internal caller.

The blocked RFC 0051 established complete Web API route inventory but did not carry the audience
and binding facts required by RFC 0026, immediate service-caller provenance, or exact
request-selected controller checks. Its appendix also duplicated the intended runtime policy.

This change is Governed because it changes authentication and authorization ownership for HTTP and
gRPC ingress. It does not change the public protobuf contract or the existing
`types.Authentication` request structure.

## Decision

Store one immutable authorization policy per deployable service under a versioned schema:

```text
pkg/authentication/policy/v1/<deployable>.yaml
```

Version 1 begins with `web-api.yaml`. It is the sole reviewed and runtime source for every HTTP and
gRPC route served by the Web API deployment, including native services, proxy services, and health
routes. A route registered by another deployment belongs to that deployment's policy even when it
uses the same protobuf service.

Every document contains exactly:

```yaml
version: 1
deployable: web-api
grpc: {}
http: {}
```

gRPC keys are canonical full method names. HTTP keys are the uppercase method plus registered route
template. Each entry declares one authorization state:

- `public`: no credential is required;
- `authenticated`: the actor and context are required, with no role-catalog permission;
- `permission`: the actor, context, and every listed permission are required;
- `controller`: middleware authenticates the allowed actor, then the controller must select and
  execute one documented request-dependent authorization path before resource or service work;
- `disabled`: the registered operation returns unavailable or unimplemented without invoking its
  controller.

The allowed actor values are `user`, `project`, `organization`, `service`, and `system`. Static
contexts are `user`, `organization`, and `project`. `context: controller` is valid only for a
`controller` entry and means the typed request selects the concrete context using the contract
below. No `any` context or implicit fallback exists.

Permission entries contain a nonempty unique permission list. Every key must exist as an active
permission definition and match the declared Organization or Project context. Public, disabled,
authenticated, and controller entries do not contain static permissions.

### Project-token policy

Every entry whose actors include `project` contains:

```yaml
project_token:
  audience: management
  binding: project
```

The only supported audience requirements are `management` and `runtime`. Management routes require
`management`. Runtime invocation and scope authorization require `runtime`. RFC 0026's acceptance
matrix permits a verified management token on a runtime route, but never permits a runtime-only
token on a management route.

Version 1 supports the project credential model that exists after migration 000016. An active
project credential is treated as a confidential management token bound to its persisted Project
and Organization. Its role definition supplies only the effective permission set. Runtime routes
still enforce the persisted Project binding. Public-widget, channel, deployment-specific, and
other narrower token classes remain unavailable until their metadata and verification path are
implemented under a separate accepted contract.

Project authentication owns credential validity, Project and Organization binding, assigned
`PROJECT_TOKEN` role, effective permissions, and the inferred management audience. It stores those
verified facts in a private authorization sidecar. It does not add fields to
`types.Authentication`.

### Internal service-caller policy

An entry may require exact immediate callers:

```yaml
caller_services:
  - issuer: workflow-api
    actor_id: 9007
```

Service assertion verification stores the verified immediate service issuer and actor ID in a
private authorization sidecar before applying delegation. Delegation may replace the effective
actor in `types.Authentication`, but it never replaces or discards the immediate caller sidecar.
When `caller_services` is present, a direct user, project credential, organization credential, or
different service caller is denied even if the effective delegated actor is otherwise allowed.

Web API verifies inbound service assertions with a secret-backed trusted-caller registry. Each
issuer entry contains exactly one actor ID and one or more HMAC keys identified by unique key IDs.
The assertion header contains the key ID, and its claims contain issuer and actor ID. The unverified
issuer and key ID may select a registry entry, but grant no authority. Web API then verifies the
HS256 signature, internal audience, issued-at and expiry bounds, issuer, configured actor ID, and
key ID before recording immediate caller identity. An unknown issuer, actor ID, or key ID is denied.
The general `AppConfig.Secret` and a key shared by multiple issuers are not valid trusted-caller
keys.

For rotation, the receiver is provisioned with the new unique key ID before the issuer signs with
it. The receiver may accept the issuer's immediately previous key during the existing five-minute
maximum assertion lifetime. The previous key is removed only after all issuer replicas use the new
key and the maximum lifetime has elapsed. Other services never receive the issuer's keys.

`ProductUsageService/CreateProductUsage` requires the exact `workflow-api` service actor 9007. It
accepts delegated user or project authorization for the selected Project and accepts a
non-delegated service actor. A delegated project actor also satisfies the declared runtime
project-token policy. Authorization completes before the controller is invoked.

### Controller-selected routes

Controller entries are limited to these selections:

| Route | Selection | Required decision |
| --- | --- | --- |
| `AuthenticationService/ScopeAuthorize` | `organization` | Organization actor and matching Organization context; authenticated only. |
| `AuthenticationService/ScopeAuthorize` | `project` | Project actor, matching Project context, and runtime project-token policy. |
| `AccessManagementService/ListPermissions` | `organization` | `organization.roles.read` in the current Organization. |
| `AccessManagementService/ListPermissions` | `project` | `project.roles.read` in the selected Project. |
| `AccessManagementService/ListRoles` | `organization` | `organization.roles.read` in the current Organization. |
| `AccessManagementService/ListRoles` | `project` | `project.roles.read` in the selected Project. |
| `AccessManagementService/ListRoles` | `project_token_selector` | Authenticated user, current Organization, and the exact Project plus `PROJECT_TOKEN` filters accepted by RFC 0039. |
| `AccessManagementService/GetRole` | `organization` | `organization.roles.read` for an eligible Organization role in the current Organization. |
| `AccessManagementService/GetRole` | `project` | `project.roles.read` for an eligible Project role in the selected Project. |
| `AccessManagementService/GetRole` | `project_token_selector` | Authenticated user and an active eligible Project-scoped `PROJECT_TOKEN` role allowed by RFC 0039. |
| `ProjectService/CreateProjectCredential` | `organization` | `organization.project_access_tokens.create` in the current Organization. |
| `ProjectService/CreateProjectCredential` | `project` | `project.access_tokens.create` in the exact selected Project. |
| `ProjectService/RevokeProjectCredential` | `organization` | `organization.project_access_tokens.revoke` in the current Organization. |
| `ProjectService/RevokeProjectCredential` | `project` | `project.access_tokens.revoke` in the exact selected Project. |

The controller calls the shared authorizer with one named selection and the typed request facts.
Missing, ambiguous, or undeclared selection denies the request. Tenant and resource validation
occurs before disclosure or mutation, and every denial uses centralized `pkg/errors` behavior.

Selection is deterministic after controller input validation:

- `ListPermissions` selects `project` only for the exact `scope = PROJECT` criterion. It selects
  `organization` for `scope = ORGANIZATION` or no scope criterion. Project selection requires the
  authenticated current Project; organization selection uses the current Organization.
- `ListRoles` selects `project_token_selector` only when the request contains both exact
  `scope = PROJECT` and `subject = PROJECT_TOKEN` criteria. Otherwise exact `scope = PROJECT`
  selects `project`; `scope = ORGANIZATION` or no scope criterion selects `organization`. Other
  criteria do not alter the selection. Duplicate, partial, or malformed selector criteria fail
  validation before authorization.
- `GetRole` first performs an authenticated, Organization-bounded metadata lookup that reads only
  role ID, owning Organization ID, scope, subject, definition type, and status. It selects
  `project_token_selector` for an active eligible Project-scoped `PROJECT_TOKEN` role,
  `project` for another active Project role, and `organization` for an active Organization role.
  The lookup returns no role data to the caller. Unknown, archived, cross-tenant, wrong-scope, and
  wrong-subject rows use the same unavailable result. The controller authorizes the selected path
  before loading permissions or constructing the response.
- Project credential create and revoke validate the requested Project ID and its current
  Organization binding, then allow either the declared Organization permission or the declared
  Project permission for that exact Project. The authorizer evaluates this as one any-of decision;
  failure returns one generic denial and does not reveal which permission or resource test failed.

`ProjectService/GetAllProjectCredential` is disabled because its current response includes stored
credential keys while the catalog defines metadata-read but no secret-read authority.
`VaultService/GetCredential` and `GetOauth2Credential` are disabled because they return provider
credential values without a secret-read permission. `VaultService/DeleteCredential` is disabled
because its response exposes the deleted stored value. Unimplemented deployment and audit RPCs are
also disabled. Restoring any disabled operation requires a separately accepted response and
permission contract.

### Middleware and startup

Credential middleware remains the sole owner of credential verification and construction of the
existing request `types.Authentication`. Private authorization sidecars retain project-token facts
and immediate service-caller identity. One final route-authorizer runs after all credential
middleware for Gin, unary gRPC, and streaming gRPC. It owns policy lookup, actor and context checks,
caller checks, token audience and binding checks, static permission checks, and controller-entry
admission.

Startup order is:

1. load application configuration and strictly parse the embedded service policy;
2. run migrations and connect the role catalog database;
3. validate permission keys, scopes, caller identities, and registered binding selectors;
4. construct credential and final authorization middleware;
5. register native, proxy, and health routes;
6. compare exact HTTP and gRPC inventories with the policy and seal it;
7. open listeners only after every validation succeeds.

Unknown fields, YAML aliases or merge keys, duplicate keys or values, blank values, unknown enums,
invalid conditional fields, absent permission keys, scope mismatch, missing route entries, stale
entries, missing project-token facts, unsupported caller provenance, missing controller selection,
and unknown runtime routes fail closed. Cross-transport HTTP and gRPC routes are independent keys.

## Contracts and Ownership

- `pkg/authentication/policy/v1/<deployable>.yaml` owns the route authorization contract for one
  deployment. No accepted RFC appendix or environment file duplicates it; RFC 0051's blocked draft
  appendix has no runtime or approval authority.
- Secret-backed application configuration owns each deployable's inbound trusted-caller registry.
  The policy owns allowed issuer and actor-ID tuples, but never contains signing keys or key IDs.
- Credential authenticators own credential validity and verified authorization sidecars.
- The role catalog and assignment tables own active role and permission resolution.
- Final route middleware owns every static policy decision before controller invocation.
- Controllers own only declared request-selected authorization plus tenant, resource, and payload
  validation using `pkg/validator` and centralized errors.
- Services own persistence, locks, transactions, and domain state transitions. Static owner/admin
  role checks are removed when equivalent route authorization activates.
- The existing `types.Authentication` and public protobuf contracts remain unchanged.
- Legacy feature permissions are not policy input or fallback. Their public removal remains a
  separate governed change.

## Alternatives Rejected

- One global policy: rejected because it couples unrelated deployments and prevents atomic route
  inventory validation.
- One file per protobuf service: rejected because a deployment registers several native and proxy
  services and would need unsafe runtime merging.
- Put permissions or token facts in `types.Authentication`: rejected because the existing request
  contract remains stable and private verified sidecars have narrower ownership.
- Admit project actors using role permissions alone: rejected because RFC 0026 also requires token
  audience and resource binding.
- Authorize Product Usage from the delegated actor alone: rejected because direct callers could
  impersonate the intended internal write path.
- Map secret-returning routes to metadata-read or use permissions: rejected because neither grants
  authority to disclose stored secrets.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Public request and response shapes remain stable. Inbound service callers must migrate from the shared application secret to their issuer-specific keys before enforcement. Enforcement intentionally enables eligible project credentials on declared routes and disables unsafe secret-returning or unimplemented routes. |
| Failure and recovery | Invalid policy or incomplete inventory prevents listener startup. Missing runtime facts deny requests. Rollback deploys the previous binary and embedded policy together. |
| Security and privacy | Static permission checks move before controllers, project tokens require audience and binding, and internal writes require an exact caller tuple verified by a key unique to that issuer. Secret-returning routes are disabled. Tokens, signing keys, secrets, and full granted permission sets are never logged. |
| Data and migration | No new migration. Enforcement waits for migrations 000016 and 000017. Existing project credentials use the documented management and Project-binding compatibility interpretation. |
| Concurrency and resources | Policies are parsed once and read immutably. Resolution propagates request cancellation and creates no background goroutines. |
| Operations and observability | Startup failures identify policy path and invalid route or key. Runtime denials record route, actor type, required permission, audience, and context IDs without credentials or personal data. |

## Rollout and Rollback

1. Add the exact versioned policy, strict parser, schema tests, and route inventory validation without
   enforcement.
2. Provision issuer-specific inbound service keys, migrate every Web API service caller from the
   shared application secret, and add verified project-token and service-caller sidecars plus
   controller-selection parity tests.
3. Complete the RFC 0049 user assignment and effective-permission cutover and verify migrations
   000016 and 000017.
4. Enable the final authorizer and remove equivalent static legacy role checks. Retain all
   request-dependent, tenant, resource, owner, lifecycle, and transaction-time checks.
5. Deploy the binary and embedded policy atomically.

Stop if any route, permission, binding selector, caller, controller selection, active assignment,
or credential fact cannot be validated. Before enforcement, rollback removes unused policy code.
After enforcement, deploy the prior binary-policy unit. No data rollback is required.

## Verification

Acceptance requires:

- exact coverage of all registered native, proxy, and health HTTP and gRPC routes;
- strict rejection of unknown fields, duplicate YAML keys or values, aliases, merge keys, invalid
  enums, missing conditional fields, stale routes, and missing routes;
- active-only user and project-token role resolution, multiple-role union, custom roles, wrong
  tenant, scope, and subject denial, cancellation, and database failure coverage;
- RFC 0026 audience acceptance and Project-binding tests for every project-token route, including
  delegated project actors;
- exact `workflow-api` actor 9007 Product Usage admission and denial for direct, absent, forged,
  wrong-key, stale-key, shared-secret, or different callers, plus issuer-key rotation coverage;
- deterministic list-filter, role-metadata, and project-credential any-of selection tests;
- every controller selection executes authorization before protected disclosure or write, with the
  bounded `GetRole` metadata classification returning no protected role content;
- Gin, unary gRPC, and streaming gRPC denial tests prove controllers are not invoked;
- unchanged `types.Authentication` and protobuf descriptor tests;
- independent high-risk review with no unresolved Critical or Major findings.

Exact commands:

```text
go test ./pkg/authentication/... ./pkg/middlewares ./api/web-api/router ./cmd/web
go test ./api/web-api/...
git diff --check
just agent-finalize "pkg/authentication,pkg/middlewares,api/web-api,cmd/web"
```

## Review and Approval

- Challenger: Codex task `task_1bb2aa4a5360`
- Outcome: Blocked after the second challenge with one Critical and four Major findings
- Prior challenge: Codex task `task_9505974fedca` returned two Major findings and no Critical
  findings. This candidate is correction round one.
- Resolved findings: Owner selected per-deployable versioning, retained project-token actors with
  audience and binding enforcement, required pre-controller Product Usage caller authorization,
  and required exact controller-selected contexts. The first Codex challenge additionally required
  issuer-specific service verification and deterministic Access Management controller selection;
  both prior Major findings are resolved in this candidate.
- Open questions: The owner must decide the selected-role permission ceiling for project-token
  creation, recent-MFA handling for authority-bearing secret creation, Organization-or-Project
  provider-credential authorization, the issuer-key-compatible rollback floor, and the required
  impact preview and confirmation for reducing a custom role assigned to live project credentials.
- Approval reference: Repository owner approval in this session on 2026-11-11
