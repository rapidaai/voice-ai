# RFC 0091: Project Credential Management Access

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-24
- Supersedes: RFC 0053 only for the CreateProjectCredential disablement and GetAllProjectCredential secret-returning-route decision

## Context

`ProjectService/CreateProjectCredential` and `ProjectService/GetAllProjectCredential` are
registered, implemented unary RPCs, but the embedded Web API authorization policy marks both
routes disabled. Policy therefore returns gRPC `Unavailable` before either controller runs.

The existing handlers and Project service already require an authenticated Principal whose
selected Project equals the request Project. Persistence repeats the Organization and Project
constraints. Creation validates the Project and selected active Project-token role, creates the
credential transactionally, records the caller as audit actor, and returns the new credential key.
Listing returns active credentials for the selected Organization and Project, including every
stored key, with the existing filters, pagination, and ordering.

User authentication already constructs a selected-Project Principal whose authorities contain the
effective permissions from both the selected Organization and selected Project. The route-policy
schema can express logical OR as complete `any_of` alternatives. No new resolver, permission,
authorizer mode, or controller selection is required.

RFC 0053 requires recent MFA, lifecycle-specific create authority, `project.access.manage`, a
selected-role permission ceiling, and locked rereads before project-credential creation can be
enabled. It also keeps listing disabled because the response contains reusable secrets and there is
no distinct secret-read permission. This RFC deliberately replaces those decisions for these two
routes. It accepts broad access-management authority as sufficient and preserves plaintext key
responses. That security decision requires independent challenge and explicit repository-owner
approval before acceptance.

This is a Governed authorization change. It changes admission and secret access, but does not
change the protobuf, request, response, persistence, client, or UI contracts.

## Decision

Enable `ProjectService/CreateProjectCredential` and
`ProjectService/GetAllProjectCredential` through one policy decision per route. Each decision is a
logical OR of exactly these complete alternatives:

1. a User Principal with an exact selected Project and `organization.access.manage`; or
2. a User Principal with the same selected-Project requirement and `project.access.manage`.

Both alternatives use Project context. Possessing the Organization permission does not admit an
Organization-only Principal and does not allow the caller to name an arbitrary Project. The
request Project must still equal the Principal's selected Project, and persistence must still
constrain both Organization and Project.

An effective selected-Project User with either permission, or both permissions, is admitted. This
includes a service-delegated User because the existing route authorizer evaluates the effective
subject and this policy adds no immediate-caller restriction. A User without either permission, a
User without a selected Project, a Project credential, an Organization credential, a
non-delegated Service, a service-delegated non-User subject, and a missing Principal are not
admitted. No Project credential audience or direct Service actor alternative is introduced.

This decision preserves all controller and service behavior after admission. In particular:

- request messages, field meanings, validation order, filters, pagination, ordering, and error
  envelopes do not change;
- response messages and field meanings do not change;
- successful create continues to return the newly created plaintext key;
- successful list continues to return the plaintext key of every matching active credential;
- creation keeps its existing active-Project and active-role checks, transaction, audit actor,
  cancellation, deadline, and failure behavior;
- listing keeps its existing tenant predicates, active-status predicate, filtering, pagination,
  ordering, cancellation, deadline, and failure behavior; and
- no database, generated artifact, SDK, UI, resolver, middleware, authorizer-engine, controller, or
  service contract changes.

The intentional compatibility change is authorization admission. Eligible requests no longer
receive policy `Unavailable`; they receive the existing success or controller-owned failure.
Authenticated principals that do not match either alternative receive policy `PermissionDenied`.
Missing or invalid authentication retains the existing `Unauthenticated` behavior.

This RFC explicitly accepts the following consequences in place of RFC 0053's prerequisites:

- neither route requires recent MFA;
- neither route requires a lifecycle-specific project-access-token permission;
- create does not require the caller to possess every permission in the selected token role;
- create adds no locked reread beyond the existing transaction; and
- list uses either broad access-management permission as authority to read reusable secrets.

## Contracts and Ownership

- The embedded Web API policy remains the sole owner of static route admission. It owns the exact
  two User, Project-context alternatives for both routes.
- User authentication remains the sole owner of credential verification, selected Organization
  and Project construction, and the effective authority set. This decision does not copy or
  recalculate permissions elsewhere.
- Service delegation remains the owner of the effective subject. A verified service-delegated
  User is intentionally evaluated as a User; this decision adds no `caller_services` restriction.
- Policy middleware remains the owner of route lookup, actor, context, and permission evaluation.
  It must deny before controller invocation when neither alternative matches.
