# RFC 0069: User Session Authentication Response

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0049 public Authentication/User duplication and OrganizationRole/ProjectRole response shapes, plus RFC 0052 and RFC 0054 public authentication-response shape and feature-permission deprecation boundaries only

## Context

`Authenticate`, `RegisterUser`, `Authorize`, and social authentication currently return an
`Authentication` message that combines a User profile, session token, Organization role projection,
Project role projections, billing data, and legacy feature permissions. The message uses singular
role-oriented names for tenant memberships, duplicates Organization assignments inside `User`, and
keeps feature permissions even though accepted route authorization does not consume them.

The internal immutable `authentication.Principal` is request-scoped security state and must not be
serialized as an authentication response. Clients instead need session identity, a token, available
tenant memberships, assigned-role display data, and the canonical effective-permission projections
retained by RFC 0050.

This is an intentional public protocol break. The repository has no released compatibility window
for this branch, and the owner requested one clean migration without a legacy response fallback.

## Decision

Replace the public `Authentication` data message with `UserSession`:

```proto
message AuthenticatedUser {
  uint64 id = 1 [jstype = JS_STRING];
  string name = 2;
  string email = 3;
  string status = 4;
}

message OrganizationMembership {
  uint64 organizationId = 1 [jstype = JS_STRING];
  string organizationName = 2;
  repeated AssignedRole roles = 3;
  repeated string effectivePermissions = 4;
}

message ProjectMembership {
  uint64 projectId = 1 [jstype = JS_STRING];
  string projectName = 2;
  repeated AssignedRole roles = 3;
  repeated string effectivePermissions = 4;
}

message UserSession {
  AuthenticatedUser user = 1;
  Token token = 2;
  optional OrganizationMembership organization = 3;
  repeated ProjectMembership projects = 4;
  optional AuthBillingPlan billingPlan = 5;
}
```

`AuthenticateResponse.data` becomes `optional UserSession data = 3`. The response envelope and all
Authentication Service method request types remain unchanged. `Authenticate`, `RegisterUser`,
`Authorize`, `Linkedin`, `Google`, and `Github` return the new data type atomically.

`GetOrganizationResponse.role = 4` and `CreateOrganizationResponse.role = 4` become
`OrganizationMembership membership = 4`. Their Organization data and Error fields remain unchanged.
The Organization controllers, SDK wrappers, onboarding, workspace overview, and contract tests move
to the new field in the same candidate.

Remove the public `Authentication`, `OrganizationRole`, `ProjectRole`, and `FeaturePermission`
messages. Their names must not be reused. There is no compatibility field, alternate response,
dual-write path, or runtime fallback.

An onboarding User session may omit `organization` and return no `projects`. A successful session
always has a nonzero User ID and a nonblank token. An Organization membership and every Project
membership contain canonical assigned-role display data plus the effective permission keys produced
by the RFC 0049 authentication owner. Projects are ordered by Project ID; roles and permission keys
retain their established deterministic ordering. The response never contains the internal
Principal, credential proof, raw role-definition objects, or route policy.

The console renames its state to `organizationMembership`, `projectMemberships`, and
`currentProjectMembership`. It removes `featurePermissions` and `isFeatureEnable`. The console may
retain effective-permission state for the future RFC 0050 rollout but does not add new permission
gating in this change.

## Contracts and Ownership

- The proto repository owns the `UserSession` wire contract. Generated Go, Python, JavaScript, and
  TypeScript artifacts are derived outputs and are never edited by hand. The root and React SDK
  proto artifact checkouts must point to the same source revision and contain identical
  `web-api.proto` bytes before generation.
- The Web API User authentication service remains the sole owner of canonical active assignments,
  role display projections, and Organization and per-Project effective permission unions.
- Web API authentication controllers map that canonical domain result to `UserSession`; they do not
  query role or permission tables and do not serialize `authentication.Principal`.
- The React SDK owns generated client exposure. The console authentication store owns persisted
  client session selection and migrates atomically to the new generated types.
- Internal callers of `Authorize` consume `UserSession` after regeneration. This RFC does not change
  credential headers, middleware selection, Principal resolution, route policy, service assertions,
  or database schema.
- RFC 0050 remains authoritative: effective permissions stay public, but the console does not use
  them to hide actions in this release.
