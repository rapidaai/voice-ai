# RFC 0053: Safe Authorization Enforcement Prerequisites

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-11
- Supersedes: RFC 0052 draft; RFC 0050 backend-handler authorization ownership only; RFC 0039 project-credential creation availability after authorization enforcement only

## Context

RFC 0052 defined a complete per-deployable Web API authorization policy and passed mechanical
coverage for all 139 registered gRPC methods and 18 registered HTTP routes. Its second independent
Codex challenge confirmed the route inventory, permission scopes, project-token declarations, and
the corrections for deterministic controller selection and issuer-specific service provenance.
The challenge still found one Critical and four Major conflicts with accepted access policy:

- `CreateProjectCredential` creates and assigns Project authority but lacks the caller permission
  ceiling required by the incorporated RFC 0026 policy;
- project-token and provider-credential secret creation lacks the required server-derived recent
  MFA assurance;
- provider-credential routes omit the accepted Organization-or-Project alternative authorization;
- rollback to the old shared-secret service authenticator would restore forgeable caller
  provenance or cause an outage after issuers migrate;
- custom project-token roles can be reduced while assigned to live credentials without the impact
  preview and confirmation required before token enforcement.

The repository has no accepted recent-MFA contract. RFC 0026 requires secret-creating and
authority-granting operations to remain disabled until that contract exists. RFC 0029 explicitly
deferred MFA only for its custom-role lifecycle API; it did not waive MFA for project-token or
provider-credential secret creation.

This is a Governed authentication and authorization change. It preserves the existing public
authentication request and `types.Authentication` structure, and it does not implement MFA.

## Decision

Retain RFC 0052's per-deployable policy source, strict schema, fail-closed inventory validation,
project-token audience and Project binding, issuer-specific service-caller registry, private
authorization sidecars, deterministic Access Management selections, and controller/service
ownership boundaries. The sole Web API policy remains:

```text
pkg/authentication/policy/v1/web-api.yaml
```

The policy changes below are mandatory before enforcement.

### Secret creation remains disabled

Version 1 marks these registered operations `disabled`:

- `ProjectService/CreateProjectCredential`;
- `VaultService/CreateProviderCredential`;
- `ConnectService/GeneralConnect`;
- every `/v1/connect-action/*`, `/v1/connect-crm/*`, and `/v1/connect-knowledge/*` HTTP route.

Disabled operations return unavailable without invoking a controller. This intentionally pauses new
project-token and provider-credential creation when policy enforcement activates. Existing active
project credentials remain eligible authentication actors on policy routes that declare `project`,
subject to their active assigned role, effective permissions, management or runtime audience, and
persisted Organization and Project binding.

Re-enabling any disabled project-token creation route requires a separately accepted recent-MFA
contract and all of these checks in one locked transaction:

1. authenticate a user and verify server-derived recent MFA from a private assurance sidecar;
2. require either `organization.project_access_tokens.create` for the current Organization and an
   active target Project in that Organization, or `project.access_tokens.create` in that exact
   Project;
3. require `project.access.manage` in the exact target Project;
4. require every effective permission contained by the selected active Project-scoped
   `PROJECT_TOKEN` role;
5. lock and reread the Organization, Project, caller assignments, selected role, active role
   permissions, and credential state before inserting the credential and immutable role relation;
6. commit the secret and role assignment atomically and return the secret only after commit.

A lifecycle-only permission can never create an assigned token. Missing assurance, any absent
permission, a stale role, tenant mismatch, cancellation, deadline, insert failure, or commit failure
writes no credential and returns no secret.

Re-enabling provider-credential creation or OAuth connection requires server-derived recent MFA
and either the matching Organization permission or Project permission. The Organization path also
requires the selected active Project to belong to the current Organization. The Project path
requires the permission in that exact Project. The callback must carry the already verified actor,
assurance, Organization, Project, provider, and redirect binding in authenticated state; it cannot
accept client-asserted assurance or change scope.

### Provider metadata authorization

`VaultService/GetAllOrganizationCredential` remains enabled as a `controller` route because its
response removes `VaultCredential.value` and returns metadata only. The controller validates the
typed request, selects the current Project, and authorizes one decision:

