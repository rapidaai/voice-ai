import { useContext } from 'react';
import { ResourceRole } from '@/models/common';
import { AuthContext } from '@/context/auth-context';

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === null) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

/**
 * getting credentials for current context user
 * @returns
 */
export const useCredential = () => {
  const { authentication, token, getCurrentProject, getCurrentOrganization } =
    useAuth();
  const currentProject = getCurrentProject?.();
  const currentOrganization = getCurrentOrganization?.();
  return [
    authentication?.user?.id || '',
    token && token.token ? token.token : '',
    currentProject?.id || '',
    currentOrganization?.id || '',
  ];
};

/**
 *
 * @returns
 */
export const useCurrentCredential = () => {
  const { authentication, token, getCurrentProject, getCurrentOrganization } =
    useAuth();
  const currentProject = getCurrentProject?.();
  const currentOrganization = getCurrentOrganization?.();
  return {
    user: authentication?.user,
    authId: authentication?.user?.id || '',
    token: token && token.token ? token.token : '',
    projectId: currentProject?.id || '',
    organizationId: currentOrganization?.id || '',
  };
};

/**
 *
 * @returns
 */
export const useResourceRole = (resource: {
  getProjectid: () => string;
  getOrganizationid: () => string;
  getCreatedby: () => string;
}): ResourceRole => {
  /**
   *
   */
  const { authentication, getProjects, getCurrentOrganization } = useAuth();

  const userId = authentication?.user?.id;
  const orgId = getCurrentOrganization?.()?.id;
  const projectIds = getProjects?.().map(project => project.id);
  if (!userId || !projectIds || !orgId) return ResourceRole.anyone;
  if (userId === resource.getCreatedby()) {
    return ResourceRole.owner;
  } else if (projectIds.includes(resource.getProjectid())) {
    return ResourceRole.projectMember;
  } else if (orgId === resource.getOrganizationid()) {
    return ResourceRole.organizationMember;
  }
  return ResourceRole.anyone;
};
