import { GetProjectDashboard } from '@rapidaai/react';
import { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb';

import { getProjectDashboardRange } from '@/clients/dashboard.client';

jest.mock('@/configs', () => ({
  connectionConfig: { endpoint: 'test-endpoint' },
}));

jest.mock('@rapidaai/react', () => {
  const actual = jest.requireActual('@rapidaai/react');
  return {
    ...actual,
    GetProjectDashboard: jest.fn(),
  };
});

describe('dashboard client', () => {
  it('calls the public dashboard API with the selected range and limits', () => {
    const fromDate = new Timestamp();
    const toDate = new Timestamp();
    const auth = {
      projectId: 'project-1',
      token: 'token-1',
      userId: 'user-1',
    };

    getProjectDashboardRange({
      fromDate,
      toDate,
      assistantLimit: 3,
      recentFailureLimit: 5,
      auth,
    });

    expect(GetProjectDashboard).toHaveBeenCalledTimes(1);
    const [connection, request, metadata] = (GetProjectDashboard as jest.Mock)
      .mock.calls[0];
    expect(connection).toEqual({ endpoint: 'test-endpoint' });
    expect(request.getFromdate()).toBe(fromDate);
    expect(request.getTodate()).toBe(toDate);
    expect(request.getAssistantlimit()).toBe(3);
    expect(request.getRecentfailurelimit()).toBe(5);
    expect(metadata).toEqual({
      authorization: 'token-1',
      'x-auth-id': 'user-1',
      'x-project-id': 'project-1',
    });
  });
});
