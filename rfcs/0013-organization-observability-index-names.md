# RFC 0013: Organization Observability Index Names

- Status: Accepted
- Owner: Observability maintainers
- Created: 2026-08-26
- Updated: 2026-08-26
- Reviewers: Independent plan challenger

## Summary

Add the organization identifier to daily OpenSearch index names for timeline records,
telemetry logs, telemetry events, and telemetry metrics. Use `global` when the organization
identifier is zero. This RFC supersedes RFC 0012 and intentionally excludes all archive
cron behavior.

## Context

The timeline collector currently writes `<prefix>-YYYYMMDD`. The telemetry exporter writes
`<prefix>-<kind>-YYYYMMDD`. Both writers already receive and store organization ID.

## Goals

- Add organization ID to all four index-name families.
- Use `global` for organization ID zero.
- Preserve prefixes, UTC dates, documents, bulk writes, and wildcard readers.

## Non-Goals

- Archive cron changes or compatibility verification.
- Retention, policies, templates, explicit index creation, or migration.
- API, schema, connector, or configuration changes.

## Scope and Ownership

### Allowed Paths

- `pkg/telemetry/providers/opensearch.go` - telemetry implementation owner.
- `pkg/telemetry/providers/opensearch_test.go` - telemetry test owner.
- `api/assistant-api/internal/observability/collectors/timeline/collector.go` - timeline implementation owner.
- `api/assistant-api/internal/observability/collectors/timeline/collector_test.go` - timeline test owner.
- `rfcs/0013-organization-observability-index-names.md` - RFC owner.
- `rfcs/0013-organization-observability-index-names/jsons/` - governance artifact owner.

### Out-of-Scope Paths

- `pkg/connectors/`
- `api/assistant-api/api/observability/`
- External cron and OpenSearch deployment configuration.

## Proposed Design

Change only the existing private index-name functions and call sites.

```text
timeline: <prefix>-<organization-segment>-<UTC YYYYMMDD>
logs:     <prefix>-logs-<organization-segment>-<UTC YYYYMMDD>
events:   <prefix>-events-<organization-segment>-<UTC YYYYMMDD>
metrics:  <prefix>-metrics-<organization-segment>-<UTC YYYYMMDD>
```

`organization-segment` is the unsigned base-10 organization ID, or `global` when the ID is
zero. Date selection remains based on occurrence time with the existing current-time
fallback and UTC formatting.

## Contracts and Compatibility

- Existing document fields and bulk payloads remain unchanged.
- Existing custom prefixes remain unchanged before the new organization segment.
- Existing wildcard telemetry readers continue matching old and new indexes.
- Existing indexes are not renamed or modified.

## Failure and Recovery

- Organization ID zero routes to `global` instead of dropping observability data.
- Existing bulk error behavior remains unchanged.
- Rollback restores the prior shared names without rewriting data.

## Security and Privacy

Organization ID already exists in each document. No permission or credential changes are
introduced.

## Observability

Focused tests inspect bulk metadata for the destination index. Existing bulk failure logs
remain unchanged.

## Data and Migration

None. Existing indexes remain unchanged.

## Rollout

The release owner must attach archive-cron dry-run output for one old and one new index name
for each enabled prefix before production deployment. The cron implementation remains out
of scope, but compatibility evidence is a rollout gate because index names are its input.

Deploy the writer change and verify new index names and existing wildcard reads. Monitor
OpenSearch index and shard counts during rollout. Organization-by-day routing intentionally
increases index count, and the release owner must stop rollout if cluster health degrades or
the operator's existing shard limit is reached.

## Rollback

Roll back the application binary. Old and new indexes remain readable through existing
wildcard patterns.

## Alternatives Considered

- A shared helper was rejected because two private formatting functions are simpler.
- Application retention behavior was rejected as out of scope.
- A literal marker segment was rejected because the requested contract is only organization
  ID or `global` followed by date.

## Testing and Verification

Required tests:

- Timeline with nonzero organization ID and with `global`.
- Log, event, and metric indexes with nonzero organization ID and with `global`.
- Custom prefixes.
- UTC occurrence dates and zero-time fallback.
- Preservation of organization ID inside documents.
- Existing bulk error behavior.

```bash
go test ./pkg/telemetry/providers/...
go test ./api/assistant-api/internal/observability/collectors/timeline/...
go test ./api/assistant-api/api/observability/...
make agent-finalize CHANGED_FILES="pkg/telemetry/providers/opensearch.go,pkg/telemetry/providers/opensearch_test.go,api/assistant-api/internal/observability/collectors/timeline/collector.go,api/assistant-api/internal/observability/collectors/timeline/collector_test.go"
```

## Acceptance Criteria

- [ ] Timeline, logs, events, and metrics include organization ID before the UTC date.
- [ ] Organization ID zero uses `global`.
- [ ] Custom prefixes and document organization fields remain unchanged.
- [ ] Existing wildcard readers require no code change.
- [ ] Focused tests and finalization pass.
- [ ] No critical or major review findings remain.

## Open Questions

None.

## Challenge Resolution

Challenge round 1 requested a complete Governed plan, explicit index-count risk, named cron
compatibility evidence, and enumerated test cases. These requirements are included without
expanding production code scope. Challenge round 2 is pending.

## Artifact Index

- `rfcs/0013-organization-observability-index-names/jsons/plan.json` - final plan.
- `rfcs/0013-organization-observability-index-names/jsons/challenge.json` - pending challenge receipt.
- `rfcs/0013-organization-observability-index-names/jsons/confirmation.json` - pending confirmation receipt.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-08-26 | Supersede RFC 0012 with a code-only index-name change | User | Direct implementation instruction |
| 2026-08-26 | Use organization ID or `global` and add no other behavior | User | Direct implementation instruction |
