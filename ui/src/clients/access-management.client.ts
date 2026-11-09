import {
  ArchiveCustomRole,
  ArchiveCustomRoleRequest,
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
  token: string;
  userId: string;
};

export const listAccessPermissions = ({
  page,
  pageSize,
  scope,
  auth,
}: {
  page: number;
  pageSize: number;
  scope: keyof typeof RoleScope;
  auth: AccessManagementAuth;
}) => {
  const request = new ListPermissionsRequest();
  const paginate = new Paginate();
  paginate.setPage(page);
  paginate.setPagesize(pageSize);
  request.setPaginate(paginate);
  request.setScope(RoleScope[scope]);
  return withConnection(ListPermissions)(request, {
    authorization: auth.token,
    'x-auth-id': auth.userId,
  });
};

export const listAccessRoles = ({
  page,
  pageSize,
  query,
  scope,
  subject,
  definitionType,
  auth,
}: {
  page: number;
  pageSize: number;
  query: string;
  scope: keyof typeof RoleScope;
  subject: keyof typeof RoleSubject;
  definitionType: keyof typeof RoleDefinitionType;
  auth: AccessManagementAuth;
}) => {
  const request = new ListRolesRequest();
  const paginate = new Paginate();
  paginate.setPage(page);
  paginate.setPagesize(pageSize);
  request.setPaginate(paginate);
  request.setQuery(query);
  request.setScope(RoleScope[scope]);
  request.setSubject(RoleSubject[subject]);
  request.setDefinitiontype(RoleDefinitionType[definitionType]);
  return withConnection(ListRoles)(request, {
    authorization: auth.token,
    'x-auth-id': auth.userId,
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
  });
};

export const reduceAccessRolePermissions = ({
  roleDefinitionId,
  permissionDefinitionIds,
  reason,
  auth,
}: {
  roleDefinitionId: string;
  permissionDefinitionIds: string[];
  reason: string;
  auth: AccessManagementAuth;
}) => {
  const request = new ReduceCustomRolePermissionsRequest();
  request.setRoledefinitionid(roleDefinitionId);
  request.setPermissiondefinitionidsList(permissionDefinitionIds);
  request.setReason(reason);
  return withConnection(ReduceCustomRolePermissions)(request, {
    authorization: auth.token,
    'x-auth-id': auth.userId,
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
  });
};
