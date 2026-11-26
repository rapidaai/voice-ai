import {
  ConnectionConfig,
  GetAllUser,
  UpdateNotificationSetting,
} from '@rapidaai/react';

import { listUsers, updateUserNotificationSettings } from '@/clients';

jest.mock('@/configs', () => ({
  connectionConfig: { endpoint: 'test-endpoint' },
}));

jest.mock('@rapidaai/react', () => {
  class NotificationSetting {
    private channel = '';
    private enabled = false;
    private eventType = '';

    setChannel(channel: string) {
      this.channel = channel;
    }

    setEnabled(enabled: boolean) {
      this.enabled = enabled;
    }

    setEventtype(eventType: string) {
      this.eventType = eventType;
    }

    getChannel() {
      return this.channel;
    }

    getEnabled() {
      return this.enabled;
    }

    getEventtype() {
      return this.eventType;
    }
  }

  class UpdateNotificationSettingRequest {
    private settings: NotificationSetting[] = [];

    addSettings(setting: NotificationSetting) {
      this.settings.push(setting);
    }

    getSettingsList() {
      return this.settings;
    }
  }

  return {
    ConnectionConfig: {
      WithDebugger: jest.fn(
        ({ authorization, userId, organizationId, projectId }) => ({
          authorization,
          'x-auth-id': userId,
          ...(organizationId ? { 'x-organization-id': organizationId } : {}),
          'x-project-id': projectId,
        }),
      ),
    },
    GetAllUser: jest.fn(),
    GetNotificationSetting: jest.fn(),
    GetUser: jest.fn(),
    NotificationSetting,
    UpdateNotificationSetting: jest.fn(),
    UpdateNotificationSettingRequest,
    UpdateUser: jest.fn(),
  };
});

describe('user client', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('lists users with pagination, criteria, and debugger metadata', () => {
    const callback = jest.fn();
    const criteria = [{ key: 'email', value: 'p_iyer', logic: 'contains' }];

    listUsers({
      page: 2,
      pageSize: 25,
      criteria,
      auth: {
        organizationId: 'organization-1',
        projectId: 'project-1',
        token: 'token-1',
        userId: 'user-1',
      },
      callback,
    });

    const metadata = (ConnectionConfig.WithDebugger as jest.Mock).mock
      .results[0].value;
    expect(GetAllUser).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      2,
      25,
      criteria,
      callback,
      metadata,
    );
    expect(ConnectionConfig.WithDebugger).toHaveBeenCalledWith({
      authorization: 'token-1',
      userId: 'user-1',
      organizationId: 'organization-1',
      projectId: 'project-1',
    });
  });

  it('omits empty project context when listing organization users', () => {
    const callback = jest.fn();

    listUsers({
      page: 1,
      pageSize: 25,
      criteria: [],
      auth: {
        organizationId: 'organization-1',
        projectId: '',
        token: 'token-1',
        userId: 'user-1',
      },
      callback,
    });

    const metadata = (GetAllUser as jest.Mock).mock.calls[0][5];
    expect(metadata).toEqual({
      authorization: 'token-1',
      'x-auth-id': 'user-1',
      'x-organization-id': 'organization-1',
    });
  });

  it('updates notification settings with nested form values', () => {
    updateUserNotificationSettings({
      values: {
        assistant: {
          created: true,
          deleted: false,
        },
        ignored: 'not-boolean',
      },
      auth: {
        organizationId: 'organization-1',
        projectId: 'project-1',
        token: 'token-1',
        userId: 'user-1',
      },
    });

    const request = (UpdateNotificationSetting as jest.Mock).mock.calls[0][1];
    const settings = request.getSettingsList();

    expect(settings).toHaveLength(2);
    expect(settings[0].getChannel()).toBe('email');
    expect(settings[0].getEventtype()).toBe('assistant.created');
    expect(settings[0].getEnabled()).toBe(true);
    expect(settings[1].getEventtype()).toBe('assistant.deleted');
    expect(settings[1].getEnabled()).toBe(false);
    const metadata = (ConnectionConfig.WithDebugger as jest.Mock).mock
      .results[0].value;
    expect(ConnectionConfig.WithDebugger).toHaveBeenCalledWith({
      authorization: 'token-1',
      userId: 'user-1',
      organizationId: 'organization-1',
      projectId: 'project-1',
    });
    expect(UpdateNotificationSetting).toHaveBeenCalledWith(
      { endpoint: 'test-endpoint' },
      request,
      metadata,
    );
  });
});
