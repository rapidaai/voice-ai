# RFC 0038: Role Catalog Interactions

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner via escalation resolution
- Date: 2026-11-08
- Supersedes: RFC 0037 full-page-create and dialog-rejection decisions only

## Context

RFC 0037 introduced the organization role catalog, role creation, and role detail and mutation
flows. The current catalog uses a separate filter strip, makes the role name the only row-level
entry point, and moves creation and detail into full-page experiences. This differs from the
Carbon listing pattern already used by the adjacent organization catalogs, makes available row
actions less explicit, and removes the catalog context while an administrator inspects or creates
a role.

This decision changes only the role-management interaction model. Existing access-management
requests, response envelopes, validation, permission compatibility, authorization, persistence,
SDK types, sidebar visibility, and role-assignment boundaries remain authoritative and unchanged.
A durable decision is required because the catalog, row actions, read-only overlay state, create
flow, existing mutation route, and dismissal behavior must remain consistent as inspection and
creation move into the catalog.

The frozen implementation plan for this decision has SHA-256
`04c502b6559da0ee149b623c5bfbffaa4b3310d34c806681530ec6a81919e3a7`. This RFC records its
durable contracts while leaving file-level execution detail in that plan.

## Decision

Keep `/organization/roles` as the role-management surface and align its structure with the
repository's Carbon organization listings. The catalog retains server-backed search, scope,
subject, type, pagination, refresh, latest-request-wins response handling, loading, empty, denied,
and retryable failure states. Search and filter controls move into the Carbon table toolbar, and
the result count, table, pagination, primary creation action, and responsive behavior follow the
same listing hierarchy as the adjacent organization catalogs.

Each row exposes exactly two inline Carbon icon-only buttons because the row has fewer than three
actions. Their accessible names and tooltips are role-qualified as `View <role name>` and
`Edit <role name>`; the actions are not hidden in an overflow menu. View is enabled for every
role. Edit is enabled for custom roles and remains present but disabled for built-in roles.

View opens the existing IBM Products SidePanel as a read-only catalog overlay without changing
the URL. The catalog owns the selected role ID and launcher reference, and closing restores focus
to the invoking View button. The panel loads the canonical complete record through the unchanged
GetRole client call rather than treating a list row as complete. It shows role name, description
with an explicit fallback, key, type, scope, subject, creator and updater identity, created and
updated timestamps, and every permission's name, key, and description with an explicit
missing-description fallback. Loading, retryable response-envelope and transport failure, denied,
empty-permission, and success states remain inside the panel. Switching roles or closing the panel
invalidates older detail requests so late completion cannot replace the current role or update
closed UI. The panel has no route, deep-link contract, or mutation control.

GetRole not-found for a stale, archived, or otherwise no-longer-visible role ID is a terminal,
non-retryable SidePanel state. It explains that the role is no longer available and provides Close
without Retry. It is never classified as a transient load failure because retrying the same ID
cannot restore visibility.

Edit navigates to the existing `/organization/roles/:roleDefinitionId` full-page RoleDetailPage.
That route and page keep their current behavior exactly, including built-in read-only behavior,
custom-role metadata update, permission removal with a required reason, archive with a required
reason, independent form and failure state, and authoritative refetch after successful metadata or
permission mutations. None of those mutations move into the SidePanel.

Both `/organization/roles` and `/organization/roles/new` render RoleCatalogPage. The pathname
comparison `location.pathname === '/organization/roles/new'` is the sole source of truth for create
modal visibility; there is no duplicate open boolean, query flag, or component-owned visibility
state. The page-header and empty-state Create role actions push `/organization/roles/new`. Modal
close and Cancel replace the current entry with `/organization/roles`, and browser Back closes the
modal by returning to the catalog pathname. Direct navigation or reload at
`/organization/roles/new` renders the catalog and reopens the modal.

Role creation uses the shared Carbon Modal, ModalHeader, scrolling ModalBody, ModalFooter, controls,
initial focus, focus containment, and focus restoration. It retains the existing key and required
name validation, immutable scope and subject rules, project-token forced project scope, complete
paginated permission loading, permission name, key, and description presentation, selection
invalidation when scope or subject changes, latest-request-wins request handling, duplicate-key
draft retention, response-envelope and transport failure display, and disabled Submit until the
complete permission catalog is loaded.

