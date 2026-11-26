# RFC 0027: Recovery Approver Separation

- Status: Superseded
- Owner: Platform team
- Approved by: Repository owner
- Date: 2026-11-07
- Supersedes: RFC 0026

## Context

Superseded by [RFC 0028](0028-token-strict-subset-reduction.md).

Accepted RFC 0026 defines dual-controlled Project Admin recovery and compromised-final-Admin
replacement. Independent review found that its second active Organization Owner must differ from
the initiating Owner, but is not explicitly required to differ from the recovery target. A target
who is also an Owner could therefore approve its own Project Admin grant.

The repository owner authorized correction of all review findings in the
`feature/access-management` delivery task on 2026-11-07. The permission catalog, roles, bundles,
credential predicates, audience rules, persistence design, and every other RFC 0026 decision remain
unchanged.

## Decision

RFC 0026 is adopted in full with this additional invariant for Project Admin recovery and
compromised-final-Admin replacement:

- the initiating Organization Owner, recovery target, and independent approver are three distinct
  authenticated humans;
- this separation applies whether the approver is another active Organization Owner or the
  platform-security break-glass approver;
- no initiator, target, approver, or system execution actor may substitute for another role in the
  same recovery transaction;
- after locking the Organization, Project, memberships, Owner assignments, and Project Admin
  assignments in the RFC 0026 order, the transaction revalidates all three identities and rejects
  equality before writing any assignment or audit event;
- separation failure rolls back the entire transaction and records no partial authority change;
- the immutable audit records all three human identities plus the system execution actor,
  incident identifier, reason, and correlation identifier.

The recovery target cannot approve its own grant. The initiating Owner cannot approve its own
request or be the target. The system execution actor performs the atomic write but grants no
authorization and cannot satisfy either human approval.

## Contracts and Ownership

- Authorization owns three-party identity separation and transaction-time revalidation.
- Authentication owns verified human identity and recent-MFA evidence for the initiator and
  independent approver.
- The Project Admin recovery workflow owns its locks, atomic assignment mutation, and immutable
  audit record.
- RFC 0026 remains the catalog source. This RFC changes no subject, permission key, role key,
  bundle, token rule, or persistence schema.
- RFC 0022 may proceed only after this RFC is Accepted and RFC 0026 is marked Superseded.

## Alternatives Rejected

- Allow a target Owner to approve its own Project Admin grant. Rejected because dual control would
  collapse into self-assignment.
- Count the system execution actor as the second approver. Rejected because execution identity does
  not provide independent human authorization.
- Remove Project Admin recovery. Rejected because active Projects still require a safe recovery
  path after external failure or final-Admin compromise.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | No runtime behavior changes until the later recovery implementation. Existing production authorization remains authoritative. |
| Failure and recovery | A recovery request fails atomically when any two required human identities are equal or change before commit. |
| Security and privacy | Three-party separation prevents target self-approval and initiator self-approval. Audit stores identity references, not factor secrets. |
| Data and migration | No catalog or production data changes. RFC 0022 remains schema-compatible. |
| Concurrency and resources | RFC 0026 lock ordering is unchanged; identity separation is revalidated while those locks are held. |
| Operations and observability | Recovery denials report a separation category and correlation identifier without exposing authentication evidence. |

## Rollout and Rollback

Acceptance atomically marks RFC 0026 Superseded and makes this RFC the Version 1 access-policy
decision. RFC 0022 may then be challenged against the unchanged catalog. Recovery remains disabled
until the separate authentication, MFA, assignment, and recovery implementation RFCs are accepted
and implemented.

Before recovery enforcement, rollback has no runtime step. After enforcement, weakening or removing
three-party separation requires a new accepted superseding RFC and migration if persisted state is
affected. Runtime must fail closed when separation cannot be verified.

## Verification

This RFC is acceptable when:

- the initiator, target, and independent approver must be three distinct authenticated humans;
- another active Owner and a platform-security approver follow the same separation rule;
- identity equality or a concurrent identity change rejects the transaction without an assignment;
- the system execution actor cannot satisfy a human approval;
- success records all three humans, the execution actor, incident, reason, and correlation ID;
- RFC 0026 catalog, roles, bundles, credential rules, audience rules, and locks remain unchanged;
- RFC layout, style, diff, and finalization checks pass.

```bash
just validate-rfc-layout
just agent-style-check
git diff --check -- rfcs/0027-recovery-approver-separation.md
just agent-finalize "rfcs/0027-recovery-approver-separation.md"
```

## Review and Approval

- Challenger: Codex `gpt-5.6-sol` and `gpt-6-astra` high-risk panel
- Outcome: Approved
- Resolved findings: Draft requires every recovery approver to differ from both initiator and target
  and revalidates that separation inside the locked transaction.
- Open questions: None
- Approval reference: `feature/access-management` delivery task, 2026-11-07
