# RFC 0039: Project Token Role Assignment

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-09
- Supersedes: RFC 0029 custom-role assignment deferral and credential-selector read authorization only; RFC 0037 Credentials assignment deferral only

## Context

Project credential creation currently accepts only a Project ID and name. The Web API creates a
standalone `project_credentials` row and returns its `rpx_` key without validating a Project role or
persisting a role association. Existing project credentials therefore cannot supply role input to a
later request-authorization cutover.

The current caller contract permits any authenticated user with Organization context to create a
project credential. It does not require an owner or admin role. The role-catalog adapter currently
limits every `ListRoles` and `GetRole` call to owners and admins, so applying role selection without
a narrow read change would break credential creation for an existing non-admin caller.

The role catalog already owns role scope, subject, definition type, tenant ownership, lifecycle
status, and active permission membership. Its three built-in project-token roles are intentionally
narrower than full Project access. `ListRoles` can return active built-in roles and active custom
roles owned by the authenticated Organization, and `GetRole` returns the selected role's active
permissions.

This is a Governed change because it adds a public protobuf request field, changes a read-
authorization boundary, adds mandatory persisted authorization data, migrates existing credentials,
changes custom-role archival behavior, and coordinates backend, generated client, SDK, and UI
releases. The assignment data remains dormant: project-token authentication and request
authorization do not read it in this change.

The original run wording said to preserve one-time secret delivery. Repository inspection proved
that wording stale, and the repository owner correction supersedes it. Create and list responses
currently return the credential key, and the UI can reveal or copy listed keys. This RFC preserves
that repeated key disclosure exactly. One-time reveal, redaction, hashing, storage, rotation,
revocation, and lost-response recovery are explicit non-goals.

This RFC partially supersedes two earlier RFCs and no other decisions. It supersedes RFC 0029 where
that RFC required impact preview and a no-active-assignment rule before a custom role could receive
any assignment. A dormant project-token association may now target a custom role, but impact preview
remains required before enforcement. It also supersedes RFC 0029's owner/admin catalog-read
authorization only for the explicitly filtered Project plus `PROJECT_TOKEN` `ListRoles` path and
eligible project-token `GetRole` path used by credential creation. Broader or unfiltered catalog
reads, `ListPermissions`, and every mutation remain owner/admin-only. It supersedes RFC 0037 only
where role assignment and changes to the Credentials experience were deferred. RFC 0029's remaining
role API, lifecycle, authorization, mutation, and tenant rules and RFC 0037's remaining
role-management UI decisions remain authoritative.

## Decision

Every new project credential request must select exactly one active Project-scoped `PROJECT_TOKEN`
role. `CreateProjectCredentialRequest` adds the wire-additive field
`uint64 roleDefinitionId = 3 [jstype = JS_STRING]`. The field is application-required: a missing or
zero value fails, and the server never selects a default. Existing fields retain their numbers and
meaning.

### Caller and selector-read authorization

Credential creation preserves its current authorization predicate. Any authenticated user with a
valid Organization context may create for an active Project in that Organization. This RFC does not
add an owner/admin check, a new membership-role check, or permission evaluation to
`CreateProjectCredential`. Project ownership and selected-role eligibility are resource validation,
not a new caller-role gate.

The Access Management read adapter gains one narrow exception so every permitted credential creator
can select and preview an eligible role:

- An owner or admin retains the existing `ListRoles` and `GetRole` behavior.
- A non-admin authenticated user with valid Organization context may call `ListRoles` only when the
  request explicitly filters scope to `PROJECT` and subject to `PROJECT_TOKEN`. Query, pagination,
  and built-in or custom definition-type filters remain available inside that exact boundary.
- This selector path uses the same authenticated-user and Organization-context predicate as
  credential creation. It does not add an owner/admin or other membership-role requirement.
- The same non-admin may call `GetRole` only for an active Project-scoped `PROJECT_TOKEN` built-in or
  an active custom role owned by the user's Organization.