- a user is allowed by `organization.provider_credentials.read_metadata` in the current
  Organization with the selected active Project constrained to that Organization, or by
  `project.provider_credentials.read_metadata` in that exact Project;
- a project actor is allowed only by `project.provider_credentials.read_metadata` in its bound
  Project and must satisfy the declared management audience and Project binding;
- organization and project alternatives form one any-of decision. Denial does not disclose which
  permission or Project predicate failed.

`project.provider_credentials.use` remains the only authority for consuming credential values in
Project content or runtime configuration. No Organization permission grants use. Existing routes
that directly return a stored credential value remain disabled.

The metadata-list controller accepts at most two unique criteria and combines them with AND. The
only accepted predicates are `name contains <value>` for a case-insensitive literal credential-name
substring and `provider = <value>` for an exact provider key. Values are trimmed, nonblank, contain
no NUL, and contain at most 200 Unicode code points. The name predicate treats `%`, `_`, and `\` as
literals. Nil criteria, duplicates, unsupported keys or logic, and invalid values fail through the
centralized invalid-criteria response before authorization or service work. The service maps the
validated slice through fixed key switches and parameterized predicates. It never uses a caller key
or logic value as a SQL identifier or operator, and it applies every criterion after the immutable
Organization, Project, and active-status predicates.

A missing pagination message defaults to page 1 with 25 items. A present message requires a page of
at least 1 and page size from 1 through 100. Count and page use the same tenant and filter predicate
in one repeatable-read, read-only transaction. Results order by creation time descending and ID
descending. Invalid pagination fails before service work through the centralized pagination error.

### Custom-role reduction preview and confirmation

`ReduceCustomRolePermissions` becomes a two-phase operation on the existing RPC. The protobuf
contract adds these fields and messages without changing existing field numbers:

```text
message RoleReductionImpact {
  uint64 organizationId = 1 [jstype = JS_STRING];
  uint64 projectId = 2 [jstype = JS_STRING];
  RoleSubject subject = 3;
  uint64 subjectId = 4 [jstype = JS_STRING];
  string assignmentStatus = 5;
}

message RoleReductionPreview {
  RoleDefinition resultingRole = 1;
  repeated RoleReductionImpact impacts = 2;
  string confirmationDigest = 3;
}

message ReduceCustomRolePermissionsRequest {
  // Existing fields 1 through 3 remain unchanged.
  string confirmationDigest = 4;
  bool preview = 5;
}

