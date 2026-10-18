import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

const mockGoToAssistantSessionList = jest.fn();
const mockGetAssistantDashboardRange = jest.fn();
const mockToastError = jest.fn();
let mockCredential = {
  authId: 'auth-1',
  token: 'token-1',
  projectId: 'project-1',
};

const createTimestamp = (seconds: number) => ({
  getSeconds: () => seconds,
  getNanos: () => 0,
});

const mockDashboard = {
  getSummary: () => ({
    getTotalsessions: () => 3,
    getActivesessions: () => 1,
    getCompletedsessions: () => 1,
    getFailedsessions: () => 1,
    getTotalmessages: () => 12,
    getUsermessages: () => 6,
    getFailurerate: () => 33.3,
    getAveragesessiondurationseconds: () => 45,
  }),
  getLatency: () => ({
    getAveragems: () => 150,
    getSttms: () => 25,
    getEosms: () => 42,
    getTtsms: () => 60,
    getLlmms: () => 120,
  }),
  getUsage: () => ({
    getTotaltokens: () => 500,
    getSttdurationseconds: () => 10,
    getTtsdurationseconds: () => 20,
    getTotaldurationseconds: () => 90,
  }),
  getSourcesList: () => [
    {
      getName: () => 'web',
      getCount: () => 8,
      getPercentage: () => 66.7,
    },
  ],
  getLanguagesList: () => [
    {
      getName: () => 'en',
      getCount: () => 6,
      getPercentage: () => 50,
    },
  ],
  getBucketsList: () => [
    {
      getStartdate: () => createTimestamp(1710000000),
      getMessagecount: () => 4,
      getSttlatencyms: () => 25,
      getEoslatencyms: () => 42,
      getTtslatencyms: () => 60,
      getLlmlatencyms: () => 120,
    },
  ],
};

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToAssistantSessionList: mockGoToAssistantSessionList,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => mockCredential,
}));

jest.mock('@/clients/assistant.client', () => ({
  getAssistantDashboardRange: (...args: unknown[]) =>
    mockGetAssistantDashboardRange(...args),
}));

jest.mock('recharts', () => {
  const Container = ({ children }: any) => <div>{children}</div>;
  const Null = () => null;
  return {
    XAxis: Null,
    Tooltip: Null,
    ResponsiveContainer: Container,
    PieChart: Null,
    Pie: Null,
    Cell: Null,
    Bar: Null,
    BarChart: Null,
    YAxis: Null,
    AreaChart: Null,
    Area: Null,
  };
});

jest.mock('@/app/components/ui/primitives/dropdown', () => ({
  Dropdown: ({ label }: any) => <div>{label}</div>,
}));