- For a non-admin, missing filters, unfiltered requests, `USER` subjects, `ORGANIZATION` scope,
  valid but non-selector filter combinations, and every broader catalog read return
  `PermissionDenied`. Malformed or unknown enum values remain `InvalidArgument` for every caller.
- A non-admin `GetRole` request for an unknown, archived, cross-tenant, wrong-scope, or wrong-subject
  role returns the same `NotFound` status and message. It does not reveal which predicate failed.
- `ListPermissions` and all role mutations retain their existing owner/admin restriction. No
  non-admin create, update, reduction, or archive access is introduced.

The narrow read returns only the existing active role metadata and active permissions needed by the
selector. It does not make another tenant's role, an archived role, an Organization role, a `USER`
role, or any mutation available.

### Transaction and persistence

The backend validates and creates the credential in one database transaction. In deterministic
order it locks and validates the active Project as belonging to the authenticated Organization,
then locks and validates the selected active role. The role must be Project-scoped and have
`PROJECT_TOKEN` subject and must be either a global built-in role or a custom role owned by the same
Organization. Any validation, cancellation, deadline, insert, or commit failure before commit rolls
back the transaction and returns no key. The key is returned only after commit.

Each `project_credentials` row owns one immutable role association through a non-null
`role_definition_id` and one constant `role_subject_type` witness constrained to `PROJECT_TOKEN`.
A composite foreign key from `(role_definition_id, role_subject_type)` references a unique
`(id, subject_type)` role-catalog key. The existing role-catalog constraint that
`PROJECT_TOKEN` implies `PROJECT` scope makes a Project-scope witness redundant, so this RFC adds no
scope witness. The subject witness is a constraint input, not a second policy source. No assignment
table or role-replacement writer is added.

The migration creates one global built-in compatibility role with this canonical metadata:

| Field | Value |
| --- | --- |
| Key | `project_token_full_access` |
| Name | `Project Token Full Access` |
| Description | `Grants a project token the full Project permission bundle captured when project-token role assignment was introduced.` |
| Scope | `PROJECT` |
| Subject | `PROJECT_TOKEN` |
| Definition type | `BUILT_IN` |
| Status | `ACTIVE` |

The compatibility role is immutable through the public role API and contains exactly the 31 active
Project permissions present when the migration is introduced. A later permission addition does not
silently expand this bundle; changing it requires a later governed catalog decision. The migration
assigns this role to every pre-existing project credential, including archived credentials, before
making the association non-null. The role is visible and selectable for a new credential, but the
UI and backend never preselect or inject it.

A custom role referenced by any active project credential cannot be archived. Archival locks the
role and checks active credential references in the same transaction, returning
`FailedPrecondition` without mutation when a reference exists. Creation and archival serialize on
the role lock. Archived credential references do not block archival. Existing monotonic permission
reduction of a custom role remains allowed and changes the shared bundle without replacing any
token's role. Impact preview and confirmation for reductions remain a prerequisite of the later
authorization-enforcement cutover.

### Public error contract

`CreateProjectCredential` continues using its existing response envelope. Expected selection and
resource failures return a failure envelope with gRPC status `OK`, so the generated callback receives
the envelope. Authentication, caller denial, context termination, and unexpected server failures
return no response message and use canonical non-`OK` gRPC status. The server never returns a
failure envelope together with a non-`OK` status.

| Condition | `CreateProjectCredentialResponse` on the wire | gRPC status and message |
| --- | --- | --- |
| Success after commit | `code=200`, `success=true`, credential data present, `error` absent | `OK` |
| Missing or zero `roleDefinitionId` | `code=400`, `success=false`, `error.errorCode=400`, `error.errorMessage="roleDefinitionId is required"`, human message `Select a project-token role and try again.` | `OK` |
| Unknown or cross-tenant role | `code=404`, `success=false`, `error.errorCode=404`, generic eligible-selection messages, no credential data | `OK` |
| Archived role | The same code 404 envelope and messages as unknown or cross-tenant | `OK` |
| Wrong-scope or wrong-subject role | The same code 404 envelope and messages as unknown or cross-tenant | `OK` |
| Inactive, unknown, or cross-tenant Project | The same code 404 envelope and messages as unknown or cross-tenant | `OK` |
| Missing or invalid authentication | No response message | `Unauthenticated`, `valid user authentication is required` |
| Authenticated non-user caller or missing Organization context | No response message | `PermissionDenied`, `project credential creation permission denied` |
| Context canceled before commit | No response message | `Canceled`, `request canceled` |
| Deadline exceeded before commit | No response message | `DeadlineExceeded`, `request deadline exceeded` |
| Insert, commit, or other internal failure | No response message; no database detail or key | `Internal`, `project credential creation failed` |

