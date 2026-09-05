# RFC 0015: User-Only Authentication for Organization Onboarding

- Status: Accepted
- Owner: authentication implementer
- Created: 2026-09-04
- Updated: 2026-09-04
- Reviewers: user challenger, independent code reviewer

## Summary

Allow a newly registered user with a valid user token, but no organization membership, to create the user's first organization. Opt in only the web service, keep the shared middleware default organization requirement, and keep `types.Authorize` unchanged for every existing endpoint.

## Context

The signup response can contain a valid user and token with no organization role. The web-api-local user principal currently requires organization context to report successful authentication. The shared user middleware also assumes organization context exists when it constructs request authentication.

The onboarding UI sends the valid token and user ID to `CreateOrganization`. Both REST and gRPC organization creation handlers call `types.Authorize`, which requires organization context for user authentication. The request therefore fails before organization creation can run.

Existing endpoints commonly use `types.Authorize` and `Authentication.Scope(AuthTypeUser)` as the organization-bearing user authorization boundary. Relaxing that shared contract would widen access across services. This RFC adds an opt-in middleware mode used only by `cmd/web` and a web-api-local authorization check used only by first-organization creation.

## Goals

- Accept valid user credentials in web-api when the user has no organization membership.
- Attach a request authentication value containing the verified user actor and user context, with no organization or project context.
- Permit only web-api organization creation to consume that user-only authentication.
- Preserve current behavior for users with an organization and for every existing organization-scoped endpoint.
- Preserve default middleware behavior for assistant-api, endpoint-api, and integration-api.
- Reject missing, malformed, mismatched, or non-user authentication.

## Non-Goals

- Changing signup, signin, token issuance, or token storage.
- Allowing multiple organization memberships or multiple organization creation.
- Relaxing organization or project authorization for any endpoint other than organization creation.
- Changing UI behavior, public request fields, protobuf schemas, database schemas, or generated files.

## Scope and Ownership

### Allowed Paths

- `api/web-api/internal/service/user/auth.type.go` - authentication implementer
- `api/web-api/internal/service/user/auth_type_test.go` - authentication implementer
- `pkg/middlewares/user_authentication.go` - authentication implementer
- `pkg/middlewares/authentication_grpc_middleware.go` - authentication implementer
- `pkg/middlewares/authentication_rpc_middleware.go` - authentication implementer
- `pkg/middlewares/authentication_middleware_test.go` - authentication implementer
- `cmd/web/web.go` - authentication implementer
- `cmd/web/authentication_middleware_test.go` - authentication implementer
- `api/web-api/api/organization.go` - authentication implementer
- `api/web-api/api/organization_test.go` - authentication implementer
- `rfcs/0015-user-only-onboarding-auth.md` - coordinator
- `rfcs/0015-user-only-onboarding-auth/jsons/` - coordinator and reviewers

### Out-of-Scope Paths

- `pkg/types/**`
- `ui/src/**`
- `protos/**`
- Database migrations and entity definitions
- Runtime authentication behavior in assistant-api, endpoint-api, and integration-api
- Organization, project, service, and system credential middleware

## Proposed Design

1. Update only the web-api-local `authPrinciple.IsAuthenticated` implementation so a valid user identity can represent successful credential authentication before organization membership exists.
2. Add an option to the shared user middleware that permits missing organization context. The default remains disabled, so every existing caller outside `cmd/web` preserves current behavior.
3. Enable the option only in `cmd/web` for Gin, unary gRPC, and stream gRPC user middleware.
4. When the option is enabled, construct `types.Authentication` from the verified user actor and user context. Populate organization and project contexts only when the principal provides them. Reject project context without organization context.
5. Keep `types.Authentication.IsAuthenticated`, `types.Authorize`, and `Authentication.Scope` unchanged. Existing endpoints therefore still reject user-only context.
6. Add a private helper in `api/web-api/api/organization.go` that reads request authentication and accepts only `AuthTypeUser` with a valid user actor whose ID matches `UserContext`.
7. Change only REST and gRPC `CreateOrganization` to use that private helper. The existing organization-present rejection remains in place.

The middleware remains the owner of credential verification and request authentication construction. `pkg/types` remains unchanged. Organization handlers remain the sole owner of the onboarding exception.

## Contracts and Compatibility

- No public request, response, protobuf, schema, configuration, or dependency contract changes.
- Existing full user authentication continues to pass `types.Authorize` unchanged.
- Shared user middleware remains organization-required unless its new option is explicitly enabled.
- Only `cmd/web` enables user-only request context construction.
- User-only authentication remains unusable by existing web-api endpoints because they continue to call `types.Authorize`.
- `CreateOrganization` accepts user-only authentication and continues to reject non-user authentication and users already attached to an organization.

## Failure and Recovery

