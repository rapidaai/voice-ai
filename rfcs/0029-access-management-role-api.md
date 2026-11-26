# RFC 0029: Access Management Role API

- Status: Superseded
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-07
- Supersedes: RFC 0028

Superseded by [RFC 0030](0030-access-management-response-envelope.md).

## Context

RFC 0022 created the inactive permission, role, and role-permission catalog. The web API does not
expose that catalog, so a UI cannot list permissions or manage custom roles. The current web client
uses gRPC services generated from the `protos/artifacts` contract submodule. Existing authentication
supplies a user, organization context, and audit actor.

This phase must make role management usable without changing authenticators, middleware,
credential formats, role assignments, permission evaluation, project tokens, MFA, or 2FA. The
repository owner has decided that MFA is a later version. This RFC adopts RFC 0028 and its
incorporated RFC 0026 policy in full except for the explicit API, assurance, audit-storage, and
rollout decisions recorded here. On acceptance, RFC 0028 becomes Superseded and this RFC becomes
the single authoritative Version 1 role-catalog lifecycle policy.

## Decision

Add an additive gRPC `AccessManagementService` with a dedicated proto contract. It owns:

- listing active permission definitions with explicit scope and token-eligibility filters;
- listing active built-in roles and active custom roles owned by the authenticated organization;
- reading one visible role and its active permissions;
- creating an organization-owned custom role and its complete permission set atomically;
- updating a custom role's display name and description;
- reducing a custom role by archiving selected active permission rows;
- archiving a custom role and its active role-permission rows atomically.

The service and RPC signatures are:

```proto
service AccessManagementService {
  rpc ListPermissions(ListPermissionsRequest) returns (ListPermissionsResponse);
  rpc ListRoles(ListRolesRequest) returns (ListRolesResponse);
  rpc GetRole(GetRoleRequest) returns (GetRoleResponse);
  rpc CreateCustomRole(CreateCustomRoleRequest) returns (CreateCustomRoleResponse);
  rpc UpdateCustomRole(UpdateCustomRoleRequest) returns (UpdateCustomRoleResponse);
  rpc ReduceCustomRolePermissions(ReduceCustomRolePermissionsRequest)
      returns (ReduceCustomRolePermissionsResponse);
  rpc ArchiveCustomRole(ArchiveCustomRoleRequest) returns (ArchiveCustomRoleResponse);
}
```

`ListPermissionsRequest` contains a scope enum and `optional bool tokenEligible`; unspecified scope
and absent token eligibility apply no filter, while present `false` and `true` filter exactly.
`ListPermissionsResponse` contains active permission definitions ordered by key and then ID, with
ID, key, scope, display name, description, and token eligibility. `ListRolesRequest` and response
pagination are defined below. Each role list item contains ID, immutable key, display name,
description, scope, subject, definition type, creation timestamp, optional update timestamp, and
optional audit actors.

`GetRoleRequest` contains `roleDefinitionId`. `GetRoleResponse` contains the visible active role
and only its active permission definitions, ordered by permission key and then ID.
`CreateCustomRoleRequest` contains key, display name, description, scope, subject, and
`permissionDefinitionIds`; its response contains the created role and active permissions.
`UpdateCustomRoleRequest` contains `roleDefinitionId`, display name, and description as complete
replacement metadata; its response contains the updated role and active permissions.
`ReduceCustomRolePermissionsRequest` contains `roleDefinitionId`, `permissionDefinitionIds`, and
reason; its response contains the updated role, remaining active permissions, and `correlationId`.
`ArchiveCustomRoleRequest` contains `roleDefinitionId` and reason; its response contains the
archived `roleDefinitionId` and `correlationId`. All identifiers use `uint64` with JavaScript
string representation. Timestamps use `google.protobuf.Timestamp`; actors use the existing
`AuditActor` contract. Responses do not use legacy success/error envelopes because canonical gRPC
status is the sole error contract.

