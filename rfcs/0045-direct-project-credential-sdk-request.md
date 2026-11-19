# RFC 0045: Direct Project Credential SDK Request

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-09
- Supersedes: RFC 0042 React SDK criteria-construction decision and RFC 0044 SDK-unchanged statement

## Context

RFC 0042 assigns Project API key criteria construction to a new React SDK options type. The
established `GetAllAssistant` SDK contract instead accepts its generated protobuf request directly.
The Project API key inventory API and its new paginated SDK entry point are unreleased, so retaining
a second options type creates inconsistency without a compatibility benefit.

## Decision

`ListProjectCredentials` accepts `GetAllProjectCredentialRequest` directly. Callers construct the
generated `Paginate` and `Criteria` messages and pass the request through unchanged, matching
`GetAllAssistant`. Do not add `ListProjectCredentialOptions` or another criteria wrapper type.

The existing unpaginated `GetAllProjectCredential(projectId, ...)` entry point remains unchanged.
The generated protobuf request remains the single source of truth for Project API key list fields.
This decision does not change backend validation, filter semantics, authorization, response
envelopes, or the access-management SDK methods.

## Contracts and Ownership

- The protobuf request owns Project API key pagination and criteria fields.
- The React SDK transports the generated request without rebuilding or interpreting it.
- UI clients own construction of the generated request from UI state.
- The new `ListProjectCredentials` signature is intentionally finalized before release. Existing
  released SDK entry points remain compatible.

## Alternatives Rejected

- Add `ListProjectCredentialOptions`: rejected because it duplicates the generated request and
  differs from the established assistant listing contract.
- Change the existing unpaginated entry point: rejected because it would break released callers.
- Build criteria inside the SDK from untyped objects: rejected because it adds another contract and
  mapping owner.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The unreleased paginated entry point changes before release. Existing released entry points remain unchanged. |
| Failure and recovery | Generated request construction errors remain local to the caller. Transport and response behavior are unchanged. |
| Security and privacy | Callers can construct arbitrary protobuf criteria. Unchanged controller validation enforces the allowlist before persistence access, and no credential secret is sent. |
| Data and migration | N/A. No schema or persisted data changes. |
| Concurrency and resources | N/A. Request construction and transport lifetimes are unchanged. |
| Operations and observability | N/A. No runtime service behavior or telemetry changes. |

## Rollout and Rollback

Verify and build the SDK, publish a new SDK version, update `ui/package.json` and `ui/yarn.lock` to
that exact version, install with the frozen lockfile, then verify and ship the UI. Stop if another
consumer of the unreleased paginated entry point is found, SDK publication fails, the lockfile does
not match the manifest, or the installed generated request lacks the criteria methods.

For rollback, restore the UI manifest and lockfile to the prior SDK version and redeploy the UI. A
published SDK version can remain unused; no package deletion or data rollback is required.

## Verification

- SDK tests prove the exact generated request instance reaches the gRPC client.
- UI client tests prove project ID, pagination, and criteria are set on the generated request.
- No `ListProjectCredentialOptions` declaration or export remains.
- The installed UI SDK version matches the manifest and lockfile and exposes the generated criteria
  methods during TypeScript and browser verification.

```sh
(cd sdks/react && npm test -- --runInBand tests/clients/project-credentials.test.ts && npm run build)
(cd ui && yarn install --frozen-lockfile && yarn checkTs)
(cd ui && yarn test --watchAll=false --runInBand src/clients/__tests__/credential.client.test.ts)
(cd ui && yarn playwright test e2e/journeys/integration/project-credential.spec.js)
just validate-rfc-layout
```

## Review and Approval

- Challenger: Codex, task `/root/criteria_implementation_review`
- Outcome: Approved
- Resolved findings: Corrected SDK/UI rollout order, installed dependency verification, and caller trust-boundary wording.
- Open questions: None
- Reviewed digest: SHA-256 `1adf3c3d634e5f94e3c77da5ffa10beaacb6e72832c0a131f9b6017392123047`
- Approval reference: Repository owner instruction to follow `GetAllAssistant` without an additional type, 2026-11-09