message ReduceCustomRolePermissionsResponse {
  // Existing fields 1 through 5 remain unchanged.
  RoleReductionPreview preview = 6;
}
```

`preview = true` requires a blank confirmation digest and performs no mutation. It locks no rows,
reads one repeatable-read snapshot, and returns the predicted resulting role with permissions
ordered by permission key then ID. Impacts include every active or invited Organization or Project
user assignment referencing the custom role and every active project credential referencing it.
User impacts identify the concrete Organization or Project and user subject ID. Project-token
impacts identify the concrete Organization, Project, and credential ID. Archived assignments and
archived credentials are excluded.

The confirmation digest is lowercase hexadecimal SHA-256 over a deterministic length-prefixed
encoding of the role ID, owning Organization, scope, subject, status, sorted active permission-pair
IDs, sorted included assignment and credential facts, sorted requested permission IDs, and sorted
resulting permission IDs. It is opaque to clients and is not persisted.

`preview = false` requires a nonblank confirmation digest. In one transaction the service first
reads candidate impacts and caller assignments, then follows the global lock order: Organization;
every currently affected Project in ascending ID order; the authorizing user's membership and every
affected user membership in ascending ID order; all caller and target role definitions in ascending
ID order with their active permission rows; active project credentials as token subjects in
ascending ID order; then caller and affected assignment rows in ascending ID order. It rereads the
complete caller and impact graphs, revalidates active membership, recomputes the caller's effective
Organization permission union from locked active assignments, and requires
`organization.roles.update` immediately before mutation. It then recomputes the same canonical
impact snapshot and compares the digest in constant time before archiving any permission pair.

Assignment and membership writers use the same Organization, Project, membership, role, token, and
assignment order. A concurrent caller-permission revocation therefore commits before confirmation
and causes denial, or waits until the reduction transaction finishes. A concurrent new affected
assignment either precedes the confirmed snapshot or observes the reduced role after this
transaction commits. A missing, malformed, or stale digest, inactive membership, or lost caller
permission fails with the centralized invalid-selection, failed-precondition, or permission-denied
error and writes nothing. Success archives only the requested active permission pairs and returns
the updated role with existing correlation behavior. Ambiguous delivery is reconciled through
`GetRole`; replay after commit cannot remove an already archived pair.

This two-phase rule applies to every custom role, including roles with no assignments. It introduces
no preview table, token, cache, expiry worker, or new persistence column. The existing monotonic
reduction exception remains available without recent MFA; the preview, reason, caller permission,
and confirmation are still mandatory.

### Service assertion migration and rollback floor

Web API inbound service authentication uses the RFC 0052 trusted-caller registry. Each issuer has
one configured actor ID and issuer-specific HMAC keys selected by key ID. Final policy enforcement
accepts only assertions verified by this registry; `AppConfig.Secret` never produces trusted
immediate-caller provenance.

Migration uses three receiver states:

1. deploy a compatibility receiver with policy enforcement off, existing handler authorization on,
   and both legacy shared-secret verification and issuer-specific verification available;
2. migrate every Web API service caller to its issuer-specific key and verify the recorded
   registry provenance;
3. deploy and retain a unique-key-only safety-floor receiver. It enforces `disabled` entries and
   exact `caller_services` constraints, while existing effective-permission handlers continue to
   authorize every other route. This immutable artifact is the rollback floor.

Only after the third state is healthy may the enforcement binary enable the remaining static and
controller policy decisions and remove equivalent handler checks. Post-enforcement rollback deploys
the unique-key-only safety floor, so secret creation stays disabled and Product Usage retains exact
caller provenance. It never deploys a binary that requires or accepts the shared application secret
for inbound service identity. Issuer key rollback follows the current-and-previous key rotation
window and never restores a key shared by multiple issuers.

## Contracts and Ownership

- `pkg/authentication/policy/v1/web-api.yaml` owns the complete Web API route decision. RFC examples
  and environment files do not duplicate it.
- Authentication owns future recent-MFA derivation. Authorization may consume only a verified
  private assurance sidecar and never reads assurance from public request fields.
- The project-credential controller owns input validation with `pkg/validator`; the project service
  owns the future locked create-and-assign ceiling, atomic persistence, and commit authority.
- Provider-credential controllers own typed request and callback-state validation; the vault service
  owns credential persistence. The route authorizer owns the Organization-or-Project any-of check.
- The Access Management controller owns preview and confirmation input validation and centralized
  error mapping. The Access Management service owns snapshot reads, digest construction, lock order,
  stale-preview rejection, and atomic permission reduction.
- The role catalog, user assignment tables, and project credential rows remain the sources of truth.
  No permission, assignment, or assurance state is copied into `types.Authentication`.
- Secret-backed application configuration owns issuer-specific inbound service keys. The policy
  owns only allowed issuer and actor-ID tuples.
- Deployment operations own preservation and identification of the unique-key-only rollback-floor
  artifact.
- Public authentication requests and the `types.Authentication` structure remain unchanged.
- Feature permissions are not an input or fallback and remain a separate deprecation change.
- Controllers do not acquire persistence ownership. Services do not perform transport validation.

## Alternatives Rejected

- Implement recent MFA in this RFC: rejected because authentication factors, sessions, recovery,
  and assurance freshness require their own accepted contract and are outside this cutover.
- Keep secret creation enabled with permission checks only: rejected because permission possession
  cannot satisfy the accepted recent-MFA prerequisite.
- Let Organization credential lifecycle permission assign a token role: rejected because it would
  let a Credential Admin mint authority the caller does not possess in the target Project.
- Require both Organization and Project lifecycle permissions: rejected because accepted policy
  defines them as alternative entry paths.
- Add a second preview RPC: rejected because additive fields on the existing reduction RPC preserve
  route inventory and keep one mutation owner while old clients fail closed.
- Persist preview rows or signed preview tokens: rejected because locked recomputation and digest
  comparison provide stale-state safety without another lifecycle or cleanup owner.
- Roll back to the old binary: rejected because it either rejects migrated issuer keys or restores
  forgeable shared-secret caller provenance.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Existing authentication requests and response shapes outside role reduction remain stable. Old reduction clients fail closed until they add preview and confirmation. Policy enforcement intentionally disables new project-token and provider-credential creation until recent MFA exists. |
| Failure and recovery | Invalid or stale preview confirmation writes nothing. Secret creation is unavailable rather than weakly authorized. The unique-key-only safety floor preserves disabled routes and exact service callers while restoring handler permission enforcement without accepting the shared secret. |
| Security and privacy | Token assignment cannot exceed the caller, secret creation cannot bypass assurance, provider metadata honors both accepted entry paths, and role reductions disclose impact only to authorized role administrators. Logs and digests never contain secrets or permission-bearing tokens. |
| Data and migration | No new table or column is introduced. Reduction reads existing assignment and credential relations. Enforcement still waits for migrations 000016 and 000017 and the RFC 0049 role-assignment cutover. |
| Concurrency and resources | Confirmation locks and rereads every digest input before mutation. Preview is a bounded database snapshot with no background worker, cache, timer, or retained resource. Existing request cancellation and deadlines propagate to every query. |
| Operations and observability | Disabled-route denials, stale previews, caller-key migration, shared-secret rejection, and policy denials are distinguishable without logging credentials. Operators must retain the exact unique-key-only safety-floor artifact. |

## Rollout and Rollback

1. Publish the additive reduction fields through the protobuf source, generated Go and Python
   artifacts, and React SDK. Any client that exposes reduction must preview, display every impact,
   and submit the returned digest before mutation. Old clients remain fail closed.
2. Implement reduction snapshot and confirmation behavior, the provider metadata any-of selection,
   strict policy parsing, and route inventory validation with enforcement off.
3. Complete RFC 0049 transactional user-role assignment and effective-permission behavior, apply
   migrations 000016 and 000017, and verify active-only permission unions.
4. Deploy the dual-verification compatibility receiver with existing effective-permission handler
   authorization, migrate all Web API service callers to issuer-specific keys, then deploy and retain
   the unique-key-only safety floor that enforces disabled routes and exact caller constraints.
5. Verify every secret-creating route is disabled, every secret-returning route remains disabled,
   custom-role reduction confirmation is active, and no inbound service assertion depends on the
   shared application secret.
6. Deploy the enforcement binary and embedded policy atomically. Existing project credentials begin
   role-permission enforcement only after all prior gates pass.

Stop on an uncovered route, unknown permission, missing token audience or binding, unconfirmed live
role reduction, a shared-secret service assertion, missing caller registry entry, stale assignment,
tenant mismatch, incomplete migration, unavailable rollback-floor artifact, or unresolved Critical
or Major finding.

Before enforcement, roll back the parser, preview, or registry stage independently while keeping
secret creation on existing handler behavior. After enforcement, deploy only the retained unique-
key-only safety floor and its matching configuration. Do not deploy the old shared-secret-only
binary. No schema rollback is required by this RFC; migration rollback remains governed by RFCs
0039, 0047, and 0049.

## Verification

Acceptance requires:

- exact policy coverage of 139 current gRPC methods and 18 HTTP routes, with no stale route and a
  generator-backed update when route inventory changes;
- disabled project-token and provider-credential creation and OAuth connection routes prove their
  controllers are not invoked;
- existing active project credentials resolve only active assigned roles, active permission pairs,
  declared audience, and exact Organization and Project binding;
- token creation contract tests prove lifecycle any-of, Project access management, complete selected-
  role permission ceiling, recent assurance, lock-and-reread, and atomic no-secret failure before a
  route may be re-enabled;
- provider metadata tests prove Organization and Project alternatives, tenant constraint, project-
  token Project-only behavior, value redaction, exact criteria allowlist, AND-only predicates,
  literal wildcard handling, invalid-criteria rejection, OR-based tenant-bypass denial, pagination
  bounds, count/page consistency, and deterministic ordering;
- preview tests cover unassigned roles, active and invited user assignments, active credentials,
  ordering, digest determinism, and archived-reference exclusion;
- confirmation tests cover success, missing or malformed digest, every stale digest input,
  cancellation, deadline, database failure, concurrent caller-permission or membership revocation,
  concurrent assignment, no partial mutation, and permission ordering by key then ID;
- service assertion tests cover issuer and actor mismatch, unknown key ID, shared-secret rejection,
  current and previous key rotation, delegation provenance, and rollback-floor compatibility;
- unchanged public authentication request, `types.Authentication`, and unrelated protobuf fields;
- `protos/access_management_contract_test.go` inspects generated descriptors and asserts every
  preserved `ReduceCustomRolePermissions` request field 1 through 3 and response field 1 through 5,
  the exact new request fields `confirmationDigest = 4` and `preview = 5`, response field
  `preview = 6`, both impact messages and their field numbers and types, and the unchanged RPC input
  and output types;
- a second backend generation and a second React generation are byte-identical to their first
  generated Access Management artifacts;
- React SDK tests assert the new preview and impact messages, all new accessors, and public index
  exports while preserving the existing reduction wrapper and response fields;
- independent high-risk review with no unresolved Critical or Major findings.

Exact commands:

```text
bash bin/artifacts-generate.sh
go test ./protos -run '^TestAccessManagementReductionContract$'

