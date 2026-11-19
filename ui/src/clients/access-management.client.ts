import {
  ArchiveCustomRole,
  ArchiveCustomRoleRequest,
  Criteria,
  CreateCustomRole,
  CreateCustomRoleRequest,
  GetRole,
  GetRoleRequest,
  ListPermissions,
  ListPermissionsRequest,
  ListRoles,
  ListRolesRequest,
  Paginate,
  ReduceCustomRolePermissions,
  ReduceCustomRolePermissionsRequest,
  RoleDefinitionType,
  RoleScope,
  RoleSubject,
  UpdateCustomRole,
  UpdateCustomRoleRequest,
} from '@rapidaai/react';
import { withConnection } from './connection';

export type AccessManagementAuth = {
  organizationId: string;
  token: string;
  userId: string;
};

export type AccessManagementCriteria = {
  key: string;
  value: string;
  logic: string;
};

export const listAccessPermissions = ({
  page,
  pageSize,
  criteria,
  auth,
}: {
  page: number;
  pageSize: number;
  criteria: AccessManagementCriteria[];
  auth: AccessManagementAuth;
}) => {
  const request = new ListPermissionsRequest();
  const paginate = new Paginate();
  paginate.setPage(page);
  paginate.setPagesize(pageSize);
  request.setPaginate(paginate);
  criteria.forEach(({ key, value, logic }) => {
    const filter = new Criteria();
    filter.setKey(key);
    filter.setValue(value);
    filter.setLogic(logic);
    request.addCriterias(filter);
  });
  return withConnection(ListPermissions)(request, {
    authorization: auth.token,
    'x-auth-id': auth.userId,
    'x-organization-id': auth.organizationId,
  });
};

export const listAccessRoles = ({
  page,
  pageSize,
  criteria,
  auth,
}: {
  page: number;
  pageSize: number;
  criteria: AccessManagementCriteria[];
  auth: AccessManagementAuth;
}) => {
  const request = new ListRolesRequest();
  const paginate = new Paginate();
  paginate.setPage(page);
  paginate.setPagesize(pageSize);
  request.setPaginate(paginate);
  criteria.forEach(({ key, value, logic }) => {
    const filter = new Criteria();
    filter.setKey(key);
    filter.setValue(value);
    filter.setLogic(logic);
    request.addCriterias(filter);
  });
  return withConnection(ListRoles)(request, {
    authorization: auth.token,
    'x-auth-id': auth.userId,
    'x-organization-id': auth.organizationId,
  });
};

export const getAccessRole = ({
  roleDefinitionId,
  auth,
}: {
  roleDefinitionId: string;
  auth: AccessManagementAuth;
}) => {
  const request = new GetRoleRequest();
  request.setRoledefinitionid(roleDefinitionId);
  return withConnection(GetRole)(request, {
    authorization: auth.token,
    'x-auth-id': auth.userId,
    'x-organization-id': auth.organizationId,
  });
};

export const createAccessRole = ({
  key,
  name,
  description,
  scope,
  subject,
  permissionDefinitionIds,
  auth,
}: {
  key: string;
  name: string;
  description: string;
  scope: keyof typeof RoleScope;
  subject: keyof typeof RoleSubject;
  permissionDefinitionIds: string[];
  auth: AccessManagementAuth;
}) => {
  const request = new CreateCustomRoleRequest();
  request.setKey(key);
  request.setName(name);
  request.setDescription(description);
  request.setScope(RoleScope[scope]);
  request.setSubject(RoleSubject[subject]);
  request.setPermissiondefinitionidsList(permissionDefinitionIds);
  return withConnection(CreateCustomRole)(request, {
    authorization: auth.token,
    'x-auth-id': auth.userId,
    'x-organization-id': auth.organizationId,
  });
};

export const updateAccessRole = ({
  roleDefinitionId,
  name,
  description,
  auth,
}: {
  roleDefinitionId: string;
  name: string;
  description: string;
  auth: AccessManagementAuth;
}) => {
  const request = new UpdateCustomRoleRequest();
  request.setRoledefinitionid(roleDefinitionId);
  request.setName(name);
  request.setDescription(description);
  return withConnection(UpdateCustomRole)(request, {
    authorization: auth.token,
    'x-auth-id': auth.userId,
    'x-organization-id': auth.organizationId,
  });
};

type AccessRolePermissionReduction = {
  roleDefinitionId: string;
  permissionDefinitionIds: string[];
  reason: string;
  auth: AccessManagementAuth;
} & ({ mode: 'preview' } | { mode: 'confirm'; confirmationDigest: string });

export const reduceAccessRolePermissions = (
  reduction: AccessRolePermissionReduction,
) => {
  const { roleDefinitionId, permissionDefinitionIds, reason, auth, mode } =
    reduction;
  const request = new ReduceCustomRolePermissionsRequest();
  request.setRoledefinitionid(roleDefinitionId);
  request.setPermissiondefinitionidsList(permissionDefinitionIds);
  request.setReason(reason);
  if (mode === 'preview') {
    request.setPreview(true);
  } else if (mode === 'confirm' && reduction.confirmationDigest.trim() !== '') {
    request.setPreview(false);
    request.setConfirmationdigest(reduction.confirmationDigest.trim());
  } else {
    return Promise.reject(
      new Error(
        'Role reduction requires an explicit preview or confirmation with a digest.',
      ),
    );
  }
  return withConnection(ReduceCustomRolePermissions)(request, {
    authorization: auth.token,
    'x-auth-id': auth.userId,
    'x-organization-id': auth.organizationId,
  });
};

export const archiveAccessRole = ({
  roleDefinitionId,
  reason,
  auth,
}: {
  roleDefinitionId: string;
  reason: string;
  auth: AccessManagementAuth;
}) => {
  const request = new ArchiveCustomRoleRequest();
  request.setRoledefinitionid(roleDefinitionId);
  request.setReason(reason);
  return withConnection(ArchiveCustomRole)(request, {
    authorization: auth.token,
    'x-auth-id': auth.userId,
    'x-organization-id': auth.organizationId,
  });
};
