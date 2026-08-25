# RFC 0012: Organization-Scoped Daily Observability Indexes

- Status: Draft
- Owner: Observability maintainers
- Created: 2026-08-25
- Updated: 2026-08-25
- Reviewers: Independent plan challenger (pending), operations owner for the archive cron (pending)

## Summary

Include the organization identifier and UTC calendar date in every OpenSearch index name
written by the assistant observability timeline collector and the shared telemetry
OpenSearch exporter. The new physical index boundary allows an external cron process to
archive complete indexes according to organization-specific retention schedules without
running document-level queries.

The application remains responsible only for selecting the destination index and writing
records. Index archival, deletion after archival, restore procedures, and retention
configuration remain owned by the external cron and its operators.

## Context

The timeline collector currently writes every organization into a shared daily index named
`rapida-timeline-YYYYMMDD`. Its index name is generated in
`api/assistant-api/internal/observability/collectors/timeline/collector.go`, while the
organization identifier is already available through `observability.Scope` and is stored
inside each indexed document.

The shared telemetry OpenSearch exporter currently writes logs, events, and metrics into
`rapida-logs-YYYYMMDD`, `rapida-events-YYYYMMDD`, and
`rapida-metrics-YYYYMMDD`. The exporter receives `telemetry.Scope.OrganizationID` and
stores it in each document, but it does not include that value in the index name.

The existing telemetry read API searches wildcard patterns `rapida-logs-*`,
`rapida-events-*`, and `rapida-metrics-*`. Those patterns match both the current names and
the proposed names, so the reader does not require a compatibility change.

An external cron will archive observability indexes. Retention can differ by organization,
which means a shared daily index is not a sufficient archival boundary. Document-level
filtering, copying, and deletion would add failure modes and would not be index archival.

## Goals

- Route every timeline record to a daily index scoped to its organization.
- Route every OpenSearch telemetry log, event, and metric to a daily index scoped to its
  organization.
- Preserve UTC date partitioning and existing configurable index prefixes.
- Preserve organization and project identifiers inside indexed documents.
- Keep existing wildcard telemetry reads compatible with old and new index names.
- Provide a deterministic fallback index for records without organization context.
- Give the external archive cron an index-level organization and date boundary.

## Non-Goals

- Implementing or scheduling the external archive cron.
- Defining organization retention periods or archive destinations.
- Deleting, closing, snapshotting, restoring, or migrating OpenSearch indexes.
- Renaming, reindexing, or backfilling existing observability indexes.
- Adding OpenSearch Index State Management policies or index templates.
- Changing observability document mappings, JSON fields, record identifiers, or payloads.
- Changing telemetry API request or response contracts.
- Changing OpenSearch connector behavior or automatic index creation settings.

## Scope and Ownership

### Allowed Paths

- `pkg/telemetry/opensearch_index.go` - implementation owner; shared organization and UTC
  date index-name contract.
- `pkg/telemetry/opensearch_index_test.go` - implementation owner; contract tests for index
  naming.
- `pkg/telemetry/providers/opensearch.go` - implementation owner; log, event, and metric
  routing.
- `pkg/telemetry/providers/opensearch_test.go` - implementation owner; exporter routing
  tests.
- `api/assistant-api/internal/observability/collectors/timeline/collector.go` -
  implementation owner; timeline routing.
- `api/assistant-api/internal/observability/collectors/timeline/collector_test.go` -
  implementation owner; timeline routing tests.
- `rfcs/0012-organization-scoped-observability-indexes.md` - RFC owner.
- `rfcs/0012-organization-scoped-observability-indexes/jsons/` - governance artifact owner.

### Out-of-Scope Paths

- `pkg/connectors/`
- `api/assistant-api/api/observability/`
- OpenSearch deployment and cluster configuration.
- External archive cron source, configuration, deployment, and credentials.
- Existing OpenSearch indexes and stored documents.

## Proposed Design

Add one shared telemetry function that constructs a daily OpenSearch index name from an
already-resolved prefix, an organization identifier, and an occurrence timestamp. This
function is the single source of truth for the organization and date suffix used by both
writers.

The naming contract is versioned with an explicit `org` marker so the archive cron can
distinguish it from every existing shared index, including indexes whose custom prefix ends
with a numeric token or the word `global`:

```text
<resolved-prefix>-org-<organization-segment>-<UTC YYYYMMDD>
```

For a nonzero organization identifier, `organization-segment` is the base-10 unsigned
integer value. For a zero organization identifier, `organization-segment` is `global`.

Timeline examples:

```text
rapida-timeline-org-42-20260825
custom-timeline-org-42-20260825
rapida-timeline-org-global-20260825
```

The timeline collector passes its configured `indexPrefix` directly to the shared naming
function together with `scope.GlobalScopeValue().OrganizationID` and the record occurrence
time.

Telemetry exporter examples:

```text
rapida-logs-org-42-20260825
rapida-events-org-42-20260825
rapida-metrics-org-42-20260825
custom-logs-org-42-20260825
```

The exporter first resolves its configured root prefix, appends the existing record kind
segment, and passes that result to the shared naming function together with
`scope.OrganizationID` and the record occurrence time.

If the occurrence timestamp is zero, the shared naming function uses the current time. It
always formats the date in UTC. No writer performs index existence checks or explicit index
creation as part of this change. The existing OpenSearch bulk behavior remains unchanged.

The index date continues to use record occurrence time to preserve existing partitioning
semantics. The archive cron must scan for eligible indexes on every run and must treat an
eligible index that reappears because of a late record as new archival work. Archive output
must be versioned or otherwise idempotent so a repeated archive does not overwrite a prior
verified artifact. The cron must remove an active index only after the current archive copy
has been verified. The maximum expected event lateness and the corresponding archive grace
period remain rollout inputs that must be recorded before this RFC can be accepted.

## Contracts and Compatibility

- Timeline index names change from `<prefix>-YYYYMMDD` to
  `<prefix>-org-<organization-segment>-YYYYMMDD`.
- Telemetry index names change from `<prefix>-<kind>-YYYYMMDD` to
  `<prefix>-<kind>-org-<organization-segment>-YYYYMMDD`.
- `organization-segment` is a base-10 organization identifier or the literal `global` when
  the identifier is zero.
- Dates remain eight decimal digits in UTC using the existing `YYYYMMDD` format.
- Existing custom prefix behavior remains unchanged before the new suffix.
- Existing wildcard telemetry readers remain compatible because `rapida-logs-*`,
  `rapida-events-*`, and `rapida-metrics-*` match both formats.
- Existing indexes remain readable and are not renamed or modified.
- The archive cron must identify the new layout by the literal `org` marker, parse the date
  from the final hyphen-delimited segment, and parse the organization from the immediately
  preceding segment. It must separately define handling for the `global` segment.
- The archive cron must support both old and new names during the compatibility window.
- Old shared names must never be interpreted as organization-scoped names. If the cron
  cannot classify a name unambiguously, it must report and skip the index without removing
  it from active storage.
- No public API, protobuf, database schema, or observability document schema changes.

## Failure and Recovery

- A zero organization identifier routes to an explicit `org-global` index instead of
  silently creating an index for numeric organization zero. The writer emits a warning with
  record kind and scope type, without payload data, whenever this fallback is used.
- Invalid or blank custom prefixes retain existing behavior and are outside this change.
- Bulk write errors retain the existing propagation and logging behavior.
- A deployment rollback resumes the previous shared index naming. Readers continue to
  search both layouts through existing wildcard patterns.
- The external cron must not archive an index unless its full name matches the approved
  prefix, organization segment, and date grammar.
- The external cron must not treat a failed or partial archive as permission to remove the
  source index. That behavior is outside this repository but is a rollout prerequisite.
- The external cron must rescan all eligible dates on every run so an old-date index
  recreated by a late record is archived again.

## Security and Privacy

- Organization identifiers are already present in observability documents. Adding the
  identifier to the index name does not introduce a new class of tenant identifier.
- OpenSearch credentials and permissions remain unchanged.
- The application must continue writing the organization identifier into each document so
  index routing can be audited against document scope.
- Index naming provides a physical archival boundary but does not replace query-time or
  OpenSearch authorization controls.
- Archive cron permissions should be limited to matching observability index prefixes and
  the required archive operations. Cron permission changes are outside this RFC's code
  scope.