backend_generation_snapshot="$(mktemp -d)"
cp protos/access-management-api.pb.go "${backend_generation_snapshot}/"
cp protos/access-management-api_grpc.pb.go "${backend_generation_snapshot}/"
cp api/document-api/app/bridges/artifacts/protos/access_management_api_pb2.py "${backend_generation_snapshot}/"
cp api/document-api/app/bridges/artifacts/protos/access_management_api_pb2.pyi "${backend_generation_snapshot}/"
cp api/document-api/app/bridges/artifacts/protos/access_management_api_pb2_grpc.py "${backend_generation_snapshot}/"
bash bin/artifacts-generate.sh
cmp "${backend_generation_snapshot}/access-management-api.pb.go" protos/access-management-api.pb.go
cmp "${backend_generation_snapshot}/access-management-api_grpc.pb.go" protos/access-management-api_grpc.pb.go
cmp "${backend_generation_snapshot}/access_management_api_pb2.py" api/document-api/app/bridges/artifacts/protos/access_management_api_pb2.py
cmp "${backend_generation_snapshot}/access_management_api_pb2.pyi" api/document-api/app/bridges/artifacts/protos/access_management_api_pb2.pyi
cmp "${backend_generation_snapshot}/access_management_api_pb2_grpc.py" api/document-api/app/bridges/artifacts/protos/access_management_api_pb2_grpc.py