- Missing or invalid credentials remain unauthenticated.
- A resolver result without a valid user identity is rejected by middleware.
- A user actor that does not match the user context is rejected by the organization onboarding helper.
- Project context without organization context is rejected.
- Existing transaction behavior for organization and owner-role creation is unchanged by this RFC.

## Security and Privacy

- The middleware exception is enabled only by `cmd/web`, and the authorization exception is limited to organization creation handlers.
- Tenant isolation is preserved because no existing organization-scoped handler changes its authorization entry point.
- No token, credential, or personal data is newly logged or persisted.
- The created organization owner role continues to use the authenticated user ID, not a request-supplied user ID.

## Observability

- Existing middleware rejection logs remain unchanged and contain no credential values.
- Existing organization creation error logs remain unchanged.
- Tests provide regression evidence for user-only context and unchanged default middleware behavior.

## Data and Migration

None.

## Rollout

Deploy the web API and shared middleware package together. Validate signup followed by first organization creation, then validate an existing organization member can still access an organization-scoped endpoint. Stop rollout if default middleware accepts user-only authentication or any endpoint other than organization creation accepts the partial context.

## Rollback

Revert the web principal, middleware option, `cmd/web` opt-in, and organization handler changes together. No data rollback is required. Organizations successfully created during rollout remain valid.

## Alternatives Considered

- Relax `Authentication.IsAuthenticated` for every user request. Rejected because many endpoints rely on `types.Authorize` as the organization-bearing authorization boundary.
- Change the shared serialized user principal. Rejected because that could affect assistant-api, endpoint-api, and integration-api.
- Skip authentication for organization creation. Rejected because it would allow unauthenticated organization creation and unaudited ownership.
- Add route-specific credential verification. Rejected because it would duplicate token verification and request context construction.
- Auto-create an organization during signup. Rejected because it changes onboarding product behavior and transaction ownership beyond the reported bug.

## Testing and Verification

- Test the local web-api user principal with and without organization context.
- Test default middleware rejection and opt-in middleware acceptance for a user-only principal across Gin, unary gRPC, and stream gRPC.
- Test that `types.Authorize` still rejects user-only context.
- Test the private organization onboarding authorization helper with matching, missing, mismatched, and non-user contexts.
- Test REST and gRPC organization creation with user-only authentication.
- Test organization creation still rejects a user with existing organization context.
- Run `gofmt` on changed Go files.
- Run `go test ./pkg/middlewares ./api/web-api/internal/service/user ./api/web-api/api ./cmd/web`.
- Run `go test ./pkg/... ./api/web-api/...`.
- Run `git diff --check`.
- Run `just agent-finalize "api/web-api/internal/service/user/auth.type.go,api/web-api/internal/service/user/auth_type_test.go,pkg/middlewares/user_authentication.go,pkg/middlewares/authentication_grpc_middleware.go,pkg/middlewares/authentication_rpc_middleware.go,pkg/middlewares/authentication_middleware_test.go,cmd/web/web.go,cmd/web/authentication_middleware_test.go,api/web-api/api/organization.go,api/web-api/api/organization_test.go,rfcs/0015-user-only-onboarding-auth.md,rfcs/0015-user-only-onboarding-auth/jsons/plan.json,rfcs/0015-user-only-onboarding-auth/jsons/challenge.json,rfcs/0015-user-only-onboarding-auth/jsons/confirmation.json"`.

## Acceptance Criteria

- [ ] A newly registered user with a valid token and user ID can create the first organization through REST and gRPC.
- [ ] Web service middleware attaches user actor and user context when organization membership is absent.
- [ ] Default user middleware still rejects missing organization context.
- [ ] `types.Authorize` still rejects that user-only context.
- [ ] Only the private organization onboarding helper accepts that user-only context.
- [ ] Existing users with organization membership retain current authentication behavior.
- [ ] Organization creation still rejects users who already have organization context.
- [ ] Invalid or mismatched user identity remains unauthenticated.
- [ ] Required tests and repository finalization pass.

## Open Questions

- Exact-digest confirmation is pending.

## Challenge Resolution

- The user challenged the first proposal because shared authentication semantics could affect other services.
- The plan was revised to keep `pkg/types` unchanged, preserve the default shared middleware behavior, opt in only `cmd/web`, and keep the user-only authorization helper private to organization creation.

## Artifact Index

- `jsons/plan.json` - revised implementation plan, accepted by challenger
- `jsons/challenge.json` - user challenge requiring minimal web-only impact, resolved
- `jsons/confirmation.json` - pending

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-09-04 | Use an explicit user-only authorization boundary for first organization creation | coordinator | `jsons/plan.json` |
| 2026-09-04 | Keep shared defaults and all `pkg/types` contracts unchanged | user challenger | `jsons/challenge.json` |
