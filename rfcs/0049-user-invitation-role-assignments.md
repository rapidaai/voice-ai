# RFC 0049: User Invitation Role Assignments

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-10
- Supersedes: RFC 0047 custom user-role assignment deferral and built-in-only writer restriction

## Context

The existing Organization invitation and Project-access APIs accept legacy role names. The UI
hard-codes `admin`, `member`, `super admin`, `writer`, and `reader`, while migration 000017 converts
the assignment tables to `role_definition_id` and permits multiple different current roles at one
scope. The current contract cannot select custom roles, cannot assign multiple roles, and cannot be
used after the role-column migration.

Invitation already owns pending user creation. If the email is unknown, the backend creates an
`INVITED` user identity, writes its invited assignments, and sends a signup email. A separate
console user-creation workflow would duplicate this ownership and is not required.

This is Governed because it changes public protobuf fields, generated SDK contracts, authorization
assignment writes, tenant validation, custom-role lifecycle checks, and coordinated releases across
the proto repository, backend, React SDK, and UI.

## Decision

Keep the existing `InviteUserToOrganization` and `AddUserToProjects` operations as the sole console
entry points. Replace legacy role-name inputs with role-definition IDs and support multiple roles.

`ProjectRoleAssignment` retains field 1 `projectId`, reserves field 2 and the name `projectRole`, and
adds `repeated uint64 roleDefinitionIds = 3 [jstype = JS_STRING]`.

`InviteUserToOrganizationRequest` retains field 1 `email` and field 3 `projectRoles`, reserves field
2 and the name `organizationRole`, and adds
`repeated uint64 organizationRoleDefinitionIds = 4 [jstype = JS_STRING]`.

`AddUserToProjectsRequest` retains its field numbers and consumes the revised
`ProjectRoleAssignment`. Response envelopes and RPC names are unchanged. The old string fields are
removed and reserved rather than reinterpreted. Older invitation clients become behaviorally
incompatible and must upgrade before the backend cutover.

`UpdateUserOrganizationRoleRequest` retains field 1 `userId`, reserves field 2 and the name
`organizationRole`, and adds
`repeated uint64 roleDefinitionIds = 3 [jstype = JS_STRING]`. The existing RPC becomes a complete
replacement of the user's current Organization role set in one transaction. An empty replacement
is rejected; removing a user remains owned by `DeleteUserFromOrganization`.

`common.proto` adds this exact public message:

```text
message AssignedRole {
  uint64 roleDefinitionId = 1 [jstype = JS_STRING];
  string key = 2;
  string name = 3;
}
```

`User` reserves field 4 and the name `role`, and adds
`repeated AssignedRole organizationRoles = 7`. `OrganizationRole` reserves fields 1 and 3 plus the
names `id` and `role`; it retains `organizationId = 2` and `organizationName = 4`, adds
`repeated AssignedRole roles = 5`, and adds `repeated string effectivePermissions = 6`.
`ProjectRole` reserves fields 1 and 3 plus the names `id` and `role`; it retains `projectId = 2` and
`projectName = 4`, adds `repeated AssignedRole roles = 5`, and adds
`repeated string effectivePermissions = 6`. These messages now represent scoped access rather than
one assignment row.

At runtime the backend loads every active assignment for the authenticated user and unions active
permission keys across its active role definitions. Organization permissions are evaluated only at
the current Organization scope; Project permissions are evaluated only at the selected Project.
The authentication response returns the same scoped role references and effective permission keys
used by server authorization. Invited assignments do not authorize requests until activation.
`GetAllUser` and invitation responses project all Organization role references instead of a single
legacy role name.

An Organization invitation requires at least one unique nonzero Organization role ID. Each role
must be active, have `ORGANIZATION` scope and `USER` subject, and be either global built-in or a
custom role owned by the current Organization. The owner role is not assignable by invitation;
ownership transfer remains a separate operation.

Every general Organization assignment writer, including invitation and
`UpdateUserOrganizationRole`, rejects `organization_owner` role 101 as a target. It also rejects an
attempt to replace the current owner's assignments. Organization creation and the separate future
ownership-transfer operation are the only owners of role 101.

Each optional Project entry requires one unique nonzero Project ID and at least one unique nonzero
role ID. Every Project must be active and belong to the current Organization. Every selected role
must be active, have `PROJECT` scope and `USER` subject, and be either global built-in or a custom
role owned by the Project's Organization. The same role cannot appear twice for one scope.

