import { createApiMetadata, withConnection } from '@/clients';

describe('createApiMetadata', () => {
  it('maps organization, project, token, and user values to API headers', () => {
    expect(
      createApiMetadata({
        organizationId: 'organization-1',
        projectId: 'project-1',
        token: 'token-1',
        userId: 'user-1',
      }),
    ).toEqual({
      authorization: 'token-1',
      'x-organization-id': 'organization-1',
      'x-project-id': 'project-1',
      'x-auth-id': 'user-1',
    });
  });
});

describe('withConnection', () => {
  it('binds the configured API connection as the first argument', () => {
    const request = jest.fn(
      (_connection, value: string) => `received ${value}`,
    );
    const connectedRequest = withConnection(request);

    expect(connectedRequest('input')).toBe('received input');
    expect(request).toHaveBeenCalledWith(expect.anything(), 'input');
  });
});