`createdDate` is always present. `updatedDate` is omitted when the database value is NULL.
`createdActor` is omitted when the persisted pair is the seeded `unknown` type with a NULL ID;
otherwise it contains the persisted nonzero ID and type. `updatedActor` is omitted when both
persisted fields are NULL; otherwise it contains the persisted nonzero ID and type. The API never
maps a database NULL actor ID to numeric zero. An actor display name is populated when resolvable
and is otherwise empty without changing the actor identity.

The public contract uses package-unique protobuf values: scope (`ROLE_SCOPE_UNSPECIFIED`,
`ROLE_SCOPE_ORGANIZATION`, `ROLE_SCOPE_PROJECT`), subject (`ROLE_SUBJECT_UNSPECIFIED`,
`ROLE_SUBJECT_USER`, `ROLE_SUBJECT_PROJECT_TOKEN`), and definition type
(`ROLE_DEFINITION_TYPE_UNSPECIFIED`, `ROLE_DEFINITION_TYPE_BUILT_IN`,
`ROLE_DEFINITION_TYPE_CUSTOM`). The service maps these values to the existing unprefixed database
values. Identifiers are `uint64` with JavaScript string representation. `ListRolesRequest`
contains `Paginate paginate`, `string query`, and scope, subject, and definition-type enum filters
whose prefixed `UNSPECIFIED` values mean no filter. A missing `paginate` means page 1 with 25 items. When
present, `page` is one-based and must be at least 1; `pageSize` must be 1 to 100. The response
returns the selected page plus `Paginated.currentPage` and the filtered `Paginated.totalItem`
count. The trimmed query may contain at most 200 Unicode characters and matches a case-insensitive
literal substring of role key or display name; `%`, `_`, and `\` have no wildcard meaning. Results
sort by case-folded display name ascending, then key ascending, then ID ascending. Every page and
the count use the same tenant and filter predicate in one repeatable-read, read-only transaction.
Permission catalog reads are not paginated because the platform-owned catalog is bounded.

Every free-text request field is validated before persistence or query construction. Query,
display name, description, and reason reject U+0000 with `InvalidArgument`; the custom-key pattern
already excludes it. Character limits count Unicode code points after the declared trimming step.

Create requires an immutable custom key matching `[a-z][a-z0-9_]{2,63}`, a display name,
description, scope, subject, and a deduplicated list of permission-definition IDs. The key is
unique within the organization across both scope types, remains reserved after archival, and may
not equal any built-in key. Scope and subject are immutable. Empty initial permission lists are
valid. Display names are trimmed and must contain 1 to 200 Unicode characters. Descriptions are
trimmed, may be empty, and must contain no more than 400 Unicode characters. Create and metadata
update validate these bounds before persistence and return `InvalidArgument` on failure.

Custom Organization user roles accept only the exact custom Organization human-role catalog in
RFC 0026. Custom Project user roles accept only the exact custom Project human-role catalog in
RFC 0026. Custom Project-token roles accept only the RFC 0026 token-eligible Project catalog and
may use only `PROJECT` scope. No other scope and subject combination is accepted.

Permission mutation is reduction-only. The request names active permission-definition IDs to
remove and includes a human reason that is trimmed and must contain 1 to 400 Unicode characters.
Role archive requires the same reason field. In one transaction the service archives only those
active role-permission rows and retains every other row. It never inserts or reactivates a
role-permission after role creation. An empty, unknown, already archived, or cross-role removal
fails without mutation. Adding or exchanging a permission requires creating a new immutable
custom key.

The service records reduction evidence on every row it archives. Migration 000015 adds nullable
`reduction_reason`, `reduction_correlation_id`, and `reduction_assurance_condition` columns to
`role_definitions` and `role_permissions`, with database checks requiring the three values on an
archived row and forbidding them on an active row. The API-supplied reason becomes
`reduction_reason`. The gRPC request-logger interceptor remains the owner of
`x-request-id`; it supplies a UUID when metadata omits it. The service reads that context value as
`reduction_correlation_id`, requires 1 to 128 characters, and fails without mutation if the
middleware contract is absent or invalid. `reduction_assurance_condition` is the server-owned
constant `RECENT_MFA_UNAVAILABLE`; clients cannot set it. Existing `scope_type`, role and
permission identifiers, and audit actor columns identify the affected scope and actor.

PostgreSQL owns ID allocation for the two catalogs this API writes. Migration 000015 creates
owned bigint sequences for `role_definitions.id` and `role_permissions.id`, initializes each from
the table's current maximum ID, and installs `nextval` column defaults. Create persistence omits
both ID columns and reads the allocated values with `RETURNING`; it must not use the embedded
process-local `Audited.BeforeCreate` ID for these inserts. Database defaults remain the sole ID
source for single-row and batch creation, so independent web-api replicas share one collision-safe
allocator. `permission_definitions` remains platform-seeded and receives no sequence in this phase.

Every query derives `organization_id` from the authenticated context. Requests never accept an
organization ID. Built-in roles are globally readable but immutable. Custom roles are readable and
mutable only inside their owning organization. Until the permission evaluator exists, every method,
including catalog reads, requires the user's existing active organization membership to be `owner`
or `admin`. This is a fail-closed subset of the accepted role-read policy. It does not grant read
access to a member that lacks `organization.roles.read` or `project.roles.read`.

The service locks and revalidates the Organization, authorizing membership, target role when
present, and affected active permission rows in that order inside every mutation transaction.
This is the Version 1 authorization adapter. A later permission-evaluator version replaces it.
This version does not require MFA for custom-role creation, display-metadata update, permission
reduction, or archival. Adding MFA later requires a separate accepted RFC.

Use gRPC status codes consistently:

- `Unauthenticated` for missing or invalid existing authentication;
- `PermissionDenied` for missing organization context, inactive membership, or any owner/admin
  adapter failure;
- `InvalidArgument` for malformed or out-of-bound fields, an empty or duplicate reduction list,
  unknown permission-definition IDs during create, unsupported enums, and pagination outside the
  contract;
- `NotFound` for nonexistent, archived, and tenant-invisible target roles through the same
  tenant-scoped active-role lookup, except the exact idempotent archive replay described below;
- `AlreadyExists` for a duplicate or built-in-reserved custom key;
- `FailedPrecondition` for built-in mutation; a reserved, scope-incompatible, subject-incompatible,
  or token-ineligible permission during create; any requested reduction pair that is unknown to
  the target role, already archived without matching the exact replay contract, or active on
  another role; and every other non-monotonic permission mutation;
- `Canceled` when the request context is canceled before a read completes or mutation commits;
- `DeadlineExceeded` when the request deadline expires before a read completes or mutation commits;
- `Internal` for remaining persistence failures after logging server-side context.

Transport and service layers propagate the caller context to every database operation. Before a
transaction commits, error mapping checks `context.Canceled` and `context.DeadlineExceeded` before
classifying a database failure as `Internal`. Once commit succeeds, the mutation is authoritative
even when cancellation, deadline expiry, or transport loss prevents the response from reaching the
client; the API does not claim rollback after commit.

The service returns one status for the complete request and never partially mutates. Duplicate
custom keys always return `AlreadyExists`, whether the existing definition is active or archived.
Missing, blank, and whitespace-only incoming request IDs are equivalent to omission, so the
unchanged interceptor replaces them with a UUID. An absent context correlation identifier after
the interceptor is `Internal`; a nonblank client-supplied request ID outside its
1-to-128-character contract is `InvalidArgument`.

Clients that require deterministic mutation replay supply a unique `x-request-id` and reuse it
after an ambiguous response. Reduction and archive treat the same authenticated organization,
target role, reason, requested permission IDs, and persisted correlation identifier as an
idempotent replay and return success with the persisted correlation identifier and current visible
role state where the response includes it. Any mismatch follows the normal `FailedPrecondition` or `NotFound` mapping. An
ambiguous create is reconciled by listing and matching its immutable organization-owned key;
replay remains `AlreadyExists`. An ambiguous metadata update is reconciled with `GetRole`. Read
reconciliation never reveals another tenant's role or audit evidence.

No new tables are introduced. Migration 000015 adds database-owned writable-catalog sequences and
extends the RFC 0022 role catalog with the reduction evidence columns defined above. Creation
inserts the complete initial set using database-allocated IDs. Reduction atomically
archives removed rows with the authenticated update actor, reason, correlation identifier, and
unavailable-assurance condition. Archive atomically records the same evidence on the custom role
and every active role-permission row it archives. No active runtime assignment can reference a
custom role in this phase. A future assignment rollout must implement the RFC 0026 impact-preview
and no-active-assignment archive contracts before assignments may target custom definitions.

## Contracts and Ownership

- `protos/artifacts` owns the new additive `access-management-api.proto` source contract.
- Generated Go and Python artifacts remain generator-owned and are never edited manually.
- `api/web-api/api` owns gRPC authentication-context extraction, transport validation, response
  mapping, and gRPC status mapping.
- `api/web-api/internal/service/accessmanagement` owns tenant-scoped persistence, the Version 1
  active owner/admin method check, lock ordering, validation, transactions, and audit writes.
- `api/web-api/router` owns service registration.
- `permission_definitions`, `role_definitions`, and `role_permissions` remain the sole persisted
  sources of truth established by RFC 0022.
- PostgreSQL sequences own new role-definition and role-permission IDs. The access-management
  service owns omission of those ID columns and retrieval of their allocated values.
- The gRPC request-logger interceptor owns correlation identifier creation. The access-management
  service owns reduction-reason validation and the atomic persistence of reduction evidence.
- Existing authentication packages, middleware order, tokens, credentials, role-assignment tables,
  and authorization behavior outside the new service do not change.
- This RFC adopts the remaining RFC 0028 and incorporated RFC 0026 policy and supersedes RFC 0028.
  Recent MFA is not a prerequisite for the custom-role lifecycle methods defined here; MFA is
  deferred to the next version by owner decision.
- SDK publication and UI consumption are deferred. The additive proto is their later contract.

## Alternatives Rejected

- Extend `OrganizationService`: rejected because role catalog lifecycle is a separate ownership
  boundary and would continue growing an already broad service.
- Add ad hoc REST endpoints: rejected because the current workspace client uses generated gRPC
  contracts and a second transport would duplicate types and error behavior.
- Add role assignments and permission enforcement in this phase: rejected because they have
  separate migration and rollout risks and the owner requested API setup first.
- Allow every authenticated member to mutate roles: rejected because that permits unreviewed
  privilege policy changes. Version 1 limits mutation to active existing owners and admins.
- Keep the RFC 0028 MFA prerequisite for these methods: rejected by owner direction because MFA is
  a next-version feature. The accepted risk is limited to role definitions, which grant nothing
  until a separately governed assignment path exists.
- Allow in-place permission replacement or expansion: rejected because it violates the immutable
  key and monotonic-reduction policy and can widen existing assignments in a future phase.
- Make built-in roles editable: rejected because platform policy must remain stable and shared.
- Add a separate audit-event table: rejected because each role-permission row can be archived only
  once, the archived catalog rows already identify the actor and affected scope, and storing the
  reduction evidence with those rows keeps one source of truth.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The gRPC service and messages are additive. Existing services and clients remain compatible. Consumers need regenerated client artifacts before using the service. |
| Failure and recovery | Multi-row writes are transactional. Cancellation or deadline expiry before commit rolls back and retains its canonical gRPC status. After commit, state is authoritative and clients reconcile or replay as defined above. A failed create, reduction, or archive leaves the prior catalog state intact. Retried create with the same key returns `AlreadyExists`; reads and metadata updates are safe to retry. Offset pages are stable for an unchanged snapshot and use an explicit ID tie-breaker; concurrent commits may move records between separate page requests, which is accepted for this administrative catalog. |
| Security and privacy | Tenant IDs come only from authenticated context. Tenant-scoped lookup makes missing and foreign roles indistinguishable. Active owner/admin membership protects every method until the evaluator exists, which is stricter than the accepted read policy. Exact custom catalogs and reduction-only mutation prevent reserved authority and in-place expansion. Deferring MFA increases account-compromise risk for role management in Version 1; the owner accepts that risk. Permission metadata contains no secrets. |
| Data and migration | Migration 000015 adds owned sequences and nullable reduction evidence without rewriting active catalog rows. Sequence initialization uses the current table maxima, including any preexisting rows. Database checks require complete evidence on archived rows and no evidence on active rows. Archive is irreversible through this API because key reuse and reactivation are forbidden. |
| Concurrency and resources | Database sequences prevent ID collision across independent replicas. Mutations lock and revalidate the Organization, active authorizing membership, target role when present, and affected permission rows. This prevents stale membership or role state from committing. The active-pair index remains defense in depth. List queries are bounded and deterministic. No goroutines or long-lived resources are introduced. |
| Operations and observability | Invalid client input returns stable status codes and causes no additional access-management service error log. The unchanged request interceptor still logs every non-nil gRPC error at error level. Persistence failures add service-owned context with operation, role ID, organization ID, and the same request ID persisted with the reduction, without tokens or credentials. |

## Rollout and Rollback

Commit the proto source and regenerate artifacts before compiling the server implementation.
Apply migration 000015, then deploy the additive server and contract together. The service may be registered immediately
because each mutation revalidates existing active owner/admin membership under lock and does not
alter other routes. SDK and UI adoption follow separately. A later MFA version replaces this
RFC's assurance exception without silently changing Version 1 behavior.

Stop rollout if contract compatibility checks fail, tenant-isolation tests fail, a mutation is
partially persisted, or existing authentication behavior changes. Roll back the application by
deploying the prior web-api binary and client artifacts while retaining the additive columns and
sequences. The
down migration must fail before altering either table when any reduction evidence exists. It may
remove the empty columns, ID defaults, and owned sequences only before the first reduction or
archive, after confirming no newer binary is running. Reapplying the up migration initializes new
sequences above every retained ID. Evidence may be removed only by a separately accepted retention
or migration decision that first preserves it in an approved durable audit store. Custom roles
already created remain inactive with respect to authorization because assignments and enforcement
are out of scope; they can remain in the catalog without affecting runtime access.

## Verification

Acceptance criteria:

- generated artifacts match the proto source and expose the seven declared RPC methods;
- permission listing applies absent, false, and true token-eligibility semantics, scope filtering,
  active-only visibility, and stable key-and-ID ordering;
- role listing applies one-based defaults and bounds, literal case-insensitive query matching,
  identical count predicates, and stable display-name, key, and ID ordering;
- role reads hide nonexistent and foreign roles identically and return only active permissions in
  stable key-and-ID order;
- seeded built-ins and never-updated roles omit unavailable audit actors and update timestamps
  without emitting a zero actor ID;
- owner/admin reads return built-ins plus only the authenticated organization's custom roles;
- create and metadata update enforce the declared display-name and description bounds before
  persistence;
- create, metadata update, permission reduction, and archive succeed without MFA for an active
  existing owner/admin and retain correct audit actors;
- reduction and archive reject blank or overlong reasons and invalid correlation identifiers, and
  persist the actor, affected scope, reason, request correlation identifier, and
  `RECENT_MFA_UNAVAILABLE` evidence atomically on every archived row;
- the down migration succeeds before reduction evidence exists; after evidence exists it refuses
  without altering either table and leaves the prior application able to run with the additive
  schema retained;
- two independent service instances concurrently creating distinct custom roles receive unique
  database-allocated role and role-permission IDs without relying on process machine IDs;
- member read and mutation attempts, inactive and unauthenticated attempts, tenant-invisible
  lookups, and built-in mutation attempts fail with the declared status codes;
- invalid scope, subject, duplicate IDs, unknown IDs, reserved permissions, built-in key shadowing,
  token-ineligible permissions, duplicate keys, invalid pagination, and U+0000 in any free-text
  field are rejected without partial persistence;
- reduction archives only requested active pairs, retains every other pair, and rejects addition,
  exchange, reactivation, empty removal, and already-archived removal;
- nonexistent and cross-organization role IDs return the same `NotFound` result;
- concurrent membership deactivation or role mutation cannot commit from stale authorization or
  create duplicate active role-permission pairs;
- canceled and deadline-expired reads return `Canceled` and `DeadlineExceeded`; pre-commit
  mutation cancellation returns the same statuses without partial changes, while post-commit
  response loss preserves the committed state and supports the declared reconciliation paths;
- exact reduction and archive replay with a reused client request ID returns success with the
  persisted correlation identifier and applicable current visible role state, while mismatched
  replay cannot alter catalog or audit evidence;
- existing authentication and organization/project APIs remain behaviorally unchanged.

Exact commands:

```text
bash bin/artifacts-generate.sh
bash bin/verify-role-catalog-migration.sh
go test ./api/web-api/api/... ./api/web-api/internal/service/accessmanagement/... ./api/web-api/router/...
just ci-contracts
just ci-go
just ci-integration
just agent-finalize "protos/artifacts/access-management-api.proto,protos/access-management-api.pb.go,protos/access-management-api_grpc.pb.go,api/document-api/app/bridges/artifacts/protos/access_management_api_pb2.py,api/document-api/app/bridges/artifacts/protos/access_management_api_pb2_grpc.py,api/document-api/app/bridges/artifacts/protos/access_management_api_pb2.pyi,api/web-api/migrations/000015_add_role_management_support.up.sql,api/web-api/migrations/000015_add_role_management_support.down.sql,api/web-api/internal/entity/role_catalog.go,api/web-api/internal/entity/role_catalog_test.go,api/web-api/api/access_management.go,api/web-api/api/access_management_test.go,api/web-api/internal/service/accessmanagement/service.go,api/web-api/internal/service/accessmanagement/service_test.go,api/web-api/router/web.go,api/web-api/router/web_test.go,bin/verify-role-catalog-migration.sh,rfcs/0028-token-strict-subset-reduction.md,rfcs/0029-access-management-role-api.md,rfcs/README.md"
```

## Review and Approval

- Challenger: Codex `gpt-5.6-sol` and `gpt-6-astra` high-risk panel
- Outcome: Approved
- Resolved findings: Owner moved MFA to a later version; the draft now makes permission mutation
  reduction-only, enforces exact custom catalogs and reserved keys, hides tenant existence, and
  defines transaction lock and revalidation order. All methods now use a fail-closed owner/admin
  adapter until permission evaluation exists. The owner chose to retain RFC 0026 reduction audit;
  this revision defines bounded metadata, request reason, server-owned correlation and assurance,
  row-level storage, atomicity, and exact verification paths. The first owner-directed review's
  list-contract, status-mapping, and evidence-preserving rollback findings are resolved here.
  The correction review's policy-lineage, complete RPC contract, request-ID, observability, and
  read-verification findings are resolved. The repository owner authorized a new correction cycle
  on 2026-11-07; it defines U+0000 rejection, optional audit-field response mapping, and
  database-owned ID allocation for concurrent replicas. The owner approved the final correction
  on 2026-11-07; it defines pre-commit cancellation, post-commit reconciliation, and
  package-unique protobuf enum values.
- Open questions: None
- Approval reference: Repository owner conversation approval on 2026-11-07
