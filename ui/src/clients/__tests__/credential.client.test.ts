import {
  ConnectionConfig,
  CreateProjectCredential,
  CreateProviderKey,
  DeleteProviderKey,
  GetAllOrganizationCredential,
  ListProjectCredentials as ListProjectCredentialsApi,
  RevokeProjectCredential,
} from '@rapidaai/react';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';

import {
  createProjectPublishableCredential,
  createProviderCredential,
  deleteProviderCredential,
  listOrganizationCredentials,
  listProjectCredentials,
  revokeProjectCredential,
} from '@/clients';

jest.mock('@/configs', () => ({
  connectionConfig: { endpoint: 'test-endpoint' },
}));

jest.mock('@rapidaai/react', () => {
  class Criteria {
    key = '';
    logic = '';
    value = '';

    setKey(value: string) {
      this.key = value;
    }

    setLogic(value: string) {
      this.logic = value;
    }

    setValue(value: string) {
      this.value = value;
    }
  }

  class Paginate {
    page = 0;
    pageSize = 0;

    setPage(value: number) {
      this.page = value;
    }

    setPagesize(value: number) {
      this.pageSize = value;
    }
  }

  class GetAllProjectCredentialRequest {
    criterias: Criteria[] = [];
    paginate?: Paginate;
    projectId = '';

    addCriterias(value: Criteria) {
      this.criterias.push(value);
    }

    setPaginate(value: Paginate) {
      this.paginate = value;
    }

    setProjectid(value: string) {
      this.projectId = value;
    }
  }

  class CreateProviderCredentialRequest {
    private credential: unknown;
    private name = '';
    private provider = '';

    setCredential(value: unknown) {
      this.credential = value;
    }

    setName(value: string) {
      this.name = value;
    }

    setProvider(value: string) {
      this.provider = value;
    }

    getCredential() {
      return this.credential;
    }

    getName() {
      return this.name;
    }

    getProvider() {
      return this.provider;
    }
  }

  return {
    ConnectionConfig: {
      WithDebugger: jest.fn(metadata => ({ debugger: metadata })),
    },
    Criteria,
    CreateProjectCredential: jest.fn(),
    CreateProviderCredentialRequest,
    CreateProviderKey: jest.fn(),
    DeleteProviderKey: jest.fn(),
    GetAllOrganizationCredential: jest.fn(),
    GetAllProjectCredentialRequest,
    ListProjectCredentials: jest.fn(),
    Paginate,
    RevokeProjectCredential: jest.fn(),
  };
});

jest.mock('google-protobuf/google/protobuf/struct_pb', () => ({
  Struct: {
    fromJavaScript: jest.fn(value => ({ value })),
  },
}));

const auth = {
  projectId: 'project-1',
  token: 'token-1',
  userId: 'user-1',
};

const headers = {
  authorization: 'token-1',
  'x-auth-id': 'user-1',
};

describe('credential client', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('lists organization credentials with debugger metadata', () => {
    const callback = jest.fn();
    const criteria = [{ key: 'provider', value: 'openai', logic: 'eq' }];

    listOrganizationCredentials({
      page: 1,
      pageSize: 100,
      criteria,
      auth,
      callback,
    });

    const metadata = (ConnectionConfig.WithDebugger as jest.Mock).mock
      .results[0].value;
    expect(GetAllOrganizationCredential).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      1,
      100,
      criteria,
      callback,
      metadata,
    );
    expect(ConnectionConfig.WithDebugger).toHaveBeenCalledWith({
      authorization: 'token-1',
      userId: 'user-1',
      projectId: 'project-1',
    });
  });

  it('creates provider credentials with debugger metadata', () => {
    createProviderCredential({
      provider: 'openai',
      name: 'Production key',
      config: { api_key: 'secret' },
      auth,
    });

    const metadata = (ConnectionConfig.WithDebugger as jest.Mock).mock
      .results[0].value;
    const request = (CreateProviderKey as jest.Mock).mock.calls[0][1];

    expect(request.getProvider()).toBe('openai');
    expect(request.getName()).toBe('Production key');
    expect(Struct.fromJavaScript).toHaveBeenCalledWith({
      api_key: 'secret',
    });
    expect(CreateProviderKey).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      request,
      metadata,
    );
  });

  it('deletes provider credentials with debugger metadata', () => {
    const callback = jest.fn();

    deleteProviderCredential({
      credentialId: 'credential-1',
      auth,
      callback,
    });

    const metadata = (ConnectionConfig.WithDebugger as jest.Mock).mock
      .results[0].value;
    expect(DeleteProviderKey).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      'credential-1',
      callback,
      metadata,
    );
  });

  it('creates and lists project API keys with auth headers', () => {
    const callback = jest.fn();

    createProjectPublishableCredential({
      projectId: 'project-1',
      name: 'publishable key',
      roleDefinitionId: '117',
      auth,
      callback,
    });
    listProjectCredentials({
      projectId: 'project-1',
      page: 2,
      pageSize: 25,
      criteria: [{ key: 'role', value: 'runtime', logic: 'contains' }],
      auth,
      callback,
    });

    expect(CreateProjectCredential).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      'project-1',
      'publishable key',
      '117',
      callback,
      headers,
    );
    expect(ListProjectCredentialsApi).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      expect.objectContaining({
        projectId: 'project-1',
        paginate: expect.objectContaining({ page: 2, pageSize: 25 }),
        criterias: [
          expect.objectContaining({
            key: 'role',
            value: 'runtime',
            logic: 'contains',
          }),
        ],
      }),
      callback,
      headers,
    );
  });

  it('revokes a project API key with auth headers', () => {
    const callback = jest.fn();

    revokeProjectCredential({
      projectId: 'project-1',
      credentialId: 'credential-1',
      auth,
      callback,
    });

    expect(RevokeProjectCredential).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      'project-1',
      'credential-1',
      callback,
      headers,
    );
  });
});
