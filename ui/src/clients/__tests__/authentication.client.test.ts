import { ChangePassword, ConnectionConfig } from '@rapidaai/react';

import { changeAccountPassword } from '@/clients';

jest.mock('@/configs', () => ({
  connectionConfig: { endpoint: 'test-endpoint' },
}));

jest.mock('@rapidaai/react', () => {
  class ChangePasswordRequest {
    private oldPassword = '';
    private password = '';

    setOldpassword(oldPassword: string) {
      this.oldPassword = oldPassword;
    }

    setPassword(password: string) {
      this.password = password;
    }

    getOldpassword() {
      return this.oldPassword;
    }

    getPassword() {
      return this.password;
    }
  }

  return {
    AuthenticateUser: jest.fn(),
    AuthorizeUser: jest.fn(),
    ChangePassword: jest.fn(),
    ChangePasswordRequest,
    ConnectionConfig: {
      WithDebugger: jest.fn(metadata => ({ debugger: metadata })),
    },
    CreatePassword: jest.fn(),
    ForgotPassword: jest.fn(),
    Github: jest.fn(),
    Google: jest.fn(),
    Linkedin: jest.fn(),
    RegisterUser: jest.fn(),
  };
});

describe('authentication client', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('changes an account password with debugger metadata', () => {
    changeAccountPassword({
      currentPassword: 'old-password',
      password: 'new-password',
      auth: {
        projectId: 'project-1',
        token: 'token-1',
        userId: 'user-1',
      },
    });

    const request = (ChangePassword as jest.Mock).mock.calls[0][1];
    expect(request.getOldpassword()).toBe('old-password');
    expect(request.getPassword()).toBe('new-password');
    expect(ConnectionConfig.WithDebugger).toHaveBeenCalledWith({
      authorization: 'token-1',
      userId: 'user-1',
      projectId: 'project-1',
    });
    const metadata = (ConnectionConfig.WithDebugger as jest.Mock).mock
      .results[0].value;
    expect(ChangePassword).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      request,
      metadata,
    );
  });
});
