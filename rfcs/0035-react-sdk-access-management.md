# RFC 0035: React SDK Access Management

- Status: Superseded
- Owner: Access Management
- Approved by: Owner
- Date: 2026-11-08
- Supersedes: N/A
- Superseded by: RFC 0036

## Context

RFC 0029 defined the access-management gRPC contract and deferred SDK adoption. The React SDK
currently generates browser clients from its `internal-bridge` submodule, but it has no generated
access-management client, connection entry point, Promise wrappers, or public exports. Consumers
therefore cannot call the seven accepted role-catalog RPCs through `@rapidaai/react`.

## Decision

Generate the access-management protobuf and gRPC-web artifacts from the pinned contract. Add one
`AccessManagementServiceClient` getter to `ConnectionConfig`, Promise wrappers for all seven unary
RPCs, and public exports for the wrappers, messages, enums, and generated service client.

The wrappers accept generated request objects and require `UserAuthInfo`. They use connection-level
user auth with the optional call-level user-auth fallback, return the generated response, and reject
with the transport error. A connection configured only with `ClientAuthInfo` remains unsupported for
these RPCs because the backend requires user authentication. The SDK adds no authorization decision.

Generation uses the locked Node dependencies, including a pinned `protoc-gen-js`, and requires the
repository script's exact supported `libprotoc 33.4` version. The script fails before deleting or
generating files when the compiler or plugins do not match the declared toolchain.

## Contracts and Ownership

- `internal-bridge` remains the sole source of truth for protobuf messages and RPC descriptors.
- The React SDK owns generated browser artifacts, `ConnectionConfig` client construction, Promise
  wrappers, package exports, and the fail-closed generation script.
- The public SDK gains additive support for `ListPermissions`, `ListRoles`, `GetRole`,
  `CreateCustomRole`, `UpdateCustomRole`, `ReduceCustomRolePermissions`, and `ArchiveCustomRole`.
- Backend services continue to own validation, authorization, tenancy, persistence, and response
  semantics.

## Alternatives Rejected

- Export only the generated client: rejected because existing SDK consumers use authenticated
  Promise wrappers and `ConnectionConfig` service getters.
- Hand-write request and response types: rejected because it would duplicate the protobuf contract.
- Add a generic unary-call abstraction: rejected because seven direct wrappers match established SDK
  patterns and keep request, response, and service method types visible.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | All exports and the service getter are additive. Existing SDK methods and generated contracts remain unchanged. The access-management wrappers intentionally accept only user auth because the backend rejects API-key auth. |
| Failure and recovery | Each wrapper rejects the existing transport error and makes no client-side state change. Consumers may retry according to the backend RPC contract. Rollback removes only the additive SDK surface. |
| Security and privacy | Calls use the existing `WithAuthContext` metadata path with `UserAuthInfo`. The change introduces no credential storage, logging, API-key support, or authorization decision in the SDK. |
| Data and migration | N/A. The SDK does not persist or migrate role data. |
| Concurrency and resources | The seven RPCs are unary and allocate no long-lived client resource beyond the existing generated client behavior. |
| Operations and observability | No new SDK telemetry is introduced. Transport and backend errors remain observable to callers through rejected Promises. |

## Rollout and Rollback

Publish the backend contract and service before publishing the React SDK. Generate artifacts from the
same pinned `internal-bridge` commit with the declared compiler and locked plugins, and publish the SDK
only after tests and the package build pass. Stop if toolchain validation fails, generation changes
unrelated contracts, or any wrapper/export test fails. Roll back by reverting the additive React SDK
commit and restoring the prior proto submodule pointer. There are no irreversible effects.

## Verification

- Generated descriptors expose all seven access-management RPCs and their request/response types.
- Each wrapper forwards its request with connection-level user auth, falls back to call-level user
  auth, resolves a success response, and rejects a transport failure.
- A clean `npm ci` supplies both protobuf plugins. The generator rejects a compiler version other
  than `libprotoc 33.4` before removing existing artifacts, then changes only the four generated
  access-management files.
- `ConnectionConfig` constructs the access-management client against the web endpoint.
- The package entry point exports the wrappers, generated types, enums, and service client.
- Run `npm ci`, `bash bin/artifacts-generate.sh`, `npm test -- --runInBand`, and `npm run build` from
  `sdks/react`.
- Run the repository's scoped `just agent-finalize` command for every changed path.

## Review and Approval

- Challenger: Codex (`react_sdk_rfc_challenge`)
- Outcome: Approved
- Resolved findings: Restricted wrappers to user authentication and defined a pinned, fail-closed protobuf generation toolchain.
- Open questions: None
- Approval reference: Owner request in this thread to update the React SDK for all access-management APIs.
