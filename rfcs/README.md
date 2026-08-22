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

An `Accepted` RFC is confirmed by exact SHA-256 through the Orca decision gate. Any byte
change after confirmation requires another challenge and confirmation before implementation.

## Index

| RFC | Title | Status |
| --- | --- | --- |
| [0001](0001-actor-aware-audit-identity.md) | Actor-Aware Audit Identity | Accepted |
| [0002](0002-jwt-only-service-auth.md) | JWT-Only Service Authentication | Accepted |
| [0003](0003-simplify-audit-backfill.md) | Simplify Audit Actor Backfill Migrations | Accepted |
