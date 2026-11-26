# RFC 0047: User Role Assignment Migration

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-10
- Supersedes: RFC 0029 user-role assignment deferral only

## Context

`user_organization_roles` and `user_project_roles` still store legacy role names in a `role`
varchar column. Their unique constraints allow only one row per user, scope, and status, so they
cannot represent multiple active roles. Authentication and authorization still read these strings
and compare them with hard-coded names.

The accepted role catalog is now the source of truth for role definitions and permissions.
Migration 000016 separately assigns every project credential to one Project `PROJECT_TOKEN` role.
User assignments must reference the same catalog without introducing new assignment tables or
combining user and token ownership in a polymorphic table.

This is a Governed data and authorization-boundary change. Applying the database migration before
the corresponding entity, service, authentication, and authorization cutover would break legacy
reads and writes. The migration may be prepared and verified independently, but it is not
deployable until that code cutover is ready in the same coordinated release.

## Decision

The existing tables remain the assignment owners. Migration 000017 converts their legacy `role`
values to catalog IDs, changes the column type to bigint, and renames it to
`role_definition_id`. No user-assignment table is added.

The migration uses this deterministic mapping:

| Assignment table | Legacy role | Role definition ID | Role key |
| --- | --- | --- | --- |
| `user_organization_roles` | `owner` | 101 | `organization_owner` |
| `user_organization_roles` | `admin` | 102 | `organization_admin` |
| `user_organization_roles` | `member` | 108 | `organization_member` |
| `user_project_roles` | `super admin` | 109 | `project_admin` |
| `user_project_roles` | `admin` | 109 | `project_admin` |
| `user_project_roles` | `writer` | 110 | `project_developer` |
| `user_project_roles` | `reader` | 112 | `project_viewer` |

The `super admin` and `admin` project values intentionally converge on `project_admin`. The
migration fails before changing data if any row contains another value. It does not silently grant
a fallback role.

Each user assignment stores `role_scope_type` and `role_subject_type` as constraint witnesses.
Organization rows require `ORGANIZATION` and `USER`; Project rows require `PROJECT` and `USER`.
Composite foreign keys use the existing role-catalog reference keys to reject unknown IDs,
wrong-scope roles, and project-token roles. The witnesses are database constraint inputs, not a
second policy source.

The old unique constraints are replaced with partial unique indexes over user, scope, and role for
current `ACTIVE` or `INVITED` rows. A user may therefore hold multiple current roles at the same
scope, while the same current role cannot be assigned twice. Archived history is not artificially
limited to one row.

Migration 000017 converts existing assignments only. The coordinated backend cutover may assign
the mapped built-in `USER` roles, but custom user-role assignment remains blocked. Enabling custom
user-role assignment requires a later accepted RFC that defines permission-reduction impact and
role-archive behavior for active user assignments. That later writer must lock and validate the
role in the assignment transaction. A custom Organization role must belong to the assignment's
Organization; a custom Project role must belong to the Project's Organization. Unknown, archived,
wrong-scope, wrong-subject, and cross-tenant roles must fail as one unavailable selection without
persisting an assignment.

Migration 000016 remains the owner of project-token assignment. `project_credentials` continues to
hold exactly one `role_definition_id` with a `PROJECT_TOKEN` subject witness. User and token
assignments share `role_definitions` and resolve permissions through `role_permissions`, but retain
separate tables and lifecycle owners.

The down migration is fail-closed. It runs only when every user assignment still references one of
the mapped built-in roles and neither table has duplicate rows for the legacy unique key of user,
scope, and status. It then restores the legacy strings and original unique constraints. Project
role 109 restores as `admin`; the old `super admin` distinction is intentionally unrecoverable
after convergence.

## Contracts and Ownership

- `user_organization_roles` remains the source of truth for Organization user-role assignments.
  Its role reference accepts only `ORGANIZATION` `USER` definitions.
- `user_project_roles` remains the source of truth for Project user-role assignments. Its role
  reference accepts only `PROJECT` `USER` definitions.
- `project_credentials` remains the source of truth for the credential's single Project
  `PROJECT_TOKEN` role under RFC 0039 and migration 000016.
- `role_definitions` remains the source of truth for role scope, subject, ownership, and lifecycle.
  `role_permissions` remains the source of truth for permission membership.
- A current assignment is a row whose status is `ACTIVE` or `INVITED`. Multiple different current
  roles are permitted at one scope; duplicate current assignment of the same role is rejected.
- The migration owner owns exact legacy mapping, constraint replacement, atomicity, verification,
  and fail-closed rollback checks.
- The later code cutover owns entity fields, write validation, role loading, permission evaluation,
  and removal of hard-coded role-name checks. It permits only built-in user-role assignment. This
  RFC does not activate catalog authorization or custom user-role assignment.
- A later custom-assignment writer owns transactional role locking and tenant validation. No custom
  Organization role may cross its `organization_id`; no custom Project role may cross the owning
  Organization of its Project.
- Existing project-token assignment data and constraints are unchanged by migration 000017.
- No public API or protobuf contract changes in this migration stage.

## Alternatives Rejected

- Add new user-role assignment tables: rejected because the existing scope-specific tables already
  own assignment lifecycle and audit data. Replacing them would add a dual-write migration without
  improving the model.
- Add one polymorphic assignment table for users and tokens: rejected because it weakens foreign
  keys, mixes different lifecycle owners, and requires nullable or type-dependent subject fields.
