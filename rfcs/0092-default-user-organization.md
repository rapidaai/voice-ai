# RFC 0092: Default User Organization

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-26
- Supersedes: RFC 0085 UI fallback decision only

## Context

RFC 0085 permits a User whose verified active memberships span multiple Organizations to receive a
tenantless Principal and the complete membership graph. It also requires the UI to leave that User
without a selected Organization until the User makes an explicit choice. The result is a successful
sign-in that opens the authenticated shell but initially leaves Project-backed content unavailable,
even when the first verified Organization contains a usable Project.

The backend membership graph remains the authorization source of truth. A UI selection only proposes
request context; the backend must continue to verify every selected Organization and Project against
active membership and reject an invalid or mismatched pair. This RFC changes only the UI fallback
chosen from that verified graph. It does not change tenant inference by the backend, the authentication
wire contract, membership ordering, role or permission derivation, or the explicit context selector.

## Decision

When the UI receives a complete Authentication with a tenantless Principal, it selects the first
Organization in the returned verified membership graph. It selects that Organization's first Project
when one is present. If the first Organization has no Project, the Organization is selected and the
Project remains unselected.

The existing precedence remains intact: a valid Principal-selected context wins, followed by any
still-valid prior UI selection regardless of whether it came from an automatic fallback or an
explicit switch. The new first-entry fallback applies only when neither source provides a valid
current context. An empty membership graph remains unselected. Project selection is always
constrained to the selected Organization.

The Organization and Project selector remains available after the fallback. A User may explicitly
switch to any Organization and Project present in the verified graph. Subsequent requests carry the
selected identifiers, and the backend remains responsible for active-membership validation and
Organization/Project binding before granting tenant-scoped authority.

The first entry means the first member of the backend's canonical verified order, currently the
lowest verified numeric ID. The UI does not add a second sorting policy. A still-valid prior UI
selection remains preferred if canonical membership order later changes.

This supersedes only RFC 0085's decision that the UI must not fall back to the first of multiple
Organizations. Every backend selection, validation, compatibility, and ownership decision in RFC 0085
remains in force.

## Contracts and Ownership

- UI authentication state owns the current Organization and Project selection. It applies Principal,
  prior-selection, and first-verified-entry precedence from the Authentication snapshot.
- Global navigation owns explicit context switching and continues to constrain Project choices to the
  selected Organization.
- The Authentication aggregate remains the sole source of verified memberships and permission
  derivation. UI fallback state does not add or modify membership, role, or authority data.
- Backend User credential resolution remains the enforcement boundary for selected Organization and
  Project identifiers. It verifies active membership and pair consistency on every request and fails
  closed for unknown, inactive, or mismatched values.
- The authentication protobuf, request headers, session model, and public API remain compatible and
  unchanged. The backend may continue to return a tenantless Principal for multiple Organizations.
- The existing multi-Organization sign-in browser flow owns end-to-end evidence for initial fallback,
  Project-backed loading, and explicit switching. Its light and dark screenshots remain required
  reviewed UI evidence.

## Alternatives Rejected

- Keep multiple Organizations unselected until an explicit choice: rejected because every tenantless
  multi-Organization sign-in begins in a reduced shell despite already verified usable membership.
- Make the backend select its first Organization: rejected because response ordering is a presentation
  concern, and this would expand the authentication contract beyond the UI fallback being corrected.
- Remove explicit switching after choosing a default: rejected because the default is convenience,
  not user intent, and Users must retain access to every verified context.
- Sort Organizations or Projects before choosing: rejected because it invents an ordering policy and
  broader compatibility contract with no current requirement.
- Trust UI-selected identifiers without backend verification: rejected because client state is not an
  authorization boundary and could permit cross-tenant access.

## Consequences and Risks

The first usable dashboard can load immediately after a tenantless multi-Organization sign-in, while
explicit switching remains available. The tradeoff is that response order influences the initial
context for a new session, so the first visible tenant may change when membership order changes.

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Existing authentication and transport contracts are unchanged. UI behavior changes from no initial tenant to the first verified Organization and its first Project when available. Valid Principal context and any still-valid prior UI selection retain precedence. |
| Failure and recovery | An empty graph stays unselected, and an Organization without Projects yields Organization-only context. Unknown, inactive, or mismatched selections continue to fail closed at the backend. Reverting stops new automatic fallback and restores the RFC 0085 unselected experience only for sessions without stored IDs. Valid persisted IDs remain selected until cleared or stale, regardless of whether they came from automatic fallback or an explicit switch. |
| Security and privacy | The UI can select only entries disclosed in the verified Authentication graph, but client state is still untrusted. Backend active-membership and Organization/Project binding checks remain mandatory, so the fallback grants no authority by itself and discloses no additional membership data. |
| Data and migration | No database, schema, token, persisted store shape, or server-side session migration is required. The existing store persists selected IDs without provenance, so an automatically chosen fallback may persist and remain preferred on refresh while it still resolves inside the verified aggregate. No cleanup or migration is needed. |
| Concurrency and resources | N/A. The selection is synchronous UI state derived from one Authentication snapshot and adds no goroutine, cache, connection, timer, or retained resource. |
| Operations and observability | No new backend metric or log is required because authorization behavior is unchanged. Browser-flow failures and screenshot differences expose incorrect default selection, missing Project loading, or broken explicit switching before rollout. |

## Rollout and Rollback

