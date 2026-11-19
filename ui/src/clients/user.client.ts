import {
  ConnectionConfig,
  GetAllUser,
  GetAllUserResponse,
  GetNotificationSetting,
  GetUser,
  NotificationSetting,
  ServiceError,
  UpdateNotificationSettingRequest,
  UpdateNotificationSetting,
  UpdateUser,
} from '@rapidaai/react';

import { ApiAuth, withConnection } from './connection';

type Criteria = {
  key: string;
  value: string;
  logic: string;
};

type UserClientCallback<TResponse> = (
  error: ServiceError | null,
  response: TResponse | null,
) => void;

export type ListUsersParams = {
  page: number;
  pageSize: number;
  criteria: Criteria[];
  auth: ApiAuth;
  callback: UserClientCallback<GetAllUserResponse>;
};

export type UpdateUserNotificationSettingsParams = {
  values: Record<string, unknown>;
  auth: ApiAuth;
};

export const getAllUser = withConnection(GetAllUser);
export const getUser = withConnection(GetUser);
export const updateUser = withConnection(UpdateUser);
export const getNotificationSetting = withConnection(GetNotificationSetting);
export const updateNotificationSetting = withConnection(
  UpdateNotificationSetting,
);

export const listUsers = ({
  page,
  pageSize,
  criteria,
  auth,
  callback,
}: ListUsersParams): void => {
  getAllUser(
    page,
    pageSize,
    criteria,
    callback,
    ConnectionConfig.WithDebugger({
      authorization: auth.token,
      userId: auth.userId,
      organizationId: auth.organizationId,
      projectId: auth.projectId,
    }),
  );
};

export const updateUserNotificationSettings = ({
  values,
  auth,
}: UpdateUserNotificationSettingsParams) => {
  const request = new UpdateNotificationSettingRequest();

  const addSettings = (prefix: string, value: unknown) => {
    if (!value || typeof value !== 'object') return;

    Object.entries(value).forEach(([key, entryValue]) => {
      const eventType = prefix ? `${prefix}.${key}` : key;

      if (typeof entryValue === 'boolean') {
        const setting = new NotificationSetting();
        setting.setChannel('email');
        setting.setEventtype(eventType);
        setting.setEnabled(entryValue);
        request.addSettings(setting);
        return;
      }

      addSettings(eventType, entryValue);
    });
  };

  addSettings('', values);

  return updateNotificationSetting(
    request,
    ConnectionConfig.WithDebugger({
      authorization: auth.token,
      userId: auth.userId,
      organizationId: auth.organizationId,
      projectId: auth.projectId,
    }),
  );
};