The generic code 404 envelope uses `eligible project or project-token role not found` as
`error.errorMessage` and `The selected project or role is unavailable.` as its human message. It is
identical for missing, foreign, archived, wrong-scope, and wrong-subject resources, so callers cannot
probe tenant or hidden-role existence. Logs may record correlation, Organization, Project, role, and
credential IDs, but never keys or cross-tenant resource details.

`ListRoles` and `GetRole` retain the Access Management convention: successful responses use
`code=200`, `success=true`, and gRPC `OK`; failures return no response and use canonical gRPC status.
Non-admin selector filter mismatch returns `PermissionDenied` with
`access management permission denied`. Hidden eligible-role mismatch returns `NotFound` with
`role not found`. Missing authentication, cancellation, deadline, and internal failures retain
`Unauthenticated`, `Canceled`, `DeadlineExceeded`, and `Internal` respectively. Mutation denial
remains `PermissionDenied`.

### Credentials workflow and deferred enforcement

Both current Credentials creation entry points use one shared role-selection workflow. It loads all
pages of active Project-scoped `PROJECT_TOKEN` roles, starts with no selection, obtains the selected
role through `GetRole`, and previews each permission's name, key, and description. Create remains
disabled until the current selection's detail request succeeds. A selection change invalidates an
older detail request, and backend validation remains authoritative if catalog state changes before
submit.

Project-token authentication, claims, middleware, and endpoint authorization remain unchanged. No
request evaluates the stored role or its permissions under this RFC.

## Contracts and Ownership

- The protobuf contract owner owns field 3 and wire compatibility. Generated Go, Python, and React
  artifacts remain generator-owned. Existing request fields are not renumbered or reinterpreted.
- The existing project-credential API remains the owner of caller authentication, Organization
  context, response-envelope construction, and the exact gRPC mapping above. It does not acquire an
  owner/admin policy before permission enforcement.
- The Access Management read adapter owns the exact non-admin selector exception. The existing
  owner/admin adapter continues to own unfiltered reads, `ListPermissions`, and every role mutation.
- The project service owns transactional Project and role validation, tenant isolation, lock order,
  exactly-one persistence, commit authority, and typed failures used by the API mapping.
- The role catalog remains the source of truth for role eligibility and active permissions. The
  credential row is the source of truth for its selected role. Its subject witness exists only for
  referential integrity; the catalog constraint remains the source of the Project-scope rule.
- The migration owner owns the atomic compatibility-role seed, exact 31-permission bundle,
  all-row backfill, non-null association, subject check, composite foreign key, verification, and
  dependency-ordered down migration.
- Access Management and the React SDK maintainers jointly own SDK publication and release guidance.
  The normal release process selects the package version; this RFC does not invent one. Publication
  does not authorize consumption before the migration and backend readiness gate passes.
- Access Management owns the rollout window, caller inventory, external notice, readiness decision,
  stop decision, and rollback trigger. React SDK maintainers support package-consumer notice.
- The Credentials UI owns role discovery, no-default selection, permission preview, stale-request
  invalidation, submit state, envelope display, transport failures, and accessibility.
- `roleDefinitionId` is the caller's only selection input. New credentials persist only that
  requested role; there is no compatibility-role fallback.
- Older clients are wire-compatible but behaviorally incompatible with creation after cutover:
  missing field 3 fails closed until they upgrade. Existing credential reads and use remain
  compatible.
- Create and list key bytes, UI reveal and copy behavior, authentication claims, and request-
  authorization paths remain unchanged.
