# RFC 0026: Access Policy Version 1

- Status: Superseded
- Owner: Platform team
- Approved by: Repository owner
- Date: 2026-11-03
- Supersedes: N/A

## Context

Superseded by [RFC 0027](0027-recovery-approver-separation.md).

The application needs one enterprise authorization policy for Organization and Project access,
custom roles, Project tokens, credential administration, and future MFA-gated operations. The
policy must avoid implicit inheritance, cross-tenant authority, self-escalation, unsafe token
reuse, and ambiguity between reusable role ownership and Project administration.

The repository owner explicitly directed that every reviewer remedy be incorporated and authorized
this Governed decision in conversation on 2026-11-03. Final acceptance still requires the
independent panel below.

This RFC is the complete Version 1 policy. It owns the canonical subjects, role keys, permission
keys, built-in bundles, delegation ceilings, and protected-role rules. Persistence may reference
this RFC only after it is Accepted. RFC 0022 owns schema only and cannot redefine this catalog.

Current production behavior is unchanged by this decision. Existing role strings and credentials
remain authoritative until separately accepted migration and enforcement RFCs are implemented.

## Decision

### Authorization graph

Version 1 uses one graph:

```text
Subject + concrete Scope -> Role Assignment -> Role Definition -> Permissions
```

Effective permissions are the union of active direct assignments in the requested scope. Absence
of an allow is a denial. There is no explicit deny, role precedence, implicit role inheritance, or
implicit Project-content access from an Organization assignment.

The backend is authoritative for every authorization decision. UI permission and assurance gates
are presentation only and never satisfy or replace server-side enforcement.

Every assignment and evaluation enforces these global invariants:

- the user subject has an active membership in the Organization that owns the concrete scope;
- the subject, role definition, assignment, Organization, and optional Project are active;
- a Project scope belongs to the same Organization as the user membership and any custom role;
- the assignment scope type equals the role scope type and the assignment subject type equals the
  role subject type;
- a Project token belongs to the same Organization and Project as its concrete scope;
- cross-Organization subjects, roles, scopes, or resources are denied without revealing whether
  the target exists.

Removing or deactivating an Organization membership atomically deactivates all of that
membership's Organization and Project assignments. Rejoining creates a new membership lifecycle
and grants no prior authority. Every role must be explicitly reassigned through the current
assignment checks. Archived assignment rows remain audit history only.

Version 1 supports exactly these assignable subjects:

```text
USER
PROJECT_TOKEN
```

Users may receive Organization or Project roles. Project tokens may receive Project token roles.
People groups, Organization-token roles, internal-service roles, and system-job roles are deferred.
They receive no role assignment and no permission key in Version 1.

An Organization is a tenant resource and authorization scope, not an assignable subject or human
identity. Organization membership links a User subject to that tenant and is a prerequisite for
User assignments and evaluation. A Project is likewise a resource and nested scope, not a subject.

Existing internal service and system traffic remains on current production authorization. Before
any endpoint accepting a service or system actor moves to Version 1 enforcement, a separate
accepted compatibility RFC must inventory every delegated and non-delegated path and define its
least-privilege authorization. That accepted compatibility RFC is a hard prerequisite for every
affected endpoint cutover. No affected endpoint may cut over through a fallback or bypass.
Signed internal delegation must preserve the originating actor, concrete scope, effective
permissions, token constraints, and authentication assurance and may never widen any of them.

### Role definition ownership

Built-in role definitions are platform-owned and reusable in every Organization. Custom role
definitions belong to one Organization. Every role declares one scope type,
`ORGANIZATION` or `PROJECT`, and one subject type, `USER` or `PROJECT_TOKEN`.

A custom Project role is reusable across Projects in its owning Organization. A role grants
nothing until a direct assignment targets a concrete Organization or Project. There is no group
inheritance and no selected-Project availability allowlist.

Built-in role keys are globally unique and immutable. Custom role keys are unique and immutable
within their Organization across both scope types. Archived keys cannot be reused. Custom roles
cannot shadow a built-in key.

Reusable custom role definitions are Organization-owned policy. A Project Admin may read
available Project roles and assign eligible roles within its Project but cannot mutate a reusable
definition.

A custom role's permission set may be reduced in place. It cannot be expanded in place. Adding a
permission requires a new immutable role key and separately authorized reassignment. This rule
applies to human and token custom roles. Built-in definitions and permission bundles are immutable
and cannot be reduced, expanded, archived, or otherwise mutated.

Custom-role creation writes the definition and its complete initial permission set atomically.
After creation, no writer may insert a new active permission pair for that role, reactivate an
archived pair, or replace one permission with another. Reduction archives existing pairs only.
Expansion always creates a new immutable role key.

Before reducing a custom role that has active assignments, the editor receives an impact preview
listing every affected concrete scope, direct user, and Project token. The reduction requires
explicit confirmation, recent MFA, and `organization.roles.update` in the role's owning
Organization. The backend
returns an opaque digest of the canonical preview snapshot, including the role definition, active
permission pairs, every active assignment with its concrete scope and subject, the exact requested
permission removals, and the complete resulting permission bundle. Confirmation must present that
digest and the same requested removal set. In the reduction transaction, the backend locks the
role, permission pairs, and active assignments, recomputes the snapshot and proposed result, and
rejects a stale or mismatched digest without changing the role. Any permission, assignment, or
requested-removal change therefore requires a new preview and confirmation.

