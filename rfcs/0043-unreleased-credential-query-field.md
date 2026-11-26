# RFC 0043: Unreleased Credential Query Field

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-09
- Supersedes: RFC 0042 Project API key query-field reservation decision only

## Context

RFC 0042 reserves every removed dedicated filter field. Independent review confirmed that the
access-management request fields are already released: their proto source is on the artifact
repository main branch and generated clients are present in published `@rapidaai/react@1.1.92`.
Those released numbers and names must remain reserved.

The Project API key `query = 4` field is different. It exists only in this feature worktree and has
never been published. Reserving it would preserve implementation history rather than a public wire
contract.

## Decision

Delete the unreleased Project API key `query` field without reserving its name or number:

```protobuf
message GetAllProjectCredentialRequest {
  Paginate paginate = 1;
  repeated Criteria criterias = 2;
  uint64 projectId = 3 [jstype = JS_STRING];
}
```

Retain RFC 0042 for released access-management fields. `ListPermissionsRequest` reserves released
field 2 and adds `criterias = 3`. `ListRolesRequest` reserves released fields 2 through 5 and adds
`criterias = 6`.

All criteria allowlists, validation, ownership, error behavior, security constraints, rollout, and
verification decisions from RFC 0042 remain unchanged except the Project API key descriptor must
show field 4 and name `query` as absent rather than reserved.

## Contracts and Ownership

- Protobuf source owns the final Project API key request shape and the released access-management
  reservations.
- Generated Go, Python, and React artifacts remain generator-owned.
- No backend, SDK, or UI behavior changes beyond RFC 0042.

## Alternatives Rejected

- Reserve Project API key field 4: rejected because no released consumer or stored wire payload
  uses it.
- Reuse released access-management field numbers: rejected because published clients already send
  those fields and reuse would be wire-incompatible.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Project API key field 4 is unreleased and can be deleted cleanly. Released access-management fields remain reserved. |
| Failure and recovery | N/A. Filter behavior and error handling remain RFC 0042. Rollback restores the complete feature contract. |
| Security and privacy | N/A. Criteria remain allowlisted and mapped to typed filters under RFC 0042. |
| Data and migration | N/A. No persisted representation or database migration exists. |
| Concurrency and resources | N/A. Field numbering does not change runtime ownership or resources. |
| Operations and observability | Generated-artifact and descriptor checks detect numbering drift before release. |

## Rollout and Rollback

Regenerate every language artifact and deliver proto, backend, SDK, and UI together. Stop on
generated drift or descriptor mismatch. Roll back the complete feature change; there is no data
rollback.

## Verification

Descriptor tests assert that `GetAllProjectCredentialRequest` contains `criterias = 2` and no field
or reservation named `query` or numbered 4. Access-management descriptor tests assert the RFC 0042
criteria numbers and all released reservations. Run the complete RFC 0042 verification suite plus:

```bash
just validate-rfc-layout
git diff --check -- rfcs/0043-unreleased-credential-query-field.md
```

## Review and Approval

- Challenger: Codex, task `/root/criteria_filter_rfc_challenge`
- Reviewed digest: SHA-256 `5cd50ff4aca4c7f9e86974ef909b21452e517d381a76d1643acaed0b352720b9`
- Outcome: Approved
- Resolved findings: The initial draft incorrectly treated released access-management fields as unreleased. The revision retains those reservations and removes only the proven-unreleased Project API key query field without reservation.
- Open questions: None
- Approval reference: Repository owner instruction that unreleased fields must not be reserved, 2026-11-09
