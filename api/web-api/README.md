# Web API permission map

This document maps the public Web API surface to the permission that must protect each operation
when endpoint enforcement is implemented. It covers the gRPC and REST services registered in
`router/web.go`.

## Current status

The permissions below are present in `permission_definitions`, but they are not yet evaluated by
these endpoints. Most handlers currently validate only authentication type and organization or
project scope. Access-management RPCs temporarily require an active legacy Owner or Admin
membership, except that an authenticated user with Organization context may list roles only with
the exact Project plus `PROJECT_TOKEN` selector filters and may read an active eligible role
returned by that selector. This narrow exception supports project-credential creation and does not
permit broader catalog reads or any mutation.

This matrix is therefore an enforcement inventory, not evidence of active protection. Each endpoint
cutover still requires the separately accepted authorization work described by RFC 0026. Until that
cutover is complete, existing production authorization remains authoritative.

Console and API access use the same operation permission:

- a `USER` role authorizes a user session, including requests made by the console;
- a `PROJECT_TOKEN` role authorizes project-token API requests;
- UI visibility is presentation only and never replaces the backend permission check;
- Organization permissions are user-only in Version 1.

`Service/Method` below is the canonical gRPC operation name. A `+` means every listed condition is
required. `scope-selected` means the request or target role determines whether the Organization or
Project permission applies.

## Authentication and user self-service

These operations establish authentication or affect only the authenticated user's own account.
They are outside the role-permission catalog and retain their dedicated authentication checks.

| API | Authorization boundary |
| --- | --- |
| `AuthenticationService/Authenticate`, `RegisterUser`, `Linkedin`, `Google`, `Github` | Public authentication flow |
| `AuthenticationService/ForgotPassword`, `CreatePassword` | Dedicated one-time-token flow |
| `AuthenticationService/VerifyToken` | Internal Service credential resolution |
| `AuthenticationService/Authorize` | Existing authenticated-context validation |
| `AuthenticationService/GetUser`, `UpdateUser`, `ChangePassword` | Authenticated user acting on self |
| `NotificationService/GetNotificationSettting`, `UpdateNotificationSetting` | Authenticated user acting on self |
| `OrganizationService/CreateOrganization` | User-only first-organization onboarding contract |

`AuthenticationService/GetAllUser` is not self-service and requires
`organization.members.read`.

## Organization and membership APIs

| API | Required permission |
| --- | --- |
| `OrganizationService/GetOrganization` | `organization.read` |
| `OrganizationService/UpdateOrganization` | `organization.update` |
| `OrganizationService/UpdateBillingInformation` | `organization.billing.manage` |
| `AuthenticationService/GetAllUser` | `organization.members.read` |
| `OrganizationService/InviteUserToOrganization` | `organization.members.invite` + access-management and target-role permission ceilings for every requested Organization or Project role |
| `OrganizationService/UpdateUserOrganizationRole` | `organization.access.manage` + every permission in the target role |
| `OrganizationService/DeleteUserFromOrganization` | `organization.members.remove` + the applicable protected-role and removal ceilings |
| `ProjectService/GetAllProject` | `organization.projects.read` |
| `ProjectService/CreateProject` | `organization.projects.create` |

No registered Web API currently maps to `organization.delete`,
`organization.ownership.manage`, `organization.billing.read`, `organization.security.read`,
`organization.security.manage`, `organization.sessions.revoke`, or
`organization.projects.archive`.

## Project and access APIs

| API | Required permission |
| --- | --- |
| `ProjectService/GetProject` | `project.read` |
| `ProjectService/UpdateProject` | `project.update` |
| `ProjectService/ArchiveProject` | `project.archive` |
| `ProjectService/AddUserToProjects` | `project.access.manage` in every target Project + every permission in each target role |
| `ProjectService/DeleteUserFromProject` | `project.access.manage` + the applicable protected-role and removal ceilings |
| `ProjectService/CreateProjectCredential` | `organization.project_access_tokens.create` or `project.access_tokens.create` |
| `ProjectService/GetAllProjectCredential` | `organization.project_access_tokens.read_metadata` or `project.access_tokens.read_metadata` |
| `ProjectService/RevokeProjectCredential` | `organization.project_access_tokens.revoke` or `project.access_tokens.revoke` |