jest.mock('@/app/components/ui/primitives/tile', () => ({
  Tile: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('react-hot-toast/headless', () => ({
  success: jest.fn(),
  error: (...args: unknown[]) => mockToastError(...args),
}));

jest.mock('@carbon/react', () => ({
  Button: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  SkeletonPlaceholder: ({ className }: any) => (
    <div className={className} data-testid="skeleton-placeholder" />
  ),
  SkeletonText: ({ className, heading, width }: any) => (
    <span
      className={className}
      data-heading={heading ? 'true' : 'false'}
      data-testid="skeleton-text"
      style={{ width }}
    />
  ),
  Toggletip: ({ children }: any) => <span>{children}</span>,
  ToggletipButton: ({ label }: any) => <button type="button">{label}</button>,
  ToggletipContent: ({ children }: any) => {
    const nodes = require('react').Children.toArray(children);
    return <span>{nodes[nodes.length - 1]}</span>;
  },
  ToggletipLabel: ({ children }: any) => <span>{children}</span>,
}));

const {
  AssistantAnalytics,
} = require('@/app/pages/assistant/view/overview/assistant-analytics');

describe('AssistantAnalytics sessions toggletip', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockCredential = {
      authId: 'auth-1',
      token: 'token-1',
      projectId: 'project-1',
    };
    mockGetAssistantDashboardRange.mockResolvedValue({
      getSuccess: () => true,
      getData: () => mockDashboard,
    });
  });

  it('shows sessions toggletip action and navigates to sessions page', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const assistant = { getId: () => 'assistant-1' } as any;
    render(<AssistantAnalytics assistant={assistant} />);

    await waitFor(() => {
      expect(screen.getByText('500 tokens used')).toBeInTheDocument();
      expect(
        screen.getByRole('button', { name: 'Go to sessions' }),
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: 'Go to sessions' }));

    expect(mockGoToAssistantSessionList).toHaveBeenCalledWith('assistant-1');
  });

  it('keeps dashboard card structure visible while metrics are loading', () => {
    mockGetAssistantDashboardRange.mockReturnValue(new Promise(() => {}));

    const assistant = { getId: () => 'assistant-1' } as any;
    render(<AssistantAnalytics assistant={assistant} />);

    expect(screen.getByText('Sessions')).toBeInTheDocument();
    expect(screen.getByText('Messages')).toBeInTheDocument();
    expect(screen.getByText('Latency')).toBeInTheDocument();
    expect(screen.getByText('Sources')).toBeInTheDocument();
    expect(screen.getByText('Message activity')).toBeInTheDocument();
    expect(screen.getAllByTestId('skeleton-text').length).toBeGreaterThan(0);
    expect(
      screen.getAllByTestId('skeleton-placeholder').length,
    ).toBeGreaterThan(0);
    expect(screen.queryByText('Total sessions')).not.toBeInTheDocument();
  });

  it('keeps sessions navigation action scoped to sessions metric', async () => {
    const assistant = { getId: () => 'assistant-1' } as any;
    render(<AssistantAnalytics assistant={assistant} />);

    await waitFor(() => {
      expect(screen.getByText('500 tokens used')).toBeInTheDocument();
      expect(screen.getByText('Failure rate')).toBeInTheDocument();
    });

    expect(
      screen.getAllByRole('button', { name: 'Go to sessions' }),
    ).toHaveLength(1);
  });

  it('includes all overview latency metrics in the latency summary', async () => {
    const assistant = { getId: () => 'assistant-1' } as any;
    render(<AssistantAnalytics assistant={assistant} />);

    await waitFor(() => {
      expect(screen.getByText('25')).toBeInTheDocument();
      expect(screen.getByText('42')).toBeInTheDocument();
      expect(screen.getByText('60')).toBeInTheDocument();
      expect(screen.getByText('120')).toBeInTheDocument();
      expect(screen.getAllByText('STT')).toHaveLength(2);
      expect(screen.getAllByText('EOS')).toHaveLength(2);
      expect(screen.getAllByText('TTS')).toHaveLength(2);
      expect(screen.getAllByText('Agent')).toHaveLength(2);
    });
  });

  it('loads dashboard using assistant id, date range, and auth context', async () => {
    const assistant = { getId: () => 'assistant-1' } as any;
    render(<AssistantAnalytics assistant={assistant} />);

    await waitFor(() => {
      expect(mockGetAssistantDashboardRange).toHaveBeenCalled();
      expect(screen.getByText('500 tokens used')).toBeInTheDocument();
    });

    expect(mockGetAssistantDashboardRange).toHaveBeenCalledWith(
      expect.objectContaining({
        assistantId: 'assistant-1',
        fromDate: expect.any(Object),
        toDate: expect.any(Object),
        auth: {
          projectId: 'project-1',
          token: 'token-1',
          userId: 'auth-1',
        },
      }),
    );
  });

  it('converts dashboard date range into protobuf timestamps', async () => {
    const assistant = { getId: () => 'assistant-1' } as any;
    render(<AssistantAnalytics assistant={assistant} />);

    await waitFor(() => {
      expect(mockGetAssistantDashboardRange).toHaveBeenCalled();
    });

    const params = mockGetAssistantDashboardRange.mock.calls[0][0];
    expect(typeof params.fromDate.getSeconds()).toBe('number');
    expect(typeof params.toDate.getSeconds()).toBe('number');
    expect(params.toDate.getSeconds()).toBeGreaterThanOrEqual(
      params.fromDate.getSeconds(),
    );
  });

  it('does not load dashboard until credential context is ready', () => {
    mockCredential = {
      authId: '',
      token: '',
      projectId: '',
    };

    const assistant = { getId: () => 'assistant-1' } as any;
    render(<AssistantAnalytics assistant={assistant} />);

    expect(mockGetAssistantDashboardRange).not.toHaveBeenCalled();
  });

  it('toasts and shows unavailable state instead of zero metrics when dashboard load fails', async () => {
    mockGetAssistantDashboardRange.mockRejectedValue(
      new Error('network failure'),
    );

    const assistant = { getId: () => 'assistant-1' } as any;
    render(<AssistantAnalytics assistant={assistant} />);

    await waitFor(() => {
      expect(mockToastError).toHaveBeenCalledWith(
        'Dashboard data is unavailable. Please try again.',
      );
    });

    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(screen.getAllByText('--').length).toBeGreaterThan(0);
    expect(screen.queryByText('0 tokens used')).not.toBeInTheDocument();
  });

  it('puts important KPIs before dashboard detail widgets', async () => {
    const assistant = { getId: () => 'assistant-1' } as any;
    render(<AssistantAnalytics assistant={assistant} />);

    await waitFor(() => {
      expect(screen.getByText('500 tokens used')).toBeInTheDocument();
      expect(screen.getByText('Assistant activity')).toBeInTheDocument();
    });

    expect(screen.getByText('Total sessions')).toBeInTheDocument();
    expect(screen.getByText('Total messages')).toBeInTheDocument();
    expect(screen.getByText('Average response latency')).toBeInTheDocument();
    expect(screen.getByText('Failed sessions')).toBeInTheDocument();
    expect(screen.getByText('Message activity')).toBeInTheDocument();
    expect(
      screen.getByText('Messages over selected range'),
    ).toBeInTheDocument();
    expect(screen.getByText('Avg session duration')).toBeInTheDocument();
    expect(screen.getByText('Usage totals')).toBeInTheDocument();
    expect(screen.queryByText('Summary Dashboard')).not.toBeInTheDocument();
  });

  it('keeps message activity after summary dashboard widgets', async () => {
    const assistant = { getId: () => 'assistant-1' } as any;
    render(<AssistantAnalytics assistant={assistant} />);

    await waitFor(() => {
      expect(screen.getByText('500 tokens used')).toBeInTheDocument();
      expect(screen.getByText('Message activity')).toBeInTheDocument();
    });

    expect(
      screen
        .getByText('Reliability')
        .compareDocumentPosition(screen.getByText('Message activity')) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });
});
