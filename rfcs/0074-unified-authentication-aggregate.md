# RFC 0074: Unified Authentication Aggregate

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0073 Authentication/Session shape, RFC 0072 Go package ownership, and RFC 0071/0072 Principal authorization wire shape only

## Context

RFC 0073 defines a User-oriented Session beside a reusable request Principal. It does not let an
internal or external consumer inspect the authenticated subject's complete Organization and Project
membership graph, assigned roles, role permissions, or Organization-owned billing plan.

The current internal `Principal` contains one selected tenant and a permission snapshot. The legacy
internal Authentication contains at most one Organization and Project. Neither matches the desired
wire contract. Billing protobufs identify subscriptions by Organization, but this repository has no
billing-service implementation or billing client that can populate the plan during authentication.

## Decision

Replace the future Session wire contract with one reusable `authentication.Authentication`
aggregate. It contains a request Principal, an optional User profile, and the verified Organization
membership graph. Each Organization owns its assigned roles, optional billing plan, and Projects.
Each Project owns its assigned roles. Each Role contains its Permission records. There is no stored
`effectivePermissions` projection.

```text
Authentication
├── Principal
├── User?
└── Organizations[]
    ├── Roles[]
    │   └── Permissions[]
    ├── BillingPlan?
    │   └── Quotas[]
    └── Projects[]
        └── Roles[]
            └── Permissions[]
```

`Principal` contains only Subject, optional Service Actor, selected Tenant, and Proof. The aggregate
constructor owns validation, deterministic ordering, deep copies, selected-tenant membership
checks, and permission derivation. Assigned roles and their Permission records are the single source
of authorization facts.

The protobuf source uses package `authentication` and Go package
`github.com/rapidaai/protos/authentication`. It is self-contained and does not import the root
`common.proto`, avoiding a Go import cycle when root Web API messages reference the authentication
subpackage. Its exact schema is:

```proto
message Identity {
  string type = 1;
  uint64 id = 2 [jstype = JS_STRING];
  string issuer = 3;
}

message ServiceActor {
  uint64 id = 1 [jstype = JS_STRING];
  string issuer = 2;
}

message Tenant {
  uint64 organizationId = 1 [jstype = JS_STRING];
  uint64 projectId = 2 [jstype = JS_STRING];
}

message ProofReference {
  string type = 1;
  string value = 2;
}

message Proof {
  string method = 1;
  string audience = 2;
  ProofReference reference = 3;
}

message Principal {
  Identity subject = 1;
  optional ServiceActor actor = 2;
  Tenant selectedTenant = 3;
  Proof proof = 4;
}

message Permission {
  uint64 id = 1 [jstype = JS_STRING];
  string key = 2;
  string scope = 3;
  string name = 4;
  string description = 5;
}

message Role {
  uint64 id = 1 [jstype = JS_STRING];
  string key = 2;
  string name = 3;
  string description = 4;
  string scope = 5;
  string subject = 6;
  string definitionType = 7;
  repeated Permission permissions = 8;
}

message BillingPlanQuota {
  uint64 id = 1 [jstype = JS_STRING];
  string resourceType = 2;
  int64 quotaLimit = 3;
  string unit = 4;
}

message BillingPlan {
  uint64 id = 1 [jstype = JS_STRING];
  string name = 2;
  string slug = 3;
  string description = 4;
  bool isDefault = 5;
  bool isActive = 6;
  int32 sortOrder = 7;
  int64 priceMonthly = 8;
  int64 priceYearly = 9;
  string currency = 10;
  string stripeUrl = 11;
  repeated BillingPlanQuota quotas = 12;
}

message Project {
  uint64 id = 1 [jstype = JS_STRING];
  string name = 2;
  repeated Role roles = 3;
}

message Organization {
  uint64 id = 1 [jstype = JS_STRING];
  string name = 2;
  repeated Role roles = 3;
  optional BillingPlan billingPlan = 4;
  repeated Project projects = 5;
}

message User {
  uint64 id = 1 [jstype = JS_STRING];
  string name = 2;
  string email = 3;
  string status = 4;
}

message Authentication {
  Principal principal = 1;
  optional User user = 2;
  repeated Organization organizations = 3;
}
```

`web-api.proto` retains a presentation envelope because an issued login token is not part of the
universal Authentication aggregate:

```proto
message AuthenticationSession {
  authentication.Authentication authentication = 1;
  Token token = 2;
}

message AuthenticateResponse {
  int32 code = 1;
  bool success = 2;
  optional Authentication data = 3 [deprecated = true];
  optional Error error = 4;
  optional AuthenticationSession session = 5;
}
```

Field 5 retains both its `session` name and tag. Its nested wire shape changes before runtime
adoption. The deprecated field 3, all legacy runtime producers, requests, RPCs, middleware,
resolvers, service assertions, and UI state remain unchanged in this phase.

Add the matching immutable `pkg/authentication.Authentication` aggregate. `User`, `Organization`,
`Project`, `Role`, `Permission`, `BillingPlan`, and `BillingPlanQuota` are exported value inputs with
the fields and Go scalar types corresponding to the schema above. Construction and access are:

```go
func NewAuthentication(Principal, *User, []Organization) (Authentication, error)
func (Authentication) Principal() Principal
func (Authentication) User() (User, bool)
func (Authentication) Organizations() []Organization
func (Authentication) Organization(uint64) (Organization, bool)
func (Authentication) SelectedOrganization() (Organization, bool)
func (Authentication) SelectedProject() (Project, bool)
func (Authentication) Permissions() []PermissionKey
```

The constructor returns only centralized `errors.AuthenticationInvalidError` for aggregate
validation failures. It copies every pointer, slice, and nested slice. Every accessor returns values
whose mutation cannot change the aggregate. Lookup misses return the zero value and `false`.

The existing private Principal permission snapshot remains a transitional request cache.
Constructing an aggregate requires its sorted unique keys to equal `Authentication.Permissions()`.
Runtime adoption must later make the Role graph authoritative before removing that cache.

### Validation and Permission Derivation

- Every persisted ID is between 1 and `math.MaxInt64`. Tenant IDs may be zero, but a Project ID
  requires a nonzero Organization ID. IDs are unique among Organizations, among Projects within an
  Organization, among Roles within one Organization or Project, and among Permissions within one
  Role. The same built-in Role and Permission IDs may recur in different assignment containers.
- Required strings are valid UTF-8, trimmed, nonblank, and contain no NUL. Optional description and
  URL strings may be empty but otherwise obey the same encoding and trimming rules.
- Organization Roles have scope `ORGANIZATION`; their Permissions have scope `ORGANIZATION`.
  Organization Permission keys start with `organization.`. Project Roles and Permissions have scope
  `PROJECT`, and Project Permission keys start with `project.`. Cross-scope keys are invalid even
  when their final derived key set matches the transitional Principal permission cache.
- User subjects accept only Roles with subject `USER`. Project credential subjects accept no
  Organization Roles and only Project Roles with subject `PROJECT_TOKEN`. Organization credential
  and direct Service subjects accept no Roles. Definition type is `BUILT_IN` or `CUSTOM`.
- Permission ID and key identify one catalog record across the aggregate. Reusing an ID with a
  different key or a key with a different ID is invalid. Within a Role, duplicate IDs or keys are
  invalid.
- User subjects require a User profile whose ID equals the Subject ID. Non-User subjects reject a
  User profile. Only active membership and assignment records may be used by later producers.
- A tenantless onboarding User may have zero or more Organizations and derives no Permissions. A
  tenantless direct Service has no Organizations. A nonzero selected Organization must exist. A
  nonzero selected Project must exist inside that selected Organization.
- A Project credential contains exactly one Organization and the selected Project. An Organization
  credential contains exactly one selected Organization. A direct Service with a selected tenant
  may contain that selected Organization and Project but no Roles. Delegated subjects follow the
  effective Subject's User, Project credential, or Organization credential rules; the Service Actor
  does not add access.
- A selected User Organization derives the union of its Organization Role Permissions. A selected
  User Project derives the union of its Organization and selected Project Role Permissions. A
  Project credential derives only its selected Project Role Permissions. Organization credential
  and direct Service subjects derive none. A zero tenant derives none.
- Derived Permission keys are deduplicated and sorted. Organizations and Projects are ordered by ID;
  Roles by ID; Permissions by key then ID; BillingPlan quotas by ID. Input order never changes the
  resulting aggregate.
- BillingPlan is optional. When present, its ID and every quota ID are valid and quota IDs and
  resource types are unique. Name, slug, currency, resource type, and unit are required. Sort order,
  prices, and quota limits are nonnegative.

BillingPlan is optional. This phase establishes its Organization ownership but does not fabricate
billing data. A later accepted wiring change must provide a real billing source before any producer
claims the field is complete.

## Contracts and Ownership

- `pkg/authentication` owns the immutable internal Authentication aggregate and validation.
- `authentication.proto` owns the matching cross-service wire contract.
- Organization owns BillingPlan. Project never owns or copies billing data.
- Role owns Permission records. No Organization, Project, Principal, or response stores a second
  effective-permission list.
- Route policy continues using the existing request Principal in this phase. Resolver, middleware,
  JWT, controller, SDK-state, and billing-source adoption are explicitly excluded.
- Raw credentials and issued session tokens remain outside Authentication. The Web API-owned
  AuthenticationSession carries the issued token for future external login adoption.
- Root and React protobuf sources remain byte-identical. Generated artifacts remain derived.

## Alternatives Rejected

- Keep `Session`: rejected because the aggregate applies to User, Project, Organization, and
  Service authentication across internal and external consumers.
- Store BillingPlan at the Authentication root: rejected because billing subscriptions are keyed by
  Organization.
- Store roles and a separate effective permission list: rejected because it duplicates the same
  authorization facts and permits divergence.
- Reuse root `AssignedRole` and billing messages: rejected because the authentication Go
  subpackage would import root protos while root Web API protos import authentication, creating an
  import cycle.
