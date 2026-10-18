import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import toast from 'react-hot-toast/headless';

import { EndpointTraces } from '@/app/pages/endpoint/view/traces';

const mockGetLogs = jest.fn();
const mockGetLog = jest.fn();
const mockAddCriterias = jest.fn();
const mockOnChangeLogs = jest.fn();
const mockSetPage = jest.fn();
const mockSetPageSize = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();

const columns = [
  { name: 'ID', key: 'id', visible: true },
  { name: 'Version', key: 'version', visible: true },
  { name: 'Source', key: 'source', visible: true },
  { name: 'Status', key: 'status', visible: true },
  { name: 'Actions', key: 'action', visible: true },
  { name: 'Total Time', key: 'timetaken', visible: true },
  { name: 'LLM Tokens', key: 'total_token', visible: true },
  { name: 'LLM Time', key: 'time_taken', visible: true },
  { name: 'Date', key: 'created_date', visible: true },
];

const makeMetric = (name: string, value: string) => ({
  getName: () => name,
  getValue: () => value,
});

const makeTimestamp = () => ({
  getSeconds: () => 1,
  getNanos: () => 0,
});

const makeLog = () =>
  ({
    getId: () => 'endpoint-log-1',
    getMetadataList: () => [],
    getOptionsList: () => [],
    getArgumentsList: () => [],
    getEndpointprovidermodelid: () => 'model-1',
    getSource: () => 'web-app',
    getStatus: () => 'SUCCESS',
    getTimetaken: () => '3000000',
    getMetricsList: () => [
      makeMetric('agent_total_token', '12'),
      makeMetric('time_taken', '5000000'),
    ],
    getCreateddate: () => makeTimestamp(),
  }) as any;

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1'],
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/stores/endpoint/endpoint-log.store', () => ({
  useEndpointLogPage: () => ({
    getLogs: mockGetLogs,
    getLog: mockGetLog,
    addCriterias: mockAddCriterias,
    endpointLogs: [makeLog()],
    onChangeLogs: mockOnChangeLogs,
    columns,
    page: 1,
    setPage: mockSetPage,
    totalCount: 1,
    criteria: [],
    pageSize: 10,
    setPageSize: mockSetPageSize,
    visibleColumn: (key: string) =>
      columns.some(c => c.key === key && c.visible),
  }),
}));

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: () => null,
}));

jest.mock('@/app/components/dialogs/endpoint/endpoint-trace-modal', () => ({
  EndpointTraceModal: ({ modalOpen }: any) =>
    modalOpen ? <section>endpoint-log-modal</section> : null,
}));

jest.mock('@/app/components/ui/composites/date-filter', () => ({
  DateFilter: () => <div>date-filter</div>,
}));

jest.mock('@/app/components/ui/feedback/status-indicator', () => ({
  CarbonStatusIndicator: ({ state }: any) => <span>Status {state}</span>,
}));

jest.mock('@/app/components/domain/indicators/source', () => ({
  SourceIndicator: ({ source }: any) => <span>Source {source}</span>,
}));

jest.mock('@/app/components/ui/primitives/pagination', () => ({
  Pagination: () => <div>pagination</div>,
}));

jest.mock('@/app/components/ui/feedback/empty-state', () => ({
  EmptyState: ({ title }: any) => <div>{title}</div>,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  IconOnlyButton: ({ iconDescription, onClick }: any) => (
    <button type="button" aria-label={iconDescription} onClick={onClick}>
      {iconDescription}
    </button>
  ),
}));

jest.mock('@/app/components/ui/primitives/buttons/copy-button', () => ({
  CopyButton: ({ children }: any) => (
    <button type="button">Copy {children}</button>
  ),
}));

jest.mock('@/app/components/layout/sections/table-section', () => ({
  ScrollableTableSection: ({ children }: any) => (
    <section data-testid="scrollable-table">{children}</section>
  ),
}));

jest.mock('@/utils/date', () => ({
  formatNanoToReadableMilli: (value: string | number) =>
    `${(Number(value) / 1_000_000).toFixed(2)} ms`,
  toDateString: () => '2026-05-10',
  toHumanReadableDateTime: () => 'May 10, 2026',
}));

