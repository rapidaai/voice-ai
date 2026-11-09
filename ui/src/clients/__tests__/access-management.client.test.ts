import {
  ArchiveCustomRole,
  CreateCustomRole,
  GetRole,
  ListPermissions,
  ListRoles,
  ReduceCustomRolePermissions,
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
      criteria: [
        { key: 'name', value: 'support', logic: 'contains' },
        { key: 'scope', value: 'PROJECT', logic: '=' },
        { key: 'subject', value: 'USER', logic: '=' },
        { key: 'type', value: 'CUSTOM', logic: '=' },
      ],
      auth,
    });

    const [request, requestMetadata] = (
      ListRoles as jest.Mock
    ).mock.calls[0].slice(1);
    expect(request.getPaginate().getPage()).toBe(2);
    expect(request.getPaginate().getPagesize()).toBe(50);
    expect(
      request.getCriteriasList().map((criteria: any) => ({
        key: criteria.getKey(),
        value: criteria.getValue(),
        logic: criteria.getLogic(),
      })),
    ).toEqual([
      { key: 'name', value: 'support', logic: 'contains' },
      { key: 'scope', value: 'PROJECT', logic: '=' },
      { key: 'subject', value: 'USER', logic: '=' },
      { key: 'type', value: 'CUSTOM', logic: '=' },
    ]);
    expect(requestMetadata).toEqual(metadata);
  });

  it('maps permission scope and pagination', async () => {
    await listAccessPermissions({
      page: 3,
      pageSize: 100,
      criteria: [{ key: 'scope', value: 'ORGANIZATION', logic: '=' }],
      auth,
    });

    const [request, requestMetadata] = (
      ListPermissions as jest.Mock
    ).mock.calls[0].slice(1);
    expect(request.getPaginate().getPage()).toBe(3);
    expect(request.getPaginate().getPagesize()).toBe(100);
    expect(request.getCriteriasList()).toHaveLength(1);
    expect(request.getCriteriasList()[0].getKey()).toBe('scope');
    expect(request.getCriteriasList()[0].getValue()).toBe('ORGANIZATION');
    expect(request.getCriteriasList()[0].getLogic()).toBe('=');
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
        criteria: [],
        auth,
      }),
    ).rejects.toEqual({ code: 14, message: 'unavailable' });
  });
});