Archiving a custom role requires recent MFA and `organization.roles.archive` in the role's owning
Organization. A role with an active assignment cannot be archived. Every assignment must be
removed or moved to another active role first.

### Permission ceilings

Creating a custom Organization role requires `organization.roles.create` and every selected
permission in the creator's effective permissions in that Organization. Creating a reusable
custom Project role requires `organization.roles.create` in the owning Organization and limits
the definition to the exact Project custom-role catalogs below. Definition creation grants no
Project authority, so it does not require choosing an arbitrary Project for a permission check.

Every human role assignment, built-in or custom, requires the scope-appropriate access-management
permission plus every permission in the target role. Organization assignments require
`organization.access.manage`; Project assignments require `project.access.manage`. Every
permission must be in the assigner's effective permission set in the concrete target Organization
or Project. A user cannot bypass the creation or assignment ceiling through a role it already
holds.

Assignment, membership, custom-role reduction, and custom-role archival transactions use one lock
order: Organization; affected Projects by ascending ID; affected memberships by ascending ID; role
definitions by ascending ID; affected token-subject rows by ascending ID; then assignment rows by
ascending ID. User and Project-token assignment creation and reassignment lock the Organization,
optional Project, authorizing-user membership, and every target role definition before reading role
permissions. Token paths then lock the token subject. All paths revalidate active membership, role
state, scope, subject type, permission ceilings, and final-role invariants before writing. Reduction
and archival use the same order and revalidate their preview or no-active-assignment condition after
locking. Consequently, a new User or Project-token assignment cannot commit from stale membership
or role state while membership removal, reduction, or archival is in progress.

The exact custom Organization human-role permission catalog is:

```text
organization.read
organization.members.read
organization.members.invite
organization.projects.read
organization.roles.read
organization.access.read
organization.project_access_tokens.read_metadata
organization.provider_credentials.read_metadata
organization.audit.read
organization.audit.export
```

Ownership, Organization deletion or update, billing, security, session revocation, member removal,
Project archival, reusable-role mutation, access management, and credential mutation are reserved
for reviewed built-in roles.

The exact custom Project human-role permission catalog is:

```text
project.read
project.roles.read
project.access.read
project.resources.read
project.resources.create
project.resources.update
project.resources.delete
project.deployments.read
project.deployments.manage
project.runtime.invoke
project.conversations.read
project.conversations.delete
project.conversations.export
project.observability.read
project.observability.export
project.provider_credentials.read_metadata
project.provider_credentials.use
project.audit.read
project.audit.export
```

The exact custom Project token-role catalog is the token-eligible Project permission catalog
defined below. Custom token roles may be assigned only to a `CONFIDENTIAL` Project token.

The exact built-in human-role assignment matrix is:

| Assigner | Built-in roles assignable to another active member |
| --- | --- |
| Organization Owner | Every Organization human role. |
| Organization Admin | `access_admin`, `credential_admin`, and `organization_member`. |
| Access Admin | `access_admin` and `organization_member`. |
| Project Admin | `project_admin`, `project_developer`, `project_operator`, `project_viewer`, and `project_auditor` in that Project. |
| Security Admin, Credential Admin, Billing Admin, Organization Auditor, Organization Member, Project Developer, Project Operator, Project Viewer, Project Auditor | None. |

Only Organization Owner may assign or remove `organization_owner`, `organization_admin`,
`security_admin`, or `billing_admin`. No caller may assign, change, or remove its own direct role
assignment.

This matrix applies only to built-in human roles and remains subject to active membership,
scope-appropriate `access.manage`, and the full permission-subset ceiling. A caller may assign a
custom role when it has active membership, the scope-appropriate `access.manage` permission, and
every permission in that custom role in the concrete target scope. The built-in matrix does not
otherwise restrict this custom-role path. An Organization role never grants authority to assign a
Project role. Organization Owner, Organization Admin, or Access Admin must hold a separate Project
Admin assignment before assigning any built-in or custom role in that Project.

Removing a human built-in or custom assignment requires the same scope-appropriate
`access.manage` permission and every permission in the assigned role. Protected-role removal
remains Owner-only, self-removal is denied, and final Owner and final Project Admin invariants take
precedence. Removing or reducing a Project token assignment requires `project.access.manage` and
every current effective token permission. Credential Admin may revoke a credential as a
lifecycle-safety action without that Project ceiling; revocation atomically deactivates every token
assignment and never reveals a replacement secret.

Membership removal is an aggregate assignment-removal operation. The ordinary path requires
`organization.members.remove` and requires the caller to satisfy the direct assignment-removal
rule above for every active assignment that the transaction would deactivate. Authorization is
evaluated from one locked snapshot before any membership or assignment changes. Organization
Owner has a narrowly scoped emergency-revocation exception for a different user's membership: it
may bypass both `project.access.manage` and the per-Project permission-subset checks solely to
deactivate that membership's existing Project assignments, but still requires
`organization.members.remove`, recent MFA, a reason, a correlation identifier, and preservation of
the final Owner and final Project Admin invariants. It cannot transfer roles, create replacement
access, reveal credentials, or remove the initiating Owner. The transaction audits the membership
and every deactivated assignment under the same actor and correlation identifier. All other
membership removals fail unless every affected assignment passes its ordinary removal ceiling.

Credential lifecycle and credential authority are separate:

