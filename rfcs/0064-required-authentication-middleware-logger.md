# RFC 0064: Required Authentication Middleware Logger

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0059 nullable-logger contract only

## Context

RFC 0059 permits a nil logger and every authentication middleware conditionally checks
`validator.NonNil(logger)` before recording a denial. Deployable command wiring always constructs
and passes its application logger. Nullable logging therefore adds repeated branches to every
credential and policy middleware without representing a supported deployment state.

## Decision

Require a configured `commons.Logger` for every HTTP, unary gRPC, and streaming gRPC authentication
and policy middleware constructor. Remove every `validator.NonNil(logger)` branch and log denial
events directly.

Do not add constructor validation, fallback loggers, functional options, or wrappers. A nil or
typed-nil logger is an invalid caller configuration and is unsupported. Tests provide a configured
test logger instead of nil.

## Contracts and Ownership

- Each deployable command owns construction and lifetime of its logger and passes it to every
  installed authentication middleware.
- `pkg/middlewares/authentication` assumes the logger dependency is valid and owns denial logging.
- Resolver, policy, credential, principal, status mapping, ordering, and controller behavior remain
  unchanged.
- Web API already satisfies the required dependency and needs no command-wiring change.

## Alternatives Rejected

- Keep nullable logging: rejected because nil is not a supported deployable state and the repeated
  checks obscure denial paths.
- Validate or panic in each constructor: rejected because it replaces repeated request-path checks
  with repeated construction checks for an impossible configured state.
- Install a fallback logger: rejected because hidden dependency substitution weakens explicit
  command ownership.
- Remove denial logging: rejected because authentication and authorization failures remain useful
  operational signals when they exclude credential values.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Internal callers that passed nil must provide a logger. No released or network contract changes. |
| Failure and recovery | An invalid nil logger can panic on a denial path. Correct command construction prevents this unsupported state; rollback restores nullable logging. |
| Security and privacy | Denial logging remains generic and does not include credential or authority values. Authentication decisions are unchanged. |
| Data and migration | N/A. No persisted data or schema changes. |
| Concurrency and resources | The shared application logger remains concurrency-safe and command-owned. No new resource is created by middleware. |
| Operations and observability | Every denial path now always emits its existing log when correctly configured. Log messages and levels are unchanged. |

## Rollout and Rollback

Remove logger guards from all authentication middleware in one candidate and replace every test nil
logger with a configured test logger. Stop on any nil middleware caller, changed status code,
test failure, or unresolved Critical or Major finding. Rollback restores the guards and nil test
callers together.

## Verification

```bash
go test ./pkg/middlewares/authentication ./cmd/web
go test -race ./pkg/middlewares/authentication ./cmd/web
go vet ./pkg/middlewares/authentication ./cmd/web
! rg -n 'validator\.NonNil\(logger\)' pkg/middlewares/authentication
! rg -n -g '*.go' 'New(Project|Organization|Service|User|Policy)(Unary|Stream|HTTP)Middleware\([^\n]*, nil\)' pkg/middlewares/authentication cmd/web
git diff --check
just agent-finalize "rfcs/0064-required-authentication-middleware-logger.md,pkg/middlewares/authentication/grpc.go,pkg/middlewares/authentication/grpc_test.go,pkg/middlewares/authentication/http.go,pkg/middlewares/authentication/http_test.go"
```

## Review and Approval

- Challenger: Independent GPT-6 Astra and GPT-5.6 Sol reviews with combined evidence panel
- Outcome: Approved
- Resolved findings: The application logger requires at least one output. The test logger enables
  console output only; passing tests remain quiet under the Go test harness.
- Open questions: None
- Approval reference: Repository owner required non-null authentication middleware loggers on
  2026-11-13. Independent reviews and the combined panel approved draft digest
  `b2aca3ab4fd0c2a014b8d708ffb0fd6cfa12b973c0fee295ba801eea2e0e01c4`.
