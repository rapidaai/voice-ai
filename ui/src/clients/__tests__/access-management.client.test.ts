import {
  ArchiveCustomRole,
  CreateCustomRole,
  GetRole,
  ListPermissions,
  ListRoles,
  ReduceCustomRolePermissions,
  RoleDefinitionType,
  RoleScope,
  RoleSubject,
  UpdateCustomRole,
} from '@rapidaai/react';
import {
  archiveAccessRole,
  createAccessRole,
  getAccessRole,
  listAccessPermissions,
  listAccessRoles,
  reduceAccessRolePermissions,
  updateAccessRole,
} from '../access-management.client';

jest.mock('@rapidaai/react', () => ({
  ...jest.requireActual('@rapidaai/react'),
  ArchiveCustomRole: jest.fn(),
  CreateCustomRole: jest.fn(),
  GetRole: jest.fn(),
  ListPermissions: jest.fn(),
  ListRoles: jest.fn(),
  ReduceCustomRolePermissions: jest.fn(),
  UpdateCustomRole: jest.fn(),
}));

const auth = { token: 'token-1', userId: 'user-1' };
const metadata = { authorization: 'token-1', 'x-auth-id': 'user-1' };

describe('access management client', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    for (const request of [
      ArchiveCustomRole,
      CreateCustomRole,
      GetRole,
      ListPermissions,
      ListRoles,
      ReduceCustomRolePermissions,
      UpdateCustomRole,
    ]) {
      (request as jest.Mock).mockResolvedValue({ getSuccess: () => true });
    }
  });

  it('maps role filters, pagination, and user authentication', async () => {
    await listAccessRoles({
      page: 2,
      pageSize: 50,
      query: 'support',
      scope: 'ROLE_SCOPE_PROJECT',
      subject: 'ROLE_SUBJECT_USER',
      definitionType: 'ROLE_DEFINITION_TYPE_CUSTOM',
      auth,
    });

    const [request, requestMetadata] = (
      ListRoles as jest.Mock
    ).mock.calls[0].slice(1);
    expect(request.getPaginate().getPage()).toBe(2);
    expect(request.getPaginate().getPagesize()).toBe(50);
    expect(request.getQuery()).toBe('support');
    expect(request.getScope()).toBe(RoleScope.ROLE_SCOPE_PROJECT);
    expect(request.getSubject()).toBe(RoleSubject.ROLE_SUBJECT_USER);
    expect(request.getDefinitiontype()).toBe(
      RoleDefinitionType.ROLE_DEFINITION_TYPE_CUSTOM,
    );
    expect(requestMetadata).toEqual(metadata);
  });

  it('maps permission scope and pagination', async () => {
    await listAccessPermissions({
      page: 3,
      pageSize: 100,
      scope: 'ROLE_SCOPE_ORGANIZATION',
      auth,
    });

    const [request, requestMetadata] = (
      ListPermissions as jest.Mock
    ).mock.calls[0].slice(1);
    expect(request.getPaginate().getPage()).toBe(3);
    expect(request.getPaginate().getPagesize()).toBe(100);
    expect(request.getScope()).toBe(RoleScope.ROLE_SCOPE_ORGANIZATION);
    expect(requestMetadata).toEqual(metadata);
  });

  it('maps create, update, reduction, archive, and get requests', async () => {
    await createAccessRole({
      key: 'support_manager',
      name: 'Support manager',
      description: 'Manages support',
      scope: 'ROLE_SCOPE_PROJECT',
      subject: 'ROLE_SUBJECT_PROJECT_TOKEN',
      permissionDefinitionIds: ['34', '35'],
      auth,
    });
    await updateAccessRole({
      roleDefinitionId: '101',
      name: 'Support lead',
      description: 'Leads support',
      auth,
    });
    await reduceAccessRolePermissions({
      roleDefinitionId: '101',
      permissionDefinitionIds: ['35'],
      reason: 'No longer required',
      auth,
    });
    await archiveAccessRole({
      roleDefinitionId: '101',
      reason: 'Replaced',
      auth,
    });
    await getAccessRole({ roleDefinitionId: '101', auth });

    const createRequest = (CreateCustomRole as jest.Mock).mock.calls[0][1];
    expect(createRequest.getKey()).toBe('support_manager');
    expect(createRequest.getPermissiondefinitionidsList()).toEqual([
      '34',
      '35',
    ]);
    const updateRequest = (UpdateCustomRole as jest.Mock).mock.calls[0][1];
    expect(updateRequest.getRoledefinitionid()).toBe('101');
    expect(updateRequest.getName()).toBe('Support lead');
    const reduceRequest = (ReduceCustomRolePermissions as jest.Mock).mock
      .calls[0][1];
    expect(reduceRequest.getPermissiondefinitionidsList()).toEqual(['35']);
    expect(reduceRequest.getReason()).toBe('No longer required');
    expect((ArchiveCustomRole as jest.Mock).mock.calls[0][1].getReason()).toBe(
      'Replaced',
    );
    expect((GetRole as jest.Mock).mock.calls[0][1].getRoledefinitionid()).toBe(
      '101',
    );
  });

  it('returns transport failures to the caller', async () => {
    (ListRoles as jest.Mock).mockRejectedValueOnce({
      code: 14,
      message: 'unavailable',
    });

    await expect(
      listAccessRoles({
        page: 1,
        pageSize: 25,
        query: '',
        scope: 'ROLE_SCOPE_UNSPECIFIED',
        subject: 'ROLE_SUBJECT_UNSPECIFIED',
        definitionType: 'ROLE_DEFINITION_TYPE_UNSPECIFIED',
        auth,
      }),
    ).rejects.toEqual({ code: 14, message: 'unavailable' });
  });
});