Every new project credential must select one active Project-scoped `PROJECT_TOKEN` role. The
credential stores that immutable role ID, while migration `000016` assigns the built-in
`project_token_full_access` compatibility role to credentials that existed before the migration.
The server never uses that compatibility role as a default for a new credential. An active
credential reference prevents archival of its custom role; archived credential references do not.
Credential create and list responses continue returning keys as before. List responses include the
assigned role ID and current role name, and new consumers use explicit pagination and name-or-role
search.

The current `ProjectService` has no rotate or reduce-access RPC. Revoke is temporarily restricted to
active Organization owners and admins until permission enforcement is activated. Future token
authority changes also require `project.access.manage` and the permission ceiling defined by
RFC 0026.

## Role catalog APIs

| API | Required permission |
| --- | --- |
| `AccessManagementService/ListPermissions`, `ListRoles`, `GetRole` | `organization.roles.read` for Organization scope; `project.roles.read` for Project scope. Before enforcement, non-admin credential creators have only the exact Project plus `PROJECT_TOKEN` selector read exception described above. |
| `AccessManagementService/CreateCustomRole` | `organization.roles.create` + the creation ceiling defined by RFC 0026 |
| `AccessManagementService/UpdateCustomRole`, `ReduceCustomRolePermissions` | `organization.roles.update` in the definition's owning Organization |
| `AccessManagementService/ArchiveCustomRole` | `organization.roles.archive` in the definition's owning Organization |

Reusable Project role definitions remain Organization-owned. `project.roles.read` permits reading
eligible definitions in a concrete Project but does not permit changing them.

## Provider credential and connector APIs

All currently registered vault operations resolve a Project context.

| API | Required permission |
| --- | --- |
| `VaultService/CreateProviderCredential` | `project.provider_credentials.create` |
| `VaultService/GetAllOrganizationCredential` | `project.provider_credentials.read_metadata` |
| `VaultService/DeleteCredential` | `project.provider_credentials.revoke` |
| `ConnectService/GeneralConnect` and OAuth connector callbacks under `/v1/connect-*` | `project.provider_credentials.create` |
| `ConnectService/GetConnectorFiles` | `project.provider_credentials.use` |

`VaultService/GetCredential` and `GetOauth2Credential` currently return credential values. They
cannot be mapped safely: Version 1 intentionally defines metadata-read and use permissions but no
secret-read permission. They must not be cut over under `read_metadata` or `use` while returning a
secret to the caller.

No registered Web API currently owns Organization-scoped provider credential lifecycle operations
or Project credential update and rotate operations.

## Project resources

The current catalog treats assistants, endpoints, knowledge bases, tools, configurations, tags,
and provider-model selections as Project resources.

| Required permission | APIs |
| --- | --- |
| `project.resources.read` | `AssistantService/GetAssistant`, `GetAllAssistant`, `GetAllAssistantProvider`, `GetAssistantConfiguration`, `GetAllAssistantConfiguration`, `GetAssistantTool`, `GetAllAssistantTool`, `GetAssistantKnowledge`, `GetAllAssistantKnowledge` |
| `project.resources.create` | `AssistantService/CreateAssistant`, `CreateAssistantProvider`, `CreateAssistantTag`, `CreateAssistantConfiguration`, `CreateAssistantTool`, `CreateAssistantKnowledge` |
| `project.resources.update` | `AssistantService/UpdateAssistantVersion`, `UpdateAssistantDetail`, `UpdateAssistantConfiguration`, `UpdateAssistantTool`, `UpdateAssistantKnowledge` |
| `project.resources.delete` | `AssistantService/DeleteAssistant`, `DeleteAssistantConfiguration`, `DeleteAssistantTool`, `DeleteAssistantKnowledge` |
| `project.resources.read` | `EndpointService/GetEndpoint`, `GetAllEndpoint`, `GetAllEndpointProviderModel` |
| `project.resources.create` | `EndpointService/CreateEndpoint`, `ForkEndpoint` |
| `project.resources.update` | `EndpointService/UpdateEndpointVersion`, `CreateEndpointProviderModel`, `CreateEndpointCacheConfiguration`, `CreateEndpointRetryConfiguration`, `CreateEndpointTag`, `UpdateEndpointDetail` |
| `project.resources.read` | `KnowledgeService/GetKnowledge`, `GetAllKnowledge`, `GetAllKnowledgeDocument`, `GetAllKnowledgeDocumentSegment` |
| `project.resources.create` | `KnowledgeService/CreateKnowledge`, `CreateKnowledgeTag`, `CreateKnowledgeDocument` |
| `project.resources.update` | `KnowledgeService/UpdateKnowledgeDetail`, `UpdateKnowledgeDocumentSegment`, `DocumentService/IndexKnowledgeDocument` |
| `project.resources.delete` | `KnowledgeService/DeleteKnowledgeDocumentSegment` |

