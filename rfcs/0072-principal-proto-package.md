# RFC 0072: Principal Protobuf Package

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0071 message naming and file ownership only

## Context

RFC 0071 placed the future Principal wire messages in `web-api.proto` with repeated
`Authentication` prefixes. The Principal is a reusable request-security contract for every service,
not a Web API presentation type. Its protobuf names should match the accepted immutable Go domain
names and live in a dedicated source file.

This remains a protobuf-only organization change. It does not remove the deprecated legacy
messages, populate the new session field, or change runtime behavior.

## Decision

Create `authentication.proto` with proto package `authentication` and the existing Go package
`github.com/rapidaai/protos`. Define the exact domain names:

```proto
message Identity {
  string type = 1;
  uint64 id = 2 [jstype = JS_STRING];
  string issuer = 3;
}

message ServiceActor {
  uint64 id = 1 [jstype = JS_STRING];
  string issuer = 2;
}

message Tenant {
  uint64 organizationId = 1 [jstype = JS_STRING];
  uint64 projectId = 2 [jstype = JS_STRING];
}

message ProofReference {
  string type = 1;
  string value = 2;
}

message Proof {
  string method = 1;
  string audience = 2;
  ProofReference reference = 3;
}

message Authorization {
  Tenant tenant = 1;
  string tenantName = 2;
  repeated AssignedRole roles = 3;
  repeated string effectivePermissions = 4;
}

message Principal {
  Identity subject = 1;
  optional ServiceActor actor = 2;
  Tenant selectedTenant = 3;
  Proof proof = 4;
  repeated Authorization authorizations = 5;
}
```

`authentication.proto` imports `common.proto` for `AssignedRole`. `web-api.proto` imports
`authentication.proto`, removes the prefixed Principal component messages introduced by RFC 0071,
and changes only `AuthenticationSession.principal` to `authentication.Principal`. The profile-only
User, Token, billing, deprecated messages, response field numbers, and all RPCs remain unchanged.

The authorization uniqueness and composition rules from RFC 0071 remain authoritative. This RFC
changes only source ownership and names.

## Contracts and Ownership

- `authentication.proto` owns the reusable Principal wire contract.
- `web-api.proto` owns `AuthenticatedUserProfile`, `AuthenticationSession`, and Web API response
  envelopes.
- Both root and React artifact checkouts contain byte-identical copies of both proto sources.
- Generated Go, Python, JavaScript, and TypeScript artifacts are derived and never edited manually.
- Runtime `pkg/authentication.Principal` remains unchanged.

## Alternatives Rejected

- Keep the messages in `web-api.proto`: rejected because it assigns a cross-service security
  contract to a presentation API.
- Keep `Authentication` prefixes: rejected because they diverge from the established domain names
  without resolving ambiguity.
- Move Token, billing, and UI profile into `authentication.proto`: rejected because they are session
  presentation data, not request Principal facts.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The new Principal messages are not populated by runtime code, so renaming them before adoption has no deployed value compatibility impact. Existing legacy fields and RPCs remain unchanged. |
| Failure and recovery | Missing imports or generated cross-file references fail compilation. Rollback restores RFC 0071 message placement and regenerates artifacts. |
| Security and privacy | Principal contents and validation requirements do not change. UI secrets remain outside `authentication.proto`. |
| Data and migration | N/A. No persisted data changes. |
| Concurrency and resources | N/A. No runtime resources change. |
| Operations and observability | N/A. No runtime producer or consumer uses the new Principal yet. |

## Rollout and Rollback

Add identical `authentication.proto` sources to both artifact checkouts, update both
`web-api.proto` imports and session references, regenerate all artifacts, and update descriptor
coverage. Stop on source drift, descriptor mismatch, consumer compilation failure, or unresolved
Critical or Major findings.

Rollback restores the RFC 0071 source placement and regenerates artifacts. No data rollback is
required.

## Verification

```bash
cmp protos/artifacts/authentication.proto sdks/react/src/clients/protos/artifacts/authentication.proto
cmp protos/artifacts/web-api.proto sdks/react/src/clients/protos/artifacts/web-api.proto
bash bin/artifacts-generate.sh
(cd sdks/react && bash bin/artifacts-generate.sh)
go test -count=1 ./protos
go test -run '^$' ./api/web-api/api ./pkg/clients/web ./pkg/authenticators ./cmd/web
(cd sdks/react && npm test -- --runInBand)
git diff --check
just agent-finalize "rfcs/0072-principal-proto-package.md,protos/artifacts/authentication.proto,protos/artifacts/web-api.proto,sdks/react/src/clients/protos/artifacts/authentication.proto,sdks/react/src/clients/protos/artifacts/web-api.proto,protos/authentication.pb.go,protos/web-api.pb.go,protos/web-api_grpc.pb.go,api/document-api/app/bridges/artifacts/protos/authentication_pb2.py,api/document-api/app/bridges/artifacts/protos/authentication_pb2.pyi,api/document-api/app/bridges/artifacts/protos/authentication_pb2_grpc.py,api/document-api/app/bridges/artifacts/protos/web_api_pb2.py,api/document-api/app/bridges/artifacts/protos/web_api_pb2.pyi,api/document-api/app/bridges/artifacts/protos/web_api_pb2_grpc.py,sdks/react/src/clients/protos/authentication_pb.js,sdks/react/src/clients/protos/authentication_pb.d.ts,sdks/react/src/clients/protos/authentication_pb_service.js,sdks/react/src/clients/protos/authentication_pb_service.d.ts,sdks/react/src/clients/protos/web-api_pb.js,sdks/react/src/clients/protos/web-api_pb.d.ts,sdks/react/src/clients/protos/web-api_pb_service.js,sdks/react/src/clients/protos/web-api_pb_service.d.ts,protos/authentication_principal_contract_test.go"
```

## Review and Approval

- Challenger: Independent GPT-6 Astra and GPT-5.6 Sol contract and security reviews, followed by a combined evidence panel
- Outcome: Approved
- Resolved findings: Added the repository-required exact finalization boundary for both proto
  sources, every generated language artifact, and descriptor coverage.
- Open questions: None
- Approval reference: Repository owner requested a dedicated `authentication.proto` and exact
  Principal domain naming on 2026-11-13. All challengers approved pre-metadata digest
  `93342ed268e3e392dec1dadffc93ce1e817061b0f31ad9c2e023ec0d96e11118`.
