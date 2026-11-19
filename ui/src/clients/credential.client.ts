import {
  ConnectionConfig,
  CreateProjectCredential,
  CreateProjectCredentialResponse,
  CreateProviderCredentialRequest,
  CreateProviderKey,
  DeleteProviderKey,
  Criteria,
  GetAllProjectCredentialRequest,
  GetAllProjectCredentialResponse,
  GetAllOrganizationCredential,
  GetAllOrganizationCredentialResponse,
  GetCredentialResponse,
  ListProjectCredentials as ListProjectCredentialsApi,
  Paginate,
  RevokeProjectCredential,
  RevokeProjectCredentialResponse,
  ServiceError,
} from '@rapidaai/react';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';

import { ApiAuth, withConnection } from './connection';

type CredentialCriteria = {
  key: string;
  value: string;
  logic: string;
};

type CredentialClientCallback<TResponse> = (
  error: ServiceError | null,
  response: TResponse | null,
) => void;

type ProjectCredentialAuth = Pick<ApiAuth, 'token' | 'userId'> &
  Required<Pick<ApiAuth, 'organizationId'>>;

export type ListOrganizationCredentialsParams = {
  page: number;
  pageSize: number;
  criteria: CredentialCriteria[];
  auth: ApiAuth;
  callback: CredentialClientCallback<GetAllOrganizationCredentialResponse>;
};

export type CreateProviderCredentialParams = {
  provider: string;
  name: string;
  config: Record<string, string>;
  auth: ApiAuth;
};

export type DeleteProviderCredentialParams = {
  credentialId: string;
  auth: ApiAuth;
  callback: CredentialClientCallback<GetCredentialResponse>;
};

export type CreateProjectPublishableCredentialParams = {
  projectId: string;
  name: string;
  roleDefinitionId: string;
  auth: ProjectCredentialAuth;
  callback: CredentialClientCallback<CreateProjectCredentialResponse>;
};

export type ListProjectCredentialsParams = {
  projectId: string;
  page: number;
  pageSize: number;
  criteria: CredentialCriteria[];
  auth: ProjectCredentialAuth;
  callback: CredentialClientCallback<GetAllProjectCredentialResponse>;
};

export type RevokeProjectCredentialParams = {
  projectId: string;
  credentialId: string;
  auth: ProjectCredentialAuth;
  callback: CredentialClientCallback<RevokeProjectCredentialResponse>;
};

const createCredentialHeaders = ({
  organizationId,
  token,
  userId,
}: ProjectCredentialAuth) => ({
  authorization: token,
  'x-auth-id': userId,
  'x-organization-id': organizationId,
});

const createCredentialDebuggerMetadata = ({
  organizationId,
  projectId,
  token,
  userId,
}: ApiAuth) =>
  ConnectionConfig.WithDebugger({
    authorization: token,
    organizationId,
    userId,
    projectId,
  });

export const createProviderKey = withConnection(CreateProviderKey);
export const deleteProviderKey = withConnection(DeleteProviderKey);
export const getAllOrganizationCredential = withConnection(
  GetAllOrganizationCredential,
);
export const getProjectCredentialsPage = withConnection(
  ListProjectCredentialsApi,
);
export const createProjectCredential = withConnection(CreateProjectCredential);
export const revokeProjectCredentialApi = withConnection(
  RevokeProjectCredential,
);

export const listOrganizationCredentials = ({
  page,
  pageSize,
  criteria,
  auth,
  callback,
}: ListOrganizationCredentialsParams): void => {
  getAllOrganizationCredential(
    page,
    pageSize,
    criteria,
    callback,
    createCredentialDebuggerMetadata(auth),
  );
};

export const createProviderCredential = ({
  provider,
  name,
  config,
  auth,
}: CreateProviderCredentialParams) => {
  const request = new CreateProviderCredentialRequest();
  request.setProvider(provider);
  request.setCredential(Struct.fromJavaScript(config));
  request.setName(name);

  return createProviderKey(request, createCredentialDebuggerMetadata(auth));
};

export const deleteProviderCredential = ({
  credentialId,
  auth,
  callback,
}: DeleteProviderCredentialParams) =>
  deleteProviderKey(
    credentialId,
    callback,
    createCredentialDebuggerMetadata(auth),
  );

export const createProjectPublishableCredential = ({
  projectId,
  name,
  roleDefinitionId,
  auth,
  callback,
}: CreateProjectPublishableCredentialParams) =>
  createProjectCredential(
    projectId,
    name,
    roleDefinitionId,
    callback,
    createCredentialHeaders(auth),
  );

export const listProjectCredentials = ({
  projectId,
  page,
  pageSize,
  criteria,
  auth,
  callback,
}: ListProjectCredentialsParams) => {
  const request = new GetAllProjectCredentialRequest();
  const paginate = new Paginate();
  request.setProjectid(projectId);
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
  return getProjectCredentialsPage(
    request,
    callback,
    createCredentialHeaders(auth),
  );
};

export const revokeProjectCredential = ({
  projectId,
  credentialId,
  auth,
  callback,
}: RevokeProjectCredentialParams) =>
  revokeProjectCredentialApi(projectId, credentialId, callback, {
    ...createCredentialHeaders(auth),
    'x-project-id': projectId,
  });