- RFC 0049 remains authoritative for canonical assignment, activation, validation, transaction,
  and effective-permission ownership. This RFC supersedes only its public response message names,
  field layout, and duplicate Organization roles inside the authenticated User profile.

## Alternatives Rejected

- Rename fields inside `Authentication`: rejected because the legacy type would retain ambiguous
  ownership and feature-permission history.
- Serialize `authentication.Principal`: rejected because it is a request-scoped security contract,
  contains selected-tenant proof facts, and is not a User session or client profile.
- Keep deprecated fields for old clients: rejected because the requested migration is clean and the
  branch has no compatibility window.
- Return only roles or only permissions: rejected because assigned roles support membership display
  while effective permissions are the accepted canonical projection retained by RFC 0050.
- Put Organization roles back inside `AuthenticatedUser`: rejected because membership state has a
  separate tenant owner and would duplicate `OrganizationMembership.roles`.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | This is an intentional atomic break for generated clients and all `Authentication` consumers. Requests, RPC names, and the response envelope remain stable. |
| Failure and recovery | A partial generator or consumer rollout does not compile. Deployment stops unless generated artifacts, Web API producers, internal clients, SDK, console, and flow tests agree on `UserSession`. |
| Security and privacy | Legacy feature permissions are removed. Principal proof and raw credentials remain private. The response keeps the existing User profile, token, membership, role-display, billing, and effective-permission exposure only. |
| Data and migration | No database data changes. The console uses a new `rpd::__user-session-v1` storage key and never reads the old key. The persistent internal `Authorize` Redis cache uses a new `user-session-v1` namespace so old `Authentication` JSON cannot decode as a partial `UserSession`. |
| Concurrency and resources | N/A. The response remains an immutable request result and introduces no retained resource or concurrent owner. |
| Operations and observability | Existing authentication success and failure status remain. Contract drift checks and end-to-end login flows become the rollout stop signal. |

## Rollout and Rollback

Update the proto source, regenerate every repository artifact, migrate Web API producers and all
repository consumers, then deploy the Web API, SDK-backed console, and internal services from one
coordinated candidate. Do not publish an intermediate generated package.

The internal Web authentication client changes its `Authorize` cache key from the unversioned
`Authorize` namespace to `Authorize/user-session-v1`. It never reads, converts, or deletes the old
entry. A rollback uses the prior unversioned namespace and therefore does not decode new-session
JSON. Tests seed the old key and prove that the new client fetches and caches a `UserSession` under
the versioned key instead.

The console changes its persisted authentication key from `rpd::__user` to
`rpd::__user-session-v1`. It does not migrate or decode the old value. A rollback reads only the old
key. Store tests seed the old key and prove that the new store starts unauthenticated.

Stop on descriptor drift, any remaining legacy generated accessor, failure of an authentication
method to produce `UserSession`, stale persisted-console-state behavior, or an unresolved Critical
or Major review finding. Rollback restores the prior proto source, all generated outputs, producers,
and consumers together. No data rollback is required; the console clears incompatible persisted
session state.

## Verification

Acceptance requires:

- descriptors contain `AuthenticatedUser`, `OrganizationMembership`, `ProjectMembership`, and
  `UserSession`, and do not contain the four removed messages;
- `AuthenticateResponse.data` resolves to `UserSession` for every authentication method;
- onboarding, Organization-only, and Project-membership sessions have deterministic projections;
- no production consumer calls a removed accessor or reads feature permissions;
- old persisted console state fails closed and is cleared;
- old unversioned Redis cache data is ignored and rollback keeps its prior cache namespace;
- generated outputs exactly match their proto sources;
- focused backend, SDK, UI store/access, and cross-language authentication flow tests pass.

Declared commands:

