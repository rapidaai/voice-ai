# RFC 0022: Role Catalog Persistence

- Status: Accepted
- Owner: Platform team
- Approved by: Repository owner
- Date: 2026-11-03
- Supersedes: N/A

## Context

The role catalog needs relational integrity before assignments or enforcement exist. RFC 0026 owns
the incorporated Version 1 permission and role catalog. RFC 0027 governs current policy; RFC 0028
will govern only after acceptance and supersession of RFC 0027. This persistence RFC becomes
eligible for challenge after RFC 0028 is Accepted and RFC 0027 is marked Superseded.

After that gate, this RFC creates and seeds only:

```text
permission_definitions
role_definitions
role_permissions
```

It does not modify current role tables or runtime behavior. Memberships, assignments, custom-role
APIs, credentials, tokens, endpoint enforcement, UI, caches, and vaults remain out of scope.

## Decision

### Permission definitions

`permission_definitions` is the seeded catalog of valid permission metadata:

```text
id               bigint, NOT NULL, primary key
key              varchar, NOT NULL, unique, immutable
scope_type       varchar, NOT NULL, ORGANIZATION or PROJECT
name             varchar, NOT NULL
description      varchar, NOT NULL
token_eligible   boolean, NOT NULL
```

The seed contains exactly the permission keys, scope types, and token eligibility accepted in RFC
0026. Seeded names and descriptions use RFC 0026's exact key-based display metadata rule. Endpoint
code will later reference stable keys and validate mappings against this catalog at startup.
Database rows own catalog membership and metadata; endpoint code owns which permission an operation
requires.

Seeded permission keys and metadata are immutable by application contract. This phase exposes no
runtime catalog writer, and the schema does not claim to prevent a privileged database operator
from updating a row. Any future writer must reject permission metadata update, archival, or
deletion before it is enabled.

Database constraints are:

```sql
CHECK (scope_type IN ('ORGANIZATION', 'PROJECT'))
CHECK (scope_type = 'PROJECT' OR token_eligible = false)
UNIQUE (key)
UNIQUE (id, scope_type, token_eligible)
```

### Role definitions

`role_definitions` stores built-in and future custom roles:

```text
organization_id   bigint, nullable, references organizations ON DELETE RESTRICT
id                bigint, NOT NULL, primary key
key               varchar, NOT NULL, immutable
name              varchar, NOT NULL
description       varchar, NOT NULL
scope_type        varchar, NOT NULL, ORGANIZATION or PROJECT
subject_type      varchar, NOT NULL, USER or PROJECT_TOKEN
definition_type   varchar, NOT NULL, BUILT_IN or CUSTOM
```

Built-in definitions are platform-owned and have no `organization_id`. Future custom definitions
require an active `organization_id`. Version 1 seeds only the exact immutable built-ins accepted in
RFC 0026, including names and descriptions derived by its exact display metadata rule.

The Organization foreign key deliberately prevents physical deletion while custom role definitions
remain. A future custom-role writer must archive its Organization's custom role permissions and
definitions after deactivating or removing every assignment. When retention permits physical
cleanup, it deletes dependent role permissions and role definitions before deleting the
Organization. The database never cascades an Organization deletion into authorization history.

Database constraints are:

```sql
CHECK (scope_type IN ('ORGANIZATION', 'PROJECT'))
CHECK (subject_type IN ('USER', 'PROJECT_TOKEN'))
CHECK (definition_type IN ('BUILT_IN', 'CUSTOM'))
CHECK (
    (definition_type = 'BUILT_IN' AND organization_id IS NULL) OR
    (definition_type = 'CUSTOM' AND organization_id IS NOT NULL)
)
CHECK (subject_type <> 'PROJECT_TOKEN' OR scope_type = 'PROJECT')
UNIQUE (id, scope_type, subject_type)

CREATE UNIQUE INDEX role_definitions_builtin_key_idx
    ON role_definitions (key)
    WHERE definition_type = 'BUILT_IN';

CREATE UNIQUE INDEX role_definitions_custom_key_idx
    ON role_definitions (organization_id, key)
    WHERE definition_type = 'CUSTOM';
```

Future custom-role writes must also reject a key reserved by a built-in definition. That write path
does not exist in this phase and requires its own accepted API and enforcement contract.

### Role permissions

`role_permissions` is the audited join:

```text
id                       bigint, NOT NULL, primary key
role_definition_id       bigint, NOT NULL
permission_definition_id bigint, NOT NULL
scope_type               varchar, NOT NULL, ORGANIZATION or PROJECT
subject_type             varchar, NOT NULL, USER or PROJECT_TOKEN
token_eligible           boolean, NOT NULL
created_date             timestamp, NOT NULL
updated_date             timestamp, nullable
status                   varchar, NOT NULL
created_actor_type       varchar, NOT NULL
created_actor_id         bigint, nullable only when created_actor_type = 'unknown'
updated_actor_type       varchar, nullable only as a pair with updated_actor_id
updated_actor_id         bigint, nullable only as a pair with updated_actor_type
```

