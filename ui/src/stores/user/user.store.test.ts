import { listUsers } from '@/clients/user.client';
import { useUserPageStore } from '@/stores/user';

jest.mock('@/clients/user.client', () => ({
  listUsers: jest.fn(),
}));

describe('useUserPageStore', () => {
  it('forwards the selected organization when listing members', () => {
    useUserPageStore
      .getState()
      .getAllUser(
        'token-1',
        'user-1',
        'organization-1',
        '',
        jest.fn(),
        jest.fn(),
      );

    expect(listUsers).toHaveBeenCalledWith(
      expect.objectContaining({
        auth: {
          organizationId: 'organization-1',
          projectId: '',
          token: 'token-1',
          userId: 'user-1',
        },
      }),
    );
  });
});