- Store multiple role IDs in an array or JSON column: rejected because it loses row-level audit,
  uniqueness, referential integrity, and ordinary relational queries.
- Keep legacy role strings and infer permissions in code: rejected because it duplicates the role
  catalog and prevents custom roles from becoming assignable.
- Add only a scalar role-definition foreign key: rejected because it would allow Organization user
  rows to reference Project roles and Project user rows to reference `PROJECT_TOKEN` roles.
- Enable custom user-role assignment in this cutover: rejected because role reduction and archival
  do not yet account for active user assignments. Schema capability does not authorize a writer.
- Rewrite project-token assignments in this migration: rejected because migration 000016 already
  owns their backfill and constraints.
- Preserve the `super admin` distinction with a new role: rejected because the approved catalog has
  one full Project user role, `project_admin`, and both legacy values have the same destination.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The table column name and type change is intentionally incompatible with the current backend. Migration 000017 must not deploy while any legacy instance is running. Existing IDs, audit fields, statuses, dates, and assignment rows are preserved. |
| Failure and recovery | The up migration is one transaction and fails on unknown legacy values, missing catalog roles, incomplete conversion, or constraint violations. The down migration fails before mutation when new roles or duplicate user-scope-status rows cannot fit the legacy schema. Recovery is a forward fix or database restore when those rollback guards fail. |
| Security and privacy | Composite foreign keys reject wrong-scope and wrong-subject grants at the database boundary. Unknown role values receive no fallback. Built-in assignment only is permitted after cutover. Custom assignment remains blocked until tenant validation and active-assignment reduction and archival behavior are accepted and implemented. This stage does not expose additional user data. |
| Data and migration | Legacy names map to stable built-in IDs. `super admin` and `admin` converge irreversibly on role 109. Multiple current roles become representable. No rows or audit history are deleted. Token assignment data is untouched. |
| Concurrency and resources | Access-exclusive locks serialize assignment writes while columns, values, and constraints change. The migration duration is proportional to both user-role tables and requires a coordinated write pause. No runtime worker or long-lived resource is added. |
| Operations and observability | Access Management owns row-count and legacy-value preflight, the coordinated deployment window, migration verification, and rollback decision. Unknown-value counts, conversion counts, constraint presence, and post-migration assignment counts are release evidence. |

## Rollout and Rollback

1. Inventory exact legacy values and row counts in both user-role tables. Stop if values outside the
   mapping exist.
2. Prepare and verify migration 000017 without deploying it. Migration 000016 must already be
   present because the role-subject catalog reference is reused.
3. Complete the backend cutover so reads and writes use role definition IDs and permission
   evaluation, while assignment writers accept only built-in `USER` roles. Verify that no code
   writes or compares legacy role strings.
4. Enter a maintenance window and stop every legacy Web API instance before migration 000017. Do
   not use a rolling deployment because a legacy instance cannot read the renamed column.
5. Apply migration 000017, deploy only the compatible backend, then verify mapping counts,
   witnesses, foreign keys, partial indexes, credential assignments, and the built-in-only writer
   rule before restoring traffic.
6. Stop or roll forward if any preflight, migration, or verification check fails. Do not bypass an
   unknown value or substitute a default role.

Before assignments use custom roles or create duplicate user-scope-status rows, rollback may run
migration 000017 down and then restore the legacy backend. After either capability is used, the
down migration intentionally fails because the legacy schema cannot represent the data. Operators
must remove or translate the incompatible assignments under an approved recovery plan, or restore
a pre-cutover snapshot. Migration 000016 rolls back separately and is not modified by this
procedure.

## Verification

Acceptance requires:

- all recognized Organization and Project legacy values map to the exact role IDs above;
- unknown legacy values make the migration fail atomically;
- the migrated role column is bigint and named `role_definition_id`;
- scope and subject witnesses are non-null and enforced by checks and composite foreign keys;
- different current roles may coexist at one scope, while a duplicate current assignment fails;
- wrong-scope, wrong-subject, and unknown role assignments fail;
- token credential assignments from migration 000016 remain unchanged;
- backend assignment tests reject custom roles until a later accepted assignment-lifecycle RFC;
- the later custom-assignment enablement includes transactional tests that reject a custom
  Organization role owned by another Organization and a custom Project role owned by another
  Project Organization;
- down and re-up preserve representable rows, and down fails for an unmapped role or duplicate
  user-scope-status assignment;
- an injected migration failure leaves the legacy columns, values, and constraints intact.

Exact commands:

```text
bash bin/verify-role-catalog-migration.sh
just validate-rfc-layout
git diff --check
just agent-finalize "api/web-api/migrations/000017_migrate_user_role_assignments.up.sql,api/web-api/migrations/000017_migrate_user_role_assignments.down.sql,bin/verify-role-catalog-migration.sh,rfcs/0047-user-role-assignment-migration.md,rfcs/README.md"
```

## Review and Approval

- Challenger: Independent Codex task `/root/user_role_migration_challenge`
- Outcome: Approved
- Resolved findings: Down migration now checks every legacy user-scope-status key; rollout stops all legacy instances; custom user-role assignment remains blocked pending its lifecycle RFC; future custom writers own transactional same-tenant validation.
- Open questions: None
- Approval reference: Repository owner instruction on 2026-11-10 to reuse the existing tables and write the user and token migration; token assignment remains owned by accepted RFC 0039 and migration 000016.
