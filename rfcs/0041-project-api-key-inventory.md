# RFC 0041: Project API Key Inventory and Revocation

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-09
- Supersedes: RFC 0039 credential-listing metadata and revocation deferral only

## Context

The Project Credentials page presents active project API keys as cards. The cards do not identify
the assigned access role, do not support search or pagination, and use inconsistent terms such as
developer key, credential, SDK authentication credential, and publishable key. Loading, empty, and
failure states are not distinct, so a failed request can look like an empty Project. There is no
public revocation operation even though the project service already has an archive path.

The `project_credentials` row already stores the immutable `role_definition_id` introduced by
RFC 0039. The list protobuf omits that role metadata, and the React SDK does not send the pagination
already present in the request. The backend also accepts arbitrary criteria but does not apply them.
This RFC therefore changes the public protobuf and React SDK contracts, introduces credential
revocation, and changes a security-sensitive UI. It is a Governed change.

RFC 0039 intentionally preserved repeated key disclosure: create and list responses return the full
key, and the UI can reveal and copy a listed key. This RFC preserves that contract. One-time secret
delivery, hashing, rotation, and lost-secret recovery remain separate security decisions.

## Decision

### Product language and inventory experience

The user-facing concept is **Project API key**. Persisted Go types and database tables retain the
established `ProjectCredential` name because renaming storage does not improve the user contract.
The page title is `Project API keys`, and its description is:

> Manage API keys used by applications and tools to access this project.

The page uses the established Carbon enterprise data-table pattern rather than credential cards.
It contains:

- a primary `Create API key` action;
- the message `Use a separate API key for each application or tool. The assigned access role
  controls what it can do, and you can revoke the key independently.`;
- a server-backed search input labelled `Search by name or role`;
- a visible-result count and pagination; and
- `Name`, `API key`, `Access role`, `Created`, and `Actions` columns.

The API key is masked by default. Its row actions are `Show API key`, `Copy API key`, and, for an
active Organization owner or admin, `Revoke API key`. Ordinary members do not see the revoke
action. The UI membership state controls only presentation; the backend membership check remains
authoritative. Reveal and copy preserve RFC 0039. Revoke requires name confirmation and displays:

> `<name>` will immediately lose access to this project. This action cannot be undone.

The main inventory has distinct loading, initial-empty, search-empty, and failure states:

- Initial empty: `No project API keys` and `Create an API key for an application or tool that needs
  access to this project.` with `Create API key`.
- Search empty: `No API keys match your search.`
- Failure: `API keys could not be loaded` and `Try again. No API keys were changed.` with
  `Try again`.

Copy and revoke failures use actionable inline or toast notifications. A successful revoke removes
the row only after the server confirms the state change. The existing compact credential selector
continues to use a compact presentation, but it adopts the same Project API key terminology and
distinct loading, empty, and failure behavior. It sequentially loads every explicit page before
presenting its choices, so credentials after the first page remain available to deployment flows.

At widths below the Carbon medium breakpoint, the toolbar controls stack and remain full width. The
table keeps all five columns at a fixed minimum width inside a horizontal scroll container. It does
not change to cards or hide the access role or actions.

### List contract

`ProjectCredential` adds two wire-additive fields using currently unused field numbers:

```protobuf
uint64 roleDefinitionId = 12 [jstype = JS_STRING];
string roleName = 13;
```

`roleDefinitionId` is the persisted association and stable role link. `roleName` is the current
read-time display projection from `role_definitions`; it is not duplicated in
`project_credentials`. Active credential references already prevent custom-role archival, and
built-in roles are immutable, so every listed active credential has a resolvable role.
Create and list responses both populate the role ID and current role name.

`GetAllProjectCredentialRequest` adds `string query = 4`. The server trims it, accepts at most 200
Unicode code points, and performs a case-insensitive literal contains search over credential name
and current role name. It does not search key material. `%`, `_`, and the escape character are
escaped before building the parameterized `ILIKE` pattern, so they are literal query characters.
The existing `criterias` field remains wire-compatible, receives `[deprecated = true]`, and is
ignored. The server does not interpolate caller-provided column names.

An omitted pagination message preserves the existing unbounded response for compatibility. When a
pagination message is present, the pair `page=0, pageSize=0` means page 1 with 25 rows. Otherwise
both values are required: page must be at least 1 and page size must be from 1 through 100. Either
partial-zero pair is invalid. Explicitly paginated results remain ordered by newest creation first,
with credential ID descending as the stable tie-breaker. The response retains `code`, `success`,
`data`, `error`, and `paginated`; an unbounded response reports current page 1 and the full count.

