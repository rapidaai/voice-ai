import {
  AddUserToProjects,
  AddUserToProjectsRequest,
  ArchiveProjectResponse,
  CreateOrganization,
  CreateOrganizationResponse,
  CreateProject,
  CreateProjectResponse,
  DeleteProject,
  DeleteUserFromOrganization,
  DeleteUserFromOrganizationRequest,
  DeleteUserFromProject,
  GetAllProject,
  GetAllProjectResponse,
  GetOrganization,
  GetOrganizationResponse,
  GetProject,
  InviteUserToOrganization,
  InviteUserToOrganizationRequest,
  ProjectRoleAssignment,
  ServiceError,
  UpdateOrganization,
  UpdateOrganizationResponse,
  UpdateProject,
  UpdateProjectResponse,
  UpdateUserOrganizationRole,
  UpdateUserOrganizationRoleRequest,
} from '@rapidaai/react';

import { withConnection } from './connection';

type WorkspaceAuth = {
  organizationId: string;
  token: string;
  userId: string;
};

type WorkspaceOnboardingAuth = Omit<WorkspaceAuth, 'organizationId'> & {
  organizationId?: undefined;
};

type WorkspaceProjectRole = {
  projectId: string;
  roleDefinitionIds: string[];
};

type WorkspaceClientCallback<TResponse> = (
  error: ServiceError | null,
  response: TResponse | null,
) => void;

export type CreateWorkspaceOrganizationParams = {
  name: string;
  size: string;
  industry: string;
  auth: WorkspaceOnboardingAuth;
  callback: WorkspaceClientCallback<CreateOrganizationResponse>;
};

export type CreateWorkspaceProjectParams = {
  name: string;
  description: string;
  auth: WorkspaceAuth;
  callback: WorkspaceClientCallback<CreateProjectResponse>;
};

export type UpdateWorkspaceProjectParams = {
  projectId: string;
  name: string;
  description: string;
  auth: WorkspaceAuth;
  callback: WorkspaceClientCallback<UpdateProjectResponse>;
};

export type InviteOrganizationUserParams = {
  email: string;
  organizationRoleDefinitionIds: string[];
  projectRoles: WorkspaceProjectRole[];
  auth: WorkspaceAuth;
};

export type AddUserToProjectsParams = {
  userId: string;
  projectRoles: WorkspaceProjectRole[];
  auth: WorkspaceAuth;
};

export type GetWorkspaceOrganizationParams = {
  auth: WorkspaceAuth;
  callback: WorkspaceClientCallback<GetOrganizationResponse>;
};

export type UpdateWorkspaceOrganizationParams = {
  organizationId: string;
  name: string;
  industry: string;
  contact: string;
  auth: WorkspaceAuth;
  callback: WorkspaceClientCallback<UpdateOrganizationResponse>;
};

export type ListWorkspaceProjectsParams = {
  page: number;
  pageSize: number;
  criteria: { key: string; value: string }[];
  auth: WorkspaceAuth;
  callback: WorkspaceClientCallback<GetAllProjectResponse>;
};

export type DeleteWorkspaceProjectParams = {
  projectId: string;
  auth: WorkspaceAuth;
  callback: WorkspaceClientCallback<ArchiveProjectResponse>;
};

export type DeleteOrganizationUserParams = {
  userId: string;
  auth: WorkspaceAuth;
};

export type UpdateOrganizationUserRoleParams = {
  userId: string;
  roleDefinitionIds: string[];
  auth: WorkspaceAuth;
};

const createWorkspaceMetadata = ({
  organizationId,
  token,
  userId,
}: WorkspaceAuth | WorkspaceOnboardingAuth) => ({
  authorization: token,
  'x-auth-id': userId,
  ...(organizationId ? { 'x-organization-id': organizationId } : {}),
});

const createProjectRoleAssignments = (projectRoles: WorkspaceProjectRole[]) =>
  projectRoles.map(row => {
    const assignment = new ProjectRoleAssignment();
    assignment.setProjectid(row.projectId);
    assignment.setRoledefinitionidsList(row.roleDefinitionIds);
    return assignment;
  });

