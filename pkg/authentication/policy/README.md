<!--
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md or contact sales@rapida.ai for commercial usage.
-->

# Authorization policies

This package owns one embedded, versioned route authorization policy for each Go deployable:

```text
v1/web-api.yaml
v1/assistant-api.yaml
v1/endpoint-api.yaml
v1/integration-api.yaml
```

Each policy answers one question: may the resolved authentication Principal invoke this exact HTTP
route or gRPC method?

## Complete schema reference

### Document keys

Every policy document contains exactly these four top-level keys. Unknown keys are rejected.

| Key | Type | Allowed value | Why it exists |
| --- | --- | --- | --- |
| `version` | Unsigned integer | `1` | Selects the policy schema understood by the parser. |
| `deployable` | String | `web-api`, `assistant-api`, `endpoint-api`, or `integration-api` | Prevents a constructor from loading another service's policy. |
| `grpc` | Nonempty map | Full gRPC method to route policy | Owns every gRPC decision for the deployable. |
| `http` | Nonempty map | HTTP method and path to route policy | Owns every HTTP decision for the deployable. |

Each constructor requires its exact deployable:

| Constructor | Required `deployable` |
| --- | --- |
| `NewWebAuthorizationPolicy` | `web-api` |
| `NewAssistantAuthorizationPolicy` | `assistant-api` |
| `NewEndpointAuthorizationPolicy` | `endpoint-api` |
| `NewIntegrationAuthorizationPolicy` | `integration-api` |

### Route keys

A gRPC key uses the generated full method name:

```yaml
grpc:
  /endpoint_api.EndpointService/GetEndpoint:
```

It must begin with `/` and contain exactly one service separator:

```text
/<protobuf package>.<service>/<method>
```

An HTTP key combines an allowed uppercase method and the registered route template:

```yaml
http:
  GET /healthz/:
  PATCH /v1/assistant/configurations/:assistantId/:id:
```

Allowed HTTP methods are `DELETE`, `GET`, `HEAD`, `OPTIONS`, `PATCH`, `POST`, and `PUT`. The path
must begin with `/`. The key must match the router template, including parameter names and trailing
slashes.

### Route-policy keys

| Key | Type | Possible values | Why it is used |
| --- | --- | --- | --- |
| `authorization` | String | `public`, `authenticated`, `permission`, `any_of`, `disabled` | Selects the decision shape. |
| `actors` | Unique nonempty list | `user`, `project`, `organization`, `service` | Restricts `Principal.Subject().Type`. |
| `context` | String | `user`, `organization`, `project`, `service` | Requires the matching Principal tenant shape. |
| `permissions` | Unique nonempty string list | `organization.*` or `project.*` keys allowed by the context | Requires every listed authority on the Principal. |
| `project_token` | Object | `audience` and `binding` | Adds Project-credential proof requirements. |
| `caller_services` | Unique object list | `issuer` and `actor_id` | Restricts the immediate verified Service caller. |
| `any_of` | Nonempty alternative list | Complete alternative policies | Expresses legitimate paths with different constraints. |
| `authenticated` | Boolean, alternatives only | `true` | Makes one `any_of` alternative authenticated-only instead of permission-based. |

All strings must be valid, trimmed, nonblank where required, and contain no NUL character. Lists do
not allow duplicate values. Two identical `any_of` alternatives are rejected.

### `actors`

| Value | Principal identity | Effective context |
| --- | --- | --- |
| `user` | User ID | `user`, `organization`, or `project` when the required tenant exists. |
| `project` | Project credential ID | `project`; `project_token` is mandatory. |
| `organization` | Organization credential ID | `organization`. |
| `service` | Verified Service identity | `service`, or a tenant context carried by the Service Principal. |

Actor and context checks both apply. Listing an actor does not bypass a missing tenant. Organization
credential and direct Service Principals carry no permissions, so those actors are useful for
`authenticated`, not `permission`.

### `context`

