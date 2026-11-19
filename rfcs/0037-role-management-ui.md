# RFC 0037: Role Management UI

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-08
- Supersedes: RFC 0034

## Context

The accepted access-management API and React SDK expose the role and permission catalog, but the
workspace UI has no route for administrators to inspect built-in roles or manage custom roles. The
existing organization navigation contains Users and Projects, while its Security page is reserved for
later MFA work. Project access remains owned by the existing Credentials experience.

This RFC retains RFC 0034's permission-list pagination request, response, defaults, limits,
transactional count, and stable ordering. It replaces only the inherited hidden user-subject
permission compatibility policy.

## Decision

Add a `Roles & permissions` organization navigation item with three routes:

- `/organization/roles` lists the visible role catalog with server-backed search, scope, subject,
  type, and pagination filters. Catalog reads use latest-request-wins sequencing so a stale response
  cannot replace newer filter or page results. The sidebar item is shown to the legacy organization
  owner and admin roles that the current backend authorizes; direct navigation still relies on the
  backend and renders a denied state.
- `/organization/roles/new` creates a custom role from metadata, immutable scope and subject, and a
  permission selection constrained to that scope. The required immutable key must match
  `^[a-z][a-z0-9_]{2,63}$`. The selector requests pages of 100 until the response total is loaded and
  preserves selection while pages are combined. Changing scope or subject invalidates every in-flight
  selector request, clears the existing permission selection, and starts a latest-request-wins load
  for the new compatible catalog.
- `/organization/roles/:roleDefinitionId` shows role metadata, permissions, and standard audit
  fields. Built-in roles are read-only. Custom roles support metadata updates, explicit permission
  removal with a required reason, and archival with a required reason. Metadata update, permission
  removal, and archive are separate actions with independent form state and success or failure
  messaging. Each successful mutation reconciles by refetching the role before another action.

The UI consumes `@rapidaai/react` 1.1.91 through one access-management client module and uses the
existing user-auth metadata path. The backend remains the authority for authorization and role
validation. The UI displays transport and response-envelope failures without optimistic mutation.

Remove the backend's hidden organization-user and project-user permission allowlists. An active
permission is valid for a custom role when its scope matches the role scope. `PROJECT_TOKEN` roles
remain restricted to project scope. This makes `ListPermissions` the complete selectable catalog
without duplicating eligibility policy in the UI or adding another API field. Only the currently
authorized organization owner or admin can create custom roles, and role assignment remains an
explicit later operation.

Permission creation and editing are not exposed because permissions are system-defined. Permission
addition after custom-role creation is not exposed because the current API is reduction-only.

## Contracts and Ownership

- The accepted protobuf contract and backend remain the source of truth for role scope, subject,
  definition type, validation, authorization, and persistence.
- The access-management service owns the same-scope permission compatibility rule and removes its
  hidden user-subject allowlists.
- The React SDK owns generated request, response, enum, and service types.
- The UI access-management client owns request construction and connection binding.
- The organization role pages own filtering, forms, loading, empty, denied, failure, and success
  states.
- Existing Users, Projects, Credentials, Security, authentication state, and role-assignment flows
  remain unchanged.

## Alternatives Rejected

- Put roles inside Security: rejected because authorization policy and MFA are separate concerns.
- Add a standalone Permissions page: rejected because permission definitions are read-only building
  blocks and have no independent UI action.
- Use dialogs for role creation and editing: rejected because the permission catalog, audit context,
  and destructive actions need a full-page layout.
- Add a generic access-management form engine: rejected because three direct pages and one shared
  permission selector are sufficient.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The organization routes, sidebar item, client exports, and React SDK dependency update are additive. The unreleased custom-role API broadens user-role creation from hidden allowlists to every same-scope permission without changing protobuf fields. Existing routes and administration flows remain unchanged. |
| Failure and recovery | Failed reads render retryable page errors. Latest-request-wins sequencing prevents stale catalog results. Failed creates, metadata updates, permission removals, and archives retain only that action's current state. Successful detail mutations refetch the role; successful archive returns to the catalog. Rollback removes the routes, pages, client, and navigation item. |
| Security and privacy | Requests use existing user-auth metadata and the backend remains authoritative. Organization owners and admins may compose any same-scope permission into a custom role; no authority is granted until a later explicit assignment. Project-token roles remain project-scoped. The UI does not persist permissions, role forms, tokens, or new secrets. Built-in mutation controls are absent. |
| Data and migration | No schema migration is required. The broader same-scope rule can create custom USER roles containing permissions rejected by the prior allowlists. These role definitions and role-permission rows must be inventoried before any policy rollback. |
| Concurrency and resources | RPCs are unary. Pages cancel no server work, but ignore settled responses after unmount and do not create streams, timers, or background resources. Concurrent server changes reconcile on explicit refresh or navigation. |
| Operations and observability | Existing toast and page error patterns expose failures to the operator. No new client telemetry or backend logging is introduced. |

## Rollout and Rollback

Publish React SDK 1.1.91 and deploy the same-scope backend rule before the UI. Stop if the installed
SDK does not expose all seven access-management methods, backend compatibility tests fail, focused UI
tests fail, or the production UI build fails. Roll back by reverting the additive UI change,
restoring the prior SDK dependency, and retaining the compatible same-scope backend rule. If an
emergency requires restoring the prior allowlists, first inventory every active custom USER role with
a permission outside those allowlists, archive each affected role or reduce it to the prior allowed
set, and verify the inventory is empty before deploying the older service. Custom-role assignment is
not available in this version, so no active subject assignment can retain the broader authority.
There are no irreversible UI effects.

## Verification

- Route and sidebar tests prove the three routes and active navigation state.
- Service tests prove organization-user and project-user custom roles accept previously excluded
  same-scope permissions, reject cross-scope permissions, and keep project tokens project-scoped.
- Client tests prove request mapping, user-auth metadata, pagination, filters, and transport errors.
- Catalog tests cover success, loading, empty, retryable failure, filtering, pagination, and
  navigation to create and detail.
- Create tests cover valid creation, key format and duplicate-key failures, project-token scope
  restriction, multi-page permission loading and selection, scope-change selection clearing,
  reversed old-scope completion, backend failure, and success navigation.
- Detail tests cover built-in read-only behavior, custom metadata update, permission removal reason,
  archive reason, independent action failure state, audit values and missing-actor fallback, role
  reconciliation, and success navigation.
- Catalog tests reverse two request completions and prove only the latest response is rendered.
- Run focused Jest tests, `yarn checkTs`, `yarn build`, the stable browser route test when route
  fixtures support the new pages, and scoped repository finalization.
- Rollback verification inventories active custom USER roles with permissions outside the prior
  allowlists and requires zero rows before restoring the old service policy.

## Review and Approval

- Challenger: Codex (`react_sdk_final_review`)
- Outcome: Approved
- Resolved findings: Defined full permission pagination, latest-request-wins catalog reads, independent detail mutations, key validation, audit fallback coverage, current owner/admin navigation visibility, same-scope permission compatibility without hidden user allowlists, stale selector invalidation, and safe policy rollback.
- Open questions: None
- Approval reference: Owner direction in this thread limiting the first UI milestone to role catalog, create role, and role detail/edit while retaining project access in Credentials and deferring MFA.