export const createOrganization = withConnection(CreateOrganization);
export const getOrganization = withConnection(GetOrganization);
export const updateOrganization = withConnection(UpdateOrganization);

export const createProject = withConnection(CreateProject);
export const getAllProject = withConnection(GetAllProject);
export const getProject = withConnection(GetProject);
export const updateProject = withConnection(UpdateProject);
export const deleteProject = withConnection(DeleteProject);

export const inviteUserToOrganization = withConnection(
  InviteUserToOrganization,
);
export const addUserToProjects = withConnection(AddUserToProjects);
export const updateUserOrganizationRole = withConnection(
  UpdateUserOrganizationRole,
);
export const deleteUserFromOrganization = withConnection(
  DeleteUserFromOrganization,
);
export const deleteUserFromProject = withConnection(DeleteUserFromProject);

export const createWorkspaceOrganization = ({
  name,
  size,
  industry,
  auth,
  callback,
}: CreateWorkspaceOrganizationParams) =>
  createOrganization(
    name,
    size,
    industry,
    createWorkspaceMetadata(auth),
    callback,
  );

export const getWorkspaceOrganization = ({
  auth,
  callback,
}: GetWorkspaceOrganizationParams) =>
  getOrganization(createWorkspaceMetadata(auth), callback);

export const updateWorkspaceOrganization = ({
  organizationId,
  name,
  industry,
  contact,
  auth,
  callback,
}: UpdateWorkspaceOrganizationParams) =>
  updateOrganization(
    organizationId,
    createWorkspaceMetadata(auth),
    callback,
    name,
    industry,
    contact,
  );

export const createWorkspaceProject = ({
  name,
  description,
  auth,
  callback,
}: CreateWorkspaceProjectParams) =>
  createProject(name, description, createWorkspaceMetadata(auth), callback);

export const updateWorkspaceProject = ({
  projectId,
  name,
  description,
  auth,
  callback,
}: UpdateWorkspaceProjectParams) =>
  updateProject(
    projectId,
    callback,
    createWorkspaceMetadata(auth),
    name,
    description,
  );

export const listWorkspaceProjects = ({
  page,
  pageSize,
  criteria,
  auth,
  callback,
}: ListWorkspaceProjectsParams) =>
  getAllProject(
    page,
    pageSize,
    criteria,
    callback,
    createWorkspaceMetadata(auth),
  );

export const deleteWorkspaceProject = ({
  projectId,
  auth,
  callback,
}: DeleteWorkspaceProjectParams) =>
  deleteProject(projectId, callback, createWorkspaceMetadata(auth));

export const inviteOrganizationUser = ({
  email,
  organizationRoleDefinitionIds,
  projectRoles,
  auth,
}: InviteOrganizationUserParams) => {
  const request = new InviteUserToOrganizationRequest();
  request.setEmail(email);
  request.setOrganizationroledefinitionidsList(organizationRoleDefinitionIds);
  request.setProjectrolesList(createProjectRoleAssignments(projectRoles));

  return inviteUserToOrganization(request, createWorkspaceMetadata(auth));
};

export const addUserToProjectsByRole = ({
  userId,
  projectRoles,
  auth,
}: AddUserToProjectsParams) => {
  const request = new AddUserToProjectsRequest();
  request.setUserid(userId);
  request.setProjectrolesList(createProjectRoleAssignments(projectRoles));

  return addUserToProjects(request, createWorkspaceMetadata(auth));
};

export const deleteOrganizationUser = ({
  userId,
  auth,
}: DeleteOrganizationUserParams) => {
  const request = new DeleteUserFromOrganizationRequest();
  request.setUserid(userId);

  return deleteUserFromOrganization(request, createWorkspaceMetadata(auth));
};

export const updateOrganizationUserRole = ({
  userId,
  roleDefinitionIds,
  auth,
}: UpdateOrganizationUserRoleParams) => {
  const request = new UpdateUserOrganizationRoleRequest();
  request.setUserid(userId);
  request.setRoledefinitionidsList(roleDefinitionIds);

  return updateUserOrganizationRole(request, createWorkspaceMetadata(auth));
};
