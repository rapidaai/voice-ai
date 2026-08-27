# RFC 0012: Organization-Scoped Daily Observability Indexes

- Status: Draft
- Owner: Observability maintainers
- Created: 2026-08-25
- Updated: 2026-08-27
- Reviewers: Independent plan challenger

## Summary

Add the organization identifier to every daily OpenSearch index written by the assistant
observability timeline collector and the shared telemetry OpenSearch exporter. When no
organization identifier is available, use the literal `global`.

The application changes only index naming. An external cron owns index archival and any
retention-specific behavior.

## Context

The timeline collector currently writes records to `<prefix>-YYYYMMDD`. The telemetry
OpenSearch exporter writes logs, events, and metrics to
`<prefix>-<kind>-YYYYMMDD`. Both writers already receive an organization identifier and
store it inside the indexed document, but neither includes it in the index name.

Retention differs by organization. The external archive cron therefore needs complete
indexes that belong to one organization and one UTC date.

Existing telemetry readers search `rapida-logs-*`, `rapida-events-*`, and
`rapida-metrics-*`. Those patterns match both current and proposed names.

## Goals

- Add organization ID to timeline, log, event, and metric index names.
- Use `global` when organization ID is zero or unavailable.
- Preserve existing prefixes, UTC daily rotation, document bodies, and bulk behavior.
- Keep existing telemetry wildcard reads compatible.
- Leave archival ownership with the external cron.

## Non-Goals

- Implementing or configuring the archive cron.
- Adding retention settings, OpenSearch policies, templates, or cleanup workers.
- Explicitly creating, deleting, closing, snapshotting, or restoring indexes.
- Migrating or renaming existing indexes.
- Changing document schemas, APIs, protobufs, databases, or OpenSearch connector behavior.

## Scope and Ownership

### Allowed Paths

- `pkg/telemetry/providers/opensearch.go` - implementation owner; log, event, and metric
  index naming.
- `pkg/telemetry/providers/opensearch_test.go` - implementation owner; exporter routing
  coverage.
- `api/assistant-api/internal/observability/collectors/timeline/collector.go` -
  implementation owner; timeline index naming.
- `api/assistant-api/internal/observability/collectors/timeline/collector_test.go` -
  implementation owner; timeline routing coverage.
- `rfcs/0012-organization-scoped-observability-indexes.md` - RFC owner.
- `rfcs/0012-organization-scoped-observability-indexes/jsons/` - governance artifact owner.

### Out-of-Scope Paths

- `pkg/connectors/`
- `api/assistant-api/api/observability/`
- OpenSearch cluster and deployment configuration.
- External archive cron source and configuration.
- Existing OpenSearch indexes.

## Proposed Design

Change the two existing private index-name functions. Do not introduce a new package,
configuration option, connector method, background worker, or index lifecycle component.

Timeline index names become:

```text
<prefix>-<organization-segment>-<UTC YYYYMMDD>
```

Examples:

```text
rapida-timeline-42-20260825
custom-timeline-42-20260825
rapida-timeline-global-20260825
```

Telemetry index names become:

```text
<prefix>-<kind>-<organization-segment>-<UTC YYYYMMDD>
```

Examples:

```text
rapida-logs-42-20260825
rapida-events-42-20260825
rapida-metrics-42-20260825
custom-logs-42-20260825
```

For a nonzero organization ID, `organization-segment` is its unsigned base-10 value. For
organization ID zero, it is `global`.

The date continues to come from the record occurrence time. A zero occurrence time falls
back to the current time, and formatting remains UTC. OpenSearch index creation continues
to occur through the existing bulk write behavior.

The archive cron must use the exact configured prefix when parsing names. After removing
that prefix, it reads the final segment as the UTC date and the preceding segment as the
organization. This avoids interpreting a token inside a custom prefix as the organization.

## Contracts and Compatibility

- Timeline changes from `<prefix>-YYYYMMDD` to
  `<prefix>-<organization-segment>-YYYYMMDD`.
- Logs change from `<prefix>-logs-YYYYMMDD` to
  `<prefix>-logs-<organization-segment>-YYYYMMDD`.
- Events change from `<prefix>-events-YYYYMMDD` to
  `<prefix>-events-<organization-segment>-YYYYMMDD`.
- Metrics change from `<prefix>-metrics-YYYYMMDD` to
  `<prefix>-metrics-<organization-segment>-YYYYMMDD`.
- Existing index prefixes remain unchanged.
- Existing document fields, including organization ID, remain unchanged.
- Existing wildcard readers continue matching old and new names.
- Existing indexes are not modified or migrated.
- The archive cron must support old and new layouts during rollout and must use the exact
  configured prefix rather than guessing prefix boundaries.

## Failure and Recovery

- Missing organization context routes to the explicit `global` segment so observability
  writes are not dropped.
- Existing bulk error propagation and logging remain unchanged.
- Rollback restores the prior shared index format without rewriting data.
- Old and new indexes can coexist and remain queryable through existing wildcard readers.
- Archive failures and late-arriving records remain owned by the external cron.

## Security and Privacy

- Organization IDs already exist in observability documents. Including them in index names
  does not add new tenant data.
