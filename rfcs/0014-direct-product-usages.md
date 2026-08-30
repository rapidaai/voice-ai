# RFC 0014: Direct Product Usages

- Status: Accepted
- Owner: Assistant and Web Platform
- Created: 2026-08-29
- Updated: 2026-08-29
- Reviewers: Independent plan challenger, billing owner

## Summary

Add `ProductUsageService.CreateProductUsages`. Assistant-api sends usage directly from its billing
collector to web-api over gRPC using the current authentication context. Web-api stores usage in
one `product_usages` table following the repository `Audited`, `Mutable`, and `Organizational`
model conventions. No outbox or SDK changes are included.

## Context

Assistant-api already emits duration usage records and has an unused billing collector publisher
boundary. Web-api has billing plan quota messages but no usage ingestion or persistence. The
technical owner explicitly selected direct delivery without an outbox and accepted that a failed
RPC is observable but not durably replayed by this change.

## Goals

- Store project-level product usage for subscription and over-usage calculations.
- Use `usageType`, positive integer `usages`, `unit`, and `occurredAt`.
- Use current PAT, project, and delegated internal authentication behavior.
- Keep assistant, conversation, and message identity out of billing records.
- Test PAT and project-key authentication paths.

## Non-Goals

- Durable retry or an outbox.
- Invoice calculation and payment collection.
- SDK generation or SDK publication.
- UI changes.

## Scope and Ownership

### Allowed Paths

- `rfcs/0014-direct-product-usages.md` and its JSON artifacts.
- `protos/artifacts/billing-api.proto`, `protos/billing-api.pb.go`, and
  `protos/billing-api_grpc.pb.go`.
- `api/web-api/migrations/000013_create_product_usages.*.sql`.
- `api/web-api/internal/entity/product_usage.go`.
- `api/web-api/internal/service/productusage/**`.
- `api/web-api/api/product_usage.go`, tests, and `api/web-api/router/web.go`.
- `pkg/clients/web/product_usage_client.go` and tests.
- `pkg/types/product_usage.go` and tests - authoritative usage type and unit pairs.
- `api/assistant-api/internal/observability/collectors/collector.go` and tests - central runtime
  collector factory used by every supported transport.
- `api/assistant-api/internal/observability/collectors/billing/**`.
- `tests/system/systemcheck/**` and the assistant smoke collection.

### Out-of-Scope Paths

- SDKs and generated Python artifacts.
- Provider implementations.
- Authentication formats or credential storage.
- Billing rating, invoices, and payments.

## Proposed Design

Add a dedicated `ProductUsageService` so the existing `BillingServiceServer` interface is not
changed. `CreateProductUsagesRequest` contains at most 100 `ProductUsage` records:

- `usageId = 1`, UUID string used for idempotency.
- `usageType = 2`, stable string matching `BillingPlanQuota.resourceType`.
- `usages = 3`, positive `int64` quantity.
- `unit = 4`, string unit such as `nanosecond`, `token`, `character`, `request`, or `byte`.
- `occurredAt = 5`, protobuf timestamp truncated to microsecond precision before sending.

`CreateProductUsagesResponse` returns `createdCount = 1` and `duplicateCount = 2`. The request is
transactional. Exact duplicates are successful; conflicting duplicate IDs fail with
`AlreadyExists` and insert nothing from that request.

The web entity embeds:

```go
gorm_model.Audited
gorm_model.Mutable
gorm_model.Organizational
```

It adds `UsageID`, `UsageType`, `Usages`, `Unit`, and `OccurredAt`. Organization and project are
derived from `types.Authorize(ctx)`, never from request fields. The handler accepts the repository's
current authenticated user, project, and service scopes when project context is present.

`pkg/types/product_usage.go` is the single registry of accepted usage type and unit pairs. The
initial six duration types are `stt_duration`, `tts_duration`, `vad_duration`, `eos_duration`,
`denoise_duration`, and `llm_duration`, each with unit `nanosecond`. Web-api rejects unknown usage
types and mismatched units. Existing external plan sources must populate `BillingPlanQuota.unit`
from this registry before quota enforcement is enabled.

The assistant collector maps `RecordUsage.Component` to `usageType`, duration nanoseconds to
`usages`, `nanosecond` to `unit`, and the record timestamp to microsecond precision. It forwards
the recorder authentication through `InternalClient.WithAuth`. It sends no scope details except
those already carried by authentication and sends no provider or arbitrary attributes.

`collectors.NewWithEnv` remains the central collector factory and returns a composite containing
configured telemetry plus billing. Every current runtime recorder already calls this factory,
including gRPC/WebRTC talk, AudioSocket, SIP sessions, SIP registration, and telephony status
reporting. Tests cover each recorder construction path and prove billing is present when enabled.

The direct call uses a five-second deadline. A failed call is returned to the recorder, logged by
the existing owner, and counted. This RFC intentionally provides no durable replay.

## Contracts and Compatibility

- The new service and messages are additive.
- `BillingPlanQuota` gains `string unit = 4`.
- Existing generated billing service interfaces remain unchanged.
- Existing usage producers and component names remain unchanged.
- Existing plan sources may omit unit until billing enforcement is introduced.
- No SDK artifacts are generated or changed.

## Failure and Recovery