Close or Cancel before submission causes no mutation and resets the form on the next open without
an additional unsaved-changes confirmation. While submission is in flight, modal-provided close
controls, outside click, Escape, Cancel, and repeat Submit cannot dismiss the modal UI or start a
second request. This guarantee does not intercept browser Back, arbitrary route navigation,
refresh, or tab close. If navigation unmounts the workflow during an in-flight request, late
success or failure is ignored and cannot navigate, reopen UI, or update unmounted state. The server
may still have committed, so a later catalog load and existing duplicate-key response are the
reconciliation paths.

On successful create, the workflow replaces the current entry with `/organization/roles` and
carries the returned ID in one-use `location.state.createdRoleDefinitionId`. It independently
requests a catalog refresh. RoleCatalogPage consumes that location state into its selected view
role, clears the consumed history state with replace, and opens the read-only GetRole SidePanel
even if the catalog refresh fails or active filters exclude the new role.

Load the SidePanel styles once from the existing IBM Products dependency. No new feature flag,
generic form framework, client cache, access-management abstraction, component stylesheet, package,
or dependency is introduced.

## Contracts and Ownership

- The access-management backend remains the sole owner of role authorization, validation,
  persistence, role scope and subject rules, permission compatibility, and mutation outcomes. No
  backend behavior or public API contract changes.
- The generated SDK and existing UI access-management client retain their request, response, enum,
  transport, and authentication contracts. The catalog, panel, and modal reuse listAccessRoles,
  getAccessRole, listAccessPermissions, and createAccessRole unchanged. No protobuf, generated
  client, SDK, response envelope, authentication, package, or dependency changes.
- The role catalog owns query, filter, pagination, refresh, row actions, selected-role state, and
  the read-only SidePanel lifecycle and launcher focus. It consumes and clears the one-use created
  role ID from location state and keeps latest-request-wins ownership of overlapping catalog and
  selected-role reads. GetRole remains the source of truth for panel content; the panel owns the
  distinction between retryable failures and terminal not-found. The SidePanel has no route or
  mutation ownership.
- The existing full-page RoleDetailPage remains the sole UI owner of custom-role metadata update,
  permission removal, archive, independent mutation progress and failure state, and mutation
  reconciliation. Its route and behavior do not change.
- The `/organization/roles/new` pathname alone owns create-modal visibility. The creation workflow
  owns its form draft, permission catalog, selected permissions, validation, create submission,
  reset, modal-control guards, and late-result cleanup. It does not own or block browser and router
  navigation.
- The existing organization role routes remain compatible. The catalog route remains canonical,
  both the catalog and new-role URLs render RoleCatalogPage, and the role-detail route continues to
  open the existing full-page mutation surface. The read-only SidePanel is intentionally catalog
  state rather than a route. Existing sidebar, Users, Projects, Credentials, Security,
  authentication, role visibility, and role-assignment flows remain unchanged.
- Deterministic access-management browser fixtures own test-only ListRoles, GetRole,
  ListPermissions, and CreateCustomRole responses. They do not become a production source of truth
  or replace existing client mapping tests.
- The existing shared Carbon modal primitives and IBM Products dependency own overlay behavior and
  presentation. The UI style entry owns loading the existing SidePanel styles once.

## Alternatives Rejected

- Keep creation and read-only inspection as full pages: rejected because it removes catalog context
  for common catalog tasks. The existing full-page mutation surface remains appropriate for
  editing and is preserved.
- Put View and Edit in an overflow menu: rejected because Carbon guidance favors visible inline
  actions when a row has fewer than three actions, and hiding these two actions would make them
  harder to discover.
- Add a route or deep link for the read-only SidePanel: rejected because viewing is a lightweight
  catalog interaction and the existing detail route already has a stable, distinct responsibility
  as the full-page mutation surface.
- Duplicate create-modal visibility in component state or a query flag: rejected because pathname
  is sufficient for CTA, Back, direct reload, and close behavior and must remain the single source
  of truth.
- Edit role fields inline in table rows: rejected because permissions, audit data, reasoned
  destructive actions, and independent mutation failures do not fit safely in a compact row.