| Value | Required Principal facts |
| --- | --- |
| `user` | Subject is a User. |
| `organization` | Organization ID is nonzero; subject is User, Organization credential, or Service. |
| `project` | Organization ID and Project ID are nonzero; subject is User, Project credential, or Service. |
| `service` | Subject is a direct, tenantless Service assertion; the Principal is not delegated and `caller_services` is present. |

Context proves the Principal's selected tenant. It does not prove that an ID in the request belongs
to that tenant. Controllers validate request data, and services enforce resource ownership and
persistence predicates.

### `permissions`

`permissions` is valid only for a `permission` decision or a permission-based `any_of` alternative.
The Principal must contain every listed key.

```yaml
permissions:
  - project.observability.read
  - project.resources.update
```

The example means:

```text
project.observability.read AND project.resources.update
```

An Organization context accepts only `organization.*` keys. A Project context accepts
`organization.*` and `project.*` keys. User and Service contexts cannot define permission policies.
The policy parser validates the namespace and string shape; it does not query a role or permission
database. Authorization compares each key with the immutable authorities already in the Principal.

### `project_token`

`project_token` is required exactly when `actors` contains `project`. It is forbidden when the
branch does not accept a Project subject.

| Key | Type | Possible values | Meaning |
| --- | --- | --- | --- |
| `audience` | String | `management`, `runtime` | Selects the minimum accepted Project proof. |
| `binding` | String | `project` | Requires nonzero Organization and Project tenant IDs on the Principal. |

`audience: management` accepts only a direct Project credential with a matching credential proof.
It rejects a delegated Project subject.

`audience: runtime` is a minimum class, not a separate direct credential type. It accepts either a
direct management credential or a Project subject delegated by a verified internal Service
assertion.

`binding: project` proves that the Principal is Project-bound. It does not compare a request field
or database row with the Principal's Project. That resource check remains mandatory at the owning
controller or service.

### `caller_services`

Each allowed caller contains:

| Key | Type | Allowed value | Meaning |
| --- | --- | --- | --- |
| `issuer` | String | Nonblank verified issuer | Matches `Principal.Caller().Issuer`. |
| `actor_id` | Unsigned integer | `1` through `MaxInt64` | Matches `Principal.Caller().ID`. |

Both values must match the same entry. Multiple entries mean caller A OR caller B. Omitting
`caller_services` means the branch does not restrict the immediate Service caller.

An empty list is equivalent to omission outside `context: service`. Service context requires at
least one allowed caller.

The current shared internal assertion identity is:

```yaml
caller_services:
  - issuer: internal
    actor_id: 1
```

The caller restriction does not add permissions and does not replace the subject or tenant.

### `any_of` alternatives

An alternative supports these keys:

- `actors`
- `context`
- `caller_services`
- `project_token`
- exactly one of `permissions` or `authenticated: true`

An alternative cannot contain `authorization` or a nested `any_of`.

### Valid key combinations

| `authorization` | Required keys | Optional keys | Forbidden keys |
| --- | --- | --- | --- |
| `public` | `authorization` | None | All other route-policy keys |
| `disabled` | `authorization` | None | All other route-policy keys |
| `authenticated` | `authorization`, `actors`, `context` | `caller_services`, conditional `project_token` | `permissions`, `any_of`, `authenticated` |
| `permission` | `authorization`, `actors`, `context`, `permissions` | `caller_services`, conditional `project_token` | `any_of`, `authenticated` |
| `any_of` | `authorization`, `any_of` | None | Top-level actors, context, callers, token, permissions, and `authenticated` |

`project_token` is conditionally required for every `authenticated`, `permission`, or `any_of`
alternative whose actors include `project`.

### Invalid document shapes

The parser rejects:

- an unknown field at any decoded schema level;
- duplicate mapping keys, duplicate list values, or duplicate alternatives;
- YAML aliases, anchors, and merge keys;
- a second YAML document or trailing document content;
- an unsupported version or mismatched deployable;
- empty gRPC or HTTP inventories;
- malformed route keys;
- an invalid authorization-field combination.

## Authorization modes

