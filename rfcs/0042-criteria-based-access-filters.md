# RFC 0042: Criteria-Based Access Filters

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-09
- Supersedes: RFC 0041 credential-list query decision, RFC 0037 role-catalog filter request contract, and RFC 0038 unchanged access-management request and client contract only

## Context

Project API key listing and role listing introduced dedicated `query` fields even though repository
list contracts use repeated `Criteria` values. Access-management lists also split filtering across
dedicated scope, subject, and definition-type fields. These APIs are new and unreleased, so this is
the point to adopt the established list-filter contract without a compatibility layer.

`GetAllProjectCredentialRequest` already contains `repeated Criteria criterias = 2`, but RFC 0041
deprecated and ignored it in favor of `string query = 4`. `ListRolesRequest` uses dedicated query
and enum fields, while `ListPermissionsRequest` uses a dedicated scope field. Replacing these public
fields changes protobuf, SDK, backend, and UI contracts and therefore supersedes those decisions.

## Decision

Use the shared `Criteria` message as the only list-filter contract for Project API keys, roles, and
permission definitions. Pagination remains separate.

```protobuf
message GetAllProjectCredentialRequest {
  Paginate paginate = 1;
  repeated Criteria criterias = 2;
  uint64 projectId = 3 [jstype = JS_STRING];
  reserved 4;
  reserved "query";
}

message ListPermissionsRequest {
  Paginate paginate = 1;
  reserved 2;
  reserved "scope";
  repeated Criteria criterias = 3;
}

message ListRolesRequest {
  Paginate paginate = 1;
  reserved 2, 3, 4, 5;
  reserved "query", "scope", "subject", "definitionType";
  repeated Criteria criterias = 6;
}
```

Supported criteria are allowlisted:

| List | Key | Logic | Meaning |
| --- | --- | --- | --- |
| Project API keys | `name` | `contains` | Case-insensitive literal credential-name substring |
| Project API keys | `role` | `contains` | Case-insensitive literal current role-name substring |
| Permission definitions | `scope` | `=` | Exact `ORGANIZATION` or `PROJECT` scope |
| Roles | `name` | `contains` | Case-insensitive literal display-name substring |
| Roles | `key` | `contains` | Case-insensitive literal immutable-key substring |
| Roles | `scope` | `=` | Exact `ORGANIZATION` or `PROJECT` scope |
| Roles | `subject` | `=` | Exact `USER` or `PROJECT_TOKEN` subject |
| Roles | `type` | `=` | Exact `BUILT_IN` or `CUSTOM` definition type |