- Move editing into the SidePanel: rejected because it would duplicate or split mutation ownership
  with the existing RoleDetailPage and make a read-only catalog inspection surface responsible for
  complex independent mutations.
- Dismiss the creation modal on outside click or add a second unsaved-changes confirmation:
  rejected because outside dismissal can discard a complex draft accidentally, while a second
  prompt adds friction that the repository owner explicitly declined. Explicit idle-state close
  and Cancel actions reset the draft.
- Block browser navigation, refresh, or tab close while create is pending: rejected because the
  modal can govern its controls but cannot safely claim ownership of browser and arbitrary route
  navigation. Late-result cleanup and later catalog reconciliation provide the bounded failure
  behavior.
- Make panel opening depend on the refreshed catalog containing the created role: rejected because
  refresh may fail or active filters may exclude the new role. The returned ID and canonical
  GetRole call provide an independent one-use handoff.
- Treat GetRole not-found as a retryable panel failure: rejected because a stale, archived, or
  otherwise no-longer-visible role cannot be restored by retrying the same ID. A terminal state
  with Close and no Retry gives the accurate recovery action.
- Add backend capabilities or a new UI authorization policy for these interactions: rejected
  because the existing API and backend authorization already own every required read and mutation.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | `/organization/roles`, `/organization/roles/new`, and `/organization/roles/:roleDefinitionId` remain valid. The new-role URL now renders the catalog plus modal, preserving push, Back, direct navigation, and reload while changing its visual presentation. The detail route and RoleDetailPage behavior remain unchanged, and the SidePanel is intentionally not bookmarkable. Existing API, client, SDK, protobuf, response-envelope, authentication, authorization, sidebar, package, dependency, and mutation contracts remain unchanged. |
| Failure and recovery | Catalog and transient read-only panel failures remain local and retryable. GetRole not-found for a stale, archived, or otherwise no-longer-visible ID is terminal, offers Close without Retry, and is never presented as transient. Creation failures retain the draft. Modal controls cannot dismiss or repeat-submit while pending, but browser or route navigation may unmount while the server still commits. Late results are ignored, and a later catalog load plus existing duplicate-key behavior reconciles an ambiguous commit. Successful creation uses one-use returned-ID state to open GetRole independently of catalog refresh success or active filters. Existing RoleDetailPage recovery remains unchanged. |
| Security and privacy | Existing user-auth metadata and backend authorization remain authoritative. Direct deep links do not grant access, built-in roles never enable Edit, and no role data, permission selection, credential, or token gains new persistent client storage. The panel and modal display only data already returned to the authorized caller, and deterministic fixtures remain test-only. |
| Data and migration | N/A. This is a UI-only interaction change with no schema, persisted representation, migration, backfill, role eligibility, or role-assignment change. Existing successful create, update, permission reduction, and archive calls keep their current data effects. |
| Concurrency and resources | Catalog, selected-role, and permission reads use separate request sequences so stale results cannot replace newer state. Selecting or closing a view invalidates older GetRole reads, scope or subject changes invalidate older permission loads, and route or component unmount invalidates create completion. Submission is single-flight. Consumed created-role location state is cleared so Back or later navigation cannot reopen the panel. No stream, timer, worker, or background resource is added. |
| Operations and observability | Existing loading, empty, denied, retry, validation, and service-error states remain operator feedback; no runtime telemetry is added. Deterministic grpc-web fixtures and a focused role journey make catalog, route, modal, panel, create, and failure behavior observable in CI. IBM Products SidePanel styles must load from the existing dependency, route and journey coverage must be recorded, and every retained light and dark catalog, modal, and panel screenshot must be visually reviewed rather than accepted only because an update command generated it. |

## Rollout and Rollback

Deploy the Carbon-aligned catalog, inline row actions, read-only IBM Products SidePanel and style
entry, pathname-governed creation modal, deterministic access-management fixtures, focused role
journey, route and journey evidence, and reviewed screenshots as one UI release. The existing
detail route and RoleDetailPage deploy unchanged. No backend, client, API, SDK, protobuf,
authentication, authorization, sidebar, package, dependency, database, or service deployment
order is required. Access Management owns the rollout.

