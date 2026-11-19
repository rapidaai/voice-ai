# RFC 0050: Defer Console Permission Gating

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-10
- Supersedes: RFC 0049 console effective-permission gating only

## Context

RFC 0049 introduced effective permission projections and required the console to use them for
action visibility. The backend authorization and authentication alignment is not ready for console
consumption. Reading the incomplete projection now hides valid actions, including organization
invitation, and couples the current UI delivery to unfinished authorization rollout work.

## Decision

Defer all console action and navigation visibility checks based on effective permissions. The UI
continues to render its existing role, invitation, credential, and navigation controls. Backend
handlers remain the authoritative authorization boundary and return their existing permission
errors. Owner-role immutability remains enforced in the UI because it is a separate domain
constraint.

The permission projection stays in the public contract for later use. A future accepted RFC must
define the complete console permission mapping before the UI consumes it.

## Contracts and Ownership

- Backend handlers continue to own authorization decisions and permission error responses.
- The console owns interaction availability but does not interpret effective permissions in this
  release.
- Protobuf, SDK, database, role, and permission contracts are unchanged.
- RFC 0049 remains authoritative except for its console effective-permission visibility rule.

## Alternatives Rejected

- Keep partial console gating: rejected because incomplete projections hide valid operations and
  produce inconsistent navigation and action behavior.
- Add role-name or built-in-role fallbacks: rejected because that creates a second authorization
  policy and reintroduces legacy coupling.
- Remove effective permissions from the contract: rejected because the projection remains required
  for the later complete console authorization rollout.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Existing UI actions remain available. No API or persisted-data contract changes. |
| Failure and recovery | Unauthorized actions can reach the backend and receive its permission error. Existing global error handling remains responsible for user feedback. |
| Security and privacy | Controls may be visible to unauthorized users, but backend enforcement remains mandatory and authoritative. No access is granted by UI visibility. |
| Data and migration | N/A. No schema or data change. |
| Concurrency and resources | N/A. No resource lifetime or concurrent behavior changes. |
| Operations and observability | Backend authorization failures remain visible through existing request logging and error responses. |

## Rollout and Rollback

Deploy the UI without effective-permission visibility checks. Stop if any handler relies on hidden
controls instead of server authorization. Roll back by restoring the prior UI build. Reintroducing
console permission gating requires an accepted permission-to-control mapping and coordinated backend
projection readiness.

## Verification

- Invitation, role management, project assignment, user removal, credential revocation, and Roles
  navigation render without effective permission data.
- Owner mutation actions remain unavailable.
- Existing backend permission failures continue to surface through global error handling.
- Run focused Jest coverage for Users, Credentials, and Sidebar behavior.
- Run `yarn checkTs`, scoped ESLint, and
  `just agent-finalize "comma,separated,changed,paths"`.

## Review and Approval

- Challenger: Independent Codex reviewer
- Outcome: Approved
- Resolved findings: No Critical or Major findings. The deferral preserves backend enforcement,
  effective permission projections, and owner immutability while limiting the change to console
  visibility.
- Open questions: None
- Approval reference: Repository owner direction on 2026-11-10 to defer UI permission integration