| Mode | Decision |
| --- | --- |
| `public` | Allows the route without a Principal. |
| `authenticated` | Requires a matching actor and context, but no permission key. |
| `permission` | Requires a matching actor, context, proof constraints, and every listed permission. |
| `any_of` | Allows the request when at least one complete alternative matches. |
| `disabled` | Always denies the route before its controller runs. |

Unknown routes are not public. Policy middleware fails closed when no route entry exists.

## What `any_of` means

`any_of` is logical OR between alternatives. Every field inside one alternative is logical AND.
Every permission listed inside one alternative is also required.

```text
alternative 1 matches
OR alternative 2 matches
OR alternative 3 matches
= allow

no complete alternative matches
= deny
```

An alternative is a complete authorization decision. Conditions from different alternatives are
never combined. For example, a caller cannot take the actor from one alternative and the
permission from another.

Each alternative contains either `permissions` or `authenticated: true`, never both. The top-level
`any_of` entry contains no actor, context, caller, token, or permission fields of its own.

This downstream policy has three valid paths:

```yaml
authorization: any_of
any_of:
  - actors: [user]
    context: project
    permissions: [project.resources.read]

  - actors: [project]
    context: project
    project_token:
      audience: management
      binding: project
    permissions: [project.resources.read]

  - actors: [project]
    caller_services:
      - issuer: internal
        actor_id: 1
    context: project
    project_token:
      audience: runtime
      binding: project
    permissions: [project.resources.read]
```

The alternatives mean:

1. A User subject has the Project context and `project.resources.read`.
2. A direct Project credential has management proof, a Project-bound tenant, and the permission.
3. A Project subject delegated through the verified internal Service caller has runtime-minimum
   proof, a Project-bound tenant, and the same permission.

The third path does not grant permission to the internal Service. The Project remains the Principal
subject. `caller_services` constrains the immediate verified Service that delegated that subject.

Use `any_of` only when legitimate callers need different complete proof or caller constraints. A
single `permission` policy is clearer when all accepted callers share the same constraints.

## Why Web API policy is different

Web API is the external credential edge. Assistant, Endpoint, and Integration are downstream
service boundaries.

At Web API, a Project API key resolves to a direct Project Principal. A management route can
therefore use one policy:

```yaml
authorization: permission
actors: [user, project]
context: project
project_token:
  audience: management
  binding: project
permissions: [project.resources.read]
```

When Web API calls a downstream service, it sends a signed internal Service assertion containing
the original subject, tenant, and permissions. The downstream resolver creates a delegated
Principal:

```text
Subject: original User or Project
Actor:   verified internal Service
Proof:   internal Service assertion
```

A delegated Project subject no longer has direct management proof. Downstream policies therefore
separate the direct Project path from the delegated Project path with `any_of`. The delegated path
requires the exact internal caller and runtime-minimum proof.

The same RPC may appear in both `web-api.yaml` and a downstream policy because each file protects a
different network boundary:

```text
client -> Web API policy -> signed delegation -> downstream service policy
```

Web API policy is authoritative for routes served by Web API. Each downstream policy is
authoritative for routes served by that deployable. Permission intent should remain aligned, while
proof and immediate-caller requirements may differ.

## Adding or changing a route

1. Edit only the YAML owned by the deployable that registers the route.
2. Use the exact gRPC full method or HTTP method and registered path template.
3. Choose the smallest complete authorization mode.
4. Add `project_token` whenever a branch accepts a Project subject.
5. Add `caller_services` when authority depends on a verified immediate Service caller.
6. Keep the route `disabled` when tenant binding, caller authority, or secret handling is not proven.
7. Run the policy and owning router tests. They compare the registered route inventory with the
   policy and reject missing or stale entries.

Policy files contain no credentials, signing keys, raw tokens, role definitions, or database
configuration.

## Verification

```bash
go test ./pkg/authentication/policy
go test ./api/web-api/router ./api/assistant-api/router ./api/endpoint-api/router ./api/integration-api/router
```