`UpdateUserOrganizationRole` is replace-set, not additive. It locks the target membership and role
definitions, validates the complete requested set, preserves current assignment rows already in
the set, archives current rows omitted from the set with audit actor and time, and inserts missing
rows. The user identity status is the sole status source: an `INVITED` identity gives every selected
assignment `INVITED`; every other eligible identity gives every selected assignment `ACTIVE`.
Preserved selected rows with a different legacy status are moved to that canonical status with
updated audit actor and time. The entire diff is one transaction.

`AddUserToProjects` remains an add-project operation. For every selected Project, its submitted
role IDs are the complete initial role set. If the user already has any active or invited role in
any selected Project, the whole request returns the existing already-in-project failure and writes
nothing. Archived history does not block a new assignment. New rows inherit the user's current
canonical identity status using the same rule. Changing roles for an existing Project requires a
separately governed replace-set operation and is not inferred from `AddUserToProjects`.

The backend validates and persists the pending identity plus all Organization and Project
assignments in one transaction. `AddUserToProjects` likewise validates and locks every selected
Project and role before inserting any assignment, then inserts all rows in one transaction. Any
invalid project or role, duplicate input, insert failure, cancellation, deadline, or commit failure
writes no identity or assignment for that request. The invitation email is sent only after commit.
Email-delivery failure retains the existing committed invitation and is reported operationally; it
does not roll back persisted access.

Assignment APIs use the same permission union as the rest of the server. Invitation requires
`organization.members.invite`; assigning or replacing Organization roles requires
`organization.access.manage`; assigning Project roles across the Organization requires
`organization.access.manage`. A request without every permission required by its selected actions
fails before resource validation. MFA remains a separate cutover.

Active or invited user-role assignments now participate in custom-role lifecycle checks. A custom
role referenced by any active or invited Organization or Project user assignment cannot be
archived. Archival locks the role and checks user assignments and project credentials in the same
transaction. Permission reduction remains monotonic and changes all assignments sharing that role;
the existing explicit reduction operation and reason remain the only mutation path.

The console keeps one `Invite user` dialog. It loads all pages of active `USER` roles from the role
catalog, separates Organization and Project roles by scope, supports searchable multi-selection,
and sends only role-definition IDs. Project access stays optional. Direct user creation is not
added.

## Contracts and Ownership

- The proto repository owns the reserved legacy fields and new repeated ID fields. Generated Go,
  Python, and React artifacts remain generator-owned.
- The Organization invitation handler owns email validation, pending identity creation, invited
  Organization and Project assignments, commit authority, and post-commit email delivery.
- The Project membership handler owns adding existing Organization users to selected Projects.
- The user authentication service owns loading active scoped role assignments, unioning active
  permissions, and producing the scoped access projection consumed by server and console checks.
- The role catalog owns role scope, subject, tenant ownership, status, and permission membership.
- User assignment tables remain the source of truth for Organization and Project user access.
- The backend transaction owner validates all selected Projects and roles before writing any row.
- Custom-role archival owns active-reference checks for user assignments and project credentials.
- The React SDK release adds the generated repeated-ID request surface and no parallel legacy
  wrapper.
- The UI invitation dialog owns role discovery, searchable selection, duplicate prevention,
  submission state, envelope errors, and refresh after success.
- The Users page owns displaying all Organization role references and replacing the Organization
  role set through the revised update request.
- Direct ownership transfer, MFA, project-token enforcement, and provider-credential redesign are
  outside this RFC.

## Alternatives Rejected

- Add a direct Create User operation: rejected because invitation already creates the pending
  identity and owns email activation.
- Keep role names in the request: rejected because names are not stable identifiers and cannot
  address custom roles safely.
- Allow only one role per scope: rejected because the accepted assignment schema intentionally
  supports multiple current roles.
- Change the type of existing protobuf fields in place: rejected because reusing a wire number with
  a different type is unsafe. Old fields are reserved and new numbers are used.
- Keep old and new inputs during a compatibility window: rejected because it creates two policy
  paths and ambiguous precedence. The release is coordinated and fail-closed.
- Add a generic polymorphic assignment RPC: rejected because Organization invitation and Project
  membership already own the required workflows and lifecycle.
- Permit archival of an assigned custom role: rejected because it would leave live assignments
  pointing at an inactive definition.
- Project one legacy role string from multiple assignments: rejected because it would hide active
  access and make custom-role authorization undefined.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Old clients no longer provide required role IDs or understand repeated role and permission projections. Proto and SDK publish first but are not consumed until a coordinated backend and UI deployment. RPC names and response envelopes remain stable. |