- OpenSearch credentials and permissions do not change.
- Physical index separation supports archival boundaries but does not replace query-time
  authorization.

## Observability

- Existing bulk failure logs remain unchanged.
- Tests inspect bulk metadata to verify destination index names.
- Operators can list indexes using the existing prefixes to verify rollout.

## Data and Migration

No database or document migration is required. Existing shared indexes remain unchanged.
New writes use organization-scoped indexes after deployment.

## Rollout

1. Update the external cron to recognize both old and new index names using its exact
   configured prefixes.
2. Deploy the application change.
3. Confirm new timeline, log, event, and metric indexes contain the expected organization
   segment and UTC date.
4. Confirm sampled documents contain the same organization ID as their index name.
5. Confirm existing telemetry reads and dashboards include both layouts.
6. Enable organization-specific archival in the external cron.

Stop rollout if index names contain the wrong organization, records disappear from existing
queries, or OpenSearch rejects writes.

## Rollback

Roll back the application binary. New writes return to the previous shared daily index
names. No data rewrite is required, and existing wildcard readers continue matching both
layouts.

The external cron must continue recognizing both layouts until old and new indexes have
completed their configured archive lifecycle.

## Alternatives Considered

- Shared indexes with document-level archival were rejected because the cron needs to
  archive complete indexes independently by organization.
- OpenSearch Index State Management was rejected because the external cron owns archival.
- An application cleanup worker was rejected because the application does not own archive
  lifecycle operations.
- A shared naming package was rejected because two small private formatting functions are
  simpler and avoid adding a new public API.
- A literal `org` marker was rejected because the cron already owns exact configured
  prefixes and can parse the final organization and date segments directly.

## Testing and Verification

Required tests:

- Timeline default prefix with a nonzero organization ID.
- Timeline custom prefix with a nonzero organization ID.
- Timeline `global` fallback.
- Telemetry log, event, and metric names with a nonzero organization ID.
- Telemetry `global` fallback for every supported record kind.
- UTC date behavior.
- Organization ID preservation inside indexed documents.
- Existing bulk error behavior.

Exact commands:

```bash
go test ./pkg/telemetry/providers/...
go test ./api/assistant-api/internal/observability/collectors/timeline/...
go test ./api/assistant-api/api/observability/...
make agent-finalize CHANGED_FILES="pkg/telemetry/providers/opensearch.go,pkg/telemetry/providers/opensearch_test.go,api/assistant-api/internal/observability/collectors/timeline/collector.go,api/assistant-api/internal/observability/collectors/timeline/collector_test.go"
```

## Acceptance Criteria

- [ ] Timeline uses `<prefix>-<organization-segment>-YYYYMMDD`.
- [ ] Logs use `<prefix>-logs-<organization-segment>-YYYYMMDD`.
- [ ] Events use `<prefix>-events-<organization-segment>-YYYYMMDD`.
- [ ] Metrics use `<prefix>-metrics-<organization-segment>-YYYYMMDD`.
- [ ] Nonzero organization IDs use unsigned base-10 formatting.
- [ ] Organization ID zero uses `global`.
- [ ] Dates remain based on occurrence time and formatted in UTC.
- [ ] Custom prefixes remain supported.
- [ ] Organization IDs remain present in documents.
- [ ] Existing wildcard reads cover old and new layouts.
- [ ] No archive lifecycle behavior is added to the application.
- [ ] Focused tests and `make agent-finalize` pass.
- [ ] No unresolved critical or major review findings remain.

## Open Questions

None.

## Challenge Resolution

Challenge round 1 requested an unambiguous archive parser contract and additional external
cron operational controls. The user selected the smaller application contract: append the
organization ID, use `global` when unavailable, and leave all archival behavior with the
cron. Challenge round 2 returned `revise` because the remaining cron, capacity, and rollout
requirements exceeded the requested application-only scope. The plan was blocked after two
revision cycles, and RFC 0013 superseded this proposal with the smaller code-only contract.

## Artifact Index

- `rfcs/0012-organization-scoped-observability-indexes/jsons/plan.json` - blocked final plan.
- `rfcs/0012-organization-scoped-observability-indexes/jsons/challenge.json` - challenge
  round 1 receipt; decision `revise`.
- `rfcs/0012-organization-scoped-observability-indexes/jsons/amendment-01-plan.json` - user
  decision and reduced-scope amendment.
- `rfcs/0012-organization-scoped-observability-indexes/jsons/amendment-01-challenge.json` -
  challenge round 2 receipt; decision `revise`, implementation unauthorized.
- No confirmation receipt exists because the RFC was blocked and superseded before approval.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-08-25 | Use organization-scoped UTC daily indexes for all OpenSearch observability writers | User | Conversation and `jsons/plan.json` |
| 2026-08-25 | Use `global` when organization ID is unavailable | User | Conversation and `jsons/amendment-01-plan.json` |
| 2026-08-25 | Keep implementation limited to adding organization to existing names | User | Conversation and `jsons/amendment-01-plan.json` |
| 2026-08-25 | Keep archive lifecycle ownership in the external cron | User | Conversation and `jsons/plan.json` |
| 2026-08-26 | Block this RFC after two revision cycles and supersede it with RFC 0013 | Coordinator | `jsons/plan.json` and RFC 0013 |
