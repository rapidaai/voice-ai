export const Permission = {
  OrganizationRead: 'organization.read',
  OrganizationUpdate: 'organization.update',
  OrganizationDelete: 'organization.delete',
  OrganizationOwnershipManage: 'organization.ownership.manage',
  OrganizationBillingRead: 'organization.billing.read',
  OrganizationBillingManage: 'organization.billing.manage',
  OrganizationSecurityRead: 'organization.security.read',
  OrganizationSecurityManage: 'organization.security.manage',
  OrganizationSessionsRevoke: 'organization.sessions.revoke',
  OrganizationMembersRead: 'organization.members.read',
  OrganizationMembersInvite: 'organization.members.invite',
  OrganizationMembersRemove: 'organization.members.remove',
  OrganizationProjectsRead: 'organization.projects.read',
  OrganizationProjectsCreate: 'organization.projects.create',
  OrganizationProjectsArchive: 'organization.projects.archive',
  OrganizationRolesRead: 'organization.roles.read',
  OrganizationRolesCreate: 'organization.roles.create',
  OrganizationRolesUpdate: 'organization.roles.update',
  OrganizationRolesArchive: 'organization.roles.archive',
  OrganizationAccessRead: 'organization.access.read',
  OrganizationAccessManage: 'organization.access.manage',
  OrganizationProjectAccessTokensCreate:
    'organization.project_access_tokens.create',
  OrganizationProjectAccessTokensReadMetadata:
    'organization.project_access_tokens.read_metadata',
  OrganizationProjectAccessTokensReduceAccess:
    'organization.project_access_tokens.reduce_access',
  OrganizationProjectAccessTokensRotate:
    'organization.project_access_tokens.rotate',
  OrganizationProjectAccessTokensRevoke:
    'organization.project_access_tokens.revoke',
  OrganizationProviderCredentialsReadMetadata:
    'organization.provider_credentials.read_metadata',
  OrganizationProviderCredentialsCreate:
    'organization.provider_credentials.create',
  OrganizationProviderCredentialsUpdate:
    'organization.provider_credentials.update',
  OrganizationProviderCredentialsRotate:
    'organization.provider_credentials.rotate',
  OrganizationProviderCredentialsRevoke:
    'organization.provider_credentials.revoke',
  OrganizationAuditRead: 'organization.audit.read',
  OrganizationAuditExport: 'organization.audit.export',

  ProjectRead: 'project.read',
  ProjectUpdate: 'project.update',
  ProjectArchive: 'project.archive',
  ProjectRolesRead: 'project.roles.read',
  ProjectAccessRead: 'project.access.read',
  ProjectAccessManage: 'project.access.manage',
  ProjectResourcesRead: 'project.resources.read',
  ProjectResourcesCreate: 'project.resources.create',
  ProjectResourcesUpdate: 'project.resources.update',
  ProjectResourcesDelete: 'project.resources.delete',
  ProjectDeploymentsRead: 'project.deployments.read',
  ProjectDeploymentsManage: 'project.deployments.manage',
  ProjectRuntimeInvoke: 'project.runtime.invoke',
  ProjectConversationsRead: 'project.conversations.read',
  ProjectConversationsDelete: 'project.conversations.delete',
  ProjectConversationsExport: 'project.conversations.export',
  ProjectObservabilityRead: 'project.observability.read',
  ProjectObservabilityExport: 'project.observability.export',
  ProjectAccessTokensCreate: 'project.access_tokens.create',
  ProjectAccessTokensReadMetadata: 'project.access_tokens.read_metadata',
  ProjectAccessTokensReduceAccess: 'project.access_tokens.reduce_access',
  ProjectAccessTokensRotate: 'project.access_tokens.rotate',
  ProjectAccessTokensRevoke: 'project.access_tokens.revoke',
  ProjectProviderCredentialsReadMetadata:
    'project.provider_credentials.read_metadata',
  ProjectProviderCredentialsCreate: 'project.provider_credentials.create',
  ProjectProviderCredentialsUpdate: 'project.provider_credentials.update',
  ProjectProviderCredentialsRotate: 'project.provider_credentials.rotate',
  ProjectProviderCredentialsRevoke: 'project.provider_credentials.revoke',
  ProjectProviderCredentialsUse: 'project.provider_credentials.use',
  ProjectAuditRead: 'project.audit.read',
  ProjectAuditExport: 'project.audit.export',
} as const;

export type PermissionKey = (typeof Permission)[keyof typeof Permission];
