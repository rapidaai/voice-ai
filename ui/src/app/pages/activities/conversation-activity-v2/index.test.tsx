import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';

import { ListingPage } from './index';

const mockListTelemetry = jest.fn();

jest.mock('@/clients', () => ({
  listTelemetry: (...args: unknown[]) => mockListTelemetry(...args),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'user-1',
    projectId: 'project-1',
    token: 'token-1',
  }),
}));

jest.mock('react-router-dom', () => ({
  useSearchParams: () => [new URLSearchParams()],
}));

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: () => null,
}));

jest.mock('@/app/components/layout/page-header', () => ({
  PageHeader: ({ title }: { title: string }) => <h1>{title}</h1>,
  PageHeaderStickyToolbar: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

jest.mock('./components/trace-query-search', () => ({
  TraceQuerySearch: () => <input aria-label="Search traces" />,
}));

jest.mock('@/app/components/ui/primitives/pagination', () => ({
  Pagination: ({ sticky }: { sticky?: boolean }) => (
    <nav aria-label="Trace pagination" data-sticky={sticky} />
  ),
}));

jest.mock('@/app/components/ui/feedback', () => ({
  EmptyState: ({ title }: { title: string }) => <div>{title}</div>,
}));

const mapFromEntries = (entries: Array<[string, string]> = []) => ({
  toArray: () => entries,
});

const metric = {
  getAttributesMap: () => mapFromEntries([['component', 'stt']]),
  getContextMap: () => mapFromEntries([['traceId', 'trace-1']]),
  getDescription: () => '',
  getId: () => 'metric-1',
  getName: () => 'stt.latency_ms',
  getOccurredat: () => ({
    toDate: () => new Date('2026-11-08T10:00:00.000Z'),
  }),
  getOrganizationid: () => 'organization-1',
  getProjectid: () => 'project-1',
  getScope: () => 'message',
  getScopeattributesMap: () =>
    mapFromEntries([
      ['assistantId', 'assistant-1'],
      ['assistantConversationId', 'conversation-1'],
      ['messageId', 'message-1'],
      ['messageRole', 'user'],
    ]),
  getValue: () => '120',
};

const telemetryRecord = {
  getEvent: () => undefined,
  getLog: () => undefined,
  getMetric: () => metric,
};

const telemetryResponse = (records: unknown[]) => ({
  getDataList: () => records,
  getError: () => undefined,
  getPaginated: () => ({ getTotalitem: () => records.length }),
  getSuccess: () => true,
});

describe('Traces listing pagination', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockListTelemetry.mockResolvedValue(telemetryResponse([telemetryRecord]));
  });

  it('keeps pagination sticky for populated traces', async () => {
    render(<ListingPage />);

    await waitFor(() => expect(mockListTelemetry).toHaveBeenCalledTimes(1));
    expect(await screen.findByText('metric-1')).toBeInTheDocument();
    expect(
      screen.getByRole('navigation', { name: 'Trace pagination' }),
    ).toHaveAttribute('data-sticky', 'true');
  });

  it('does not render pagination for an empty trace result', async () => {
    mockListTelemetry.mockResolvedValue(telemetryResponse([]));

    render(<ListingPage />);

    expect(await screen.findByText('No traces found')).toBeInTheDocument();
    expect(
      screen.queryByRole('navigation', { name: 'Trace pagination' }),
    ).not.toBeInTheDocument();
  });
});