| Failure and recovery | Validation and writes are atomic. Pre-commit failure leaves no identity or assignment. Post-commit email failure leaves a retryable invited membership and must be visible in logs. |
| Security and privacy | Scope, subject, status, tenant ownership, permission unions, caller authorization, and owner-role exclusion are server-enforced. Generic unavailable-role failures prevent cross-tenant role probing. The console uses server-produced effective permissions rather than role names. Emails and role IDs contain no new secret material. |
| Data and migration | Migration 000017 must be applied in the same maintenance cutover described by RFC 0047. New writes use only role-definition IDs and may create multiple rows per scope. No new table is added. |
| Concurrency and resources | Invitation and AddUserToProjects lock Projects and roles in ascending ID order before any assignment insert. Role archival locks the role and checks references in the same transaction. Email delivery is post-commit and does not hold database locks. |
| Operations and observability | Release order is proto, React SDK, backend plus migration, then UI. Logs identify correlation, Organization, Project, role, and invited user IDs without exposing passwords or tokens. Assignment or email failures are actionable separately. |

## Rollout and Rollback

1. Commit and push the accepted protobuf changes to the proto repository main branch.
2. Pull that proto commit into the React SDK, run `bin/artifacts-generate.sh`, update the wrapper
   types if required, bump the package patch version, verify, commit, and push main. The SDK release
   workflow publishes the package. Publication does not authorize any consumer to upgrade yet.
3. Update generated backend artifacts and implement the transactional role-ID invitation and
   Project membership paths plus custom-role archival checks.
4. Inventory external callers and require their upgrade readiness. During the RFC 0047 maintenance
   window, stop legacy backend instances and hold console traffic, apply migration 000017, deploy
   the compatible backend, install the released SDK in the UI, and deploy the compatible UI before
   restoring traffic. The new SDK must not be consumed against the old backend, and the old UI must
   not run against the new backend.

Stop if the proto or SDK release fails, the package is unavailable, migration preflight fails,
legacy instances remain, or backend assignment verification fails. Before new multiple or custom
assignments exist, keep traffic held, stop the new backend, restore compatible old UI assets, run
migration 000017 down, start the old backend, and only then restore traffic. A published SDK
version is immutable; pin consumers to the prior version or publish a corrective patch rather than
unpublish it. After new assignments exist, use a forward fix or approved data translation because
the legacy schema cannot represent them.

## Verification

Acceptance requires:

- protobuf compilation proves reserved legacy fields and repeated string-safe uint64 ID accessors;
- generated React artifacts expose Organization and Project role-definition ID lists;
- old role-name setters are absent from the released request surface;
- authentication and user responses expose all assigned role references, and authentication
  exposes the exact effective permission union used by server authorization;
- backend tests cover built-in and same-tenant custom assignments, multiple roles, duplicate IDs,
  wrong scope, wrong subject, archived roles, owner-role exclusion, cross-tenant roles, atomic
  rollback, mixed legacy assignment statuses converging to the identity-derived status, and
  assigned-role archival rejection;
- UI tests cover catalog loading, searchable Organization and Project role selection, multiple
  selections, optional Project access, validation, success, and envelope or transport failure;
- existing invitation identity creation and post-commit email behavior remain covered.

Exact commands:

```text
# Proto repository
proto_descriptor="$(mktemp)"
protoc -I . --descriptor_set_out="${proto_descriptor}" common.proto access-management-api.proto web-api.proto
git diff --check

# React SDK
bash bin/artifacts-generate.sh
npm test -- --runInBand
npm run build
git diff --check

# Backend repository after proto update
bash bin/artifacts-generate.sh
go test ./api/web-api/api ./api/web-api/internal/service/user ./api/web-api/internal/service/accessmanagement
just agent-finalize "api/web-api/api,api/web-api/internal/entity,api/web-api/internal/service,protos"

# UI after released SDK installation
cd ui && yarn test --watchAll=false --runInBand src/app/components/dialogs/workspace/__tests__/workspace-dialogs.test.tsx src/app/pages/workspace/user/index.test.tsx src/clients/__tests__/workspace.client.test.ts
cd ui && yarn checkTs
cd ui && yarn lint
just ui-browser-test
just agent-finalize "ui/src/app/components/dialogs/workspace,ui/src/app/pages/workspace/user,ui/src/clients"
```

No generated artifact is edited manually. Temporary descriptor output is deleted after the proto
check.

## Review and Approval

- Challenger: Independent Codex task `/root/user_role_migration_challenge`
- Outcome: Approved
- Resolved findings: Added every legacy writer and runtime projection to the ID contract; defined permission-union enforcement, atomic invitation and Project assignment, exact protobuf fields, owner exclusion, replace and add semantics, coordinated release and rollback, executable verification, and identity-derived assignment status.
- Open questions: None
- Approval reference: Repository owner approval on 2026-11-10 to proceed with invitation, release the proto first, then the React SDK, then update the UI.
