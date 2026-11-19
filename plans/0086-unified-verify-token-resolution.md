# Unified VerifyToken Credential Resolution Plan

## Change Contract

- Outcome: One internal `AuthenticationService.VerifyToken` RPC resolves User, Project, or
  Organization credentials into the canonical Authentication aggregate. The separate
  `AuthenticationResolverService` and legacy public token lookup are removed.
- Acceptance: exact oneof protocol and reserved tags; service-only policy; identical credential
  verification and aggregate output for every family; all internal callers use the existing
  Authentication service client; no stale service, generated stub, SDK wrapper, or UI wrapper
  remains.
- Non-goals: Service assertion format, credential persistence schema, Principal structure,
  permission derivation, middleware ordering, route authorization semantics outside VerifyToken,
  and `Authorize` session refresh.
- Primary risk: an incomplete atomic cutover can leave a consumer calling a removed method or make
  raw credential resolution public. Descriptor, policy, consumer, and repository-search tests must
  fail on either condition.

## Scope and Ownership

- Proto sources and generated root, Python, JavaScript, and TypeScript artifacts own the protocol.
- Web Authentication API and router own dispatch, failure mapping, construction, and registration.
- Web policy owns direct Service admission for the single route.
- Web internal authentication client owns oneof construction and aggregate decoding.
- Assistant, Endpoint, Integration, and the composite Rapida client own constructor wiring.
- React SDK and UI authentication client own removal of the unusable public wrapper.
- Web API and React SDK READMEs own the documented visibility and intended caller.
- Package-local tests own success, rejection, descriptor, policy, and selection evidence.
- Dirty-worktree changes outside the final path manifest remain untouched.

## Implementation

1. Add failing descriptor, policy, handler, and client tests for the single VerifyToken contract.
2. Replace the legacy request and response, remove the resolver service from authentication proto,
   and regenerate all artifacts.
3. Move credential dispatch into the existing Web Authentication gRPC owner and delete the
   separate handler and registration.
4. Update the internal Web client and its consumers to use `AuthenticationServiceClient`.
5. Replace three policy routes with the service-only VerifyToken route.
6. Remove the public React and UI wrappers, stale tests, and stale generated resolver stubs.
7. Run focused tests, race tests, generated parity, repository search, finalization, and independent
   review.

## Verification

```bash
cmp protos/artifacts/authentication.proto sdks/react/src/clients/protos/artifacts/authentication.proto
cmp protos/artifacts/web-api.proto sdks/react/src/clients/protos/artifacts/web-api.proto
go test -count=1 ./protos ./pkg/authentication/... ./pkg/clients/web ./pkg/clients/rapida ./api/web-api/api ./api/web-api/router ./cmd/assistant ./cmd/endpoint ./cmd/integration
go test -race -count=1 ./pkg/clients/web ./pkg/clients/rapida ./api/web-api/api
(cd sdks/react && npm test -- --runInBand)
(cd ui && yarn test --watchAll=false --runInBand)
git diff --check
just agent-finalize "$(paste -sd, plans/0086-finalize-paths.txt)"
```

The full Web API package currently fails `TestWebRPCAuthenticationContract` for unrelated proxy
handlers. This implementation must preserve and report that blocker; delivery remains prohibited
until the separate gate is resolved and finalization passes.

## Approval

- RFC: `rfcs/0086-unified-verify-token-resolution.md`
- Challenger: Independent Codex protocol and lifecycle review
- Owner approval: Repository owner explicitly directed this replacement on 2026-11-15.
- Decision: Approved. The implementation may proceed; delivery remains blocked by the recorded
  pre-existing Web API contract-test failure.
