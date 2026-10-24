import React from 'react';
import { act, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import userEvent from '@testing-library/user-event';

import {
  DashboardAssistantHealth,
  DashboardFailureCode,
  DashboardSectionState,
  DashboardUsageComponent,
} from '@rapidaai/react';

import {
  dashboardActivityLabel,
  formatDashboardDuration,
  getDashboardStartDate,
  HomePage,
} from '@/app/pages/main/web-dashboard';
import { getProjectDashboardRange } from '@/clients/dashboard.client';

jest.mock('@/clients/dashboard.client', () => ({
  getProjectDashboardRange: jest.fn(),
}));

let mockCurrentCredential: {
  authId: string;
  projectId: string;
  token: string;
  user?: { name?: string };
} = {
  authId: 'user-1',
  projectId: 'project-1',
  token: 'token-1',
  user: { name: 'Jordan Lee' },
};

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => mockCurrentCredential,
}));

const mockGetProjectDashboardRange =
  getProjectDashboardRange as jest.MockedFunction<
    typeof getProjectDashboardRange
  >;

const timestamp = (value: string) => ({
  toDate: () => new Date(value),
});

const comparison = (value: number, change: number) => ({
  getValue: () => value,
  getPreviousvalue: () => value - change,
  hasChangepercent: () => true,
  getChangepercent: () => change,
});

function dashboardResponse({
  sessions = 12486,
  analyticsState = DashboardSectionState.DASHBOARD_SECTION_STATE_READY,
  usageState = DashboardSectionState.DASHBOARD_SECTION_STATE_READY,
  partial = false,
  failureDurationMs = 94000,
}: {
  sessions?: number;
  analyticsState?: number;
  usageState?: number;
  partial?: boolean;
  failureDurationMs?: number;
} = {}) {
  const analytics = {
    getProjectstate: () => ({
      getAssistantcount: () => 3,
      getHasassistants: () => true,
      getHasactivity: () => true,
    }),
    getSummary: () => ({
      getSessions: () => comparison(sessions, 14.2),
      getSuccessrate: () => ({
        hasValuepercent: () => true,
        getValuepercent: () => 94.8,
        hasChangepoints: () => true,
        getChangepoints: () => 2.1,
      }),
      getResponselatency: () => ({
        hasValuems: () => true,
        getValuems: () => 684,
        hasChangems: () => true,
        getChangems: () => -58,
      }),
      getFailedsessions: () => ({
        getValue: () => 649,
        hasSharepercent: () => true,
        getSharepercent: () => 5.2,
      }),
    }),
    getActivityList: () => [
      {
        getFromdate: () => timestamp('2026-10-16T00:00:00Z'),
        getSessions: () => 428,
        getFailedsessions: () => 18,
      },
      {
        getFromdate: () => timestamp('2026-10-17T00:00:00Z'),
        getSessions: () => 486,
        getFailedsessions: () => 21,
      },
    ],
    getHealth: () => ({
      getFailedsessions: () => 649,
      getSlowsessions: () => 27,
      getSlowsessionthresholdms: () => 1000,
      getHealthyassistants: () => 1,
      getWatchassistants: () => 1,
      getNeedsattentionassistants: () => 1,
      getNodataassistants: () => 0,
    }),
    getAssistantsList: () => [
      {
        getAssistantid: () => 'assistant-1',
        getName: () => 'Support bot',
        getSessions: () => 5392,
        hasSuccessratepercent: () => true,
        getSuccessratepercent: () => 97.2,
        hasResponselatencyms: () => true,
        getResponselatencyms: () => 612,
        getLastactivityat: () => timestamp('2026-10-17T08:28:00Z'),
        getHealth: () =>
          DashboardAssistantHealth.DASHBOARD_ASSISTANT_HEALTH_HEALTHY,
      },
    ],
    getRecentfailuresList: () => [
      {
        getConversationid: () => 'conversation-1',
        getAssistantid: () => 'assistant-1',
        getAssistantname: () => 'Support bot',
        getOccurredat: () => timestamp('2026-10-17T08:25:00Z'),
        getChannel: () => 'Phone',
        getDurationms: () => failureDurationMs,
        getFailurecode: () =>
          DashboardFailureCode.DASHBOARD_FAILURE_CODE_PROVIDER,
      },
    ],
  };

  const dashboard = {
    getPartial: () => partial,
    getAnalytics: () => ({
      getState: () => analyticsState,
      getData: () =>
        analyticsState === DashboardSectionState.DASHBOARD_SECTION_STATE_READY
          ? analytics
          : undefined,
    }),
    getUsage: () => ({
      getState: () => usageState,
      getComponentsList: () => [
        {
          getComponent: () =>
            DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_STT,
          getDurationms: () => 7200000,
        },
        {
          getComponent: () =>
            DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_LLM,
          getDurationms: () => 5400000,
        },
      ],
    }),
  };

  return {
    getSuccess: () => true,
    getData: () => dashboard,
  } as never;
}