The React SDK adds `ListProjectCredentials(connectionConfig, options, callback, authHeader)`, where
`options` contains Project ID, page, page size, and query. The existing exported
`GetAllProjectCredential(connectionConfig, projectId, callback, authHeader)` signature remains and
continues omitting pagination and preserves its unbounded response. The new inventory and compact
selector use the new operation and always send explicit pagination. Existing TypeScript call sites
continue to compile and retain their observable list behavior.

### Revocation contract

`ProjectService` adds:

```protobuf
rpc RevokeProjectCredential(RevokeProjectCredentialRequest)
    returns (RevokeProjectCredentialResponse);

message RevokeProjectCredentialRequest {
  uint64 projectId = 1 [jstype = JS_STRING];
  uint64 credentialId = 2 [jstype = JS_STRING];
}

message RevokeProjectCredentialResponse {
  int32 code = 1;
  bool success = 2;
  uint64 id = 3 [jstype = JS_STRING];
  Error error = 4;
}
```

The API controller validates Project ID, credential ID, pagination, and query inputs with the
existing validator package. Expected invalid-input and hidden-resource conditions use
operation-specific platform errors from `pkg/errors` in the response envelope. Authentication,
authorization, cancellation, deadline, and unexpected storage failures return canonical non-OK
gRPC status and no response message. The service owns only tenant-scoped database selection and
state transition; it does not duplicate request validation or response construction.

Revocation atomically changes one active credential in the authenticated Organization and requested
Project to archived status and records the update audit actor. A retry for that same archived
credential in the same Organization and Project is an idempotent success with its ID. Unknown,
cross-Project, and cross-Organization IDs have one indistinguishable not-found result. The operation
never returns the credential object or key. No key or hidden resource detail is logged or returned.
Revocation is irreversible through the public API.

Create and list retain their existing caller predicate: an authenticated user with valid
Organization context. Revoke is restricted to a user with an active owner or admin membership in
that Organization until role-permission enforcement is accepted. Membership is read from the
established organization-membership source, not trusted only from caller input. Every other caller
receives `PermissionDenied` with `project API key revocation permission denied`, without a response
message. This RFC does not enable role-permission enforcement. The operation inventory maps the
future revocation cutover to
`organization.project_access_tokens.revoke` or `project.access_tokens.revoke` according to the
caller scope already defined by the permission catalog.

### Public error contract

Expected request and hidden-resource failures return a failure envelope with gRPC `OK` so generated
callbacks receive the response. Authentication, authorization, cancellation, deadline, and
unexpected server failures return no response message with the canonical non-OK gRPC status. The
server never returns a failure envelope together with non-OK gRPC status.

The list endpoint uses these platform errors:

| Condition | HTTP-shaped `code` | `error.errorCode` | Machine message | Human message |
| --- | ---: | ---: | --- | --- |
| Invalid Project ID | 400 | 1026101 | `invalid project identifier` | `Select a valid project and try again.` |
| Invalid pagination pair or range | 400 | 1026102 | `invalid project API key pagination` | `Choose a page and page size between 1 and 100.` |
| Query longer than 200 Unicode code points | 400 | 1026103 | `project API key query is too long` | `Shorten the search to 200 characters or fewer.` |

The revoke endpoint uses these platform errors:

| Condition | HTTP-shaped `code` | `error.errorCode` | Machine message | Human message |
| --- | ---: | ---: | --- | --- |
| Invalid Project ID | 400 | 1026201 | `invalid project identifier` | `Select a valid project and try again.` |
| Invalid credential ID | 400 | 1026202 | `invalid project API key identifier` | `Select a valid API key and try again.` |
| Unknown or hidden credential | 404 | 1026203 | `project API key not found` | `The selected API key is unavailable.` |

Success uses `code=200`, `success=true`, no error, and the requested page or revoked credential ID.
Failure envelopes use `success=false` and omit data. Invalid user authentication returns
`Unauthenticated` with `valid user authentication is required`. Missing Organization context on
list returns `PermissionDenied` with `project API key listing permission denied`. A non-owner or
non-admin revoke caller, missing Organization context, or missing active Organization membership
returns `PermissionDenied` with `project API key revocation permission denied`. Cancellation returns
`Canceled` with `request canceled`; deadline expiry returns `DeadlineExceeded` with
`request deadline exceeded`; unexpected list failure returns `Internal` with
`project API key listing failed`; and unexpected revoke failure returns `Internal` with
`project API key revocation failed`.