- Populate synthetic billing data: rejected because this repository has no authoritative billing
  source.
- Cut over middleware, JWT, and UI in the same candidate: rejected because the requested sequence
  establishes the shared contract before linking producers and consumers.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The unused field 5 keeps its `session` name and tag while its nested message type changes. Binary framing remains length-delimited, but nested wire, JSON, and generated source shapes intentionally change before adoption. Deprecated field 3 and all operational producers remain unchanged. Authentication Go output moves to a collision-free subpackage. |
| Failure and recovery | Constructor validation fails closed on malformed graphs, duplicate IDs, invalid selected tenants, or permission-cache mismatch. Generation and compilation stop on import or descriptor drift. |
| Security and privacy | Raw tokens remain excluded. Organization billing becomes visible to consumers that receive the aggregate, so later producers must authorize that disclosure. Roles and permissions remain tenant-bound. |
| Data and migration | N/A. No persisted records or database schema change. |
| Concurrency and resources | The aggregate is immutable and owns copied slices. No goroutines or retained external resources are introduced. |
| Operations and observability | No runtime producer uses the new field. Later adoption requires billing-source failure and freshness behavior. |

## Rollout and Rollback

Add the immutable internal aggregate and focused tests. Replace the future protobuf Session shape,
move generated Go authentication output to its subpackage, update the unused Web API field 5,
regenerate all languages, and update descriptor coverage. Stop on contract drift, import cycles,
consumer compilation failure, mutable aliasing, or unresolved Critical or Major review findings.

Rollback restores RFC 0073 sources, removes the new internal aggregate, regenerates artifacts, and
restores field 5 to `authentication.Session`. No data rollback is required.

## Verification

```bash
cmp protos/artifacts/authentication.proto sdks/react/src/clients/protos/artifacts/authentication.proto
cmp protos/artifacts/web-api.proto sdks/react/src/clients/protos/artifacts/web-api.proto
bash bin/artifacts-generate.sh
(cd sdks/react && bash bin/artifacts-generate.sh)
go test -count=1 ./pkg/authentication ./pkg/errors ./protos/...
go test -race -count=1 ./pkg/authentication
go test -run '^$' ./api/web-api/api ./pkg/clients/web ./pkg/authenticators ./cmd/web
(cd sdks/react && npm test -- --runInBand)
git diff --check
just agent-finalize "rfcs/0074-unified-authentication-aggregate.md,pkg/authentication/authentication.go,pkg/authentication/authentication_test.go,pkg/errors/authentication_principal.go,pkg/errors/authentication_principal_test.go,bin/artifacts-generate.sh,protos/artifacts/authentication.proto,protos/artifacts/web-api.proto,sdks/react/src/clients/protos/artifacts/authentication.proto,sdks/react/src/clients/protos/artifacts/web-api.proto,protos/authentication/authentication.pb.go,protos/web-api.pb.go,protos/web-api_grpc.pb.go,api/document-api/app/bridges/artifacts/protos/authentication_pb2.py,api/document-api/app/bridges/artifacts/protos/authentication_pb2.pyi,api/document-api/app/bridges/artifacts/protos/authentication_pb2_grpc.py,api/document-api/app/bridges/artifacts/protos/web_api_pb2.py,api/document-api/app/bridges/artifacts/protos/web_api_pb2.pyi,api/document-api/app/bridges/artifacts/protos/web_api_pb2_grpc.py,sdks/react/src/clients/protos/authentication_pb.js,sdks/react/src/clients/protos/authentication_pb.d.ts,sdks/react/src/clients/protos/authentication_pb_service.js,sdks/react/src/clients/protos/authentication_pb_service.d.ts,sdks/react/src/clients/protos/web-api_pb.js,sdks/react/src/clients/protos/web-api_pb.d.ts,sdks/react/src/clients/protos/web-api_pb_service.js,sdks/react/src/clients/protos/web-api_pb_service.d.ts,protos/authentication_principal_contract_test.go"
```

Descriptor tests must assert every Authentication, Principal, User, Organization, Project, Role,
Permission, BillingPlan, and BillingPlanQuota field number and type; repeated and optional
semantics; `web_api.AuthenticationSession` fields; field 5's exact full name; absence of
`authentication.Session` and `authentication.Authorization`; and legacy field 3.

Internal aggregate tests must include both cross-scope rejection directions, including a
Project-selected User whose transitional Principal permission cache otherwise equals the invalid
derived key set.

## Review and Approval

- Challenger: Independent GPT-6 Astra and GPT-5.6 Sol contract and security reviews, followed by a combined evidence panel
- Outcome: Approved
- Resolved findings: Added the exact protobuf and Go contracts, subject-specific validation and derivation, the Web-owned token envelope, complete supersession metadata, and cross-scope permission-key rejection.
- Open questions: None
- Approval reference: Repository owner approved the unified internal/external Authentication graph,
  Organization-owned billing, and Role-owned Permissions on 2026-11-13.
  All challengers approved pre-metadata digest
  `45ef40bc36c5689ed815591644116a9da88a37863b48bdfb8929bfe41c1bd4a0`.