Criteria in one request are combined with AND and keys must be unique. Text values are trimmed,
nonempty, at most 200 Unicode code points, and treat `%`, `_`, and `\` literally. Enum values must
match the listed persisted values. Unsupported keys, logic, duplicates, blank values, NUL, and
excess criteria fail before service query construction. Project API key listing accepts at most two
criteria, permission listing at most one, and role listing at most five.

Controllers own validation and map criteria to typed service filters. Services never interpolate
caller-provided keys or logic into SQL. Existing pagination, tenant isolation, authorization,
ordering, response envelopes, and omitted-filter behavior remain unchanged.

Invalid Project API key criteria return the existing list failure-envelope shape with gRPC `OK`.
Error code `1026103` is retained but its unreleased query-specific declaration becomes
`ListProjectAPIKeyInvalidCriteria`, with machine message `invalid project API key criteria` and
human message `Use supported filters and values, then try again.` Every invalid criterion
condition listed above maps to that one error and reveals no storage detail. Access-management
permission and role criteria failures return canonical gRPC `InvalidArgument`, consistent with its
existing invalid-filter behavior. Unexpected persistence failures retain `Internal`, and
authentication and authorization behavior does not change.

The Project API key inventory replaces its free-text search with the established structured
`QuerySearch` pattern and exposes `name` and `role` filters. The role catalog uses the same pattern
with `name`, `key`, `scope`, `subject`, and `type`. SDK and UI request types expose `criteria`
arrays and no `query` property. Proto, backend, SDK, and UI change together with no legacy path.

## Contracts and Ownership

- Protobuf source owns reserved removed fields and new repeated criteria fields. Generated
  artifacts remain generator-owned.
- Web API controllers own criteria validation and conversion to typed service filters.
- `pkg/errors` owns the Project API key invalid-criteria platform error at code `1026103`.
- Access-management and project services own tenant-scoped parameterized predicates and never
  receive raw criterion keys or logic.
- The React SDK owns generated `Criteria` construction from typed criteria arrays.
- The UI owns structured filter state and sends only allowlisted keys and logic.
- Authentication, authorization, role assignment, persistence, response envelopes, pagination,
  and ordering do not change.

## Alternatives Rejected

- Keep dedicated query and enum fields: rejected because it preserves a second list-filter contract.
- Keep query as deprecated compatibility: rejected because these APIs are new and unreleased.
- Pass arbitrary keys and logic to GORM: rejected because caller-selected SQL fragments are unsafe.
- Use one synthetic multi-column search criterion: rejected because it hides field ownership and
  does not match the structured Users and Projects filter experience.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The unreleased proto, backend, SDK, and UI contracts change together. Removed field numbers and names are reserved. |
| Failure and recovery | Invalid credential criteria return the defined failure envelope; invalid access-management criteria return `InvalidArgument`. Both fail before database access. List requests are read-only and safe to retry. Rollback restores every coordinated layer together. |
| Security and privacy | Allowlisted mappings prevent SQL identifier and operator injection. API key values remain excluded from filtering. Tenant and authorization predicates remain mandatory. |
| Data and migration | N/A. No table, row, backfill, or persisted representation changes. |
| Concurrency and resources | Existing count and page consistency remains. Criteria add bounded predicates and no long-lived resource. |
| Operations and observability | Existing list error monitoring remains. Rollout stops on contract drift, incorrect filtering or totals, cross-tenant visibility, or generated-artifact drift. |

## Rollout and Rollback

Generate protobuf artifacts, update backend handling, update and publish the React SDK, then update
the UI consumer as one feature delivery. No dual-read period is needed because the affected APIs
are unreleased. Roll back UI, SDK, backend, and protobuf source together. No data rollback exists.

## Verification

Acceptance requires evidence that query fields and dedicated access-list filters are absent, field
numbers and names are reserved, only allowlisted criteria are accepted, literal contains behavior
is escaped, criteria combine with AND, counts and rows share predicates, and omitted criteria retain
unfiltered results. Proto descriptor tests assert the new criteria field numbers, repeated message
types, and reserved removed field numbers and names. Stable route tests retain serious accessibility
checks for both changed listings.

```bash
bash bin/artifacts-generate.sh
(cd sdks/react && bash bin/artifacts-generate.sh)
go test ./api/web-api/api ./api/web-api/internal/service/accessmanagement ./api/web-api/internal/service/project
(cd sdks/react && npm test -- --runInBand && npm run build)
(cd ui && yarn test --watchAll=false src/clients/__tests__/access-management.client.test.ts src/clients/__tests__/credential.client.test.ts src/app/pages/workspace/role/role-query-search.test.tsx src/app/pages/external-integration/rapida-credentials/index.test.tsx)
(cd ui && yarn checkTs)
just ui-browser-test
just validate-rfc-layout
just agent-finalize "protos/artifacts/access-management-api.proto,protos/artifacts/web-api.proto,pkg/errors/project_api_key.go,pkg/errors/project_api_key_test.go,api/web-api/api/access_management.go,api/web-api/api/access_management_test.go,api/web-api/api/project.go,api/web-api/api/project_credential_test.go,api/web-api/internal/service/accessmanagement/service.go,api/web-api/internal/service/accessmanagement/service_test.go,api/web-api/internal/service/project/service.go,api/web-api/internal/service/project/service_test.go,ui/src/clients/access-management.client.ts,ui/src/clients/__tests__/access-management.client.test.ts,ui/src/clients/credential.client.ts,ui/src/clients/__tests__/credential.client.test.ts,ui/src/app/pages/workspace/role/role-query-search.tsx,ui/src/app/pages/workspace/role/role-query-search.test.tsx,ui/src/app/pages/workspace/role/index.tsx,ui/src/app/pages/external-integration/rapida-credentials/index.tsx,ui/src/app/pages/external-integration/rapida-credentials/index.test.tsx"
```

## Review and Approval

- Challenger: Codex, task `/root/criteria_filter_rfc_challenge`
- Reviewed digest: SHA-256 `05417e9e6cfc374cf69891b24ac20b042d5df64e40af9893078b69a571251a2b`
- Outcome: Approved
- Resolved findings: The accepted revision supersedes current RFC 0037 and RFC 0038 request decisions, defines exact invalid-criteria error behavior, and includes descriptor, browser accessibility, and scoped finalization gates.
- Open questions: None
- Approval reference: Repository owner instruction to use another Codex and obtain approval, 2026-11-09