The join has these executable PostgreSQL constraints:

```sql
FOREIGN KEY (role_definition_id, scope_type, subject_type)
    REFERENCES role_definitions (id, scope_type, subject_type)
FOREIGN KEY (permission_definition_id, scope_type, token_eligible)
    REFERENCES permission_definitions (id, scope_type, token_eligible)
CHECK (scope_type IN ('ORGANIZATION', 'PROJECT'))
CHECK (subject_type IN ('USER', 'PROJECT_TOKEN'))
CHECK (subject_type <> 'PROJECT_TOKEN' OR token_eligible = true)

CREATE UNIQUE INDEX role_permissions_active_pair_idx
    ON role_permissions (role_definition_id, permission_definition_id)
    WHERE status = 'ACTIVE';
```

The composite foreign keys enforce scope agreement and token eligibility without triggers. The
schema does not enforce built-in immutability. In this phase the migration is the only writer and
the runtime has no catalog mutation path. Before a runtime writer exists, its accepted contract
must reject update, archival, or deletion of built-in definitions and bundles. A later Governed
migration may add a new built-in key.

A future custom-role writer must create a definition and its complete initial permission pairs in
one transaction. After that transaction, it may archive existing pairs for a reduction but must
reject every insertion or reactivation of a permission pair for that role. Expansion uses a new
immutable role key. The partial unique index prevents duplicate active pairs but does not replace
this writer invariant.

### Audit and lifecycle

All three tables use the repository's audited and mutable field names:

```text
id                   bigint, NOT NULL, primary key
created_date         timestamp, NOT NULL
updated_date         timestamp, nullable
status               varchar, NOT NULL
created_actor_type   varchar, NOT NULL
created_actor_id     bigint, nullable only when created_actor_type = 'unknown'
updated_actor_type   varchar, nullable only as a pair with updated_actor_id
updated_actor_id     bigint, nullable only as a pair with updated_actor_type
```

No table adds `created_by`, `updated_by`, or `archived_date`. Every table has:

```sql
CHECK (status IN ('ACTIVE', 'ARCHIEVE'))
CHECK (
    (created_actor_type = 'unknown' AND created_actor_id IS NULL) OR
    (created_actor_type IN ('user', 'project', 'organization', 'service', 'system')
        AND created_actor_id IS NOT NULL
        AND created_actor_id > 0)
)
CHECK (
    (updated_actor_type IS NULL AND updated_actor_id IS NULL) OR
    (updated_actor_type IS NOT NULL
        AND updated_actor_id IS NOT NULL
        AND updated_actor_type IN ('user', 'project', 'organization', 'service', 'system')
        AND updated_actor_id > 0)
)
```

The migration seed uses `(unknown, NULL)`. This is honest migration provenance and matches the
repository's historical audit backfill representation. It does not invent a durable system actor.
Future runtime writes require an authenticated durable actor. The accepted authentication and MFA
RFC must provision the durable Project Admin recovery execution actor required by RFC 0026 before
that workflow is enabled. That future actor does not retroactively replace the migration seed
provenance.

`ARCHIEVE` is the established repository lifecycle literal, despite its spelling. It is defined by
`pkg/types/enums/record_state.go` and enforced by existing web migrations 000006 and 000013. The
catalog migration and its tests preserve that exact value for compatibility.

### Migration behavior

The migration is single-run and transactional. It does not claim idempotent replay. It:

1. creates `permission_definitions`;
2. creates `role_definitions`;
3. creates `role_permissions`;
4. inserts exactly the accepted RFC 0026 permission catalog;
5. inserts exactly the accepted RFC 0026 global built-in roles;
6. inserts exactly the accepted RFC 0026 role bundles;
7. validates expected counts, scope agreement, token eligibility, and complete bundles;
8. commits only when every validation succeeds.

Unknown permissions fail through foreign keys. Invalid enum values fail through checks. Duplicate
built-in keys and duplicate active pairs fail through unique indexes. The current runtime never
reads these tables in this phase.

## Contracts and Ownership

- RFC 0028 governs policy and incorporates RFC 0026 as the catalog source. This RFC owns only
  persistence.
- The database catalog owns valid permission metadata and built-in role composition after seed.
- Endpoint owners later map operations to stable permission keys and validate them at startup.
- The Platform team owns schema, migration, exact seed, rollback, and migration verification.
- Current role-string tables remain runtime-authoritative until a later accepted cutover RFC.
- Built-in catalog immutability is an application contract. This phase exposes no runtime writer;
  a future writer must enforce the contract before it is enabled.
- Custom-role no-expansion is an application writer contract. Initial permission pairs are atomic;
  later writes may archive pairs but cannot insert or reactivate them for the existing role.
- Seeded permission-definition immutability is the same application contract; future writers must
  reject metadata update, archival, and deletion.
