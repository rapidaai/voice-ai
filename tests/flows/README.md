# End-to-End Flows

Flows are grouped first by user journey and then by public client. Every client directory has its own executable `run.sh`, fixture setup, assertions, and cleanup so it can run independently.

Current matrix:

| Flow | Node.js SDK | React SDK | REST |
| --- | --- | --- | --- |
| Sign in | Yes | Yes | Yes |
| Sign up, create organization, create project | Yes | Yes | Not available |
| Create assistant | Yes | Yes | Not available |
| Create model, AgentKit, WebSocket, and AgentFlow providers | Yes | Yes | Not available |
| Create vault credential | Yes | Yes | Not available |

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