The React SDK adds `RevokeProjectCredential` without changing an existing function signature. If a
client loses the success response, retrying the same request succeeds without returning secret
data. After any revoke transport failure the UI reloads the current page: an absent row is reported
as `API key was revoked`; a still-present row is retained and reported as
`API key revocation could not be confirmed. Try again.`

### Secret handling and non-goals

List responses continue returning full key bytes, and the UI masks them until the user asks to
reveal or copy. This RFC does not add one-time delivery, secret hashing, key rotation, role
replacement, multiple roles, personal API keys, authentication changes, or access-policy
enforcement. Those changes require separate accepted decisions.

No database migration is required. Existing role associations and audit columns remain the source
of truth.

## Contracts and Ownership

- The protobuf owner owns additive fields 12 and 13, request field 4, the revoke RPC, response
  envelope, and wire compatibility. Generated Go, Python, and React artifacts remain
  generator-owned.
- The Project API controller owns user authentication, Organization context, validator-based input
  checks, platform-error mapping, response construction, and canonical gRPC status mapping.
- The project service owns the tenant-scoped role join, deterministic ordering, count and page
  query, and idempotent active-to-archived state transition.
- `project_credentials.role_definition_id` remains the role-association source of truth.
  `role_definitions.name` remains the display-name source of truth. No display-only field is added
  to a persisted entity.
- The React SDK owns request construction for page, page size, query, and revocation while
  preserving its established callback, function signature, and authentication conventions.
- The Credentials UI owns terminology, search and pagination state, loading and failure states,
  masked presentation, confirmation, and accessible feedback.
- Existing key disclosure, role immutability, audit conventions, create/list caller authentication,
  and omitted-pagination behavior remain compatible. New consumers use explicit bounded pages.
  Revoke has the temporary owner/admin restriction specified above.

## Alternatives Rejected

- Keep cards and add the role name: rejected because cards do not provide the scan, search,
  pagination, or action density expected from an enterprise inventory.
- Add a database column for role name: rejected because it would duplicate mutable catalog data and
  require synchronization.
- Add transient role display fields to the persistence entity: rejected because a listing
  projection keeps query output separate from stored state.
- Continue using generic criteria for search: rejected because caller-selected column names are an
  unsafe and ambiguous public query contract. A named query field has one documented meaning.
- Make omitted pagination bounded immediately: rejected because it would silently truncate existing
  SDK and direct RPC callers. New consumers use explicit bounds while the legacy omission remains
  compatible until a separately coordinated contract change.
- Delete credentials: rejected because audited archival is the established lifecycle and preserves
  evidence.
- Rename database and Go credential types to API key: rejected because it creates migration and
  code churn without changing product behavior.
- Change to one-time secret delivery now: rejected because RFC 0039 explicitly preserved repeated
  disclosure and secret storage redesign has a larger compatibility and recovery surface.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Protobuf additions are wire-compatible. Omitted pagination remains unbounded, the existing SDK signature retains its behavior, and new consumers send explicit bounded pages. The compact selector exhausts all explicit pages. Existing key bytes and create behavior remain unchanged. |
| Failure and recovery | List failure does not replace current rows with an empty state. Same-scope revoke retries are idempotent, and the UI reloads after an ambiguous transport result. A revoked key cannot be restored through the public API; recovery requires creating a replacement key with an explicitly selected role. |
| Security and privacy | Search never reads key material, hidden resources share one not-found result, and revocation is tenant and Project scoped. Revoke is owner/admin-only until permission enforcement exists and returns only an ID. Full key disclosure remains an explicitly accepted existing risk and is masked by default in the UI. No permission enforcement is implied by displaying an assigned role. |
| Data and migration | N/A. Existing role and audit columns are sufficient, and no persistent shape changes. The role name is joined at read time. |
| Concurrency and resources | A locked active-to-archived transition gives revocation one state-transition owner and prevents two archive writes while allowing idempotent success for a later same-scope retry. Stable ordering prevents duplicate or skipped rows within a static result set; concurrent inserts can shift offset pages and are accepted for this administrative inventory. No worker or long-lived resource is added. |
| Operations and observability | Controllers log operation, correlation, Organization, Project, credential, and failure class without key bytes. Alerting uses existing Web API error monitoring. Release owners stop rollout on incorrect pagination totals, cross-tenant visibility, key logging, or revoke ambiguity. |

