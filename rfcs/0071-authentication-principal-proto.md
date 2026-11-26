# RFC 0071: Authentication Principal Protobuf Contract

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0054 future Principal wire representation and RFC 0070 future UI response direction only; current runtime Principal behavior remains authoritative

## Context

The immutable request `authentication.Principal` and the public authentication response currently
have different structures. Internal services also depend on credential-specific and legacy
authentication response messages. The intended final model is one versioned Principal wire
contract for User, Project credential, Organization credential, and Service assertion identities,
while the UI receives that same Principal inside a session response containing its profile, token,
and billing data.

This RFC defines only the protobuf target. It does not change runtime Principal construction,
middleware, resolvers, controllers, service behavior, UI state, route policy, or database data.

## Decision

Add these messages to `web-api.proto`:

```proto
message AuthenticationIdentity {
  string type = 1;
  uint64 id = 2 [jstype = JS_STRING];
  string issuer = 3;
}

message AuthenticationServiceActor {
  uint64 id = 1 [jstype = JS_STRING];
  string issuer = 2;
}

message AuthenticationTenant {
  uint64 organizationId = 1 [jstype = JS_STRING];
  uint64 projectId = 2 [jstype = JS_STRING];
}

message AuthenticationProofReference {
  string type = 1;
  string value = 2;
}

message AuthenticationProof {
  string method = 1;
  string audience = 2;
  AuthenticationProofReference reference = 3;
}

message PrincipalAuthorization {
  AuthenticationTenant tenant = 1;
  string tenantName = 2;
  repeated AssignedRole roles = 3;
  repeated string effectivePermissions = 4;
}

message AuthenticatedUserProfile {
  uint64 id = 1 [jstype = JS_STRING];
  string name = 2;
  string email = 3;
  string status = 4;
}

message AuthenticationPrincipal {
  AuthenticationIdentity subject = 1;
  optional AuthenticationServiceActor actor = 2;
  AuthenticationTenant selectedTenant = 3;
  AuthenticationProof proof = 4;
  repeated PrincipalAuthorization authorizations = 5;
}

message AuthenticationSession {
  AuthenticationPrincipal principal = 1;
  AuthenticatedUserProfile user = 2;
  Token token = 3;
  optional AuthBillingPlan billingPlan = 4;
}
```

`PrincipalAuthorization` becomes the target single source of truth for assigned-role display and
effective permissions. Authorizations are scope-specific: an Organization authorization has an
Organization ID and zero Project ID; a Project authorization has both IDs. There may be at most one
authorization for each exact tenant. `AuthenticatedUserProfile` intentionally has no role field.

A later runtime RFC will define immutable Go construction with these authority-composition rules:

- an empty selected tenant derives no authorities;
- an Organization-selected User derives permissions from the matching Organization authorization;
- a Project-selected User requires the matching Organization authorization and derives its union
  with the matching Project authorization when one exists. A missing Project authorization is an
  empty Project-permission contribution and does not invalidate an otherwise verified Project
  selection;
- a Project credential derives permissions from its matching Project authorization;
- Organization credential and direct Service subjects retain their accepted empty-authority
  behavior;
- an Organization-selected User without its matching Organization authorization is invalid rather
  than an empty-permission fallback;

The union is deduplicated and sorted. Organization permissions remain owned by the Organization
authorization and are not copied into each Project authorization. This protobuf-only change does
not alter the current Go Principal.

Add `optional AuthenticationSession session = 5` to `AuthenticateResponse`. Mark the existing
`data = 3` field deprecated without changing its tag or type. Mark the legacy `Authentication`,
`OrganizationRole`, `ProjectRole`, `FeaturePermission`, and `ScopedAuthentication` messages
deprecated. Existing RPCs, requests, responses, field tags, and runtime producers remain unchanged.
The new `session` field is intentionally absent until a later accepted implementation migrates
producers and consumers.

String fields preserve the accepted closed vocabulary used by the current Principal:
`user`, `project`, `organization`, `service`; `user_session`, `project_credential`,
`organization_credential`, `service_assertion`; empty, `management`, `rapida_internal`; and
`user_session`, `credential`, `signing_key`. A future decoder must reject unknown or invalid
cross-field combinations rather than treating protobuf strings as trusted values.

## Contracts and Ownership

- `protos/artifacts/web-api.proto` is the source of truth. The React artifact checkout must contain
  identical bytes and the generated Go, Python, JavaScript, and TypeScript outputs are derived.
