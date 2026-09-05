# End-to-End Flows

Flows are grouped first by user journey and then by public client. Every client directory has its own executable `run.sh`, fixture setup, assertions, and cleanup so it can run independently.

Current matrix:

| Flow | Node.js SDK | React SDK | REST |
| --- | --- | --- | --- |
| Sign in | Yes | Yes | Yes |
| Sign up, create organization, create project | Yes | Yes | Not available |
| Create assistant | Yes | Yes | Yes |
| Create model, AgentKit, WebSocket, and AgentFlow providers | Yes | Yes | Yes |
| Create vault credential | Yes | Yes | Not available |
| Create assistant, phone deployment, and outbound call | Yes | Yes | Yes |

Every assistant flow runs once with a personal access token and once with the project's `x-api-key`. SDK clients use `ConnectionConfig.WithPersonalToken` and `ConnectionConfig.WithSDK`; REST clients send the corresponding authentication headers directly.

The REST provider flow creates one assistant per provider because the REST surface accepts providers during assistant creation. The SDK flows additionally verify attaching providers to an existing assistant.

The onboarding flow has no REST variant because organization and project creation are currently exposed through gRPC and gRPC-web only.

Vault credential updates are not included because the current Vault service and released SDKs do not expose an update operation. They expose create, get/list, and archive/delete operations only.

Run all flows locally with:

```sh
just flows
```

CI uses the equivalent command:

```sh
just ci-flows
```

In GitHub Actions, each flow/client combination is shown in a collapsible log group. Failures emit an error annotation with the exact flow and client, and the job summary includes a result table from `flow-reports/flows.md`.