The generic resource permissions do not distinguish assistants, endpoints, knowledge bases, tools,
or their child objects. That breadth is part of the current catalog and must be treated as one
authority boundary during enforcement.

## Deployments and runtime

| API | Required permission |
| --- | --- |
| Every `AssistantDeploymentService/Get*` and `GetAll*` RPC | `project.deployments.read` |
| Every `AssistantDeploymentService/Create*` and `Disable*` RPC | `project.deployments.manage` |
| `Deployment/Invoke` | `project.runtime.invoke` |

`Deployment/Invoke` is the Web API runtime entry point used for deployed execution. It currently
accepts user, project, or service authentication with a valid scope and does not evaluate
`project.runtime.invoke`.

`Deployment/Probe` and `Deployment/Update` are unimplemented and have no active permission mapping.
Runtime endpoints outside Web API require their own service-specific cutover inventory before token
enforcement.

## Conversations, observability, usage, and audit

| API | Required permission |
| --- | --- |
| `AssistantService/GetAllAssistantMessage`, `GetAllConversationMessage`, `GetAllMessage`, `GetAllAssistantConversation`, `GetAssistantConversation` | `project.conversations.read` |
| `AssistantService/GetAssistantDashboard` | `project.observability.read` |
| `AssistantService/GetAssistantHTTPLog`, `GetAllAssistantHTTPLog`, `GetAssistantToolLog`, `GetAllAssistantToolLog` | `project.observability.read` |
| `AssistantService/RetryAssistantHTTPLog` | `project.observability.read` + `project.resources.update` |
| `EndpointService/GetAllEndpointLog`, `GetEndpointLog` | `project.observability.read` |
| `KnowledgeService/GetAllKnowledgeLog`, `GetKnowledgeLog` | `project.observability.read` |
| `ObservabilityService/GetAllTelemetry` | `project.observability.read` |
| `DashboardService/GetProjectDashboard` | `project.observability.read` |
| `ProductUsageService/GetProductUsages` | `project.observability.read` |
| `ProductUsageService/GetOrganizationUsages` | `organization.billing.read` |
| `AuditLoggingService/GetAuditLog`, `GetAllAuditLog` | `project.audit.read` |

No registered Web API currently maps to conversation delete/export, observability export, Project
audit export, or Organization audit read/export permissions.

`ProductUsageService/CreateProductUsage` is an internal write and must be restricted through the
future service-actor compatibility contract, not granted through a User or Project Token role.
`AuditLoggingService/CreateMetadata` is unimplemented.

## Enforcement prerequisites and gaps

Before any row in this document becomes active authorization:

1. The endpoint must declare and evaluate its required permission on the backend.
2. User and Project Token assignments must exist for the concrete Organization or Project.
3. Project-token endpoints must also enforce token audience and resource binding.
4. Internal service delegation must preserve the originating actor, scope, permissions, and token
   constraints under a separately accepted compatibility contract.
5. Secret-returning credential RPCs require contract correction because the accepted catalog has no
   secret-read permission.
6. Denials must be tested for both user sessions and project tokens wherever both subjects are
   eligible.

The permission catalog is seeded by `migrations/000014_create_role_catalog.up.sql`. RFC 0026 remains
the authoritative Version 1 policy; this README is the Web API operation inventory used to plan
later endpoint cutovers.