Stop promotion if focused role tests, route or journey coverage, type checking, scoped lint, the
production build, the focused browser journey, the complete browser gate, or accessibility checks
fail; if either inline action targets the wrong surface; if Back or direct reload violates the
pathname contract; if successful create depends on catalog refresh; if stale or archived GetRole
not-found exposes Retry; if a screenshot update command can touch an image outside the six allowed
role baselines; or if any light or dark catalog, modal, or panel screenshot has not been visually
reviewed and accepted.

Rollback restores the prior catalog row link and full-page element at the preserved new-role URL,
and removes the inline actions, read-only panel, modal integration, SidePanel style entry,
deterministic fixture and journey additions, and their screenshot baselines. The existing detail
route and RoleDetailPage require no rollback because this decision does not change them. All three
routes and backend contracts work before, during, and after rollback, so no compatibility window
or data repair is required. Roles created through the modal use the unchanged API and remain valid
after rollback. There are no irreversible effects and no feature-flag cleanup.

## Verification

- Catalog tests prove the Carbon listing hierarchy, server-backed toolbar search and filters,
  refresh, pagination, result count, latest-request-wins behavior, and loading, empty, denied, and
  retryable failure states.
- Row-action tests prove exactly two inline Carbon icon-only actions are used instead of an
  overflow menu; their names are exactly `View <role name>` and `Edit <role name>`; View is enabled
  for every role; Edit is enabled for custom roles and present but disabled for built-in roles;
  View opens the SidePanel without navigation and restores launcher focus on close; and Edit
  navigates to the existing detail route.
- Panel tests prove the canonical GetRole request; complete role and audit content; every
  permission's name, key, description, and fallback; empty permissions; loading; retryable
  response-envelope and transport failures; terminal stale or archived not-found copy with Close
  and no Retry; denial; retry for transient failures; catalog-owned selection; role switch; close
  behavior without a route; stale completion after switch or close; and absence of mutation calls
  and controls.
- Existing RoleDetailPage regression tests prove the route remains full-page and its built-in
  read-only behavior, custom metadata update, reasoned permission removal, reasoned archive,
  independent action failures, and post-mutation reconciliation remain unchanged.
- Creation tests prove both URLs render RoleCatalogPage; pathname is the only modal-visibility
  owner; CTA push; close, Cancel, and success replace; Back; direct new-route reload; initial and
  restored focus; complete permission paging and name, key, and description content; validation;
  project-token scope restriction; selection clearing and stale-response rejection; failed-create
  draft retention; guarded modal close controls, outside click, Escape, Cancel, and repeat Submit;
  browser or route unmount being allowed; late-result ignore; one-use
  `createdRoleDefinitionId` state and consumption; independent catalog refresh; and SidePanel open
  when refresh fails or filters exclude the new role.
- Request assertions prove the existing user-auth metadata, method selection, request fields, and
  response handling are unchanged. No backend, client, API, SDK, protobuf, authentication,
  authorization, sidebar, package, dependency, persistence, visibility, or assignment file changes.
- Deterministic grpc-web fixtures cover ListRoles, GetRole, ListPermissions, and CreateCustomRole.
  The focused role journey proves listing, actions, full panel details, edit routing, pathname and
  history behavior, terminal non-retryable GetRole not-found for a stale ID, create success handoff,
  refresh-failure independence, accessibility, and reconciliation. Route-manifest and
  journey-catalog evidence marks the catalog, direct create route, panel, edit, and create journeys
  covered and removes prior fixture-based deferrals.
- Capture and visually review light and dark screenshots for the populated role catalog, direct
  create modal, and opened details panel. Confirm action affordances, focus and scroll layout,
  permission descriptions, content hierarchy, contrast, clipping, and overlay stacking before
  retaining a baseline.
- Run the following exact commands:

```bash
cd ui && yarn test --watchAll=false --runInBand src/app/pages/workspace/role/index.test.tsx src/app/pages/workspace/role/create.test.tsx src/app/pages/workspace/role/role-details-side-panel.test.tsx src/app/pages/workspace/role/detail.test.tsx src/app/routes/__tests__/breadcrumb-metadata.test.tsx src/styles/__tests__/carbon-components.test.ts src/clients/__tests__/access-management.client.test.ts
cd ui && yarn checkTs
cd ui && yarn eslint src/app/pages/workspace/role src/app/pages/workspace/index.tsx src/app/routes/organization.tsx src/app/routes/__tests__/breadcrumb-metadata.test.tsx src/styles/__tests__/carbon-components.test.ts e2e/fixtures.js e2e/mocks/clients/access-management.js e2e/journeys/organization/roles.spec.js e2e/journey-catalog.js e2e/route-manifest.js
cd ui && yarn check:e2e-routes
cd ui && yarn build
cd ui && yarn playwright test e2e/journeys/organization/roles.spec.js --config=playwright.config.js
cd ui && yarn playwright test e2e/screenshot-journey.spec.js --config=playwright.config.js --grep "^organization\.roles(?:\.create)? matches screenshot journey in (?:light|dark) mode$" --update-snapshots
cd ui && yarn playwright test e2e/journeys/organization/roles.spec.js --config=playwright.config.js --grep "^organization\.roles details panel screenshot in (?:light|dark) mode$" --update-snapshots
just ui-browser-test
just agent-finalize "ui/src/app/pages/workspace/role/index.tsx,ui/src/app/pages/workspace/role/index.test.tsx,ui/src/app/pages/workspace/role/create.tsx,ui/src/app/pages/workspace/role/create.test.tsx,ui/src/app/pages/workspace/role/role-details-side-panel.tsx,ui/src/app/pages/workspace/role/role-details-side-panel.test.tsx,ui/src/app/pages/workspace/index.tsx,ui/src/app/routes/organization.tsx,ui/src/app/routes/__tests__/breadcrumb-metadata.test.tsx,ui/src/styles/carbon.scss,ui/src/styles/__tests__/carbon-components.test.ts,ui/e2e/mocks/clients/access-management.js,ui/e2e/fixtures.js,ui/e2e/journeys/organization/roles.spec.js,ui/e2e/journey-catalog.js,ui/e2e/route-manifest.js,ui/e2e/screenshots/organization/roles-light.png,ui/e2e/screenshots/organization/roles-dark.png,ui/e2e/screenshots/organization/roles-create-light.png,ui/e2e/screenshots/organization/roles-create-dark.png,ui/e2e/screenshots/organization/roles-details-panel-light.png,ui/e2e/screenshots/organization/roles-details-panel-dark.png"
just agent-style-check
```

The first title-filtered screenshot command may update only the light and dark catalog and create
images. The second may update only the light and dark details-panel images. Do not run an
unfiltered screenshot update command. After both commands, visually inspect and accept all six
images before treating screenshot verification as passed.

## Review and Approval

- Challenger: Round-2 independent Orca task `task_479419c62caa`
- Outcome: Approved
- Frozen plan: SHA-256 `04c502b6559da0ee149b623c5bfbffaa4b3310d34c806681530ec6a81919e3a7`
- Resolved findings: Replaced ambiguous action labels with exactly two role-qualified inline
  buttons; made canonical GetRole and complete permission content explicit; kept Edit and mutations
  on the existing detail page; made pathname the sole modal-visibility owner; defined push,
  replace, Back, and reload behavior; narrowed pending-request guards to modal controls; defined
  route-unmount cleanup and later reconciliation; made the created-role panel independent of
  refresh through one-use location state; and required deterministic fixtures, route and journey
  evidence, focused browser coverage, and reviewed light and dark screenshots. Round-2 task
  `task_479419c62caa` identified the final two major findings and reached the two-cycle limit. The
  owner escalation resolution then added terminal non-retryable GetRole not-found handling with
  unit and browser coverage and replaced the broad screenshot update with two title-filtered
  commands bounded to the six allowed role images.
- Open questions: None
- Approval reference: Repository-owner escalation resolution task `task_e610ab4bfeb2`, dispatch
  `ctx_1058d8395370`, approves this exact amended plan digest and RFC for implementation within the
  unchanged UI-only scope. Original scope message `msg_687df67f8b6c` and the escalation resolution
  prohibit backend, client, API, SDK, protobuf, authentication, authorization, sidebar, package,
  and dependency changes.
