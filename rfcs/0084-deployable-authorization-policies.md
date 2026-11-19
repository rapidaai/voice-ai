# RFC 0084: Deployable Authorization Policies

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner direction on 2026-11-14
- Date: 2026-11-14
- Supersedes: N/A

## Context

The authorization policy package embeds only `web-api.yaml`, and its constructor is named
`NewServicePolicy`. The name hides the owning deployable and cannot safely select policies for the
Assistant, Endpoint, or Integration services. Those Go services already authenticate a unified
Principal, but they do not yet own versioned route authorization catalogs.

This is a Governed authorization contract. The catalogs are added without installing enforcement
middleware in the three additional services.

## Decision

Expose one constructor and one embedded Version 1 catalog for each Go deployable:

```text
NewWebAuthorizationPolicy()         v1/web-api.yaml
NewAssistantAuthorizationPolicy()   v1/assistant-api.yaml
NewEndpointAuthorizationPolicy()    v1/endpoint-api.yaml
NewIntegrationAuthorizationPolicy() v1/integration-api.yaml
```

The constructors share one private strict parser owned by `ServicePolicy`. There is no generic
public constructor, caller-supplied source, runtime file lookup, or deployable switch. Each
constructor validates the expected deployable name as well as the existing schema, duplicate-key,
alias, route-shape, and typed policy invariants.

The Web policy is unchanged except for its constructor name. Endpoint routes retain the permission
decisions already reviewed in the Web catalog. Assistant management, deployment, knowledge,
document, observability, and dashboard routes use the same permissions as their Web-facing RPCs.
Every downstream permission route that accepts a Project subject has separate direct and delegated
branches. The direct branch requires management audience and Project binding. The delegated branch
requires the verified `internal` Service caller with actor ID 1, runtime minimum audience, the same
Project binding, and the same permissions. This preserves Web-proxied Project callers without
turning an arbitrary Service assertion into management authority.

Assistant conversation listing requires `project.conversations.read`; phone-call creation and live
talk streams require `project.runtime.invoke`. Both `GetAllConversationMessage` RPC aliases and the
two Talk metric mutations remain `disabled` because their current persistence paths do not prove
that caller-supplied conversation or message IDs belong to the Principal's Project. Telephony
provider callback and context routes also remain `disabled` until a signed provider-callback or
opaque-context authorization contract is accepted. Health routes are public.

Integration provider execution (`Chat`, `StreamChat`, `Embedding`, `Reranking`, and
`VerifyCredential`) remains `disabled`. Existing runtime-invoker and SIP paths do not carry one
proven, uniform provider-use authority, so enabling those operations requires a separately accepted
deployment-bound caller and authority-propagation contract. Audit reads accept only a user or
Project subject delegated by the verified `internal` Service caller with actor ID 1 and retain
`project.audit.read`; a direct Service subject cannot carry permissions and is not an authorization
branch. Audit metadata creation remains disabled, and health routes are public.

This change creates and validates catalogs only. It does not install route-authorization
middleware, remove controller checks, modify authentication, or change a public protobuf contract.
Document API remains owned by its Python runtime and is outside this Go catalog change.

Assistant has one explicit optional route group. `KnowledgeService` and `DocumentService` are
registered only when OpenSearch is configured. The Assistant catalog contains that complete group;
inventory verification selects it only in the configured state and proves exact equality in both
the 65-RPC standard state and the 78-RPC OpenSearch state. No other route is optional.

## Contracts and Ownership

- Each `v1/<deployable>.yaml` file is the sole route-decision source for its named deployable.
- `ServicePolicy` owns strict parsing and immutable lookup behavior.
- Explicit constructors own compile-time selection of their embedded catalog.
- Authentication middleware continues to own Principal construction.
- Controllers and services retain current authorization behavior until a separate accepted wiring
  change installs a catalog.
- Document API owns any future Python authorization catalog and loader.

## Alternatives Rejected

- Keep `NewServicePolicy`: rejected because it conceals that it always loads Web policy.
- Accept a deployable string or YAML bytes: rejected because callers could select or inject the
  wrong security contract.
