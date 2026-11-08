# End-to-End Flows

Flows are grouped first by user journey and then by public client. Every client directory has its own executable `run.sh`, fixture setup, assertions, and cleanup so it can run independently.

## Flow contract

`manifest.txt` is the source of truth for journey inclusion and execution order. A journey owns only its display name and supported clients in `<journey>/run.sh`. Each `<journey>/<client>/run.sh` owns that client's fixture lifecycle and invokes the public client implementation. Shared fixture code belongs in `project-fixture.sh` only when multiple journeys require the same pre-existing project. Onboarding and project lifecycle fixtures remain local because project creation is the behavior under test.

Use this order in every client runner:

1. Export deterministic fixture values and configurable service or database hosts.
2. Source `process-lifecycle.sh` and register the client cleanup function.
3. Clean stale fixture state, then seed prerequisites.
4. Exercise SDK commands through `run_flow_process` with a bounded timeout, or
   bound every REST request directly.
5. Assert the public response in the client implementation.
6. Assert persistent side effects from the runner when the response alone is insufficient.
7. Let the registered cleanup remove all fixture state.

Fixture identifiers must be flow-specific and repeatable. Use reserved `.invalid` email addresses, never real recipients. Database access must use `${FLOW_POSTGRES_HOST:-postgres}`. Service endpoints must retain their existing environment override. A test must not depend on data created by another flow or leave persistent data behind.

To add or extend a flow:

1. Add the journey directory and its standard journey `run.sh`.
2. Add one directory per supported client with an executable `run.sh`.
3. Add the journey to `manifest.txt` in its required execution order.
4. Update the matrix below and document any intentional API limitation.
5. Run `just ci-flow-contract`, then the affected client suite.

`just ci-flow-contract` enforces the manifest, React coverage for every journey, standard journey declaration, declared client directories, cleanup traps, bounded calls, configurable PostgreSQL host, valid JavaScript, unique numeric fixture identifiers, reserved `.invalid` email addresses, and shared fixture execution order. Regression cases verify that malformed flows fail the contract. It runs as part of `just ci-check` so structural drift fails CI.

Current matrix:

| Flow | Node.js SDK | React SDK | REST |
| --- | --- | --- | --- |
| Sign in | Yes | Yes | Yes |
| Reject invalid credentials, unauthorized project access, missing permission, and invalid identifiers | Not available | Yes | Yes |
| Sign up, create organization, create project | Yes | Yes | Not available |
| Create, update, and add user to project | Not available | Yes | Not available |
| Create assistant | Yes | Yes | Yes |
| Create, get, list, update, version, and archive assistant | Yes | Yes | Not available |
| Create, get, list, update, and archive assistant tools and configurations | Yes | Yes | Not available |
| Create model, AgentKit, WebSocket, and AgentFlow providers | Yes | Yes | Yes |
| Create, verify, version, and deploy endpoint | Not available | Yes | Not available |
| Create vault credential | Yes | Yes | Not available |
| Create assistant, phone deployment, and outbound call | Yes | Yes | Yes |

Unless noted below, every assistant flow runs once with a personal access token and once with the project's `x-api-key`. SDK clients use `ConnectionConfig.WithPersonalToken` and `ConnectionConfig.WithSDK`; REST clients send the corresponding authentication headers directly.

The authorization boundary flow rejects invalid personal tokens and project API keys, cross-project access, users without project permission, and invalid or nonexistent resource identifiers. React exercises these cases through the released SDK and REST verifies the corresponding HTTP status contract. Both variants assert that rejected writes create no assistant records.

The Node assistant lifecycle covers both authentication modes. Its React variant uses a personal access token because the current web proxy rejects project API-key scope for provider listing, detail updates, version updates, and deletion.

The assistant components flow covers tools, storage, analysis, authentication, webhook, and telemetry. Node covers tools with both authentication modes; configuration operations and the React web proxy currently allow personal access tokens only.

The endpoint flow uses a personal access token because endpoint creation does not currently allow the SDK authentication scope.

The REST provider flow creates one assistant per provider because the REST surface accepts providers during assistant creation. The SDK flows additionally verify attaching providers to an existing assistant.

The onboarding flow has no REST variant because organization and project creation are currently exposed through gRPC and gRPC-web only.

Vault credential updates are not included because the current Vault service and released SDKs do not expose an update operation. They expose create, get/list, and archive/delete operations only.

Run all flows locally with:

```sh
just flows
```

Run one SDK integration suite against services already listening on the local
development ports:

```sh
just test-integration-node
just test-integration-react
```

Pass a manifest journey name to run only that journey:

```sh
just test-integration-node assistant-components
just test-integration-react assistant-components
```

Local SDK dependencies are installed into private immutable directories and
atomically published by the flow lockfile checksum. Interrupted installations
never become the active cache entry. Set `FLOW_DEPENDENCY_CACHE_DIR` to choose a
different private cache directory. React SDK request and response logging is
disabled by default; set `FLOW_SDK_DEBUG=true` only when the full trace is
required. The Node SDK's similarly named constructor flag selects insecure
local gRPC transport and must remain enabled for the plaintext development ports.

Set `FLOW_POSTGRES_HOST`, `WEB_API_ENDPOINT`, `ASSISTANT_API_ENDPOINT`,
`WEB_API_REACT_ENDPOINT`, and `ASSISTANT_API_REACT_ENDPOINT` to override the
localhost defaults provided by these recipes. When the assistant API runs in a
container, set `FLOW_ARI_HOST` to a host address that the container can reach.

CI runs each client independently with:

```sh
just ci-flows-node
just ci-flows-react
just ci-flows-rest
```

`just ci-flows` remains the combined local CI entry point. In GitHub Actions,
each client suite is divided into two isolated shards. `FLOW_SHARD_INDEX` and
`FLOW_SHARD_TOTAL` select a deterministic manifest subset and default to `1`.
Flow/client combinations are shown in collapsible log groups, unsupported
clients are reported explicitly, durations appear in the result table, and
every shard uploads its report artifact. After all shards finish, the summary
job publishes one table containing every flow/client status and duration, then
uploads the combined table. On same-repository pull requests, the job also
creates or updates one PR comment containing the table. Fork pull requests keep
the Actions summary because GitHub does not grant their token comment-write
permission. Failures also emit an annotation with the exact flow and client.