- No assigned-role metadata is added to credential responses. No generic assignment subsystem,
  user assignment, multiple role, reassignment, permission snapshot, or feature flag is introduced.

## Alternatives Rejected

- Require owner/admin for credential creation: rejected because it breaks the existing caller
  contract before permission enforcement and would leave the selector authorization mismatch
  unresolved for non-admin credential creators.
- Open the full role catalog to every credential creator: rejected because the selector needs only
  active Project-scoped `PROJECT_TOKEN` roles. Broader reads and all mutations retain existing
  restrictions.
- Add a separate eligible-role endpoint: rejected because exact `ListRoles` filters and a guarded
  `GetRole` path already express the necessary read boundary.
- Return different errors for foreign, archived, wrong-scope, or wrong-subject IDs: rejected because
  distinct public results would create a resource-existence oracle.
- Default new credentials to the compatibility role: rejected because it would bypass explicit
  selection and create accidental full access. Full access is automatic only for migration rows.
- Let old clients omit the role during a transition: rejected because it would create unassigned
  credentials and weaken the exactly-one invariant.
- Store both scope and subject witnesses: rejected because `PROJECT_TOKEN` already implies `PROJECT`
  through an existing database constraint. A second witness duplicates catalog policy.
- Use a general assignment or token-history table: rejected because one immutable role has no
  assignment lifecycle or history event. A non-null scalar expresses exactly one more directly.
- Reuse `project_admin`: rejected because it is a `USER` role. Expanding an existing project-token
  role would silently change an accepted least-privilege bundle.
- Add permission enforcement now: rejected because assignment correctness and authorization cutover
  have different rollout, observability, impact-preview, and rollback requirements.
- Change secret disclosure in this RFC: rejected because the owner correction requires current
  repeated disclosure to remain, and secret redesign is separable public-contract work.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Field 3 is wire-additive, but creation becomes behaviorally breaking because missing or zero role IDs fail. The database constraint prevents an old backend from creating an unassigned credential after migration, even if it ignores field 3 from a newer client. Existing non-admin callers retain creation and gain only the eligible selector reads they need. Old clients remain fail-closed until upgraded; existing reads, keys, authentication, and credential use remain unchanged. |
| Failure and recovery | Expected selection failures are stable response envelopes with gRPC `OK`; authentication, denial, cancellation, deadline, insert, commit, and internal failures use the exact statuses above. Migration and creation are transactional. Every pre-commit failure returns no key and leaves no row. Ambiguous post-commit delivery retains current reconciliation behavior. UI list and detail failures are retryable, stale details cannot replace a newer selection, and submit-time backend rejection is authoritative. |
| Security and privacy | Explicit no-default selection prevents accidental full access. Non-admin reads are constrained to active eligible roles and permissions; unfiltered, `USER`, `ORGANIZATION`, archived, and cross-tenant reads stay unavailable. Generic 404 mapping prevents role and tenant probing. Project and custom-role ownership are rechecked under lock. Persisted associations grant no runtime authority until a separate accepted enforcement RFC. Existing repeated key disclosure is intentionally unchanged and remains future security work. |
| Data and migration | A built-in role and 31 role-permission pairs are added, and every active or archived project credential is backfilled before the relation becomes non-null. The row adds only `role_definition_id` and a constant `PROJECT_TOKEN` subject witness; the existing subject-to-scope constraint proves Project scope. Exact metadata, counts, zero nulls, and referential constraints are migration invariants. The down migration preserves credential rows and keys while removing role-assignment data in dependency order. |
| Concurrency and resources | Project and role locks give creation one transaction owner and serialize create-versus-archive races. Either creation commits and archival fails on the active reference, or archival commits and creation rejects the inactive role. A commit failure rolls back both credential and association. UI request generations prevent late detail responses and double submit. No long-lived worker, stream, timer, or connection is introduced. |
| Operations and observability | Access Management owns a maximum 30-minute coordinated creation-outage window, caller inventory, readiness observation, and rollback. Known external callers receive notice before migration. Deployment state and probes must prove that no old backend is reachable before client consumption. A 10-minute no-progress interval or the 30-minute maximum triggers rollback. Logs carry operational IDs but never keys or hidden resource details. |

