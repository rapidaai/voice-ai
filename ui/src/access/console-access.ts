import { Permission, type PermissionKey } from './permissions';

export type AccessScope = 'organization' | 'project';

export type ScopedAccessRequirement = {
  scope: AccessScope;
  permissions: readonly [PermissionKey, ...PermissionKey[]];
};

export type AccessRequirement = {
  anyOf: readonly [ScopedAccessRequirement, ...ScopedAccessRequirement[]];
};

const organizationAccess = (
  permission: PermissionKey,
  ...permissions: PermissionKey[]
): ScopedAccessRequirement => ({
  scope: 'organization',
  permissions: [permission, ...permissions],
});

const projectAccess = (
  permission: PermissionKey,
  ...permissions: PermissionKey[]
): ScopedAccessRequirement => ({
  scope: 'project',
  permissions: [permission, ...permissions],
});

const allowAny = (
  requirement: ScopedAccessRequirement,
  ...requirements: ScopedAccessRequirement[]
): AccessRequirement => ({
  anyOf: [requirement, ...requirements],
});

export const ConsoleAccess = {
  organization: {
    read: allowAny(organizationAccess(Permission.OrganizationRead)),
    update: allowAny(organizationAccess(Permission.OrganizationUpdate)),
    delete: allowAny(organizationAccess(Permission.OrganizationDelete)),
    ownership: {
      manage: allowAny(
        organizationAccess(Permission.OrganizationOwnershipManage),
      ),
    },
    billing: {
      read: allowAny(organizationAccess(Permission.OrganizationBillingRead)),
      manage: allowAny(
        organizationAccess(Permission.OrganizationBillingManage),
      ),
    },
    security: {
      read: allowAny(organizationAccess(Permission.OrganizationSecurityRead)),
      manage: allowAny(
        organizationAccess(Permission.OrganizationSecurityManage),
      ),
      revokeSessions: allowAny(
        organizationAccess(Permission.OrganizationSessionsRevoke),
      ),
    },
    members: {
      read: allowAny(organizationAccess(Permission.OrganizationMembersRead)),
      invite: allowAny(
        organizationAccess(
          Permission.OrganizationMembersInvite,
          Permission.OrganizationAccessManage,
        ),
      ),
      remove: allowAny(
        organizationAccess(Permission.OrganizationMembersRemove),
      ),
    },
    projects: {
      read: allowAny(organizationAccess(Permission.OrganizationProjectsRead)),
      create: allowAny(
        organizationAccess(Permission.OrganizationProjectsCreate),
      ),
      archive: allowAny(
        organizationAccess(Permission.OrganizationProjectsArchive),
      ),
    },
    roles: {
      read: allowAny(organizationAccess(Permission.OrganizationRolesRead)),
      create: allowAny(organizationAccess(Permission.OrganizationRolesCreate)),
      update: allowAny(organizationAccess(Permission.OrganizationRolesUpdate)),
      archive: allowAny(
        organizationAccess(Permission.OrganizationRolesArchive),
      ),
    },
    access: {
      read: allowAny(organizationAccess(Permission.OrganizationAccessRead)),
      manage: allowAny(organizationAccess(Permission.OrganizationAccessManage)),
    },
    audit: {
      read: allowAny(organizationAccess(Permission.OrganizationAuditRead)),
      export: allowAny(organizationAccess(Permission.OrganizationAuditExport)),
    },
  },
  project: {
    read: allowAny(projectAccess(Permission.ProjectRead)),
    update: allowAny(projectAccess(Permission.ProjectUpdate)),
    archive: allowAny(projectAccess(Permission.ProjectArchive)),
    roles: {
      read: allowAny(projectAccess(Permission.ProjectRolesRead)),
    },
    access: {
      read: allowAny(projectAccess(Permission.ProjectAccessRead)),
      manage: allowAny(projectAccess(Permission.ProjectAccessManage)),
    },
    resources: {
      read: allowAny(projectAccess(Permission.ProjectResourcesRead)),
      create: allowAny(projectAccess(Permission.ProjectResourcesCreate)),
      update: allowAny(projectAccess(Permission.ProjectResourcesUpdate)),
      delete: allowAny(projectAccess(Permission.ProjectResourcesDelete)),
    },
    deployments: {
      read: allowAny(projectAccess(Permission.ProjectDeploymentsRead)),
      manage: allowAny(projectAccess(Permission.ProjectDeploymentsManage)),
    },
    runtime: {
      invoke: allowAny(projectAccess(Permission.ProjectRuntimeInvoke)),
    },
    conversations: {
      read: allowAny(projectAccess(Permission.ProjectConversationsRead)),
      delete: allowAny(projectAccess(Permission.ProjectConversationsDelete)),
      export: allowAny(projectAccess(Permission.ProjectConversationsExport)),
    },
    observability: {
      read: allowAny(projectAccess(Permission.ProjectObservabilityRead)),
      export: allowAny(projectAccess(Permission.ProjectObservabilityExport)),
    },
    audit: {
      read: allowAny(projectAccess(Permission.ProjectAuditRead)),
      export: allowAny(projectAccess(Permission.ProjectAuditExport)),
    },
  },
  projectAccessTokens: {
    create: allowAny(
      organizationAccess(Permission.OrganizationProjectAccessTokensCreate),
      projectAccess(Permission.ProjectAccessTokensCreate),
    ),
    readMetadata: allowAny(
      organizationAccess(
        Permission.OrganizationProjectAccessTokensReadMetadata,
      ),
      projectAccess(Permission.ProjectAccessTokensReadMetadata),
    ),
    reduceAccess: allowAny(
      organizationAccess(
        Permission.OrganizationProjectAccessTokensReduceAccess,
      ),
      projectAccess(Permission.ProjectAccessTokensReduceAccess),
    ),
    rotate: allowAny(
      organizationAccess(Permission.OrganizationProjectAccessTokensRotate),
      projectAccess(Permission.ProjectAccessTokensRotate),
    ),
    revoke: allowAny(
      organizationAccess(Permission.OrganizationProjectAccessTokensRevoke),
      projectAccess(Permission.ProjectAccessTokensRevoke),
    ),
  },
  providerCredentials: {
    readMetadata: allowAny(
      organizationAccess(
        Permission.OrganizationProviderCredentialsReadMetadata,
      ),
      projectAccess(Permission.ProjectProviderCredentialsReadMetadata),
    ),
    create: allowAny(
      organizationAccess(Permission.OrganizationProviderCredentialsCreate),
      projectAccess(Permission.ProjectProviderCredentialsCreate),
    ),
    update: allowAny(
      organizationAccess(Permission.OrganizationProviderCredentialsUpdate),
      projectAccess(Permission.ProjectProviderCredentialsUpdate),
    ),
    rotate: allowAny(
      organizationAccess(Permission.OrganizationProviderCredentialsRotate),
      projectAccess(Permission.ProjectProviderCredentialsRotate),
    ),
    revoke: allowAny(
      organizationAccess(Permission.OrganizationProviderCredentialsRevoke),
      projectAccess(Permission.ProjectProviderCredentialsRevoke),
    ),
    use: allowAny(projectAccess(Permission.ProjectProviderCredentialsUse)),
  },
} as const;