The `organization.project_access_tokens.*` and `organization.provider_credentials.*` permissions
use the Organization as the requested authorization scope. The caller must have the matching
permission through an active assignment in that Organization. The target Project is a constrained
resource: it must be active and belong to the same Organization. These permissions authorize only
the named credential metadata or lifecycle operation. They grant no Project role, Project content
permission, token authority, runtime authority, or authority over a Project in another
Organization. Project token assignment and assigned-token rotation remain subject to the separate
Project ceilings below.

For token lifecycle operations, the caller needs the operation's matching Organization permission
or matching Project permission below. These are alternative entry paths; both are never required.
The Organization path also requires the target-Project constraint above. The Project path requires
the permission through an active assignment in that concrete Project.

| Project-token operation | Required authorization |
| --- | --- |
| Create an unassigned token | `organization.project_access_tokens.create` or `project.access_tokens.create`. |
| Read token metadata | `organization.project_access_tokens.read_metadata` or `project.access_tokens.read_metadata`. |
| Reduce an assigned token's role, audience, or binding | Matching `*.reduce_access` plus `project.access.manage` and every current effective token permission. |
| Rotate an unassigned token | `organization.project_access_tokens.rotate` or `project.access_tokens.rotate`. |
| Rotate an assigned token | Matching `*.rotate` plus `project.access.manage` and every current effective token permission. |
| Revoke a token | `organization.project_access_tokens.revoke` or `project.access_tokens.revoke`; revocation is a monotonic safety action and needs no Project ceiling. |
| Create and assign atomically | Matching `*.create` plus `project.access.manage` and every permission in the target token role; return the new secret once to that caller. |
| First role assignment to an existing unassigned token | Matching `*.rotate` plus `project.access.manage` and every permission in the target token role; atomically revoke the old secret, assign the role, and return the replacement secret once to that caller. |
| Add or replace a role after any active assignment | Use replacement with matching `*.create`, `project.access.manage`, and every permission in the resulting token. |
| Replace to increase permissions, audience, or binding | Matching `*.create` plus `project.access.manage` and every permission in the resulting token; atomically revoke the old token. |

For provider credentials, the matching Organization or Project permission is likewise an
alternative entry path for `read_metadata`, `create`, `update`, `rotate`, and `revoke`. Only
`project.provider_credentials.use` authorizes using a provider credential in Project content or
runtime configuration; no Organization permission grants use. Token and provider operations fail
closed when neither exact lifecycle predicate is satisfied.

- Credential Admin may create an unprivileged Project credential, rotate it while it has no active
  role assignment, and revoke it at any time as a lifecycle-safety action.
- A lifecycle-only create or rotate operation does not assign a role.
- First assignment of authority to an existing unassigned token always rotates its secret in the
  same transaction, so any lifecycle-only caller that saw the old secret cannot exercise the new
  authority.
- Rotating an assigned token requires `project.access.manage` plus every effective permission of
  that token in the target Project. Rotation revokes the old secret and returns the replacement
  secret once through a confidential response only to the caller satisfying that ceiling.
- Assigning a Project token role requires `project.access.manage` in the target Project and every
  permission contained by the token role.
- Organization Admin or Credential Admin cannot grant a token content or runtime authority unless
  the user separately satisfies that Project ceiling.
- A token never inherits the permissions of its creator or rotator.

Rotation changes only the secret. It preserves the token subject, role assignments, audience,
token class, Project, and resource binding.

Once a token role has an active assignment, its permission bundle cannot expand. Any increase in a
token's effective permissions, including adding or replacing a role or widening audience or
resource binding, requires a new token subject. The old token is revoked and all its assignments
are deactivated. Only permission, audience, or binding reduction may occur on an existing token,
and the reduction takes effect on the next request and remains audited.

Every Project token assignment, removal, rotation, revocation, audience change, binding change, or
replacement transaction follows the shared order above, including locking every target role before
reading its state or permissions and locking the token subject before reading its assignments. It
then rereads lifecycle, class, audience, binding, roles, role permissions, and every active
assignment and applies the permission ceiling and no-expansion rules to that locked state.
Replacement locks the old token before creating the new subject and atomically revokes the old
token and its assignments. Concurrent first-role assignments therefore serialize; after one
commits, the other observes an active assignment and must use replacement or fail without mutation.

Every Project token has one immutable token class, one closed audience value, and the required
binding below:

| Token class | Audience | Eligible roles | Required binding |
| --- | --- | --- | --- |
| `PUBLIC_WIDGET` | `WIDGET_RUNTIME` | `runtime_invoker` only | Exactly one deployment. |
| `CHANNEL` | `CHANNEL_RUNTIME` | `runtime_invoker` only | Exactly one typed channel or deployment binding. |
| `CONFIDENTIAL` | `RUNTIME` | `runtime_invoker` or a custom token role whose permissions are a subset of `project.runtime.invoke` | Exactly one deployment. |
| `CONFIDENTIAL` | `MANAGEMENT` | `runtime_invoker`, `automation_client`, `observer_client`, or an eligible custom Project token role | The token's Project, optionally narrowed by exact typed resource bindings selected at issuance. |

`WIDGET_RUNTIME`, `CHANNEL_RUNTIME`, and `RUNTIME` authorize only backend endpoints classified as
runtime invocation and only the effective `project.runtime.invoke` permission. `MANAGEMENT`
authorizes backend endpoints mapped to any token-eligible permission granted by the assigned role,
including `project.runtime.invoke`. Every token-capable endpoint declares exactly one required
audience. Authorization uses this closed acceptance matrix in addition to permission and binding:

| Endpoint required audience | Accepted token audience |
| --- | --- |
| `WIDGET_RUNTIME` | `WIDGET_RUNTIME` only |
| `CHANNEL_RUNTIME` | `CHANNEL_RUNTIME` only |
| `RUNTIME` | `RUNTIME` or `MANAGEMENT` |
| `MANAGEMENT` | `MANAGEMENT` only |

An absent or unknown endpoint audience, or a token audience not listed for it, is denied. Thus a
qualified `MANAGEMENT` token can invoke runtime and management endpoints, while a `RUNTIME` token
can invoke only runtime endpoints.

Audience and binding changes are ordered by the set of requests they permit. `RUNTIME` is a strict
subset of `MANAGEMENT`; changing `RUNTIME` to `MANAGEMENT` requires replacement, while changing
`MANAGEMENT` to `RUNTIME` is a reduction when the new deployment binding is within the old allowed
resource set. Within one audience, removing allowed resources is a reduction. Adding resources,
removing a binding constraint, or changing to an incomparable resource is widening and requires
replacement. `PUBLIC_WIDGET` and `CHANNEL` audience and bindings are immutable; changing either
requires a new token of the required class and revocation of the old token.

The authorization context carries token class, audience, Project, and resource binding from
verified token metadata. Request input cannot assert them. A token presented outside any binding
is denied. `PUBLIC_WIDGET` and `CHANNEL` tokens can never receive a custom or management-capable
role. Provider signature verification is preferred for channel callbacks. Confidential tokens
must not appear in widgets, browser bundles, phone URLs, query parameters, logs, or analytics.

### Protected roles and membership

Only Organization Owner may appoint or remove Organization Owner, Organization Admin, Security
Admin, or Billing Admin. Only Organization Owner may remove the membership of a user holding any
of those roles. Membership removal evaluates protected active assignments before changing state.
An Organization always retains at least one active Owner. Every active, non-archived Project
retains at least one active Project Admin. Removing a membership or assignment that would remove
the final Project Admin is rejected unless the same transaction transfers Project Admin to another
active member.

Every assignment or membership mutation that can deactivate an Owner first locks the Organization
row. Every mutation that can deactivate a Project Admin first locks that Project row. A membership
mutation spanning Projects locks the Organization first and then every affected Project in
ascending ID order. After acquiring all locks, the transaction rereads active memberships and
assignments, rechecks every final-role invariant, applies all deactivations and transfers, and
rechecks the invariants immediately before commit. A failed check rolls back the entire mutation.
This serialization prevents concurrent removals from each relying on an assignment the other
transaction removes.

Organization and Project bootstrap and Project Admin recovery are explicit system workflows, not
self-assignment.

### Bootstrap workflows

Authorization owns two system workflows that are not exposed as ordinary role-assignment APIs:

- Organization bootstrap atomically creates the Organization, the initiating authenticated user's
  active membership, and the first `organization_owner` assignment. It is valid only while the new
  Organization has no membership or Owner assignment.
- Project bootstrap requires `organization.projects.create` in the owning Organization, then
  atomically creates the Project and assigns `project_admin` to the initiating authenticated
  user. The user must already have an active membership in that Organization, and the Project must
  have no prior assignment. A member without `organization.projects.create` is denied before any
  Project or assignment is created.

Organization bootstrap is a narrow system-owned exception to the ordinary human-assignment
ceiling. It bypasses that ceiling only for the first `organization_owner` assignment in the same
transaction that creates a new Organization with no membership or Owner assignment. It cannot add
or replace an Owner in an existing Organization.

Project bootstrap is a narrow system-owned exception to the ordinary Project assignment ceiling.
It applies only inside the transaction creating a new empty Project and cannot add an assignment
to an existing Project. Only the built-in Organization Owner and Organization Admin bundles
contain `organization.projects.create`; custom Organization roles cannot authorize bootstrap. If
an active Project has no Project Admin after an external failure, Organization Owner may use a
recent-MFA recovery workflow to assign Project Admin to a different active member. The initiating
Owner cannot be the recovery target. A second active Organization Owner must independently approve
the recovery with recent MFA. When the Organization has only one active Owner, an independent
platform-security break-glass approver may provide the second approval through the dedicated
recovery workflow. That approver must be an authenticated human distinct from the initiating Owner
and target, use recent MFA, and receives no Organization or Project permission from the approval.
Both paths require a reason, incident identifier, correlation identifier, and immutable audit of
both approvers. Recovery is not exposed as the ordinary assignment API.

The same dual-control workflow may replace a compromised final Project Admin before the Project
becomes adminless. The initiating Organization Owner and independent approver follow the same
identity, recent-MFA, separation, reason, incident, and audit rules. In one transaction, it locks
the Organization and Project, revalidates both approvals and active memberships, assigns
`project_admin` to a different active member, deactivates the compromised Admin assignment, and
rechecks that the replacement is the active Project Admin before commit. It cannot target the
initiating Owner, retain the compromised assignment, change any other role, or remove membership.
Membership removal, if required, is a later operation after the safe replacement commits.