jest.mock('@carbon/react', () => {
  const React = require('react');
  const Div = ({ children, className, ...props }: any) =>
    React.createElement('div', { className, ...props }, children);
  return {
    Table: ({ children, className }: any) => (
      <table data-testid="endpoint-log-table" className={className}>
        {children}
      </table>
    ),
    TableHead: ({ children }: any) => <thead>{children}</thead>,
    TableRow: ({ children }: any) => <tr>{children}</tr>,
    TableHeader: ({ children }: any) => <th>{children}</th>,
    TableBody: ({ children }: any) => <tbody>{children}</tbody>,
    TableCell: ({ children, className }: any) => (
      <td className={className}>{children}</td>
    ),
    TableToolbar: Div,
    TableToolbarContent: Div,
    TableToolbarSearch: ({ placeholder }: any) => (
      <input placeholder={placeholder} />
    ),
    Loading: () => <div>loading</div>,
  };
});

jest.mock('react-hot-toast/headless', () => ({
  error: jest.fn(),
}));

describe('Endpoint logs table', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetLogs.mockImplementation(
      (_endpointId, _projectId, _token, _userId, _onError, onSuccess) => {
        onSuccess([makeLog()]);
      },
    );
    mockGetLog.mockImplementation(
      (
        _endpointId,
        _logId,
        _projectId,
        _token,
        _userId,
        _onError,
        onSuccess,
      ) => {
        onSuccess(makeLog());
      },
    );
  });

  it('aligns endpoint log table cells with platform log tables', async () => {
    render(
      <EndpointTraces
        currentEndpoint={
          {
            getId: () => 'endpoint-1',
          } as any
        }
      />,
    );

    await waitFor(() => expect(mockGetLogs).toHaveBeenCalledTimes(1));
    expect(
      screen.getByPlaceholderText('Search endpoint logs'),
    ).toBeInTheDocument();
    expect(screen.getByTestId('scrollable-table')).toBeInTheDocument();
    expect(screen.getByTestId('endpoint-log-table')).toHaveClass('min-w-max');

    expect(screen.queryByText('Trace ID')).not.toBeInTheDocument();
    expect(screen.getByText('Actions')).toBeInTheDocument();
    expect(screen.getByText('Date')).toBeInTheDocument();
    expect(screen.getByText('endpoint-log-1')).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Copy trace-123' }),
    ).not.toBeInTheDocument();
    expect(screen.getByText('vrsn_model-1')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Copy vrsn_model-1' }),
    ).toBeInTheDocument();
    expect(screen.getByText('3.00 ms')).toBeInTheDocument();
    expect(screen.getByText('5.00 ms')).toBeInTheDocument();
    expect(screen.getByText('12')).toBeInTheDocument();
    expect(screen.getByText('May 10, 2026')).toBeInTheDocument();
  });

  it('uses flat row actions to open endpoint log detail', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <EndpointTraces
        currentEndpoint={
          {
            getId: () => 'endpoint-1',
          } as any
        }
      />,
    );

    await waitFor(() => expect(mockGetLogs).toHaveBeenCalledTimes(1));
    await user.click(screen.getByRole('button', { name: 'View detail' }));

    expect(mockGetLog).toHaveBeenCalledWith(
      'endpoint-1',
      'endpoint-log-1',
      'project-1',
      'token-1',
      'user-1',
      expect.any(Function),
      expect.any(Function),
    );
    expect(screen.getByText('endpoint-log-modal')).toBeInTheDocument();
  });

  it('keeps the detail modal closed when endpoint log detail fails', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockGetLog.mockImplementationOnce(
      (_endpointId, _logId, _projectId, _token, _userId, onError) => {
        onError('Unable to get endpoint log');
      },
    );

    render(
      <EndpointTraces
        currentEndpoint={
          {
            getId: () => 'endpoint-1',
          } as any
        }
      />,
    );

    await waitFor(() => expect(mockGetLogs).toHaveBeenCalledTimes(1));
    await user.click(screen.getByRole('button', { name: 'View detail' }));

    expect(toast.error).toHaveBeenCalledWith('Unable to get endpoint log');
    expect(screen.queryByText('endpoint-log-modal')).not.toBeInTheDocument();
  });
});
