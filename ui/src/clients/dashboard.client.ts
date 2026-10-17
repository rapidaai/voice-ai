import {
  GetProjectDashboard,
  GetProjectDashboardRequest,
} from '@rapidaai/react';
import type { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb';

import { connectionConfig } from '@/configs';
import { ApiAuth, createApiMetadata } from './connection';

export type GetProjectDashboardRangeParams = {
  fromDate: Timestamp;
  toDate: Timestamp;
  assistantLimit?: number;
  recentFailureLimit?: number;
  auth: ApiAuth;
};

export const getProjectDashboardRange = ({
  fromDate,
  toDate,
  assistantLimit,
  recentFailureLimit,
  auth,
}: GetProjectDashboardRangeParams) => {
  const request = new GetProjectDashboardRequest();
  request.setFromdate(fromDate);
  request.setTodate(toDate);
  if (assistantLimit !== undefined) request.setAssistantlimit(assistantLimit);
  if (recentFailureLimit !== undefined)
    request.setRecentfailurelimit(recentFailureLimit);

  return GetProjectDashboard(
    connectionConfig,
    request,
    createApiMetadata(auth),
  );
};