- Use one cross-service catalog: rejected because route inventory and rollout ownership belong to
  each deployable.
- Mark unresolved callbacks public: rejected because transport reachability is not proof of a
  verified provider or call context.
- Wire enforcement in this change: rejected because catalog construction and runtime activation
  need separate verification and rollback boundaries.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The Web constructor is intentionally renamed in current consumers. No network contract changes. New catalogs are not enforced yet. |
| Failure and recovery | Invalid embedded policy fails its constructor. Reverting the constructor and catalog changes restores the previous build without data recovery. |
| Security and privacy | Explicit catalog ownership prevents cross-service selection. Unknown callback authorization remains disabled rather than guessed. No secrets enter policy files. |
| Data and migration | N/A. No schema, stored data, or migration changes. |
| Concurrency and resources | Catalogs are immutable after construction and create no goroutines or retained external resources. |
| Operations and observability | Startup behavior changes only where a constructor is explicitly called. Additional services are not wired in this change. |

## Rollout and Rollback

Rename the Web constructor and add the three new embedded catalogs and focused tests. Do not wire
the additional catalogs into service commands. A later accepted change may install one deployable
at a time after exact registered-route coverage and denial behavior are verified. Assistant policy
activation is prohibited while either conversation-message alias, either metric mutation, or any
registered telephony callback or context route is disabled. Tenant-bound Talk persistence and a
separately accepted callback/context authentication contract must replace those decisions before
deployable-wide enforcement. Integration activation is prohibited while provider execution routes
are disabled; a separately accepted deployment-bound caller and authority contract must replace
those decisions first.

Rollback removes the new constructors and catalogs and restores the Web constructor call sites.
There is no data rollback and no compatibility window because no service enforcement changes.

## Verification

Acceptance requires every constructor to load only its expected deployable, reject malformed or
mismatched input, return detached route policies, and expose the declared route inventory. Router
tests must construct each Go deployable's registered gRPC and HTTP routes and prove exact set
equality with its catalog, including missing and stale route detection. Assistant verification must
prove both exact inventories for its declared OpenSearch-dependent route group.

```text
go test ./pkg/authentication/policy ./api/web-api/router ./api/assistant-api/router ./api/endpoint-api/router ./api/integration-api/router ./cmd/web
go test -race ./pkg/authentication/policy
git diff --check
just agent-finalize "rfcs/0084-deployable-authorization-policies.md,pkg/authentication/policy/service_policy.go,pkg/authentication/policy/service_policy_test.go,pkg/authentication/policy/v1/web-api.yaml,pkg/authentication/policy/v1/assistant-api.yaml,pkg/authentication/policy/v1/endpoint-api.yaml,pkg/authentication/policy/v1/integration-api.yaml,api/web-api/router/web_test.go,api/assistant-api/router/authorization_policy_test.go,api/endpoint-api/router/authorization_policy_test.go,api/integration-api/router/authorization_policy_test.go,cmd/web/web.go"
```

## Review and Approval

- Challenger: Codex tasks `/root/auth_cutover_assistant` and `/root/jwt_rfc_astra_review`
- Outcome: Approved
- Resolved findings: Exact registered-route equality tests are now required for all three added
  deployables. Integration permits only delegated user or Project subjects from the exact internal
  caller instead of an impossible direct-Service permission branch. Downstream Project permission
  decisions separate direct management credentials from delegated internal callers. Unbound
  conversation-message reads and metric mutations and Integration provider execution are disabled.
  Assistant and Integration activation remain blocked until the stated tenant, callback, caller,
  and authority contracts are accepted. Assistant's OpenSearch-dependent Knowledge and Document
  group is verified in both absent and present configurations. Implementation review also required
  all parsed catalog state to become private so callers cannot mutate validated route decisions;
  lookups return detached snapshots and scalar identity is exposed only through accessors.
- Open questions: None
- Approval reference: Repository owner approved explicit per-deployable constructors and catalogs
  in this session on 2026-11-14.