## Observability

- Existing bulk write failure logs remain the primary application diagnostic.
- A warning identifies every write routed to `org-global` so operators can distinguish
  expected infrastructure records from missing tenant context.
- Tests inspect bulk metadata to prove that each record is routed to the expected physical
  index.
- Operators can list indexes by prefix and organization segment to verify rollout.
- No new application metric or log is required because this change only alters deterministic
  routing metadata.

## Data and Migration

No database schema or document migration is required.

Existing indexes retain their current names and contents. New application versions begin
writing to organization-scoped daily indexes immediately after deployment. During the
compatibility window, OpenSearch therefore contains both old shared indexes and new
organization-scoped indexes.

The external archive cron must recognize both layouts until all old shared indexes have
passed the maximum supported retention period and have been archived under the existing
operational process.

## Rollout

1. Update the archive cron parser and dry-run validation to recognize both old and new index
   layouts by the literal `org` marker. Do not enable new archival actions yet.
2. Deploy the application index-routing change.
3. Verify that timeline, log, event, and metric writes create organization-scoped daily
   indexes and retain matching organization identifiers in their documents.
4. Verify that telemetry reads and dashboards continue to find both old and new indexes.
5. Record the current active primary and replica shard counts, the projected peak from the
   organization count and retention horizon, the operator-approved hard threshold, the
   maximum expected event lateness, and the archive grace period in
   `jsons/operational-readiness.json`.
6. Enable organization-specific archive schedules in the cron after the new names are
   observed and validated.
7. Stop rollout if organization-scoped indexes are missing, organization IDs disagree with
   document scope, wildcard reads omit either layout, the cron cannot classify an index,
   late writes are not re-archived, or index growth reaches the approved threshold.

## Rollback

Roll back the application binary to restore the previous shared daily index names. No data
rewrite is required. Existing wildcard readers continue to include indexes produced before,
during, and after rollback.

Disable archival of the new organization-scoped pattern before rollback if the cron cannot
safely process a mixed layout. Indexes already archived by the external cron are not
restored by this application rollback and remain governed by the cron's recovery process.

The cron must continue rescanning eligible dates after rollback until the final
organization-scoped index has passed the maximum lateness and archive grace windows.

## Alternatives Considered

- Keep shared daily indexes and filter by `organizationId` during archival. Rejected because
  it requires document-level copy and removal rather than atomic index archival.
- Use OpenSearch Index State Management. Rejected because retention and archival are owned
  by an external cron.
- Add an application cleanup worker. Rejected because the application does not own archive
  scheduling or lifecycle operations.
- Create one index per retention tier. Rejected because organization retention schedules can
  differ independently and the archive cron requires an organization boundary.
- Add organization identifiers only to timeline indexes. Rejected because logs, events, and
  metrics require the same independent archival behavior.
- Duplicate index formatting in each writer. Rejected because index naming is an operational
  contract and should have one source of truth.

## Testing and Verification

Required test categories:

- Shared index-name contract for nonzero organization identifiers.
- Explicit `global` fallback for organization identifier zero.
- Warning emission when the `global` fallback is used.
- UTC conversion and zero-time fallback.
- Default and custom prefix preservation.
- Timeline log, event, metric, metadata, and usage routing through the organization-scoped
  name.
- Telemetry log, event, and metric routing through organization-scoped names.
- Preservation of organization identifiers inside documents.
- Existing wildcard telemetry query compatibility through focused API tests or direct
  assertion of unchanged patterns.
- Existing bulk error propagation.
- Old and new default and custom-prefix parser fixtures supplied by the archive cron owner.

Exact verification commands:

```bash
go test ./pkg/telemetry/...
go test ./api/assistant-api/internal/observability/collectors/timeline/...
go test ./api/assistant-api/api/observability/...
make agent-finalize CHANGED_FILES="pkg/telemetry/opensearch_index.go,pkg/telemetry/opensearch_index_test.go,pkg/telemetry/providers/opensearch.go,pkg/telemetry/providers/opensearch_test.go,api/assistant-api/internal/observability/collectors/timeline/collector.go,api/assistant-api/internal/observability/collectors/timeline/collector_test.go"
```