Deploy the UI fallback only after the RFC 0085 backend membership validation is present. The UI and
its focused state tests, multi-Organization browser flow, and reviewed light and dark screenshots ship
as one candidate. There is no mixed-version server constraint because the wire and backend contracts
do not change.

Stop rollout if the UI selects an entry absent from the Authentication graph, pairs a Project with the
wrong Organization, fails to load the first Project-backed view, prevents explicit switching, or if
the backend accepts an inactive or mismatched selection. Also stop for an unexplained screenshot
difference or an unresolved Critical or Major review finding.

Rollback reverts the UI fallback and its browser expectations, which stops new automatic selection
and restores the RFC 0085 unselected behavior for sessions without stored IDs. Valid automatically or
explicitly persisted IDs remain selected until browser storage is cleared or the IDs become stale.
The backend validation and tenantless multi-Organization response remain deployed. No migration,
data conversion, or persisted-ID cleanup is required.

## Verification

Acceptance requires:

- focused UI state coverage proving a tenantless multi-Organization Authentication selects the first
  verified Organization and its first Project;
- edge coverage proving an empty graph stays unselected, an Organization without Projects stays at
  Organization scope, valid Principal and any still-valid prior UI selection retain precedence, and
  invalid cross-Organization Project pairs are rejected by the UI state boundary;
- the existing multi-Organization sign-in browser flow proving the default Project-backed view loads,
  its dashboard requests carry Organization `202` and Project `302`, the selector still exposes both
  verified Organizations, and an explicit switch carries Organization `203` and Project `303`;
- reviewed light and dark screenshots at the existing 1024 by 900 viewport for the `defaulted`,
  `selector`, and `switched` states;
- retained backend coverage from RFC 0085 for active membership and Organization/Project binding;
- type checking, route-catalog validation, full UI verification, and the stable browser contract;
- scoped repository finalization, independent Governed review with no unresolved Critical or Major
  finding, and PR readiness against `feature/access-management` before delivery.

```bash
# Retain the expected pre-production failure after changing the focused regression test.
(cd ui && CI=true yarn test --watch=false --runInBand src/stores/auth/__tests__/authentication.store.test.ts)

# Focused state and retained selector behavior after implementation.
(cd ui && CI=true yarn test --watch=false --runInBand src/stores/auth/__tests__/authentication.store.test.ts src/app/components/layout/navigation/__tests__/actionable-header.test.tsx)

# Journey registration and the focused light/dark browser flow, including request metadata.
(cd ui && yarn check:e2e-routes)
(cd ui && CI=true yarn e2e e2e/journeys/auth/authentication.spec.js --grep "multi-organization sign-in" --config=playwright.config.js)

# Existing backend membership and receiving-resolver checks, with no backend edits.
go test -count=1 ./api/web-api/internal/service/user ./pkg/authentication/resolver

# Full non-browser UI and stable browser gates.
(cd ui && yarn checkTs)
(cd ui && yarn verify:ui)
just ui-browser-test

# Repository checks over the exact writable implementation set.
git diff --check
just agent-style-check
just agent-finalize "ui/src/stores/auth/authentication.store.ts,ui/src/stores/auth/__tests__/authentication.store.test.ts,ui/e2e/journey-catalog.js,ui/e2e/journeys/auth/authentication.spec.js,ui/e2e/mocks/clients/authentication.js,ui/e2e/mocks/clients/dashboard.js,ui/e2e/screenshots/auth/multi-organization-signin/defaulted/lg-light.png,ui/e2e/screenshots/auth/multi-organization-signin/defaulted/lg-dark.png,ui/e2e/screenshots/auth/multi-organization-signin/selector/lg-light.png,ui/e2e/screenshots/auth/multi-organization-signin/selector/lg-dark.png,ui/e2e/screenshots/auth/multi-organization-signin/switched/lg-light.png,ui/e2e/screenshots/auth/multi-organization-signin/switched/lg-dark.png"

# Run after the scoped commit exists against the confirmed PR base.
just agent-pr-ready feature/access-management
```

All six light and dark images must be inspected after generation. The defaulted state must show Acme
Corporation and Customer Support with loaded dashboard data, the selector must show both Organizations
and the current marker, and the switched state must show Northwind Group and Sales Operations without
new clipping, contrast, or focus regressions.

## Review and Approval

- Challenger: Codex plan challenger `term_66f4b4e4372c96a31331f223d3a888f4`
- Outcome: Approved
- Resolved findings: After the initial `REVISE` verdict, defined provenance-independent precedence
  as valid Principal context, any still-valid prior UI selection, then the first verified
  Organization. Recorded persisted automatic fallback behavior, aligned verification with the full
  metadata, backend, UI, browser, screenshot, finalization, and PR-readiness scope, and clarified
  that rollback stops new fallback while valid stored IDs remain selected until cleared or stale.
- Open questions: None
- Approval reference: Independent challenge receipt `run_id run_3a61ab8dd5ec`, `task_id
  task_8e3853126797`, decision `approve`, RFC `rfcs/0092-default-user-organization.md`, reviewed Draft
  SHA-256 `9b8892ff8214c1b336872a5d9ac50b2cd5c337f8957882bc3459f665ba102a29`, reviewed at
  `2026-11-26T12:54:11.367Z`. Repository-owner approval is the user's 2026-11-26 instruction,
  `perfect do this and commit and push`, followed by `create PR to feature/acces-management`; the
  confirmed remote base is `feature/access-management`.
