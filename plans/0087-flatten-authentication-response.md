# Flatten Authentication Response Plan

## Change Contract

- Outcome: Successful User authentication methods return `authentication` and `token` directly in
  `AuthenticateResponse`; `AuthenticationSession` is absent from sources, generated artifacts, and
  production consumers.
- Acceptance: published tags and names are reserved; all six response producers populate both
  fields; UI, browser fixtures, and Document consumers decode the flat fields; generated languages
  agree; no legacy wrapper or accessor remains; the console is verified against a packed SDK
  `1.1.95` candidate and pins that version for deployment.
- Non-goals: error-envelope redesign, token issuance, Principal or aggregate shape, `VerifyToken`,
  authorization policy, credential headers, persistence, and database migrations.
- Primary risk: a partial protocol rollout can make successful login responses appear empty.

## Ownership and Allowed Paths

- RFC and plan: `rfcs/0087-*`, `plans/0087-*`.
- Proto sources and generated Go, Python, JavaScript, and TypeScript authentication artifacts.
- Web Authentication response mapper and focused response tests.
- React SDK exports and contract tests.
- Console authentication store, sign-in/sign-up consumers, types, and focused tests.
- React SDK and console package metadata needed for the `1.1.95` release and exact dependency pin.
- Console authentication gRPC-Web fixture plus sign-in, sign-up, and onboarding browser journeys.
- Document API authentication bridge and focused tests.
- No unrelated controller, middleware, policy, schema, or role-management changes.

## Implementation

1. Change descriptor tests to require flat fields and reject `AuthenticationSession`.
2. Update and push the authoritative proto as a follow-up commit without rewriting history.
3. Align the React proto checkout and regenerate every language artifact.
4. Update the Web response mapper and its six-method coverage.
5. Update SDK exports, package version, console state, login consumers, wire fixture, dependency
   pin, and Document bridge. Add missing-authentication, missing-token, failed-response, and legacy
   wrapper rejection tests.
6. Build and pack SDK `1.1.95`, record its SHA-256, install that exact candidate into the console,
   and run console type, build, focused Jest, and browser authentication/onboarding checks.
7. Search for stale wrapper symbols, run focused and broader checks, finalize, and obtain an
   independent review of the fixed manifest.

## Verification

Use the commands declared in RFC 0087. Record the Python environment limitation if `pytest` is not
available. Repository delivery remains blocked until the existing full Web API contract gate is
clean.

## Approval

- RFC: `rfcs/0087-flatten-authentication-response.md`
- Owner approval: repository owner explicitly approved complete wrapper removal on 2026-11-16.
- Challenger: Independent GPT-6 Astra protocol and rollout review. Approved with no open questions
  after adding browser fixture coverage, packed SDK verification, and negative Document cases.
