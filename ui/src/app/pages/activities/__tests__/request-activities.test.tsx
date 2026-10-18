import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { ListingPage } from '@/app/pages/activities/webhook-activities';
import { RetryHTTPLog } from '@rapidaai/react';

const mockGetActivities = jest.fn();
const mockAddCriterias = jest.fn();
const mockSetCriterias = jest.fn();
const mockOnChangeActivities = jest.fn();
const mockSetPage = jest.fn();
const mockSetPageSize = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();

const makeLog = () =>
  ({
    getId: () => 'http-log-1',
    getSourcerefid: () => 'req-123',
    getAssistantid: () => 'assistant-1',
    getAssistantconversationid: () => 'session-1',
    getSourceevent: () => 'conversation.begin',
    getHttpmethod: () => 'POST',
    getHttpurl: () => 'https://api.example.com/request',
    getResponsestatus: () => '200',
    getTimetaken: () => '1000000',
    getRetrycount: () => 0,
    getCreateddate: () => undefined,
  }) as any;

jest.mock('@rapidaai/react', () => {
  class ConnectionConfig {}

  class RetryAssistantHTTPLogRequest {
    projectId = '';
    id = '';
    setProjectid(value: string) {
      this.projectId = value;
    }
    setId(value: string) {
      this.id = value;
    }
    getProjectid() {
      return this.projectId;
    }
    getId() {
      return this.id;
    }
  }

  return {
    ConnectionConfig,
    RetryAssistantHTTPLogRequest,
    RetryHTTPLog: jest.fn(),
  };
});

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['u1', 't1', 'p1'],
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/stores/activity/webhook-log.store', () => ({
  useWebhookLogPage: () => ({
    getActivities: mockGetActivities,
    addCriterias: mockAddCriterias,
    setCriterias: mockSetCriterias,
    webhookLogs: [makeLog()],
    onChangeActivities: mockOnChangeActivities,
    columns: [
      { name: 'Request ID', key: 'sourcerefid', visible: true },
      { name: 'Session ID', key: 'sessionid', visible: true },
      { name: 'Event', key: 'event', visible: true },
      { name: 'Endpoint', key: 'endpoint', visible: true },
      { name: 'Actions', key: 'action', visible: true },
      { name: 'Http status', key: 'responsestatus', visible: true },
      { name: 'Time Taken', key: 'timetaken', visible: true },
      { name: 'Retry Count', key: 'retrycount', visible: true },
      { name: 'Date', key: 'created_date', visible: true },
    ],
    page: 1,
    setPage: mockSetPage,
    totalCount: 1,
    criteria: [],
    pageSize: 10,
    visibleColumn: (key: string) =>
      [
        'sourcerefid',
        'sessionid',
        'event',
        'endpoint',
        'action',
        'responsestatus',
        'timetaken',
        'retrycount',
        'created_date',
      ].includes(key),
    setPageSize: mockSetPageSize,
    setColumns: jest.fn(),
  }),
}));

jest.mock('@/app/pages/assistant/actions/hooks/use-confirmation', () => {
  const React = require('react');
  return {
    useConfirmDialog: ({ title = 'Confirm' }: { title?: string } = {}) => {
      const [isOpen, setIsOpen] = React.useState(false);
      const [onConfirm, setOnConfirm] = React.useState<() => void>(
        () => () => {},
      );
      return {
        showDialog: (cb: () => void) => {
          setOnConfirm(() => cb);
          setIsOpen(true);
        },
        ConfirmDialogComponent: () =>
          isOpen ? (
            <button
              onClick={() => {
                onConfirm();
                setIsOpen(false);
              }}
            >
              {title}
            </button>
          ) : null,
      };
    },
  };
});

jest.mock('@/app/components/dialogs/activity', () => ({
  RequestLogDialog: ({ modalOpen }: any) =>
    modalOpen ? <div>request-log-modal</div> : null,
}));

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: () => null,
}));

jest.mock('@/app/components/ui/composites/query-search', () => ({
  QuerySearch: ({ placeholder }: any) => <input placeholder={placeholder} />,
  parseQuerySearchFilters: () => [],
}));

jest.mock('@/app/components/layout/blocks/page-title-with-count', () => ({
  PageTitleWithCount: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/layout/blocks/page-header-block', () => ({
  PageHeaderBlock: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/ui/table/table-link', () => ({
  TableLink: ({ children, href }: any) => <a href={href}>{children}</a>,
}));

jest.mock('@/app/components/ui/primitives/pagination', () => ({
  Pagination: () => <div>pagination</div>,
}));

jest.mock('@/app/components/ui/feedback/empty-state', () => ({
  EmptyState: ({ title }: any) => <div>{title}</div>,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  IconOnlyButton: ({
    iconDescription,
    children,
    renderIcon: _renderIcon,
    ...props
  }: any) => (
    <button aria-label={iconDescription} {...props}>
      {children || iconDescription}
    </button>
  ),
}));

jest.mock('@/utils/date', () => ({
  formatNanoToReadableMilli: () => '1 ms',
  toDateString: () => '2026-05-10',
  toHumanReadableDateTime: () => 'May 10, 2026',
}));

jest.mock('@/app/components/domain/indicators/http-status', () => ({
  HttpStatusSpanIndicator: ({ status }: any) => <span>{status}</span>,
}));

jest.mock('@carbon/react', () => {
  const React = require('react');
  const Div = ({ children }: any) => React.createElement('div', null, children);
  return {
    Table: Div,
    TableHead: Div,
    TableRow: Div,
    TableHeader: Div,
    TableBody: Div,
    TableCell: Div,
    TableToolbar: Div,
    TableToolbarContent: Div,
    TableToolbarSearch: ({ placeholder }: any) =>
      React.createElement('input', { placeholder }),
    Loading: () => React.createElement('div', null, 'loading'),
    Tag: ({ children }: any) => React.createElement('span', null, children),
    Link: ({ href, children, className }: any) =>
      React.createElement('a', { href, className }, children),
  };
});

jest.mock('react-hot-toast/headless', () => ({
  success: jest.fn(),
  error: jest.fn(),
}));

describe('Request logs listing', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetActivities.mockImplementation(
      (_projectId, _token, _userId, _onError, onSuccess) => {
        onSuccess([makeLog()]);
      },
    );
    (RetryHTTPLog as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
    });
  });

  it('renders request log data using HTTP log fields', async () => {
    render(<ListingPage />);

    await waitFor(() => expect(mockGetActivities).toHaveBeenCalledTimes(1));
    expect(screen.getByText('Request Logs')).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText(
        'Search for logID, requestID, endpoint, status and more',
      ),
    ).toBeInTheDocument();
    expect(screen.getByText('req-123')).toBeInTheDocument();
    expect(screen.getByText('conversation.begin')).toBeInTheDocument();
  });

  it('retries request only after confirm dialog acceptance', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<ListingPage />);

    await waitFor(() => expect(mockGetActivities).toHaveBeenCalledTimes(1));
    await user.click(screen.getByRole('button', { name: 'Retry request' }));

    expect(RetryHTTPLog).not.toHaveBeenCalled();
    await user.click(screen.getByRole('button', { name: 'Retry request?' }));

    await waitFor(() => expect(RetryHTTPLog).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(mockGetActivities).toHaveBeenCalledTimes(2));
  });
});
