# User Organization Selection Plan

## Change Contract

- Outcome: A User credential may explicitly select an active Organization independently of a
  Project, and a User with multiple Organization memberships can authenticate without an arbitrary
  tenant being chosen.
- Acceptance: field 4 and the optional transport header propagate end to end; active membership and
  Organization/Project consistency are verified; Project-only and sole-Organization behavior stay
  compatible; an unselected multi-Organization User receives a tenantless Principal and the full
  membership graph; malformed or unauthorized selections fail closed.
- Non-goals: role or permission semantics, aggregate shape, billing ownership, token format,
  service assertions, route policy, database schema, caching, and service-specific domain UI.
- Risk: accepting a caller-provided Organization without checking the same canonical assignment
  snapshot could cross tenant boundaries. Service tests must prove mismatched and inactive
  selections are rejected.

## Scope and Ownership

- Protobuf sources and generated Go, Python, JavaScript, and TypeScript artifacts own field 4.
- `pkg/types`, Go authentication middleware, the Document API middleware, and React SDK headers own
  the optional `x-organization-id` transport value.
- `pkg/authentication/resolver` and `pkg/clients/web` own selected-tenant propagation and remote
  response verification.
- Web authentication resolver/controller and User authentication service own canonical membership
  verification and tenant selection.
- Web session producers own the same multi-Organization default behavior; none may repeat the old
  single-Organization rejection.
- Web command CORS owns browser admission for the new header.
- UI authentication state owns explicit Organization selection and metadata; global navigation owns
  the Organization selector.
- The protected shell owns membership onboarding and keeps authenticated Users with memberships in
  the shell even before tenant selection or Project creation.
- Access Management and Workspace clients own their Organization-scoped metadata. Call sites pass
  `organizationId` from `useCredential` or `useCurrentCredential`; generic clients never import the
  authentication store.
- Focused package tests and descriptor tests own success, compatibility, and denial evidence.
- Existing dirty-worktree changes outside these paths are out of scope and must remain untouched.

## Implementation

1. Add failing descriptor, resolver, middleware, client, controller, and service tests for explicit
   Organization selection and multi-Organization behavior.
2. Add protobuf field 4 and regenerate artifacts from both authoritative source checkouts.
3. Propagate the optional Organization ID through transport credentials, resolvers, and the common
   Web resolver RPC.
4. Replace every single-Organization session and resolver rejection with deterministic tenant
   selection from the active Organization and Project assignments loaded for the request.
5. Add the optional header to Python and React SDK authentication boundaries, Web CORS, and UI
   request metadata.
6. Store and expose an explicit current Organization in the UI. Multi-Organization sessions must
   not select the first entry; the global navigation selector establishes the choice and constrains
   Project selection.
7. Separate verified membership presence from current tenant selection in the protected shell, and
   propagate Organization metadata through Workspace User listing and member deletion.
8. Run focused tests, race tests, generated-source parity, explicit finalization, and independent
   code review.

## Verification

```bash
cmp protos/artifacts/authentication.proto sdks/react/src/clients/protos/artifacts/authentication.proto
go test -count=1 ./pkg/authentication/... ./pkg/middlewares/authentication ./pkg/clients/web ./protos ./api/web-api/api ./api/web-api/internal/service/user
go test -race -count=1 ./pkg/authentication/... ./pkg/middlewares/authentication ./pkg/clients/web ./api/web-api/internal/service/user
python3 -m pytest api/document-api/tests/bridges/test_auth_bridge.py api/document-api/tests/middlewares/test_principal_authentication_middleware.py
(cd sdks/react && npm test -- --runInBand)
(cd ui && yarn test --watchAll=false --runInBand)
(cd ui && yarn checkTs)
just ui-browser-test
git diff --check
just agent-finalize "$(paste -sd, plans/0085-finalize-paths.txt)"
```

## Approval

- RFC: `rfcs/0085-user-organization-selection.md`
- Challenger: Independent Codex plan and contract review
- Owner approval: Repository owner explicitly selected UI-first propagation on 2026-11-14.
- Decision: Approved. No concrete scope or coverage blocker remains.