## Rollout and Rollback

Access Management owns the coordinated release:

1. Generate and verify protobuf artifacts, backend behavior, React SDK behavior, and UI states.
2. Deploy the backend first. Older clients ignore additive response fields, retain their unbounded
   omitted-pagination behavior, and do not call the new RPC.
3. Publish the generated React SDK with a package-version increment after backend readiness.
4. Update the UI package and deploy the inventory experience after list and revoke smoke tests pass.
5. Update the Web API operation inventory with the revoke permission mapping for future
   enforcement. This step does not activate permission checks.

Stop rollout on generated-artifact drift, incorrect counts, unresolved roles, cross-tenant data,
key material in logs or revoke responses, a failed same-scope revoke retry, or an unresolved
critical or major finding.

Rollback the UI first, then the SDK consumer, then the backend. Additive fields can remain deployed
while clients roll back. Keys revoked before rollback remain archived and cannot be restored; issue
replacement keys when needed. No database rollback exists because there is no migration.

## Verification

Acceptance requires evidence that:

- list responses expose the persisted role ID and current role name without duplicating role data;
- omitted-unbounded, zero-pair default, explicit, maximum, and invalid pagination behavior and
  stable ordering match this RFC;
- query searches name and role case-insensitively and never searches or interpolates key material;
- revoke succeeds for the initial call and a same-scope archived retry, returns only the credential
  ID, and returns the same hidden not-found result for unknown, cross-Project, and
  cross-Organization credentials;
- controller tests cover validator failures, platform error envelopes, authentication,
  cancellation, deadline, and unexpected storage failures;
- SDK tests prove page, page size, query, credential ID, auth forwarding, and compilation of the
  unchanged `GetAllProjectCredential` signature;
- UI tests cover loaded rows, role presentation, search, pagination, reveal, copy, revoke confirmation,
  revoke success and failure, hidden revoke actions for ordinary members, loading, initial empty,
  search empty, list failure, responsive layout, both color modes, keyboard use, and serious
  accessibility findings; and
- existing create flow, compact selector, role assignment, repeated key reveal, and copy behavior do
  not regress.

Run these exact gates plus the implementation task's scoped `just agent-finalize` command:

```bash
bash bin/artifacts-generate.sh
(cd sdks/react && bash bin/artifacts-generate.sh)
(cd sdks/react && npm test -- --runInBand)
(cd sdks/react && npm run build)
go test ./api/web-api/internal/service/project ./api/web-api/api ./pkg/errors
(cd ui && yarn test --watchAll=false src/clients/__tests__/credential.client.test.ts src/app/pages/external-integration/rapida-credentials/index.test.tsx src/app/components/domain/cards/__tests__/cards.test.tsx)
(cd ui && yarn checkTs)
(cd ui && yarn verify:ui)
just ui-browser-test
just validate-rfc-layout
just agent-style-check
git diff --check
bin/agent-review --working-tree --risk high
```

Any configured external reviewer transport must pass `bin/agent-egress` and have explicit user
authorization first.

## Review and Approval

- Challenger: Codex RFC challenger, task `/root/api_key_inventory_rfc_challenge`
- Reviewed digest: SHA-256 `562f80757da28b99378019517b1ea0fb512aeb9621c973e17cbd3e392ab4d4b6`
- Outcome: Approved
- Resolved findings: The revision restricts revoke to active Organization owners/admins, returns
  only the ID, preserves the existing SDK signature through a new options operation, makes
  same-scope retries idempotent, defines exact validation and error contracts, escapes literal
  search characters, marks criteria deprecated in protobuf, and fixes responsive table behavior.
- Open questions: None
- Approval reference: Repository owner, current Codex coordination chat, 2026-11-09. The owner
  approved the proposed Project API key inventory direction with `sure go with it`; the governed
  challenge tightened compatibility and destructive-action authorization without expanding that
  approved scope.

This RFC is Accepted. The independent challenger approved the exact candidate digest above, the
repository owner approved implementation progression, and no open question or critical or major
finding remains. A material decision change requires a new accepted RFC that supersedes this record.
