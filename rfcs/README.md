# Requests for Comments

This directory contains design proposals for changes that affect multiple services,
public contracts, persistent data, security boundaries, or operational behavior.

## Lifecycle

RFCs use the following statuses:

- `Draft`: under discussion; implementation must not begin.
- `Accepted`: final design bytes are ready for exact-digest confirmation; status alone does not authorize implementation.
- `Implemented`: the accepted design has shipped.
- `Superseded`: replaced by another RFC.
- `Rejected`: considered but not selected.

Material design changes discovered during implementation require the RFC to return to
`Draft` or be superseded by a follow-up RFC.

## Naming

RFC files use a four-digit sequence followed by a short descriptive name:

```text
0001-actor-aware-audit-identity.md
```

The coordinator reserves the next unused path before creating an Orca development Run.
Reservation is single-owner for that Run; a colliding or pre-existing path must be rejected.
The RFC author may edit only that reserved file. The RFC index is updated separately after
the RFC is accepted. Reservations use an atomic directory under Git metadata and remain
until the exact-digest confirmation task is created; abandoned reservations require
coordinator cleanup.

Start new RFCs from [`TEMPLATE.md`](TEMPLATE.md). The required metadata and sections make
scope, contracts, risks, verification, rollout, and decisions reviewable without inventing
a new structure for each proposal.

## Artifact Layout

Keep the RFC document at `rfcs/NNNN-short-name.md`. Store every JSON artifact associated
with that RFC under a matching directory using the full RFC stem:

```text
rfcs/
├── 0004-example-change.md
└── 0004-example-change/
    └── jsons/
        ├── plan.json
        ├── confirmation.json
        ├── amendment-01-plan.json
        ├── amendment-01-confirmation.json
        └── operational-readiness.json
```

This includes plans, amendments, challenge receipts, approval/confirmation receipts,
inventories, and operational-readiness records. Do not add new JSON artifacts directly
under `rfcs/`. The full RFC stem is required because RFC numbers are not currently unique.

Historical JSON artifacts moved into this layout retain their original embedded path values
so existing hashes and approvals remain auditable. New artifacts must record the current
`rfcs/<rfc-stem>/jsons/<name>.json` path and must not overwrite an existing receipt; use a
new amendment-specific filename instead.

An `Accepted` RFC is confirmed by exact SHA-256 through the Orca decision gate. Any byte
change after confirmation requires another challenge and confirmation before implementation.

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