- The Project controllers remain the owners of typed request validation, request-to-selected-
  Project comparison, transport error mapping, and response construction.
- The Project service remains the owner of Organization and Project predicates, active Project and
  role selection, creation transaction, list query, ordering, persistence, and audit actor fields.
- Access Management owns the authorization decision and diagnosis. Web API operations owns staged
  deployment and binary rollback. Security owns incident classification, credential containment,
  and exposure response.
- Public protobuf descriptors, generated artifacts, React SDK signatures, and UI behavior remain
  compatible and unchanged. The intentional public behavior change is that authorized calls can
  now reach the existing secret-returning handlers.
- RFC 0053 remains authoritative for every other route and decision. This RFC does not weaken or
  replace its provider-credential, OAuth, role-reduction, service-assertion, or other authorization
  requirements.

## Alternatives Rejected

- Keep both routes disabled under RFC 0053: rejected because it does not meet the requested access
  rule, although it has the smallest secret-exposure surface.
- Require recent MFA, lifecycle-specific permissions, a selected-role permission ceiling, and
  additional locked rereads: rejected for this decision because those safeguards change the
  requested authorization and behavior contract. Their absence is an explicitly accepted risk,
  not an implementation omission.
- Redact list keys or return a create key only once: rejected because either choice changes the
  existing response behavior.
- Require both `organization.access.manage` and `project.access.manage`: rejected because the
  requirement is logical OR. A single permission list would also express AND in the current policy
  schema.
- Admit an Organization-context User and let the request select the Project: rejected because it
  weakens the selected-Project tenant boundary and conflicts with the existing controller and
  service contract.
- Add permission checks in both controllers: rejected because policy already owns static route
  admission and duplicate checks would create competing sources of truth.
- Add a feature flag or mutable runtime policy: rejected because the embedded reviewed policy and
  deployed binary are the existing authorization source and rollback mechanism.
- Add UI-only access gating: rejected because presentation is not authorization and the clients
  require no wire-contract change.

## Consequences and Risks

The benefit is that authorized access managers can use the existing Project credential workflows
without new protocol or persistence work. The cost is deliberate authority broadening for secret
creation and disclosure, with less assurance and a wider permission surface than RFC 0053.

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Public request and response shapes remain unchanged. Eligible callers change from policy `Unavailable` to existing success or handler failure. Authenticated ineligible callers receive `PermissionDenied`; missing or invalid authentication remains `Unauthenticated`. |
| Failure and recovery | Policy denial invokes no controller. Existing create remains transactional, so a failed transaction returns no created key. A successful credential is durable and binary rollback does not remove it. Revoke is required to contain credentials created during an unsafe window. |
| Security and privacy | Either broad access-management permission grants creation and plaintext read of reusable Project keys. A service-delegated effective User with either permission is also admitted. No immediate-caller restriction, recent MFA, lifecycle-specific secret permission, selected-role permission ceiling, or list access audit event is added. Exact selected-Project and Organization predicates remain mandatory, and secrets must not enter logs, metrics, traces, or errors. |
| Data and migration | No schema or data migration occurs. Existing credential rows and keys are unchanged. New credentials created while enabled remain after rollback until explicitly revoked. |
| Concurrency and resources | No new goroutine, channel, cache, timer, connection, or resource owner is introduced. Existing database transaction, cancellation, deadline, lock, query, and cleanup behavior remains unchanged. |
| Operations and observability | Existing request and policy-denial logs provide route, status, duration, and error outcomes. Creation retains nonsecret audit actor data. Listing has no dedicated safe access audit event, so retrospective key-disclosure scope can be incomplete and incident containment may require broad revocation. |

## Rollout and Rollback

Implementation and deployment must not begin until this RFC has an approved independent challenge,
explicit repository-owner approval, and `Accepted` status.

Each Web API instance deploys its binary and embedded policy as one reviewed unit. No schema
migration or client deployment is required. Before production, run the full authorization matrix
and create and list regression suites. In nonproduction, exercise both direct and delegated User
admission for both permission alternatives, create a dedicated smoke credential, verify create and
list preserve its key, revoke it, and verify it is no longer listed. Confirm that no key appears in
logs before production begins.

Production uses the existing rolling deployment. During that window old instances return policy
`Unavailable` while new instances apply this RFC. No sticky routing is assumed, so an eligible
caller may receive transient `Unavailable` until every instance is updated. Old instances never
invoke the controller, so they cannot create or disclose a credential. Do not create or list a
credential solely as a production smoke test because the list operation has no Project-attributed
access audit event.