- The `role_definitions.organization_id` foreign key uses `ON DELETE RESTRICT`. A future custom-role
  writer owns assignment deactivation, role-permission archival, role-definition archival, and
  retention cleanup before an Organization may be physically deleted.
- The migration seed alone uses `(unknown, NULL)`. A future authentication and MFA RFC must
  provision the durable Project Admin recovery execution actor before that workflow is enabled.
- No API, assignment, membership, credential, token, UI, cache, vault, or runtime enforcement
  contract changes in this phase.

## Alternatives Rejected

- Accept persistence before policy. Rejected because a Draft policy cannot authorize a catalog
  that differs from the currently accepted RFC.
- Store only a permission string on `role_permissions`. Rejected because SQL could not validate an
  unknown key through a foreign key.
- Store bundles as JSON. Rejected because relational foreign keys and unique pairs provide direct
  integrity and impact queries.
- Add assignments now. Rejected because the inactive catalog can be verified and rolled back
  independently.
- Attribute seed rows to `(system, 1)`. Rejected because no durable Project Admin recovery execution
  actor exists in this phase. A future authentication and MFA RFC may provision one for runtime
  recovery, but it cannot truthfully attribute earlier migration seed rows.
- Seed built-ins for each current Organization. Rejected because later Organizations would require
  a catch-up path. Global immutable built-ins are complete for current and future Organizations.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The migration is additive and unused by the current runtime, so existing authorization remains unchanged. |
| Failure and recovery | Schema and seed are one transaction. Any constraint, count, eligibility, or bundle mismatch rolls back all three tables. |
| Security and privacy | Foreign keys validate permission membership. Check constraints validate scope, subject type, and token eligibility. No grant, token audience, or secret is created. |
| Data and migration | Existing data is untouched. Permission, built-in role, and bundle rows are global and seeded once with honest migration attribution. |
| Concurrency and resources | Inserts have fixed catalog size and do not rewrite current role tables. |
| Operations and observability | Verification reports counts and mismatched keys or bundles without logging credentials or secret values. |

## Rollout and Rollback

Do not challenge or accept this RFC until RFC 0028 is Accepted and RFC 0027 is Superseded. After
that transition, run the additive
migration before any catalog reader ships. Stop on any validation failure.

The down migration drops `role_permissions`, `role_definitions`, then
`permission_definitions`. Rollback is safe because this phase creates no custom roles,
assignments, or runtime readers.

## Verification

The change is acceptable when:

- RFC 0028 is Accepted and RFC 0027 is Superseded before this RFC is challenged;
- exactly three catalog tables are created;
- all enum-like and audit pairs have executable check constraints;
- lifecycle constraints use the established `ARCHIEVE` repository literal;
- composite foreign keys can be created against the declared unique column sets;
- the active role-permission pair has a partial unique index;
- the seed exactly matches the accepted RFC 0026 catalog and bundles;
- every seeded permission and built-in role name and description matches RFC 0026's exact
  key-based display metadata rule;
- Organization permissions cannot be marked token-eligible;
- seed audit provenance is `(unknown, NULL)`;
- no runtime writer can mutate seeded permission metadata or built-in bundles in this phase;
- a future custom-role writer atomically creates initial pairs and rejects every later insertion
  or reactivation for that role, using a new role key for expansion;
- the role-definition Organization foreign key uses `ON DELETE RESTRICT`, and a database test
  proves an Organization with a dependent custom definition cannot be physically deleted;
- custom-role archival cleanup and the durable Project Admin recovery execution actor remain unavailable until
  their owning accepted RFCs are implemented;
- unknown permissions, invalid enum values, duplicate keys, duplicate active pairs, scope
  mismatches, and ineligible token permissions are rejected;
- current role and authorization behavior remains unchanged;
- PostgreSQL tests cover up, invalid seed, count, constraint, and down paths;
- entity tests cover table names, nullable seed audit identity, and persisted fields;
- RFC, style, diff, and repository finalization checks pass.

Expected commands after RFC 0028 acceptance and RFC 0027 supersession are:

```bash
just validate-rfc-layout
just agent-style-check
git diff --check
go test ./api/web-api/internal/entity/...
just ci-integration
just agent-finalize "api/web-api/migrations,api/web-api/internal/entity,rfcs/0022-role-permission-persistence-migration.md"
```

## Review and Approval

- Challenger: Codex `gpt-5.6-sol` and `gpt-6-astra` high-risk panel
- Outcome: Approved
- Resolved findings: Draft specifies policy-first sequencing, honest seed attribution, future
  authentication ownership of the Project Admin recovery execution actor, explicit NOT NULL fields, NULL-safe
  actor checks, enum checks, composite reference uniqueness, active-pair uniqueness,
  application-owned built-in immutability, restricted Organization deletion with archival cleanup
  ownership, non-circular catalog ownership, RFC 0026-only seed attribution, and explicit shared
  audit fields on the role-permission join.
- Open questions: None
- Approval reference: `feature/access-management` delivery task, 2026-11-07