Both bootstrap workflows audit the initiating authenticated user as the created actor and record one
correlation identifier for the resource and assignment transaction. Project Admin recovery is a
separate system-owned transaction. Its durable authenticated system execution actor does not grant
authorization; the verified initiating Owner and independent human approval above do. The
transaction locks the Organization, Project, active memberships, Owner assignments, and Project
Admin assignments; revalidates target eligibility, both approvers, recent MFA, and either the
missing-admin recovery condition or the exact compromised-final-Admin replacement condition;
writes the assignment changes and immutable audit atomically; and fails without mutation when any
input changed. It never uses `unknown`, a client assertion, or an unauthenticated actor as
runtime provenance. The authentication and MFA prerequisite must define the execution actor,
platform-security approver identity, recent-MFA proof, incident record, immutable dual-approval
audit, and operator separation, so Project Admin recovery cannot be enabled earlier.

Organization Owner account or role recovery is outside this RFC and remains disabled. Neither the
Project Admin recovery workflow nor its execution actor may create, replace, or reactivate an
Organization Owner assignment. A separate accepted authorization RFC must define Owner-recovery
initiator eligibility, target eligibility, distinct approvers, assurance, tenant verification,
transaction locks, final-Owner behavior, immutable audit, and rollback before any such endpoint is
implemented or enabled.

### Canonical permission catalog

The exact Organization permission keys are:

```text
organization.read
organization.update
organization.delete
organization.ownership.manage
organization.billing.read
organization.billing.manage
organization.security.read
organization.security.manage
organization.sessions.revoke
organization.members.read
organization.members.invite
organization.members.remove
organization.projects.read
organization.projects.create
organization.projects.archive
organization.roles.read
organization.roles.create
organization.roles.update
organization.roles.archive
organization.access.read
organization.access.manage
organization.project_access_tokens.create
organization.project_access_tokens.read_metadata
organization.project_access_tokens.reduce_access
organization.project_access_tokens.rotate
organization.project_access_tokens.revoke
organization.provider_credentials.read_metadata
organization.provider_credentials.create
organization.provider_credentials.update
organization.provider_credentials.rotate
organization.provider_credentials.revoke
organization.audit.read
organization.audit.export
```

Organization permissions are user-only. Organization-token lifecycle and Organization-token roles
are deferred together. No `organization.access_tokens.*` or `organization.runtime.invoke`
permission is seeded.

The exact Project permission keys are:

```text
project.read
project.update
project.archive
project.roles.read
project.access.read
project.access.manage
project.resources.read
project.resources.create
project.resources.update
project.resources.delete
project.deployments.read
project.deployments.manage
project.runtime.invoke
project.conversations.read
project.conversations.delete
project.conversations.export
project.observability.read
project.observability.export
project.access_tokens.create
project.access_tokens.read_metadata
project.access_tokens.reduce_access
project.access_tokens.rotate
project.access_tokens.revoke
project.provider_credentials.read_metadata
project.provider_credentials.create
project.provider_credentials.update
project.provider_credentials.rotate
project.provider_credentials.revoke
project.provider_credentials.use
project.audit.read
project.audit.export
```

Token-eligible Project permissions are:

```text
project.read
project.resources.read
project.resources.create
project.resources.update
project.resources.delete
project.deployments.read
project.deployments.manage
project.runtime.invoke
project.conversations.read
project.conversations.delete
project.conversations.export
project.observability.read
project.observability.export
project.provider_credentials.read_metadata
project.provider_credentials.use
project.audit.read
project.audit.export
```

Seeded catalog display metadata is derived exactly from each immutable key. Split the key on `.`
and `_`; for each nonempty lowercase ASCII token, uppercase only its first ASCII letter and preserve
the remaining bytes; join tokens with one space. That result is `name`. Permission `description` is
the literal `Allows <key>.`, substituting the unchanged permission key. Built-in role `description`
is the literal `Built-in role <key>.`, substituting the unchanged role key. For example,
`organization.project_access_tokens.create` has name `Organization Project Access Tokens Create`
and description `Allows organization.project_access_tokens.create.`; `project_admin` has name
`Project Admin` and description `Built-in role project_admin.`. This rule owns every seeded name and
description; future custom-role display metadata is user-supplied and outside this seed.

### Canonical built-in roles

`ALL_ORGANIZATION` and `ALL_PROJECT` refer to the complete catalogs above. They are specification
notation, never stored wildcard permissions.

