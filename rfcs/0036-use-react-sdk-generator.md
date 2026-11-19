# RFC 0036: Use the React SDK Generator

- Status: Accepted
- Owner: Access Management
- Approved by: Owner
- Date: 2026-11-08
- Supersedes: RFC 0035

## Context

RFC 0035 accepted React SDK access-management support but selected `libprotoc 33.4` and a new
JavaScript plugin. The React SDK already owns `bin/artifacts-generate.sh`, and its checked-in
artifacts match the installed protobuf 3 toolchain. Running the existing script with
`libprotoc 3.20.3` produces the four new access-management artifacts without changing any existing
generated contract.

## Decision

Retain the SDK surface, user-authentication contract, ownership, rollout, and rollback decisions from
RFC 0035. Use the React SDK's existing `bin/artifacts-generate.sh` unchanged with its compatible
`libprotoc 3.20.3` compiler and locked `ts-protoc-gen` dependency. Do not add another generator,
dependency, or generation abstraction.

## Contracts and Ownership

- `internal-bridge` remains the protobuf source of truth.
- The React SDK continues to own its existing generator and checked-in browser artifacts.
- The public wrappers, generated types, service client, and `ConnectionConfig` getter remain exactly
  as accepted in RFC 0035.
- This decision changes no backend, authentication, or role-management behavior.

## Alternatives Rejected

- Add a new JavaScript generator dependency: rejected because the existing protobuf 3 compiler
  already generates byte-compatible artifacts.
- Upgrade every generated JavaScript file: rejected because unrelated generated churn would expand
  the public compatibility boundary.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Existing generated files remain byte-for-byte unchanged. Four additive access-management files are introduced. |
| Failure and recovery | Generation failure leaves no committed change. Restore the prior proto pointer and omit the four new files to roll back. |
| Security and privacy | Unchanged from RFC 0035. Wrappers require user authentication and add no credential storage or authorization logic. |
| Data and migration | N/A. The SDK persists no role data. |
| Concurrency and resources | N/A. Generation is an offline build step and the RPCs remain unary. |
| Operations and observability | The compatible compiler must be available when regenerating artifacts. Runtime error behavior remains rejected Promises. |

## Rollout and Rollback

Run the existing generator with `libprotoc 3.20.3`, verify no existing generated file changes, then
test and build the SDK. Publish only after the backend service is available. Roll back by reverting
the additive SDK commit and proto pointer. There are no irreversible effects.

## Verification

- Run `PATH="/opt/homebrew/opt/protobuf@3/bin:$PATH" bash bin/artifacts-generate.sh` from `sdks/react`.
- Verify only the proto pointer and four access-management generated files differ under
  `src/clients/protos`.
- Run the wrapper tests, full SDK tests, package build, and scoped repository finalization accepted
  by RFC 0035.

## Review and Approval

- Challenger: Codex (`react_sdk_rfc_challenge`)
- Outcome: Approved
- Resolved findings: Confirmed the existing generator and protobuf 3 compiler preserve every existing generated contract.
- Open questions: None
- Approval reference: Owner direction in this thread to use the React SDK's existing artifact generator.