go test ./pkg/authentication/... ./pkg/authenticators ./pkg/middlewares
go test ./api/web-api/api ./api/web-api/internal/service/accessmanagement ./api/web-api/router ./cmd/web
go test ./api/web-api/...

cd sdks/react
bash bin/artifacts-generate.sh
react_generation_snapshot="$(mktemp -d)"
cp src/clients/protos/access-management-api_pb.js "${react_generation_snapshot}/"
cp src/clients/protos/access-management-api_pb.d.ts "${react_generation_snapshot}/"
cp src/clients/protos/access-management-api_pb_service.js "${react_generation_snapshot}/"
cp src/clients/protos/access-management-api_pb_service.d.ts "${react_generation_snapshot}/"
bash bin/artifacts-generate.sh
cmp "${react_generation_snapshot}/access-management-api_pb.js" src/clients/protos/access-management-api_pb.js
cmp "${react_generation_snapshot}/access-management-api_pb.d.ts" src/clients/protos/access-management-api_pb.d.ts
cmp "${react_generation_snapshot}/access-management-api_pb_service.js" src/clients/protos/access-management-api_pb_service.js
cmp "${react_generation_snapshot}/access-management-api_pb_service.d.ts" src/clients/protos/access-management-api_pb_service.d.ts
npm test -- --runInBand tests/clients/access-management.test.ts tests/index.test.ts
npm test -- --runInBand
npm run build
cd ../..

cd ui
yarn test --watch=false --runInBand src/clients/__tests__/access-management.client.test.ts
yarn checkTs
yarn eslint src/clients/access-management.client.ts src/clients/__tests__/access-management.client.test.ts
cd ..