| Key | Scope | Subject | Exact permission set |
| --- | --- | --- | --- |
| `organization_owner` | Organization | User | `ALL_ORGANIZATION` |
| `organization_admin` | Organization | User | `ALL_ORGANIZATION` except `organization.delete`, `organization.ownership.manage`, both billing permissions, both security permissions, and `organization.sessions.revoke` |
| `access_admin` | Organization | User | `organization.read`, all member permissions, `organization.projects.read`, all organization role permissions, and both organization access permissions |
| `security_admin` | Organization | User | `organization.read`, both security permissions, `organization.sessions.revoke`, `organization.access.read`, and both audit permissions |
| `credential_admin` | Organization | User | `organization.read`, all project access-token permissions, and all provider-credential permissions |
| `billing_admin` | Organization | User | `organization.read`, `organization.billing.read`, and `organization.billing.manage` |
| `organization_auditor` | Organization | User | `organization.read`, `organization.billing.read`, `organization.security.read`, `organization.members.read`, `organization.projects.read`, `organization.roles.read`, `organization.access.read`, and both audit permissions |
| `organization_member` | Organization | User | `organization.read` and `organization.projects.read` |
| `project_admin` | Project | User | `ALL_PROJECT` |
| `project_developer` | Project | User | `project.read`, all resource permissions, `project.deployments.read`, `project.runtime.invoke`, `project.conversations.read`, `project.observability.read`, `project.provider_credentials.read_metadata`, and `project.provider_credentials.use` |
| `project_operator` | Project | User | `project.read`, `project.resources.read`, both deployment permissions, `project.runtime.invoke`, `project.conversations.read`, both observability permissions, and `project.audit.read` |
| `project_viewer` | Project | User | `project.read`, `project.resources.read`, `project.deployments.read`, `project.conversations.read`, and `project.observability.read` |
| `project_auditor` | Project | User | `project.read`, `project.roles.read`, `project.access.read`, `project.deployments.read`, both observability permissions, and both audit permissions |
| `runtime_invoker` | Project | Project token | `project.runtime.invoke` |
| `automation_client` | Project | Project token | `project.read`, all resource permissions, both deployment permissions, `project.runtime.invoke`, `project.observability.read`, `project.provider_credentials.read_metadata`, and `project.provider_credentials.use` |
| `observer_client` | Project | Project token | `project.read`, `project.resources.read`, `project.deployments.read`, `project.conversations.read`, `project.observability.read`, `project.observability.export`, `project.audit.read`, and `project.audit.export` |

### Authentication assurance

Role possession and authentication assurance remain separate. Recent MFA is required for Owner
and protected-role changes, Access Admin or Credential Admin assignment or change, Organization
deletion, billing and security policy, custom-role creation, custom-role permission reduction,
custom-role archival, creating, changing, or removing a Project Admin assignment, Project token
lifecycle and role assignment, provider-credential lifecycle, and Organization-wide session
revocation. Membership
removal that would deactivate any Project Admin assignment is a Project Admin removal and requires
recent MFA, including the Owner emergency-revocation path.

Authentication owns factors, OTP secrets, challenges, recovery codes, sessions, authentication
methods, verification timestamps, break-glass assurance, and the durable Project Admin recovery
execution identity.
Authorization consumes only server-derived assurance.

The backend evaluates permission before assurance so an MFA challenge never reveals an operation
to an unauthorized actor. Service, system, Organization-token, and Project-token actors cannot
satisfy a human MFA requirement.

An accepted authentication and MFA RFC is a hard prerequisite before any MFA-gated or recovery
operation is enforced. It must define recent MFA, recovery, audited break-glass issuance and
expiry, and the durable system actor used by Owner or Project Admin recovery. Until then, every
authority-granting, permission-increasing, secret-creating, secret-rotating, or recovery operation
requiring recent MFA is denied; missing assurance is never treated as satisfied.

This enforcement prerequisite is not a dependency for accepting this authorization policy or for
persisting its inactive catalog. Organization bootstrap uses the initiating authenticated User as
its audit actor and does not use the Project Admin recovery execution actor. MFA-gated and recovery endpoints remain
disabled until the separate authentication and MFA RFC supplies the assurance contract and durable
recovery identity defined above.

An already authenticated actor holding the underlying non-MFA permission may perform only these
monotonic emergency reductions when recent MFA is unavailable: revoke an access token, revoke a
provider credential, reduce an active custom-role permission bundle, or revoke sessions. The
operation must issue no replacement secret, add no permission, widen no audience or binding, and
create no assignment. It records actor, reason, affected scope, correlation identifier, and the
unavailable-assurance condition. All other MFA-gated operations fail closed. Break-glass recovery
is unavailable until the authentication RFC is accepted and implemented.

## Contracts and Ownership

- This RFC is the sole Version 1 policy source for subjects, permission keys, role keys, built-in
  bundles, delegation ceilings, and protected roles.
- Platform policy owns immutable built-in definitions.
- Organization role-management permissions own custom reusable definitions subject to permission
  ceilings.
- Organization membership owns whether a user may receive or exercise authority in that tenant.
- Organization Owner, Organization Admin, Access Admin, and Project Admin own only the built-in
  assignment paths explicitly listed above, subject to scope access management and full permission
  ceilings. Custom assignments always use the same checks.
- Credential lifecycle owns secret creation, rotation, and revocation but grants no token
  authority.
- Organization credential-lifecycle permissions target only active Projects belonging to the
  assigned Organization and grant no Project content, runtime, role, or token authority.
- Project token assignment owns token authority and enforces the Project permission ceiling.
- Token authentication owns immutable class, audience, and resource-binding metadata;
  authorization enforces it on every request.
- Authentication owns identity and assurance. Authorization does not own factors or sessions.
- Authorization owns Organization and Project bootstrap transactions; authentication owns their
  initiating user identity.
- A future authentication and MFA RFC owns break-glass assurance and provisions the durable
  Project Admin recovery execution actor before that recovery workflow is enforced.
- A future internal-actor compatibility RFC owns every service and system endpoint cutover; no
  affected path may rely on legacy fallback after that cutover.
- Persistence must implement this RFC after acceptance and may not add, omit, or reinterpret a
  subject, permission, role, or bundle.
- UI gating remains presentation only; backend endpoint owners enforce permission and assurance.
- Internal delegation compatibility must preserve actor, scope, effective permissions, token
  constraints, and assurance without widening them.

## Alternatives Rejected

- Keep people groups in Version 1. Rejected because direct assignment satisfies current needs and
  avoids inheritance and self-escalation rules.
