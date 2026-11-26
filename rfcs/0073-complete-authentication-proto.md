# RFC 0073: Complete Authentication Protobuf Contract

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0072 session-type ownership only

## Context

RFC 0072 moved the reusable Principal wire contract into `authentication.proto` but left the
future authenticated User session types in `web-api.proto`. The resulting file is not a complete
authentication contract: consumers must combine the authentication package with Web API
presentation messages to represent an authenticated session.

The repository already has top-level Go messages named `User` and `BillingPlan`. All generated Go
protobufs currently share package `protos`, so introducing more top-level messages with those names
would not compile.

## Decision

Keep the existing Principal messages in `authentication.proto` and add a complete session contract:

```proto
message Session {
  message User {
    uint64 id = 1 [jstype = JS_STRING];
    string name = 2;
    string email = 3;
    string status = 4;
  }

  message Token {
    uint64 id = 1 [jstype = JS_STRING];
    string token = 2;
    string tokenType = 3;
    bool isExpired = 4;
  }

  message BillingPlan {
    string planSlug = 1;
    string planName = 2;
    map<string, int64> quotas = 3;
  }

  Principal principal = 1;
  User user = 2;
  Token token = 3;
  optional BillingPlan billingPlan = 4;
}
```

The nested messages make session ownership explicit and avoid collisions with unrelated global
messages. `Session.User` is profile-only. Assigned roles and effective permissions remain solely in
`Principal.authorizations`.

Remove `AuthenticatedUserProfile` and `AuthenticationSession` from `web-api.proto` and change
`AuthenticateResponse.session = 5` to `authentication.Session` without changing its field number.
Keep the deprecated `Authentication` response, its `Token` and `AuthBillingPlan` dependencies, all
requests, response envelopes, RPCs, and runtime producers unchanged. The new Session remains
unpopulated until an accepted atomic consumer-cutover RFC.

## Contracts and Ownership

- `authentication.proto` owns Principal identity, proof, tenant authorization, and the future User
  session payload.
- `web-api.proto` owns Web API requests, response envelopes, RPC services, and deprecated legacy
  response types during migration.
- Internal request authentication communicates `authentication.Principal`. UI authentication may
  later consume `authentication.Session` after the coordinated runtime cutover.
- Root and React proto sources remain byte-identical. Generated Go, Python, JavaScript, and
  TypeScript artifacts remain derived outputs.

## Alternatives Rejected

- Keep session messages in `web-api.proto`: rejected because it leaves the authentication contract
  split across owners.
- Add top-level `User` and `BillingPlan`: rejected because the generated Go package already owns
  those names.
- Move the legacy `Token` and `AuthBillingPlan` immediately: rejected because it expands this
  protobuf-only correction into a runtime and SDK consumer migration.
- Put roles on `Session.User`: rejected because `Principal.authorizations` is the single source of
  authenticated roles and effective permissions.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Field 5 retains its number and message wire shape. Its protobuf full type name changes before any runtime producer or consumer uses it. Deprecated field 3 remains unchanged. |
| Failure and recovery | Descriptor, generated artifact, source parity, and consumer compilation checks stop a partial move. Rollback restores the RFC 0072 message placement and regenerates artifacts. |
| Security and privacy | Principal still excludes secrets. Session intentionally contains the existing UI token payload but remains unpopulated. Roles and permissions are not duplicated onto the User profile. |
| Data and migration | N/A. No persisted data changes and no browser or cache migration occurs. |
| Concurrency and resources | N/A. Protobuf generation introduces no runtime resources. |
| Operations and observability | N/A. No runtime producer reads or writes the new Session field. |

## Rollout and Rollback

Update both proto source checkouts, regenerate all language artifacts, and update descriptor tests.
Do not change controllers, middleware, resolvers, internal callers, or UI state. Stop on source
drift, generated artifact mismatch, consumer compilation failure, or unresolved Critical or Major
review findings.

Rollback restores `AuthenticatedUserProfile` and `AuthenticationSession` to `web-api.proto`,
restores the RFC 0072 field type, and regenerates all artifacts. No data rollback is required.

## Verification

Descriptor coverage must assert every field number and message type in `Session`, `Session.User`,
`Session.Token`, and `Session.BillingPlan`; prove that `BillingPlan.quotas` is a map; prove that
`AuthenticateResponse.session = 5` resolves to `authentication.Session`; and prove that
`web_api.AuthenticatedUserProfile` and `web_api.AuthenticationSession` are absent.

```bash
cmp protos/artifacts/authentication.proto sdks/react/src/clients/protos/artifacts/authentication.proto
cmp protos/artifacts/web-api.proto sdks/react/src/clients/protos/artifacts/web-api.proto
bash bin/artifacts-generate.sh
(cd sdks/react && bash bin/artifacts-generate.sh)
go test -count=1 ./protos
go test -run '^$' ./api/web-api/api ./pkg/clients/web ./pkg/authenticators ./cmd/web
(cd sdks/react && npm test -- --runInBand)
git diff --check
just agent-finalize "rfcs/0073-complete-authentication-proto.md,protos/artifacts/authentication.proto,protos/artifacts/web-api.proto,sdks/react/src/clients/protos/artifacts/authentication.proto,sdks/react/src/clients/protos/artifacts/web-api.proto,protos/authentication.pb.go,protos/web-api.pb.go,protos/web-api_grpc.pb.go,api/document-api/app/bridges/artifacts/protos/authentication_pb2.py,api/document-api/app/bridges/artifacts/protos/authentication_pb2.pyi,api/document-api/app/bridges/artifacts/protos/authentication_pb2_grpc.py,api/document-api/app/bridges/artifacts/protos/web_api_pb2.py,api/document-api/app/bridges/artifacts/protos/web_api_pb2.pyi,api/document-api/app/bridges/artifacts/protos/web_api_pb2_grpc.py,sdks/react/src/clients/protos/authentication_pb.js,sdks/react/src/clients/protos/authentication_pb.d.ts,sdks/react/src/clients/protos/authentication_pb_service.js,sdks/react/src/clients/protos/authentication_pb_service.d.ts,sdks/react/src/clients/protos/web-api_pb.js,sdks/react/src/clients/protos/web-api_pb.d.ts,sdks/react/src/clients/protos/web-api_pb_service.js,sdks/react/src/clients/protos/web-api_pb_service.d.ts,protos/authentication_principal_contract_test.go"
```

## Review and Approval

- Challenger: Independent GPT-6 Astra and GPT-5.6 Sol contract and security reviews, followed by a combined evidence panel
- Outcome: Approved
- Resolved findings: Closed the draft governance marker and made exact nested Session descriptor coverage mandatory.
- Open questions: None
- Approval reference: Repository owner approved the complete authentication structure on 2026-11-13.
  All challengers approved pre-metadata digest
  `0c0a4a2d7865cea117501a29acaf0ffcfac046704a251ff9ea704fba1b5990ae`.