Stop the rollout on any confirmed cross-Project or cross-Organization access, any admission of a
non-User or unprivileged User, any credential secret in telemetry, any unexpected response-contract
change, or any `Internal` result in the focused nonproduction smoke. An unresolved Critical or
Major review finding also stops rollout. Existing route-level request logs are evidence for route,
status, duration, and deployment window only; this RFC does not claim a Project-scoped error-rate
or containment metric.

Rollback deploys the prior reviewed Web API artifact, which restores both routes to disabled and
requires no schema reversal. Rollback cannot retract keys already returned and does not remove
credentials created while enabled. Operations must identify credentials created during the
enabled window and Security must revoke any credential whose creation or exposure is suspect.
Because list has no dedicated Project-attributed access audit event, any suspected unauthorized
list admission during the deployment window requires deployment-wide containment: Security must
revoke every active Project credential in the deployment's database scope before declaring
containment complete. Do not infer a narrower Project set from logs that do not record it.

There are no irreversible schema effects. Secret disclosure and external copying are irreversible;
credential revocation and replacement are the only containment path.

## Verification

Acceptance criteria are:

- each named route has exactly two nonduplicate User, Project-context policy alternatives, one
  requiring only `organization.access.manage` and one requiring only `project.access.manage`;
- on both routes, direct and service-delegated selected-Project effective Users with either
  permission or both permissions are admitted;
- on both routes, an unprivileged User, Organization-only User, Project credential, Organization
  credential, non-delegated Service, service-delegated non-User subject, and missing Principal are
  denied before controller invocation;
- request Project mismatch and Organization or Project persistence mismatch cannot disclose or
  create a credential;
- existing create and list tests retain their validation, error, transaction, audit, pagination,
  filtering, ordering, cancellation, and deadline behavior, and list adds an explicit assertion
  that its response retains the stored plaintext key;
- route inventory remains exact and unrelated policies remain unchanged; and
- the authoritative Web API permission documentation states the broad OR rule and the preserved
  plaintext key behavior.

Run these exact commands against the implementation candidate:

```text
go test ./pkg/authentication/policy -run '^Test(NewWebAuthorizationPolicyRouteAlternatives|WebProjectCredentialAuthorizationMatrix)$'
go test ./pkg/authentication/authorizer -run '^TestRouteAuthorizerEvaluatesRouteAlternatives$'
go test ./pkg/middlewares/authentication -run '^(TestPolicyGRPCAuthentication|TestGRPCAuthenticationRunsBeforePolicy)$'
go test ./api/web-api/api -run '^(TestCreateProjectCredential|TestGetAllProjectCredential)'
go test ./api/web-api/internal/service/project -run '^TestProjectCredential'
go test ./api/web-api/router ./cmd/web
go test ./pkg/authentication/... ./pkg/middlewares/authentication ./api/web-api/api ./api/web-api/internal/service/project ./api/web-api/router ./cmd/web
git diff --check
just agent-style-check
just agent-finalize "pkg/authentication/policy/v1/web-api.yaml,pkg/authentication/policy/service_policy_test.go,api/web-api/api/project_credential_test.go,api/web-api/README.md"
bin/agent-review --base REVIEW_BASE_COMMIT --risk high --allow-external
```

The high-risk review runs only against a clean, committed candidate range whose base commit is
recorded in place of `REVIEW_BASE_COMMIT` and whose worktree has no tracked or untracked changes. The
`--allow-external` flag enables the configured two-family panel but does not provide user consent.
Run it only after `bin/agent-egress` validation and separate explicit user authorization for that
external egress. Passing tests and review do not substitute for the challenge and owner-approval
gates.

## Review and Approval

- Challenger: Orca task `task_5e2efb72a012`, dispatch `ctx_6026325c4257`
- Outcome: Approved
- Resolved findings: After the repository-owner-directed post-escalation challenge, the governed artifact path matches this RFC and requires no nonexistent JSON sidecars. Service-delegated effective User admission is explicit and covered by the matrix. The implementation scope permits an exact list plaintext-key regression assertion. Rollout defines mixed-version routing, nonproduction smoke cleanup, supported route-level evidence, and deployment-wide containment when Project attribution is unavailable. The exact finalization command includes the API test, and high-risk review enables the configured external panel only against a clean committed range with separate egress consent. The challenger reported no Critical, Major, or Minor findings.
- Open questions: None
- Approval reference: Repository owner direction in the current coordination session on 2026-11-24 to keep the broad permission contract simple, then challenge and unblock the corrected RFC