- Seed Organization-token permissions without Organization-token roles. Rejected because the
  lifecycle and reduction semantics would have no Version 1 authorization subject.
- Allow Project Admin to mutate a reusable Project role. Rejected because the change would affect
  Projects outside that administrator's authority.
- Allow custom-role creation or assignment beyond the caller's permissions. Rejected because it
  bypasses billing, security, and content separation.
- Let Credential Admin grant token roles. Rejected because credential lifecycle authority does not
  imply the content or runtime authority carried by a token.
- Treat every Project token alike. Rejected because public widget and provider-channel credentials
  must never receive management-capable roles.
- Combine human and token audiences on one role. Rejected because token replacement and mutation
  contracts are stricter.
- Restrict reusable roles through a selected-Project allowlist. Rejected because direct
  assignments already determine where a role grants authority.
- Seed service or system roles without verified permission limits. Rejected because authentication
  identity alone does not define safe application authority.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Current runtime roles and credentials remain unchanged until separate migration and enforcement RFCs are accepted and implemented. |
| Failure and recovery | Direct assignments, single-audience roles, default deny, and immutable expansion paths reduce ambiguous authority. Unknown subjects, scopes, roles, and permissions fail closed. |
| Security and privacy | Active same-Organization membership, tenant matching, access-management checks, and permission ceilings prevent cross-tenant use and delegation beyond the caller. Membership removal enforces every affected assignment ceiling except for an audited, recent-MFA Owner emergency revocation that preserves final-role invariants. No secret-reveal permission exists. |
| Data and migration | No production data changes occur here. Persistence must seed this exact catalog only after acceptance. |
| Concurrency and resources | Final Owner and Project Admin mutations serialize through locked Organization and Project rows in deterministic order and recheck invariants before commit. Role reduction locks its preview snapshot. This policy creates no runtime resource. |
| Operations and observability | Future enforcement must audit role and token lifecycle changes, report denial categories, and avoid secret values in logs. |

## Rollout and Rollback

This RFC has no direct rollout. Acceptance order is strict:

1. obtain an approving external high-risk two-family panel for this exact candidate;
2. obtain explicit owner authorization for acceptance;
3. record the combined panel path, digest, and owner approval in the referenced task or pull
   request, then put only that concrete task or pull-request reference in this RFC;
4. change this RFC to `Accepted` before merge;
5. challenge and accept the persistence RFC against this accepted catalog;
6. implement the inactive catalog;
7. accept and implement the authentication and MFA RFC before any assurance-gated or recovery
   operation is enabled;
8. accept and implement the internal service and system compatibility RFC before any affected
   endpoint moves to Version 1 enforcement;
9. use separate accepted RFCs for membership, assignment, token, and endpoint cutovers.

Acceptance authorizes the persistence RFC challenge, not runtime enforcement. Current production
behavior remains authoritative until the separately accepted persistence and later cutover RFCs are
implemented.

Before enforcement, rollback removes the unused catalog. After enforcement, a new superseding
policy and migration are required. Runtime must never introduce group, undocumented internal-actor,
or Organization-token fallback behavior.

## Verification

This RFC is acceptable when:

- Version 1 has exactly User and Project Token assignable subjects;
- the permission catalog contains no group, Organization-token, service, or system permission;
- every built-in role has one canonical key, scope, subject, and exact bundle;
- every assigner has an exact built-in role-assignment matrix;
- custom Organization, Project human, and Project token roles have exact eligible catalogs and
  concrete creation and assignment ceilings;
- Project Admin cannot mutate reusable definitions;
- custom Organization-role creation enforces the creator's Organization permission subset; custom
  Project human-role and token-role creation uses the fixed eligible catalog without selecting an
  arbitrary Project; every assignment enforces the full permission subset in its concrete scope;
- custom-role creation writes its complete initial permission set atomically, later writes cannot
  add or reactivate a permission pair, and assigned-role reductions show affected scopes, direct
  users, and Project tokens before confirmation;
- every human built-in or custom assignment requires scope-appropriate access management and the
  full permission subset, and Organization roles cannot assign Project roles without a separate
  Project Admin assignment;
- Credential Admin can manage Project credential lifecycle but cannot grant token authority;
- creating and assigning a token requires both lifecycle and Project ceilings; first role assignment
  to an existing unassigned token atomically invalidates its old secret and releases the replacement
  only to the caller satisfying both ceilings;
- Organization credential-lifecycle authorization requires an active Organization assignment and
  rejects a target Project outside that Organization without granting Project content authority;
- rotating an assigned token requires its full Project ceiling, and every effective-permission
  increase replaces and revokes the old token;
- token rotation changes only the secret and preserves subject, assignments, audience, class,
  Project, and resource binding;
- removal authority is explicit for human built-in, human custom, and Project token assignments;
- assigned roles cannot expand in place;
- custom-role reduction requires `organization.roles.update`, and archival requires
  `organization.roles.archive`, in the definition's owning Organization;
- custom-role reduction confirmation is bound to the exact removals, resulting bundle, and an
  atomically revalidated role, permission, and assignment snapshot; stale or mismatched
  confirmation changes nothing and requires a new preview;
- roles with active assignments cannot be archived;
- public widget and channel tokens are runtime-only and enforce audience and resource binding;
- token audiences are limited to `WIDGET_RUNTIME`, `CHANNEL_RUNTIME`, `RUNTIME`, and `MANAGEMENT`;
  endpoint permission sets, required bindings, and widening or reduction rules match the closed
  audience contract;
