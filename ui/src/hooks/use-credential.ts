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
  const { currentUser, token, currentProjectRole, organizationRole } =
    useAuth();
  return [
    currentUser && currentUser.id ? currentUser.id : '',
    token && token.token ? token.token : '',
    currentProjectRole && currentProjectRole.projectid
      ? currentProjectRole.projectid
      : '',
    organizationRole && organizationRole.organizationid
      ? organizationRole.organizationid
      : '',
  ];
};

/**
 *
 * @returns
 */
export const useCurrentCredential = () => {
  const { currentUser, token, currentProjectRole, organizationRole } =
    useAuth();
  return {
    user: currentUser,
    authId: currentUser && currentUser.id ? currentUser.id : '',
    token: token && token.token ? token.token : '',
    projectId:
      currentProjectRole && currentProjectRole.projectid
        ? currentProjectRole.projectid
        : '',
    organizationId:
      organizationRole && organizationRole.organizationid
        ? organizationRole.organizationid
        : '',
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
  const { currentUser, projectRoles, organizationRole } = useAuth();

  const userId = currentUser && currentUser.id;
  const orgId = organizationRole && organizationRole.organizationid;
  const projectIds = projectRoles?.map(x => x.projectid);
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
