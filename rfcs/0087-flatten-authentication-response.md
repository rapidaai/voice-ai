# RFC 0087: Flatten Authentication Response

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-16
- Supersedes: RFC 0074 and RFC 0076 authentication response wrapper only

## Context

The unreleased authentication contract wraps the canonical `authentication.Authentication` and
the issued `Token` in `web_api.AuthenticationSession`, then wraps that value again in
`AuthenticateResponse`. `AuthenticationSession` has no independent lifecycle, is used by no other
response, and adds one accessor layer to every login, registration, social login, and session
refresh consumer.

The authoritative proto containing this wrapper was pushed to `internal-bridge/main` in commit
`cb0ffa4`, but no generated consumer release or coordinated deployment has occurred. The repository
owner requires the wrapper to be removed before the remaining consumers are committed.

## Decision

Delete `AuthenticationSession`. `AuthenticateResponse` directly carries the canonical
Authentication aggregate and its issued User session token:

```proto
message AuthenticateResponse {
  reserved 3, 5;
  reserved "data", "session";

  int32 code = 1;
  bool success = 2;
  optional Error error = 4;
  authentication.Authentication authentication = 6;
  Token token = 7;
}
```

Tags `3` and `5`, and names `data` and `session`, remain reserved. The follow-up commit does not
amend, force-push, or otherwise rewrite the published proto history. A successful authentication
response contains both `authentication` and `token`; a failed response contains neither. The Token
does not become part of `authentication.Authentication`.

`Authenticate`, `RegisterUser`, `Authorize`, `Linkedin`, `Google`, and `Github` continue returning
`AuthenticateResponse`. Requests, RPC names, credential headers, Principal construction, route
policy, persistence, token issuance, and `VerifyToken` remain unchanged.

## Contracts and Ownership

- The proto repository owns the flat response and reserved field history.
- Web Authentication controllers own populating both success fields from one service result.
- `authentication.Authentication` remains the credential-free identity, tenant, role, permission,
  and billing aggregate.
- React SDK and generated artifacts expose the flat accessors without a compatibility wrapper. The
  SDK package version advances from `1.1.94` to `1.1.95`; the console pins that exact release before
  deployment.
- The console authentication store owns persisting the Authentication and Token values separately.
- Console browser fixtures own wire-accurate flat authentication responses for sign-in, sign-up,
  and onboarding journeys.
- The Document API bridge owns decoding the flat response for its User authentication flow.

## Alternatives Rejected

- Retain `AuthenticationSession`: rejected because it has one consumer boundary and no independent
  lifecycle or reuse.
- Put the token inside `authentication.Authentication`: rejected because reusable authentication
  facts must not contain a raw credential.
- Reuse tag `5` for `authentication`: rejected because that tag was published as a different message
  type and mixed generated clients could decode it incorrectly.
- Keep both flat fields and `session`: rejected because the branch is unreleased and the owner
  requested one clean contract without a compatibility path.
- Remove the existing response envelope in the same change: rejected because status-envelope
  cleanup is unrelated and would expand the public break.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | This intentionally changes an unreleased public response. All generated artifacts, producers, and consumers must move together. Published tags and names are reserved rather than reused. |
| Failure and recovery | Missing either success field, a failed envelope, or the legacy wrapper fails consumer validation. Partial regeneration or rollout is a stop condition. |
| Security and privacy | The raw token remains separate from the reusable Authentication aggregate and is returned only by User session-issuing methods. |
| Data and migration | No database or persisted-record migration is required. The console already stores Authentication and Token separately. |
| Concurrency and resources | N/A. The response is request-scoped and owns no retained resource. |
| Operations and observability | RPC names and status behavior remain unchanged. Descriptor and consumer tests detect contract drift before rollout. |

## Rollout and Rollback

Push a new proto commit after `cb0ffa4`, update both submodule pointers, regenerate every language,
then update Web, SDK, console, browser fixtures, and Document consumers in one repository candidate.
Build and pack SDK version `1.1.95`, install that exact local candidate into the console for type,
build, and browser verification, then pin the console dependency and lockfile to the published
`1.1.95` artifact before deployment. Record the packed or published artifact SHA-256. Do not publish
or deploy an intermediate generated package.

Stop if `AuthenticationSession`, `getSession`, or `session.authentication` remains in a production
consumer or browser fixture, if a successful response omits either flat field, if a legacy wrapper
is accepted, or if generated sources do not match the same proto revision. Rollback restores the
prior proto revision, SDK `1.1.94`, console dependency pin, generated artifacts, producers, and
consumers together. No data rollback is required.

## Verification

```bash
cmp protos/artifacts/web-api.proto sdks/react/src/clients/protos/artifacts/web-api.proto
bash bin/artifacts-generate.sh
(cd sdks/react && bash bin/artifacts-generate.sh)
go test -count=1 ./protos ./api/web-api/api
(cd sdks/react && npm test -- --runInBand)
(cd sdks/react && npm run build)
# Pack SDK 1.1.95 into a mktemp directory, record its SHA-256, and install it into ui with
# npm install --no-save --package-lock=false before the following console checks.
(cd ui && CI=true yarn test --watch=false --runInBand src/stores/auth src/app/pages/authentication)
(cd ui && yarn checkTs && yarn build)
(cd api/document-api && python3 -m pytest -q tests/bridges/test_auth_bridge.py)
just ui-browser-test
git diff --check
just agent-finalize "$(paste -sd, plans/0087-finalize-paths.txt)"
```

Document bridge tests explicitly reject missing Authentication, missing Token, a failed response,
and the legacy `session` wrapper. They also prove that the returned aggregate excludes the Token.
Browser verification includes authentication sign-in/sign-up and onboarding journeys using the
wire fixture carrying response fields `6` and `7`.

The pre-existing full Web API proxy-principal contract failure remains outside this RFC and still
blocks repository delivery. Focused authentication tests must pass, but they cannot waive that
repository gate.

## Review and Approval

- Challenger: Independent GPT-6 Astra protocol and rollout review
- Outcome: Approved
- Resolved findings: Added wire-accurate browser fixtures and required browser tests. Added a built
  and packed SDK `1.1.95` candidate, console installation/type/build evidence, dependency pinning,
  artifact digest, and explicit negative Document bridge cases.
- Open questions: None
- Approval reference: Repository owner directed complete removal of the unreleased
  `AuthenticationSession` wrapper on 2026-11-16.