- audience verification proves a qualified `MANAGEMENT` token can access `MANAGEMENT` and `RUNTIME`
  endpoints, a `RUNTIME` token cannot access `MANAGEMENT`, `MANAGEMENT` to `RUNTIME` removes only
  management access, and `RUNTIME` to `MANAGEMENT` requires replacement;
- protected-role membership removal is Owner-only and preserves the final Owner;
- concurrent assignment and membership removals serialize on deterministically ordered Organization
  and Project row locks, then recheck final Owner and Project Admin invariants before commit;
- User and Project-token assignment creation and reassignment use the shared Organization, Project,
  membership, role, token-subject, and assignment lock order and cannot commit from stale
  membership or role state;
- enforcement integration tests race removals of distinct Owner or Project Admin assignments and
  prove that at most one commits when both would otherwise remove the final role;
- ordinary membership removal satisfies the removal ceiling for every deactivated assignment;
- Owner emergency membership revocation works without any Project assignment by waiving both
  `project.access.manage` and Project permission-subset checks only for deactivation, while retaining
  recent MFA, audit, non-self, and final-role safeguards;
- Owner emergency membership revocation is recent-MFA, non-self, reasoned, correlated, fully
  audited, and preserves the final Owner and final Project Admin;
- every active Project preserves a final Project Admin through transfer or audited
  Organization Owner-approved Project Admin recovery;
- Project Admin recovery targets a member other than the initiating Owner and requires independent
  recent-MFA approval from another active Owner or, for a single-Owner Organization, an independent
  platform-security break-glass approver;
- creating, changing, directly removing, or membership-removing a Project Admin assignment requires
  recent MFA;
- a compromised final Project Admin can be replaced only by the dual-controlled recent-MFA
  workflow that atomically assigns a different active member and deactivates the compromised
  assignment under Organization and Project locks;
- active same-Organization membership and tenant matching are required for every user assignment
  and evaluation, membership removal atomically deactivates assignments, and rejoining restores no
  prior authority;
- Organization bootstrap is a narrow exception only for the first Owner of a new empty
  Organization; Project bootstrap requires `organization.projects.create`; both workflows are
  system-owned, transactional, and use authenticated audit provenance, and no custom Organization
  role may contain that permission;
- Access Admin and Credential Admin assignment or change requires recent MFA;
- an accepted authentication and MFA RFC is implemented before assurance-gated or recovery
  enforcement, including provisioning the durable Project Admin recovery execution actor;
- Organization Owner account and role recovery remains disabled until a separate authorization RFC
  defines and receives approval for that workflow;
- unavailable or unverifiable recent MFA permits only the enumerated audited monotonic emergency
  reductions and denies every authority increase, replacement-secret operation, or recovery;
- an accepted internal service and system compatibility RFC precedes every affected endpoint
  cutover, with no legacy fallback or undocumented bypass;
- signed internal delegation cannot widen actor, scope, effective permissions, token constraints,
  or assurance;
- concurrent Project token assignment and metadata mutations serialize on the token-subject row;
  integration tests prove concurrent first-role assignments cannot both commit;
- Project-token assignment and reassignment lock and revalidate target role definitions before
  reading permissions or writing assignments;
- every credential operation enforces the exact Organization-or-Project lifecycle predicate, and
  assigned-token reduction, rotation, replacement, and role assignment enforce their Project
  ceilings;
- every seeded permission and built-in role name and description matches the deterministic key-based
  display metadata rule;
- every endpoint enforces authorization on the backend and treats UI gating as presentation only;
- permission is evaluated before assurance, and machine actors cannot satisfy human MFA;
- persistence approval is explicitly sequenced after this policy acceptance;
- final panel evidence and owner approval are stored in the referenced task or pull request, while
  this RFC stores only that concrete reference;
- current production authorization remains unchanged until separately accepted migration and
  enforcement RFCs are implemented;
- RFC layout, style, diff, and finalization checks pass.

```bash
just validate-rfc-layout
just agent-style-check
git diff --check -- rfcs/0026-access-policy-v1.md
just agent-finalize "rfcs/0026-access-policy-v1.md"
```

## Review and Approval

- Challenger: Codex `gpt-5.6-sol` and `gpt-6-astra` high-risk panel
- Outcome: Approved
- Resolved findings: The owner authorized this new Governed cycle in conversation on 2026-11-03.
  This accepted policy preserves all approved Version 1 decisions and adds backend-only enforcement,
  authorization-before-assurance ordering, non-widening internal delegation, dual-control Project
  recovery, custom-role no-expansion persistence, rotation invariants, and impact preview.
  Correction pass 1 makes RFC 0026 the sole seed source, defines constrained Organization-scope
  credential authorization, and constrains Organization bootstrap to the first Owner. Correction
  pass 2 closes aggregate membership-removal ceilings and single-Owner Project Admin recovery. The
  post-escalation owner-directed correction makes every Project Admin removal MFA-gated, binds
  custom-role reduction to a fresh snapshot, distinguishes Organization scope from subjects, and
  separates policy acceptance from future assurance enforcement. The final owner-directed pass
  defines compromised-final-Admin replacement, explicit custom-role mutation permissions, and
  token-subject serialization. Both allegedly missing access-token keys were already present.
- Open questions: None
- Approval reference: `feature/access-management` delivery task, 2026-11-07
