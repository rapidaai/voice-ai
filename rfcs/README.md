# Requests for Comments

This directory contains design proposals for changes that affect multiple services,
public contracts, persistent data, security boundaries, or operational behavior.

## Lifecycle

RFCs use the following statuses:

- `Draft`: under discussion; implementation must not begin.
- `Accepted`: the required reviewer or owner approved the design; implementation may begin.
- `Implemented`: the accepted design has shipped.
- `Superseded`: replaced by another RFC.
- `Rejected`: considered but not selected.

An accepted RFC is a stable decision record. After acceptance, change only its status or
reference links. A material decision change requires a new RFC whose `Supersedes` field
points to the previous RFC.

## Naming

RFC files use a four-digit sequence followed by a short descriptive name:

```text
0001-actor-aware-audit-identity.md
```

Choose the next unused path before drafting. A colliding or pre-existing path must be
rejected. The RFC author may edit only that selected file. Update the RFC index after the
RFC is accepted.

Start new RFCs from [`TEMPLATE.md`](TEMPLATE.md). The format records context, decision,
contracts and ownership, rejected alternatives, consequences and risks, rollout and
rollback, verification, and approval. Its risk table requires an impact statement or a
reasoned `N/A` for compatibility, failure recovery, security, data, concurrency, and
operations. Keep file-level scope, task breakdown, and execution evidence in the plan or
pull request.

The layered format is enforced for RFC 0017 and later. Earlier RFCs retain their accepted
structure and are not rewritten solely to match the current template.

## Approval Record

Keep each RFC as one Markdown decision record at `rfcs/NNNN-short-name.md`. Record the
challenger, outcome, approver, and approval reference in that file. Keep implementation,
verification, and code-review evidence in the task or pull request. Do not create RFC
sidecar receipts or approval digests.

A layered RFC cannot become `Accepted` while a risk-table row is blank, an open question
remains, or approval is unrecorded. The independent challenge checks both the decision and
the completeness of those fields.

Before acceptance, revise the draft to resolve challenge findings. After acceptance, do
not rewrite the decision. Create and approve a new RFC for a material change, set its
`Supersedes` field, then mark the old RFC `Superseded` with a link to its replacement.

## Index

| RFC | Title | Status |
| --- | --- | --- |
| [0001](0001-actor-aware-audit-identity.md) | Actor-Aware Audit Identity | Accepted |
| [0002](0002-jwt-only-service-auth.md) | JWT-Only Service Authentication | Accepted |
| [0002](0002-linux-ci-system-coverage.md) | Linux CI System Coverage | Accepted |
| [0003](0003-native-sip-assistant-phone-resolution.md) | Native SIP Party Identity Resolution | Accepted |
| [0003](0003-simplify-audit-backfill.md) | Simplify Audit Actor Backfill Migrations | Accepted |
| [0004](0004-consolidate-authentication-middleware.md) | Separate Authentication Middleware by Credential Class | Accepted |
| [0005](0005-refine-authentication-middleware-contracts.md) | Refine Authentication Middleware Contracts | Accepted |
| [0006](0006-remove-audit-backfill-id-validation.md) | Simplify Audit Actor Migrations | Accepted |
| [0022](0022-role-permission-persistence-migration.md) | Role Catalog Persistence | Accepted |
| [0026](0026-access-policy-v1.md) | Access Policy Version 1 | Superseded |
| [0027](0027-recovery-approver-separation.md) | Recovery Approver Separation | Superseded |
| [0028](0028-token-strict-subset-reduction.md) | Token Reduction and Recovery Separation | Accepted |