## Rollout and Rollback

Treat caller notice, migration, backend, SDK consumption, and UI deployment as one coordinated
release owned by Access Management. The first-party credential-creation outage window opens when the
migration commits and closes when the updated UI passes its creation smoke test. Its maximum length
is 30 minutes. Old external clients remain fail-closed beyond that window until they upgrade.

1. Before scheduling migration, Access Management inventories every known caller: both repository
   Credentials entry points, the UI credential client, the published React SDK wrapper, registered
   external SDK consumers, and known direct gRPC integrations. Each known external owner receives
   the cutover time, required field, package availability, fail-closed behavior, and rollback contact.
   A known critical caller that cannot upgrade or explicitly accept a creation outage postpones the
   migration.
2. Complete generation, migration rehearsal, focused tests, and independent review. Confirm exact
   compatibility-role metadata, the 31-permission bundle, complete backfill, the subject-only
   integrity witness, error mapping, non-admin selector boundary, and rollback path.
3. Access Management and React SDK maintainers may publish the generated SDK package for release
   preparation without prescribing a version here. Release guidance marks field 3 as not yet
   consumable. Publication is not permission for the UI or an external client to upgrade or send
   the field.
4. Apply the transactional database migration first. Its non-null and composite foreign-key
   constraints are the guardrail: old backends and old clients can no longer create credentials,
   and an old backend that ignores field 3 from a new client cannot insert an unassigned row.
5. Deploy the backend that supports, requires, validates, and persists field 3. Access Management
   observes the deployment controller until 100 percent of replicas report the new revision and no
   old revision remains registered or reachable. Readiness probes then prove selected-role creation,
   missing-role failure, non-admin selector access, broader-read denial, and persisted association.
   Only this recorded readiness observation authorizes client consumption.
6. After readiness, deploy the UI with the published SDK package and notify external consumers that
   upgrades may begin. Close the first-party window only after selection, preview, creation,
   unchanged key presentation, and database-persistence smoke tests pass. Old clients continue to
   receive the missing-role failure envelope until upgraded.

Stop immediately on generation drift, incomplete backfill, incorrect metadata or permission counts,
an unexpected scope witness, a reachable old backend, an unassigned insert, incorrect public error
mapping, authorization mismatch, selector failure, or an unresolved critical or major finding. If
no additional backend replica becomes ready for 10 consecutive minutes after migration, or if the
30-minute maximum would be exceeded before UI smoke completion, trigger rollback rather than
extending the window.

Before external consumption is authorized, rollback by stopping UI promotion, confirming no new
client is sending field 3, restoring the prior backend, and then running the dependency-ordered down
migration. This preserves credentials and keys and restores the prior unassigned-creation contract.

After external consumption is authorized, do not roll schema or backend back to a version that
ignores field 3 while creation remains reachable. Prefer forward repair. An emergency full rollback
must first block project-credential creation at the operational boundary, coordinate updated
clients, roll back UI consumption, and only then roll back backend and schema. Because request
authorization is deferred, removing dormant associations does not change runtime authority, but it
does remove the new creation invariant and requires a later migration before enforcement.

## Verification

Acceptance requires evidence that:

- the protobuf descriptor and generated Go, Python, and React outputs expose field 3 as `uint64`
  with JavaScript string mapping without changing existing fields;
- the React SDK and UI require and forward one role ID while preserving authentication, callback,
  response, and repeated key behavior;
- migration up, failure rollback, down, and re-up prove exact compatibility metadata, 31 active
  permission pairs, active and archived row backfill, zero nulls, invalid-reference rejection,
  no-default inserts, credential/key preservation, exactly one subject witness, and no scope witness;
- database tests prove the subject witness accepts only `PROJECT_TOKEN`, the composite foreign key
  rejects other role subjects, and the existing catalog check implies Project scope;
- an authenticated non-admin user with Organization context can create with a built-in or same-
  Organization custom eligible role and is not subjected to an owner/admin creation check;
