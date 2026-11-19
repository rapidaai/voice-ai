import { useContext } from 'react';

import { AuthContext } from '@/context/auth-context';

import type { AccessRequirement } from './console-access';

export type AccessDecision = 'loading' | 'allowed' | 'denied';
export type AccessProjectionStatus = 'loading' | 'ready';

export function useAccess(
  requirement: AccessRequirement,
  projectionStatus: AccessProjectionStatus,
): AccessDecision {
  const { getCurrentOrganization, getCurrentProject } = useContext(AuthContext);

  if (projectionStatus === 'loading') {
    return 'loading';
  }

  for (const alternative of requirement.anyOf) {
    const roles =
      alternative.scope === 'organization'
        ? getCurrentOrganization?.()?.rolesList
        : getCurrentProject?.()?.rolesList;

    if (!roles) {
      continue;
    }

    const grantedPermissions = new Set(
      roles.flatMap(role =>
        role.permissionsList.map(permission => permission.key),
      ),
    );
    if (
      alternative.permissions.every(permission =>
        grantedPermissions.has(permission),
      )
    ) {
      return 'allowed';
    }
  }

  return 'denied';
}