- `AuthenticationPrincipal` owns the future wire representation of subject, optional Service actor,
  selected tenant, proof, roles, and effective permissions.
- `AuthenticationSession` owns the future UI bootstrap response and embeds the same Principal wire
  message. It separately carries a profile-only User, raw session token, and billing data. Assigned
  roles appear only in `principal.authorizations`.
- The existing messages and `AuthenticateResponse.data` remain operationally authoritative during
  this protobuf-only candidate. No code may read or populate `session` yet.
- The runtime `pkg/authentication.Principal` remains immutable and unchanged in this candidate.

## Alternatives Rejected

- Change field 3 from `Authentication` to `AuthenticationSession`: rejected because both are
  length-delimited messages and old clients could silently decode incompatible bytes.
- Put the raw token, User profile, or billing data in `AuthenticationPrincipal`: rejected because
  request authorization must not retain secrets or presentation data.
- Reuse `ScopedAuthentication`: rejected because it cannot represent proof, delegation, or complete
  tenant authorization.
- Use protobuf enums for closed values: rejected for this phase because the accepted Principal and
  route policy vocabularies are stable strings. Runtime construction remains the fail-closed owner.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Existing fields and RPCs remain wire-compatible. New clients can compile against the target messages, but `session` remains absent until the runtime migration. |
| Failure and recovery | Descriptor or artifact drift stops the candidate. Runtime behavior is unchanged, so rollback restores only proto sources and generated artifacts. |
| Security and privacy | The Principal message excludes raw credentials and secrets. The session wrapper carries the existing raw token only for the future UI response. The profile cannot duplicate role assignments. No receiver may trust string values without constructor validation. |
| Data and migration | N/A. No database or persisted-browser state changes occur. |
| Concurrency and resources | N/A. Schema generation introduces no runtime resources. |
| Operations and observability | N/A. No producer populates the new field and no deployment behavior changes. |

## Rollout and Rollback

Update both proto artifact checkouts identically, regenerate all repository artifacts, and publish
no runtime behavior change. The new field remains unset. A later accepted RFC must migrate runtime
producers, internal services, and UI consumers before removal of deprecated messages.

Stop on descriptor mismatch, source-checkout drift, a generated artifact mismatch, an existing
consumer compile failure, or unresolved Critical or Major review findings. Rollback restores the
two proto sources and regenerates artifacts. No data rollback is required.

## Verification

```bash
cmp protos/artifacts/web-api.proto sdks/react/src/clients/protos/artifacts/web-api.proto
bash bin/artifacts-generate.sh
(cd sdks/react && bash bin/artifacts-generate.sh)
go test -count=1 ./protos
(cd sdks/react && npm test -- --runInBand)
go test -run '^$' ./api/web-api/api ./pkg/clients/web ./pkg/authenticators ./cmd/web
git diff --check
just agent-finalize "rfcs/0071-authentication-principal-proto.md,protos/artifacts/web-api.proto,sdks/react/src/clients/protos/artifacts/web-api.proto,protos/web-api.pb.go,protos/web-api_grpc.pb.go,api/document-api/app/bridges/artifacts/protos/web_api_pb2.py,api/document-api/app/bridges/artifacts/protos/web_api_pb2.pyi,api/document-api/app/bridges/artifacts/protos/web_api_pb2_grpc.py,sdks/react/src/clients/protos/web-api_pb.js,sdks/react/src/clients/protos/web-api_pb.d.ts,sdks/react/src/clients/protos/web-api_pb_service.js,sdks/react/src/clients/protos/web-api_pb_service.d.ts,protos/authentication_principal_contract_test.go"
```

## Review and Approval

- Challenger: Independent GPT-6 Astra and GPT-5.6 Sol contract and security reviews, followed by a combined evidence panel
- Outcome: Approved
- Resolved findings: Replaced the role-bearing `User` with `AuthenticatedUserProfile`; defined
  scope-specific authorization uniqueness and Organization-plus-Project authority composition; and
  narrowed RFC 0054 supersession to the future wire boundary.
- Open questions: None
- Approval reference: Repository owner approved the Principal/session structure and constrained the
  current candidate to protobuf sources, generated artifacts, and descriptor coverage on 2026-11-13.
  All challengers approved pre-metadata digest
  `ea09e308a085a72cd5f14676a9c4e9066e4453fbb5582e02439bb35564ac768b`.