- Invalid requests return `InvalidArgument`.
- Missing authentication returns `Unauthenticated`.
- Authentication without project context returns `PermissionDenied`.
- Conflicting duplicate IDs return `AlreadyExists`.
- Database failures return `Internal` and roll back the request.
- Assistant publish failures do not retry after the five-second call and may lose usage. This is
  the accepted no-outbox tradeoff and must be observable before billing enforcement is enabled.

## Security and Privacy

- Current credentials and middleware remain unchanged.
- Tenant ownership comes only from authenticated organization and project context.
- No credential is persisted or logged.
- No assistant, conversation, message, provider, prompt, transcript, trace, or arbitrary
  attribute is stored.

## Observability

- Count assistant publish success and failure by usage type and unit.
- Count web-api created, duplicate, rejected, and persistence failure outcomes.
- Do not use usage ID, organization ID, or project ID as metric labels.

## Data and Migration

Migration `000013` creates `product_usages` with:

- Standard `Audited`: `id`, `created_date`, `updated_date`.
- Standard `Mutable`: `status`, created actor fields, updated actor fields.
- Standard `Organizational`: `organization_id`, `project_id`.
- `usage_id varchar(36)` not null.
- `usage_type varchar(100)` not null.
- `usages bigint` not null with `usages > 0`.
- `unit varchar(32)` not null.
- `occurred_at timestamp without time zone` not null at microsecond precision.
- Indexes on `(project_id, usage_type, occurred_at)` and
  `(organization_id, usage_type, occurred_at)`.
- A unique constraint on `(organization_id, project_id, usage_id)`.

The table is append-only through this API. `Mutable` is retained for consistency with current
repository entities, but this feature exposes no update or delete RPC.

## Rollout

1. Generate Go protobuf artifacts only.
2. Apply web-api migration and deploy the product usage service.
3. Deploy assistant collector publishing.
4. Run focused tests and both smoke authentication modes.
5. Monitor failures before enabling any later billing enforcement.

## Rollback

- Disable assistant collector registration.
- Roll back assistant code before web-api code.
- Retain the ledger table for audit unless removal is explicitly approved.

## Alternatives Considered

- A durable outbox was rejected by the technical owner for this scope.
- Assistant-specific attribution was rejected because billing is project-level.
- Extending `BillingService` was rejected to avoid changing its generated server interface.

## Testing and Verification

- Proto descriptor tests cover the service, method, fields, and quota unit field.
- Web tests cover PAT, project, and delegated service auth with project context; missing project;
  positive quantity validation; exact duplicate; conflicting duplicate; and rollback.
- Web tests prove the same usage ID is accepted independently for different projects.
- Collector tests cover mapping, timestamp precision, excluded fields, success, and gRPC failure.
- `systemcheck product-usage-smoke` seeds a PAT and project credential, calls
  `ProductUsageService.CreateProductUsages` once with each credential, and queries PostgreSQL to
  assert the expected organization, project, usage type, quantity, unit, and idempotency behavior.
- Existing assistant smoke still runs once with PAT headers and once with project API key.

```text
go test ./api/web-api/api ./api/web-api/internal/service/productusage
go test ./api/assistant-api/internal/observability/collectors/billing ./pkg/clients/web
go test ./tests/system/cmd/protobuf-descriptor ./tests/system/systemcheck
go run ./tests/system/cmd/systemcheck product-usage-smoke --web-address localhost:9001 --postgres-host localhost --auth-mode both
make agent-finalize CHANGED_FILES="$(git diff --name-only --diff-filter=ACMR HEAD | paste -sd, -)"
```

## Acceptance Criteria

- [ ] `CreateProductUsages` persists project usage using current authentication.
- [ ] The table follows `Audited`, `Mutable`, and `Organizational` conventions.
- [ ] Retries with the same ID and payload do not create duplicate usage.
- [ ] Timestamps are compared and stored at microsecond precision.
- [ ] Unknown usage types and mismatched units are rejected by one shared registry.
- [ ] Usage ID uniqueness is scoped to organization and project.
- [ ] Assistant/message attribution is absent from the contract and table.
- [ ] The collector calls web-api directly without an outbox.
- [ ] Every supported recorder path receives the central billing collector.
- [ ] PAT and project-key smoke paths both create and verify product usage.
- [ ] Focused tests and `agent-finalize` pass.

## Open Questions

None.

## Challenge Resolution

- All current recorder construction paths use the central `collectors.NewWithEnv` factory, which
  owns billing collector registration.
- One shared registry defines valid usage type and unit pairs and supplies future quota units.
- Idempotency uniqueness is scoped by organization and project.
- The product usage smoke directly invokes and verifies ingestion under PAT and project keys.

## Artifact Index

- `jsons/plan.json` - governed implementation plan, awaiting challenge.
- `jsons/challenge.json` - first independent challenge, recorded.
- `jsons/amendment-01-plan.json` - first amendment, recorded.
- `jsons/amendment-01-challenge.json` - final independent challenge, pending.
- `jsons/confirmation.json` - technical owner confirmation, represented by the conversation after
  challenge approval.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-08-29 | Use current credentials | Technical owner | Conversation confirmation |
| 2026-08-29 | Use `CreateProductUsages` with usage type, quantity, and unit | Technical owner | Conversation confirmation |
| 2026-08-29 | Use one audited organizational table and no outbox | Technical owner | Conversation confirmation |