Operational verification before enabling archival:

```text
curl -fsS "$OPENSEARCH_URL/_cat/indices/rapida-*?h=index,pri,rep,docs.count,store.size&format=json"
curl -fsS "$OPENSEARCH_URL/_cluster/health?level=indices"
curl -fsS -H 'Content-Type: application/json' "$OPENSEARCH_URL/rapida-*-org-*-*/_search" -d '{"size":100,"_source":["organizationId"],"query":{"match_all":{}}}'
go test ./api/assistant-api/api/observability/... -run TestGetAllTelemetry
```

The archive cron owner must add its environment-specific dry-run and archive verification
commands to `jsons/operational-readiness.json`. That artifact must also record parser
fixtures for old and new default and custom prefixes, the approved shard threshold, the
measured shard counts, the maximum lateness, and the archive grace period. Absence of this
evidence blocks rollout.

## Acceptance Criteria

- [ ] Timeline records use `<prefix>-org-<organization-segment>-YYYYMMDD`.
- [ ] Telemetry logs use `<prefix>-logs-org-<organization-segment>-YYYYMMDD`.
- [ ] Telemetry events use `<prefix>-events-org-<organization-segment>-YYYYMMDD`.
- [ ] Telemetry metrics use `<prefix>-metrics-org-<organization-segment>-YYYYMMDD`.
- [ ] Nonzero organization identifiers use their base-10 representation.
- [ ] Zero organization identifiers use the literal `global`.
- [ ] Writes using the `global` fallback emit a warning without payload data.
- [ ] Dates are derived from record occurrence time and formatted in UTC.
- [ ] Existing custom prefixes remain supported.
- [ ] Organization identifiers remain present in indexed documents.
- [ ] Existing telemetry wildcard reads cover old and new index layouts.
- [ ] No application code lists, archives, closes, or deletes indexes.
- [ ] Focused tests and `make agent-finalize` pass.
- [ ] The archive cron owner confirms mixed-layout support before rollout.
- [ ] The archive cron proves ambiguous names are skipped, late eligible indexes are
  re-archived, and archive verification precedes source removal.
- [ ] Operational readiness records measured and projected shard counts, a hard capacity
  threshold, maximum event lateness, and archive grace period.
- [ ] No unresolved critical or major review findings remain.

## Open Questions

- Who owns the external archive cron compatibility confirmation?
- What hard active-shard threshold has the OpenSearch operator approved?
- What maximum event lateness and archive grace period has the archive cron owner approved?
- How long must the cron retain compatibility with the old shared index layout?
- Which current record producers are legitimately allowed to emit with organization ID zero?

## Challenge Resolution

Challenge round 1 returned `REVISE` for ambiguous mixed-layout parsing, missing shard
capacity gates, undefined late-write behavior, unconstrained zero-organization routing, and
insufficient operational commands. This revision adds the explicit `org` marker, safe
classification rules, repeated late-index archival requirements, zero-organization
warnings, and an operational-readiness artifact contract. Operator-owned values and exact
cron commands remain unresolved, so the RFC remains `Draft` and requires another
independent challenge after those inputs are recorded.

## Artifact Index

- `rfcs/0012-organization-scoped-observability-indexes/jsons/plan.json` - initial task plan;
  revised after challenge round 1.
- `rfcs/0012-organization-scoped-observability-indexes/jsons/challenge.json` - challenge
  round 1 receipt; decision `revise`.
- `rfcs/0012-organization-scoped-observability-indexes/jsons/confirmation.json` - not created;
  reserved for the exact-digest confirmation receipt.
- `rfcs/0012-organization-scoped-observability-indexes/jsons/operational-readiness.json` -
  not created; required before rollout.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-08-25 | Use organization-scoped UTC daily indexes for all OpenSearch observability writers | User and RFC owner | `jsons/plan.json` |
| 2026-08-25 | Keep archive lifecycle ownership in the external cron | User and RFC owner | `jsons/plan.json` |
| 2026-08-25 | Keep the RFC in Draft pending independent challenge and exact-digest confirmation | RFC owner | Governed workflow requirement |
| 2026-08-25 | Add an explicit `org` marker and operational safety gates | RFC owner | `jsons/challenge.json` |