describe('Dashboard HomePage', () => {
  beforeAll(() => {
    Element.prototype.scrollIntoView = jest.fn();
  });

  afterAll(() => {
    delete (Element.prototype as Partial<Element>).scrollIntoView;
  });

  beforeEach(() => {
    jest.clearAllMocks();
    mockCurrentCredential = {
      authId: 'user-1',
      projectId: 'project-1',
      token: 'token-1',
      user: { name: 'Jordan Lee' },
    };
    mockGetProjectDashboardRange.mockResolvedValue(dashboardResponse());
  });

  it('renders the project dashboard from the public dashboard API', async () => {
    const { container } = render(<HomePage />);

    expect(screen.queryByText('Welcome, Jordan!')).not.toBeInTheDocument();
    expect(screen.queryByText('Project overview')).not.toBeInTheDocument();
    expect(await screen.findByText('12,486')).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Dashboard', level: 1 }),
    ).toBeInTheDocument();
    expect(screen.getByText('94.8%')).toBeInTheDocument();
    expect(screen.getByText('Component usage')).toBeInTheDocument();
    expect(screen.getByText('Recent failures')).toBeInTheDocument();
    expect(screen.getByText('Top assistants')).toBeInTheDocument();
    expect(screen.getByText('Healthy')).toBeInTheDocument();
    const supportLinks = screen.getAllByRole('link', { name: 'Support bot' });
    expect(supportLinks.map(link => link.getAttribute('href'))).toEqual(
      expect.arrayContaining([
        '/deployment/assistant/assistant-1',
        '/deployment/assistant/assistant-1/sessions/conversation-1',
      ]),
    );
    expect(screen.queryByText('Sample data')).not.toBeInTheDocument();
    expect(container.querySelectorAll('[data-dashboard-kpi]')).toHaveLength(4);
    expect(container.querySelectorAll('[data-dashboard-bar]')).toHaveLength(2);
    expect(mockGetProjectDashboardRange).toHaveBeenCalledWith(
      expect.objectContaining({
        assistantLimit: 3,
        recentFailureLimit: 5,
        auth: {
          projectId: 'project-1',
          token: 'token-1',
          userId: 'user-1',
        },
      }),
    );
  });

  it('uses the assistant dashboard surface and bordered card layout', async () => {
    const { container } = render(<HomePage />);

    expect(await screen.findByText('12,486')).toBeInTheDocument();
    expect(container.querySelector('[data-dashboard-page]')).toHaveClass(
      'bg-gray-100',
      'dark:bg-[#161616]',
    );
    expect(container.querySelector('[data-dashboard-kpi-strip]')).toHaveClass(
      'gap-4',
    );

    const cards = container.querySelectorAll(
      '[data-dashboard-kpi], [data-dashboard-panel]',
    );
    expect(cards).toHaveLength(6);
    cards.forEach(card => {
      expect(card).toHaveClass(
        'border',
        'border-gray-200',
        'bg-white',
        'dark:border-gray-800',
        'dark:bg-[#262626]',
      );
    });
  });

  it('uses canonical Carbon typography across dashboard content', async () => {
    render(<HomePage />);

    expect(await screen.findByText('12,486')).toHaveClass(
      'text-[2rem]',
      'font-normal',
      'leading-[2.5rem]',
      'tracking-[0]',
      'tabular-nums',
    );
    expect(screen.queryByText('Project overview')).not.toBeInTheDocument();
    expect(
      screen.queryByText(
        'Monitor sessions, reliability, latency, and assistant health.',
      ),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Conversation activity' }),
    ).toHaveClass(
      'text-[1rem]',
      'font-[600]',
      'leading-[1.5rem]',
      'tracking-[0]',
    );
    expect(
      screen.getByText('Sessions and failures from the last 30 days'),
    ).toHaveClass(
      'text-[0.875rem]',
      'font-normal',
      'leading-[1.125rem]',
      'tracking-[0.16px]',
    );
    expect(screen.getByText('Based on success rate and latency')).toHaveClass(
      'text-[0.75rem]',
      'font-normal',
      'leading-[1rem]',
      'tracking-[0.32px]',
    );
    expect(screen.getByText('Speech to text')).toHaveClass(
      'text-[0.875rem]',
      'font-normal',
      'leading-[1.125rem]',
      'tracking-[0.16px]',
    );
    screen.getAllByRole('link', { name: 'Support bot' }).forEach(link => {
      expect(link.firstElementChild).toHaveClass(
        'text-[0.875rem]',
        'font-[600]',
        'leading-[1.125rem]',
        'tracking-[0.16px]',
      );
    });
  });

  it('shows a layout skeleton while the first request is pending', () => {
    mockGetProjectDashboardRange.mockReturnValue(new Promise(() => {}));

    const { container } = render(<HomePage />);

    expect(
      screen.getByRole('status', { name: 'Loading project dashboard' }),
    ).toBeInTheDocument();
    expect(screen.queryByText('Top assistants')).not.toBeInTheDocument();
    expect(
      container.querySelectorAll('[data-dashboard-loading-card]'),
    ).toHaveLength(8);
  });

  it('shows a retry action when the initial request fails', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    mockGetProjectDashboardRange
      .mockRejectedValueOnce(new Error('unavailable'))
      .mockResolvedValueOnce(dashboardResponse());

    render(<HomePage />);

    expect(
      await screen.findByRole('heading', {
        name: 'Project dashboard is unavailable',
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Check your connection and try again.'),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByText('12,486')).toBeInTheDocument();
    expect(mockGetProjectDashboardRange).toHaveBeenCalledTimes(2);
  });

  it('renders the available section when the response is partial', async () => {
    mockGetProjectDashboardRange.mockResolvedValue(
      dashboardResponse({
        analyticsState:
          DashboardSectionState.DASHBOARD_SECTION_STATE_UNAVAILABLE,
        partial: true,
      }),
    );

    render(<HomePage />);

    expect(await screen.findByText('Component usage')).toBeInTheDocument();
    expect(screen.getByText('Analytics are unavailable')).toBeInTheDocument();
    expect(screen.getByText('Partial data')).toBeInTheDocument();
  });

  it('keeps analytics visible when usage is unavailable', async () => {
    mockGetProjectDashboardRange.mockResolvedValue(
      dashboardResponse({
        usageState: DashboardSectionState.DASHBOARD_SECTION_STATE_UNAVAILABLE,
        partial: true,
      }),
    );

    render(<HomePage />);

    expect(await screen.findByText('12,486')).toBeInTheDocument();
    expect(
      screen.getByText('Component usage is unavailable'),
    ).toBeInTheDocument();
    expect(screen.getByText('Top assistants')).toBeInTheDocument();
  });

  it('renders a simple assistant CTA when the dashboard has no activity', async () => {
    mockGetProjectDashboardRange.mockResolvedValue(
      dashboardResponse({
        analyticsState: DashboardSectionState.DASHBOARD_SECTION_STATE_EMPTY,
        usageState: DashboardSectionState.DASHBOARD_SECTION_STATE_EMPTY,
      }),
    );

    render(<HomePage />);

    expect(
      await screen.findByRole('heading', {
        name: 'Your dashboard is not ready yet',
        level: 1,
      }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('heading', { name: 'Dashboard', level: 1 }),
    ).not.toBeInTheDocument();
    expect(screen.getByText('Getting started')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Create an assistant and run a test conversation. Sessions, latency, and health will appear here after the first session.',
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: 'Create assistant' }),
    ).toHaveAttribute('href', '/deployment/assistant/create-assistant');
    expect(
      screen.queryByRole('link', { name: 'View assistants' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Add credential' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('combobox', { name: 'Date range' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('combobox', { name: 'Auto-refresh' }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('Recommended setup')).not.toBeInTheDocument();
    expect(screen.queryByText('No analytics yet')).not.toBeInTheDocument();
    expect(
      screen.queryByText('No component usage yet'),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText('No recent failures yet'),
    ).not.toBeInTheDocument();
  });

  it('replaces the empty dashboard when the project changes', async () => {
    mockGetProjectDashboardRange
      .mockResolvedValueOnce(
        dashboardResponse({
          analyticsState: DashboardSectionState.DASHBOARD_SECTION_STATE_EMPTY,
          usageState: DashboardSectionState.DASHBOARD_SECTION_STATE_EMPTY,
        }),
      )
      .mockResolvedValueOnce(dashboardResponse({ sessions: 212 }));

    const { rerender } = render(<HomePage />);

    await screen.findByRole('heading', {
      name: 'Your dashboard is not ready yet',
    });
    mockCurrentCredential = {
      ...mockCurrentCredential,
      projectId: 'project-2',
    };
    rerender(<HomePage />);

    expect(await screen.findByText('212')).toBeInTheDocument();
    expect(
      screen.queryByRole('heading', {
        name: 'Your dashboard is not ready yet',
      }),
    ).not.toBeInTheDocument();
  });

  it('preserves section fallbacks when only analytics are empty', async () => {
    mockGetProjectDashboardRange.mockResolvedValue(
      dashboardResponse({
        analyticsState: DashboardSectionState.DASHBOARD_SECTION_STATE_EMPTY,
      }),
    );

    render(<HomePage />);

    expect(await screen.findByText('No analytics yet')).toBeInTheDocument();
    expect(screen.getByText('Component usage')).toBeInTheDocument();
    expect(
      screen.queryByRole('heading', {
        name: 'Your dashboard is not ready yet',
      }),
    ).not.toBeInTheDocument();
  });

  it('ignores a stale response after the project changes', async () => {
    let resolveFirst: (value: never) => void = () => undefined;
    const firstRequest = new Promise<never>(resolve => {
      resolveFirst = resolve;
    });
    mockGetProjectDashboardRange
      .mockReturnValueOnce(firstRequest)
      .mockResolvedValueOnce(dashboardResponse({ sessions: 3248 }));

    const { rerender } = render(<HomePage />);
    mockCurrentCredential = {
      ...mockCurrentCredential,
      projectId: 'project-2',
    };
    rerender(<HomePage />);

    expect(await screen.findByText('3,248')).toBeInTheDocument();
    await act(async () => {
      resolveFirst(dashboardResponse({ sessions: 12486 }));
      await firstRequest;
    });
    expect(screen.getByText('3,248')).toBeInTheDocument();
    expect(screen.queryByText('12,486')).not.toBeInTheDocument();
  });

  it('hides old project data before the next project request completes', async () => {
    const { rerender } = render(<HomePage />);
    expect(await screen.findByText('12,486')).toBeInTheDocument();

    let resolveNext: (value: never) => void = () => undefined;
    const nextRequest = new Promise<never>(resolve => {
      resolveNext = resolve;
    });
    mockGetProjectDashboardRange.mockReturnValueOnce(nextRequest);
    mockCurrentCredential = {
      authId: 'user-1',
      projectId: 'project-2',
      token: 'token-2',
      user: { name: 'Jordan Lee' },
    };

    rerender(<HomePage />);

    expect(screen.queryByText('12,486')).not.toBeInTheDocument();
    expect(
      screen.getByRole('status', { name: 'Loading project dashboard' }),
    ).toBeInTheDocument();

    await act(async () => {
      resolveNext(dashboardResponse({ sessions: 212 }));
      await nextRequest;
    });
    expect(await screen.findByText('212')).toBeInTheDocument();
  });

  it('formats exact ranges, bucket labels, and duration carry correctly', () => {
    const endDate = new Date('2026-03-08T12:00:00-04:00');
    expect(
      endDate.getTime() - getDashboardStartDate('24h', endDate).getTime(),
    ).toBe(24 * 60 * 60 * 1000);
    expect(
      endDate.getTime() - getDashboardStartDate('7d', endDate).getTime(),
    ).toBe(7 * 24 * 60 * 60 * 1000);
    expect(formatDashboardDuration(7_180_000)).toBe('2 hr');

    const firstBucket = timestamp('2026-10-17T08:00:00Z') as never;
    const secondBucket = timestamp('2026-10-17T08:30:00Z') as never;
    expect(dashboardActivityLabel(firstBucket, '24h')).not.toBe(
      dashboardActivityLabel(secondBucket, '24h'),
    );
    expect(dashboardActivityLabel(firstBucket, '7d')).not.toBe(
      dashboardActivityLabel(timestamp('2026-10-17T12:00:00Z') as never, '7d'),
    );
  });

  it('provides accessible chart and table descriptions', async () => {
    render(<HomePage />);

    expect(
      await screen.findByRole('img', {
        name: 'Conversation activity showing 12,486 sessions and 649 failures over Last 30 days',
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('table', { name: 'Recent failures' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('table', { name: 'Top assistants' }),
    ).toBeInTheDocument();
  });
});