git diff --check
just agent-finalize "rfcs/0053-safe-authorization-enforcement-prerequisites.md,config/config.go,config/config_test.go,pkg/authentication/policy/v1/web-api.yaml,pkg/authentication/policy.go,pkg/authentication/policy_test.go,pkg/authentication/authorizer.go,pkg/authentication/authorizer_test.go,pkg/types/jwt.go,pkg/types/jwt_test.go,pkg/authenticators/service_authenticator.go,pkg/authenticators/service_authenticator_test.go,pkg/middlewares/authorization_middleware.go,pkg/middlewares/authorization_middleware_test.go,pkg/errors/access_management.go,pkg/errors/access_management_test.go,pkg/errors/provider_credential.go,pkg/errors/provider_credential_test.go,protos/artifacts/access-management-api.proto,protos/access-management-api.pb.go,protos/access-management-api_grpc.pb.go,protos/access_management_contract_test.go,api/document-api/app/bridges/artifacts/protos/access_management_api_pb2.py,api/document-api/app/bridges/artifacts/protos/access_management_api_pb2.pyi,api/document-api/app/bridges/artifacts/protos/access_management_api_pb2_grpc.py,api/web-api/api/access_management.go,api/web-api/api/access_management_test.go,api/web-api/api/vault.go,api/web-api/api/vault_test.go,api/web-api/api/testdata/authentication-policy-matrix.json,api/web-api/internal/service/accessmanagement.service.go,api/web-api/internal/service/accessmanagement.service_test.go,api/web-api/internal/service/accessmanagement/service.go,api/web-api/internal/service/accessmanagement/service_test.go,api/web-api/internal/service/vault.service.go,api/web-api/internal/service/vault.service_test.go,api/web-api/internal/service/vault/service.go,api/web-api/internal/service/vault/service_test.go,api/web-api/router/web.go,api/web-api/router/web_test.go,cmd/web/web.go,cmd/web/authentication_middleware_test.go,sdks/react/src/clients/protos/artifacts/access-management-api.proto,sdks/react/src/clients/protos/access-management-api_pb.js,sdks/react/src/clients/protos/access-management-api_pb.d.ts,sdks/react/src/clients/protos/access-management-api_pb_service.js,sdks/react/src/clients/protos/access-management-api_pb_service.d.ts,sdks/react/src/clients/access-management.ts,sdks/react/src/index.ts,sdks/react/tests/clients/access-management.test.ts,sdks/react/tests/index.test.ts,sdks/react/package.json,sdks/react/package-lock.json,ui/src/clients/access-management.client.ts,ui/src/clients/__tests__/access-management.client.test.ts,ui/package.json,ui/yarn.lock"
```

This is the complete approved implementation boundary. If implementation proves another source,
generated, configuration, wrapper, or test file necessary, work pauses for an explicit plan update
instead of silently expanding the finalization scope. UI component or route files are excluded; if
a rendered consumer is added, its focused happy-path and failure-path test becomes required and the
UI commands and exact-file boundary must be updated before that edit.

## Review and Approval

- Challenger: Codex task `task_c519c464cadc`
- Outcome: Approved
- Prior challenge: Codex task `task_e05670aa7e49` returned three Major findings and no Critical
  findings. This candidate is correction round one.
- Final scheduled challenge: Codex task `task_461c27a52714` returned one Major verification-contract
  finding and no other Critical or Major finding. The repository owner authorized one additional
  correction cycle limited to that finding on 2026-11-11.
- Resolved findings: Draft incorporates all five blocking findings from Codex task
  `task_1bb2aa4a5360`. The first RFC 0053 challenge additionally required locked caller-permission
  revalidation, fixed provider metadata criteria, and complete protobuf, SDK, and focused UI
  verification. The final challenge confirmed the caller revalidation and metadata criteria fixes,
  plus all five original blockers, but found the verification boundary incomplete. The extra owner-
  authorized correction now adds executable descriptor assertions, backend and React generation-
  drift checks, and the complete exact-file finalization boundary. The targeted final challenger
  approved that correction and the unchanged authorization policy; its sole formatting Minor was
  resolved by removing the extra policy EOF blank line.
- Open questions: None
- Approval reference: Repository owner accepted this exact RFC and policy for implementation in
  this session on 2026-11-11