```bash
bash bin/artifacts-generate.sh
(cd sdks/react && bash bin/artifacts-generate.sh)
go test -count=1 ./api/web-api/api ./api/web-api/internal/service/user ./pkg/clients/web ./pkg/authenticators ./protos
go test -race -count=1 ./api/web-api/api ./api/web-api/internal/service/user ./pkg/clients/web ./pkg/authenticators ./protos
go vet ./api/web-api/api ./api/web-api/internal/service/user ./pkg/clients/web ./pkg/authenticators ./protos
(cd sdks/react && npm test -- --runInBand)
(cd ui && yarn test --watchAll=false src/stores/auth src/access src/hooks)
(cd ui && yarn verify:ui)
test "$(git -C protos/artifacts rev-parse HEAD)" = "$(git -C sdks/react/src/clients/protos/artifacts rev-parse HEAD)"
cmp protos/artifacts/web-api.proto sdks/react/src/clients/protos/artifacts/web-api.proto
! rg -n '\b(Authentication|OrganizationRole|ProjectRole|FeaturePermission|FeaturePermissions)\b|\b(GetOrganizationRole|GetProjectRoles|GetFeaturePermission|GetFeaturePermissions|IsFeatureEnabled)\b' api/web-api/api/auth.go api/web-api/api/organization.go api/web-api/internal/service/user/auth.type.go pkg/clients/web pkg/authenticators/user_authenticator.go pkg/types/user_principle.go
! rg -n '\b(Authentication|OrganizationRole|ProjectRole|FeaturePermission)\b' sdks/react/src/clients/protos/web-api_pb.d.ts
! rg -n 'getOrganizationrole|getProjectrolesList|getFeaturepermissionsList|featurePermissions|isFeatureEnable' ui/src tests/flows sdks/react/src --glob '!**/node_modules/**' --glob '!**/clients/protos/**'
just ci-flow-contract
bash tests/flows/run-local.sh nodejs signup-organization-project
bash tests/flows/run-local.sh react signup-organization-project
git diff --check
just agent-finalize "rfcs/0069-user-session-response.md,protos/artifacts/web-api.proto,protos/web-api.pb.go,protos/web-api_grpc.pb.go,protos/authentication_session_contract_test.go,api/document-api/app/bridges/artifacts/protos/web_api_pb2.py,api/document-api/app/bridges/artifacts/protos/web_api_pb2.pyi,api/document-api/app/bridges/artifacts/protos/web_api_pb2_grpc.py,api/web-api/api/auth.go,api/web-api/api/auth_test.go,api/web-api/api/organization.go,api/web-api/api/organization_test.go,api/web-api/internal/service/user/auth.type.go,api/web-api/internal/service/user/auth.type_test.go,pkg/types/user_principle.go,pkg/types/user_principle_test.go,pkg/clients/web/auth_client.go,pkg/clients/web/auth_client_test.go,pkg/authenticators/user_authenticator.go,pkg/authenticators/user_authenticator_test.go,sdks/react/src/clients/protos/artifacts/web-api.proto,sdks/react/src/clients/protos/web-api_pb.js,sdks/react/src/clients/protos/web-api_pb.d.ts,sdks/react/src/clients/protos/web-api_pb_service.js,sdks/react/src/clients/protos/web-api_pb_service.d.ts,sdks/react/src/index.ts,sdks/react/src/clients/auth.ts,sdks/react/src/clients/organization.ts,sdks/react/tests/clients/user-session-contract.test.ts,ui/src/stores/auth/authentication.store.ts,ui/src/stores/auth/authentication.store.test.ts,ui/src/types/types.authentication.ts,ui/src/access/use-access.ts,ui/src/access/__tests__/access.test.tsx,ui/src/hooks/use-credential.ts,ui/src/hooks/__tests__/use-credential.test.tsx,ui/src/app/pages/workspace/overview/index.tsx,ui/src/app/pages/user-onboarding/user-organization/index.tsx,ui/e2e/fixtures.js,tests/flows/signup-organization-project/react/signup-organization-project.js,tests/flows/signup-organization-project/nodejs/signup-organization-project.js"
```

## Review and Approval

- Challenger: Independent GPT-6 Astra and GPT-5.6 Sol contract and security reviews
- Outcome: Approved
- Resolved findings: Added Organization response migration, narrow RFC 0049 supersession,
  versioned Redis and browser persistence namespaces, artifact parity checks, enforcing legacy
  scans, concrete finalization paths, and executable Node.js and React onboarding flows.
- Open questions: None
- Approval reference: Repository owner requested a clean migration to a new authentication response
  type and approved the scoped roles and effective-permissions response on 2026-11-13. Challengers approved digest
  `4badafa163a0c5f2c9034751ce889d9d91c3a7fcd58cb7783997853c2dfc9268`.
