# RFC 0014: Direct Product Usages

- Status: Accepted
- Owner: Assistant and Web Platform
- Created: 2026-08-29
- Updated: 2026-09-05
- Reviewers: Independent plan challenger, billing owner

## Summary

Add `ProductUsageService.CreateProductUsage`. Assistant-api sends usage directly from its billing
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
- `tests/smoke/product-usage/**`, `tests/smoke/run.sh`, and the assistant smoke collection.

### Out-of-Scope Paths

- SDKs and generated Python artifacts.
- Provider implementations.
- Authentication formats or credential storage.
- Billing rating, invoices, and payments.

## Proposed Design

Add a dedicated `ProductUsageService` so the existing `BillingServiceServer` interface is not
changed. `CreateProductUsageRequest` contains one usage record:

- `usageType = 1`, stable string matching `BillingPlanQuota.resourceType`.
- `usages = 2`, positive `int64` quantity.
- `unit = 3`, string unit such as `nanosecond`, `token`, `character`, `request`, or `byte`.
- `occurredAt = 4`, protobuf timestamp truncated to microsecond precision before sending.

`CreateProductUsage` stores one row with a server-assigned audited bigint ID and returns that row in
`GetProductUsageResponse`. Callers do not provide an idempotency key, so repeated requests create
independent usage rows.

The web entity embeds:

```go
gorm_model.Audited
gorm_model.Mutable
gorm_model.Organizational
```

It adds `UsageType`, `Usages`, `Unit`, and `OccurredAt`. Organization and project are
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

Each runtime constructs the billing collector with an explicit `billing.Config`. The process-owned
`RapidaClient.ProductUsage` is shared by gRPC/WebRTC talk, AudioSocket, SIP sessions, SIP
registration, and telephony status reporting. Tests cover each recorder construction path.

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
- Count web-api created, rejected, and persistence failure outcomes.
- Do not use record ID, organization ID, or project ID as metric labels.

## Data and Migration

Migration `000013` creates `product_usages` with:

- Standard `Audited`: `id`, `created_date`, `updated_date`.
- Standard `Mutable`: `status`, created actor fields, updated actor fields.
- Standard `Organizational`: `organization_id`, `project_id`.
- `usage_type varchar(100)` not null.
- `usages bigint` not null with `usages > 0`.
- `unit varchar(32)` not null.
- `occurred_at timestamp without time zone` not null at microsecond precision.
- Indexes on `(organization_id, project_id, usage_type, occurred_at)` and
  `(organization_id, usage_type, occurred_at)`.

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
  positive quantity validation; repeated events; and rollback.
- Collector tests cover mapping, timestamp precision, excluded fields, non-positive duration,
  success, and gRPC failure.
- `product-usage-smoke` seeds a PAT and project credential, calls
  `ProductUsageService.CreateProductUsage` once with each credential, and queries PostgreSQL to
  assert the expected organization, project, usage type, quantity, and unit.
- Existing assistant smoke still runs once with PAT headers and once with project API key.

```text
go test ./api/web-api/api ./api/web-api/internal/service/productusage
go test ./api/assistant-api/internal/observability/collectors/billing ./pkg/clients/web
go test ./scripts/contracts/protobuf-descriptor
tests/smoke/run.sh product-usage-smoke
just agent-finalize "$(git diff --name-only --diff-filter=ACMR HEAD | paste -sd, -)"
```

## Acceptance Criteria

- [x] `CreateProductUsage` persists project usage using current authentication.
- [x] The table follows `Audited`, `Mutable`, and `Organizational` conventions.
- [x] The database assigns the product usage ID.
- [x] Timestamps are compared and stored at microsecond precision.
- [x] Unknown usage types and mismatched units are rejected by one shared registry.
- [x] Assistant and message attribution are absent from the contract and table.
- [x] The collector calls web-api directly without an outbox.
- [x] Every supported recorder path receives the billing collector.
- [x] PAT and project-key smoke paths both create and verify product usage.
- [x] Focused tests and `agent-finalize` pass.

## Open Questions

None.

## Challenge Resolution

- All current recorder construction paths configure billing explicitly with the process-owned
  product usage client.
- One shared registry defines valid usage type and unit pairs and supplies future quota units.
- Amendment 02 replaced batch idempotency with singular creation and server-assigned IDs.
- The product usage smoke directly invokes and verifies ingestion under PAT and project keys.

## Artifact Index

- `jsons/plan.json` - governed implementation plan, updated to the approved contract.
- `jsons/challenge.json` - initial independent challenge, superseded by approved amendments.
- `jsons/amendment-01-plan.json` - first amendment.
- `jsons/amendment-01-challenge.json` - approved first amendment challenge.
- `jsons/amendment-02-plan.json` - confirmed singular-create contract amendment.
- `jsons/amendment-02-challenge.json` - approved singular-create contract challenge.
- `jsons/amendment-02-review.json` - approved implementation review for the singular contract.
- `jsons/confirmation.json` - technical owner confirmation linked to Amendment 02.
- `jsons/inventory.json` - implementation and verification artifact inventory.
- `jsons/operational.json` - delivery, timeout, and rollback decisions.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-08-29 | Use current credentials | Technical owner | Conversation confirmation |
| 2026-08-29 | Initially approve batched product usage creation | Technical owner | Base confirmation |
| 2026-08-29 | Use one audited organizational table and no outbox | Technical owner | Conversation confirmation |
| 2026-08-30 | Replace batching and caller IDs with singular creation and server-assigned IDs | Technical owner | Amendment 02 |