- non-admin read tests prove exact Project plus `PROJECT_TOKEN` `ListRoles` and eligible `GetRole`
  success, while unfiltered, `USER`, `ORGANIZATION`, archived, cross-tenant, wrong-scope, and wrong-
  subject reads are denied or hidden exactly as specified;
- owner/admin read behavior, `ListPermissions`, and every mutation restriction remain unchanged,
  and non-admin mutation attempts remain `PermissionDenied`;
- API mapping tests assert every envelope field, gRPC code, and public message in the error table,
  including identical unknown, foreign, archived, scope, and subject results with no leaked detail;
- backend failure tests cover cancellation, deadline, insert failure, and an injected commit failure,
  proving no row, association, or key response survives;
- backend concurrency tests prove create-versus-archive serialization and no credential can commit
  against a role archived before its creation commit;
- authentication regression tests prove project-token claims and request middleware do not read or
  enforce the selected role;
- UI and browser tests cover complete pagination, empty initial selection, permission preview,
  non-admin success, broader-read denial, loading, empty, retry, stale completion, archived-before-
  submit, envelope and transport failures, double submit, unmount, accessibility, and unchanged key
  reveal and copy behavior;
- rollout evidence contains the known-caller inventory and notice, Access Management owner,
  deployment-controller readiness observation, probes, timestamps, 10-minute no-progress trigger,
  and 30-minute maximum; and
- a new independent exact-digest challenge has no unresolved critical or major finding.

Run the approved task plan's scoped finalization plus these exact gates:

```bash
bash bin/artifacts-generate.sh
(cd sdks/react && bash bin/artifacts-generate.sh)
(cd sdks/react && npm test -- --runInBand)
(cd sdks/react && npm run build)
go test ./api/web-api/internal/entity ./api/web-api/internal/service/project ./api/web-api/internal/service/accessmanagement ./api/web-api/api
bash bin/verify-role-catalog-migration.sh
(cd ui && yarn test --watchAll=false src/clients/__tests__/credential.client.test.ts src/app/components/domain/project-credentials/create-project-token.test.tsx src/app/pages/external-integration/rapida-credentials/index.test.tsx src/app/components/domain/cards/__tests__/cards.test.tsx)
(cd ui && yarn checkTs)
(cd ui && yarn verify:ui)
just ui-browser-test
just ci-contracts
just ci-integration
just validate-rfc-layout
just agent-style-check
git diff --check
bin/agent-review --working-tree --risk high
```

Any external reviewer transport must first pass `bin/agent-egress` and have explicit user
authorization. The implementation task or pull request retains the exact formatter and
`just agent-finalize` changed-path command; file-level execution detail is not duplicated here.

## Review and Approval

- Challenger: Codex plan challenger, Orca task `task_bf5187f23b9c`, dispatch `ctx_c2300b043e7c`
- Reviewed digest: SHA-256 `e33a45f9e6ad82944e32bdbf25cbe51be8fdb09dda23bd9124d27b057ebc2616`
- Outcome: Approved
- Revision round: Targeted challenge after the repository-owner Option A escalation resolution
- Resolved findings: The targeted challenger confirmed that the sole blocking Major was resolved with no remaining targeted finding. The approved candidate supersedes RFC 0029's owner/admin read rule only for the explicitly filtered Project plus `PROJECT_TOKEN` `ListRoles` path and eligible project-token `GetRole` path used by credential creation. Broader catalog reads, `ListPermissions`, and every mutation remain owner/admin-only. The six findings resolved in the prior round remain unchanged.
- Open questions: None
- Approval reference: Repository owner, current Codex coordination chat, 2026-11-09. The owner selected Option A in response to the two-option escalation and instructed `now start next`; the coordinator stated that this would be recorded as implementation approval, and the owner continued the acceptance flow. With the targeted exact-digest approval recorded above, implementation is authorized within this RFC's accepted scope.

This RFC is Accepted. The targeted challenger approved the exact decision candidate, the repository
owner approved Option A and implementation progression, and no open question or targeted finding
remains. A material decision change requires a new accepted RFC that supersedes this record.
