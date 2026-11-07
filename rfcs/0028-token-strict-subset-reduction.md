# RFC 0028: Token Reduction and Recovery Separation

- Status: Accepted
- Owner: Platform team
- Approved by: Repository owner
- Date: 2026-11-07
- Supersedes: RFC 0027

## Context

RFC 0027 adopts RFC 0026 and adds recovery approver separation. Independent persistence review
found conflicting incorporated rules for changing an assigned Project token's role: one permits
in-place permission reduction, while another requires replacement for every role replacement.

The repository owner authorized a new correction cycle in the `feature/access-management` delivery
task on 2026-11-07. This RFC resolves that contradiction and closes one recovery edge case found
during challenge. All other RFC 0026 subjects, permissions, roles, bundles, credential predicates,
audiences, locks, and assurance rules remain unchanged.

## Decision

RFC 0027 and its incorporated RFC 0026 policy are adopted with this replacement rule:

- an assigned Project token may change role assignments in place only when the resulting effective
  permission set is a proper subset of its current effective permission set;
- the token subject, secret, class, and Project remain unchanged;
- audience and binding must remain unchanged or become strictly narrower under the RFC 0026
  audience and binding order;
- the caller needs the matching `organization.project_access_tokens.reduce_access` or
  `project.access_tokens.reduce_access`, plus `project.access.manage` and every current effective
  token permission;
- the transaction follows the RFC 0026 lock order, locks and rereads every current and target role,
  the token subject, and assignments, then recomputes current and resulting effective permissions;
- an equal, incomparable, or larger resulting permission set is not a reduction and cannot change
  the existing token;
- every non-reduction role addition, removal, or replacement creates a new token subject under the
  create and Project ceilings, atomically revokes the old token and secret, and deactivates its
  assignments;
- a stale role, permission, audience, binding, or assignment snapshot fails without mutation.

An in-place strict-subset reduction preserves the existing secret because it can only remove
authority. Token replacement remains mandatory whenever a change could preserve different
authority, add authority, or make authority incomparable.

RFC 0027 three-party separation remains the rule for missing-admin recovery. Compromised-final-Admin
replacement has a fourth human identity and requires all four to be distinct: compromised Admin,
initiating Organization Owner, replacement target, and independent approver. The transaction locks
the RFC 0026 resources, rereads all four identities, and rejects any equality before changing an
assignment or writing an audit event. A compromised Admin cannot initiate or approve its own
replacement and cannot be the replacement target. When no valid distinct initiating Owner exists,
replacement fails closed; Organization Owner account recovery remains outside this policy.
The immutable replacement audit records the compromised Admin, initiating Owner, replacement target,
independent approver, system execution actor, incident identifier, reason, and correlation
identifier.

For compromised-final-Admin replacement, the platform-security approver may provide the independent
approval whenever no eligible distinct active Organization Owner remains after excluding the
compromised Admin, initiating Owner, and replacement target. This extends RFC 0026's single-Owner
break-glass condition only for this replacement case. The platform-security approver still must be
distinct from all three humans, use recent MFA, satisfy operator separation, and follow the same
incident, reason, immutable-audit, and transaction-lock requirements. If an eligible Owner
approver exists, that Owner path is required. When no eligible Owner approver remains, the
platform-security fallback is permitted and sufficient. Replacement fails closed only when the
applicable path cannot supply one qualified independent approver.

## Contracts and Ownership

- Authorization owns the proper-subset comparison over effective permission keys.
- Project-token lifecycle owns atomic in-place reduction and replacement.
- Role definitions and assignments are reread under the RFC 0026 lock order before comparison.
- RFC 0026 remains the incorporated catalog source. RFC 0027 is preserved for missing-admin
  recovery and superseded only for compromised-final-Admin replacement by the four-party rule here.
- Authorization owns four-party separation for compromised-final-Admin replacement and three-party
  separation for missing-admin recovery.
- RFC 0022 may proceed only after this RFC is Accepted and RFC 0027 is marked Superseded.

## Alternatives Rejected

- Replace the token for every permission decrease. Rejected because it adds secret rotation to a
  monotonic reduction without improving authority containment.
- Allow equal or incomparable role replacement in place. Rejected because a different role can
  change future behavior or exchange one authority for another without being a strict reduction.
- Compare role keys instead of effective permissions. Rejected because multiple assignments and
  reusable roles make role identity an invalid authority comparison.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | No runtime behavior changes until later token enforcement. Existing production tokens remain authoritative. |
| Failure and recovery | A stale or non-subset change rolls back without changing the token, secret, or assignments. |
| Security and privacy | Existing secrets survive only monotonic authority reduction. Any other role change replaces and revokes the token. Four-party separation prevents a compromised Admin from authorizing its own replacement. |
| Data and migration | The permission and role catalog is unchanged. RFC 0022 remains schema-compatible. |
| Concurrency and resources | Existing Organization, Project, role, token, and assignment lock ordering serializes comparison and mutation. |
| Operations and observability | Audit records old and resulting permission-key sets, actor, reason, and correlation ID without recording secrets. |

## Rollout and Rollback

Acceptance marks RFC 0027 Superseded and makes this RFC the Version 1 policy decision. RFC 0026
remains Superseded and remains the incorporated catalog source. RFC 0022 may then be challenged
against the unchanged catalog.

There is no runtime rollout in this RFC. Before token enforcement, rollback has no runtime step.
After enforcement, changing the strict-subset rule requires a new accepted superseding RFC and any
required data migration. Enforcement fails closed when the comparison cannot be completed.

## Verification

This RFC is acceptable when:

- an in-place role change commits only when resulting effective permissions are a proper subset;
- equal, incomparable, or larger permission sets require replacement;
- audience and binding remain unchanged or become narrower for an in-place reduction;
- the caller satisfies matching reduction permission, Project access management, and every current
  effective token permission;
- current and target roles, token state, and assignments are locked and reread before comparison;
- concurrent or stale changes fail without mutation;
- replacement atomically revokes the old token, secret, and assignments;
- missing-admin recovery retains three distinct humans, while compromised-final-Admin replacement
  requires four distinct humans including the compromised Admin;
- compromised-final-Admin replacement permits the platform-security approver whenever exclusions
  leave no eligible distinct active Owner approver;
- with exactly two Owners where one is the compromised Admin and the other initiates replacement, a
  qualified distinct platform-security approver can approve and the transaction can commit;
- any identity equality in a replacement transaction fails before assignment or audit mutation;
- the immutable replacement audit records all four humans plus the system execution actor;
- RFC 0026 catalog remains unchanged; RFC 0027 three-party separation remains unchanged for
  missing-admin recovery and is replaced by four-party separation only for compromised Admins;
- RFC layout, style, diff, and finalization checks pass.

```bash
just validate-rfc-layout
just agent-style-check
git diff --check -- rfcs/0028-token-strict-subset-reduction.md
just agent-finalize "rfcs/0028-token-strict-subset-reduction.md"
```

## Review and Approval

- Challenger: Codex `gpt-5.6-sol` and `gpt-6-astra` high-risk panel
- Outcome: Approved
- Resolved findings: Draft permits only proper-subset role changes in place, requires replacement
  for equal, incomparable, or larger authority, and requires four distinct humans for
  compromised-final-Admin replacement.
- Open questions: None
- Approval reference: `feature/access-management` delivery task, 2026-11-07
