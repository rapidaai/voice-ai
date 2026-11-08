import React from 'react';
import { act, render, screen, waitFor, within } from '@testing-library/react';
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

const mockLineChart = jest.fn(
  ({ options }: { options: { accessibility?: { svgAriaLabel?: string } } }) => (
    <div
      data-testid="carbon-line-chart"
      role="img"
      aria-label={options.accessibility?.svgAriaLabel}
    />
  ),
);

jest.mock('@carbon/charts-react', () => ({
  Alignments: { RIGHT: 'right' },
  ChartTheme: { G100: 'g100', WHITE: 'white' },
  LegendPositions: { TOP: 'top' },
  LineChart: (properties: unknown) => mockLineChart(properties as never),
  ScaleTypes: { LABELS: 'labels', LINEAR: 'linear' },
}));

jest.mock('@/clients/dashboard.client', () => ({
  getProjectDashboardRange: jest.fn(),
}));

jest.mock('@/theme/theme-provider', () => ({
  useTheme: () => ({ resolvedMode: 'light' }),
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

let pageHeaderIntersectionCallback: IntersectionObserverCallback;

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
    window.IntersectionObserver = jest.fn(callback => {
      pageHeaderIntersectionCallback = callback;
      return {
        disconnect: jest.fn(),
        observe: jest.fn(),
        root: null,
        rootMargin: '0px',
        takeRecords: jest.fn(() => []),
        thresholds: [0],
        unobserve: jest.fn(),
      };
    }) as unknown as typeof IntersectionObserver;
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
      screen.getByRole('heading', { name: 'Workspace Overview', level: 1 }),
    ).toBeInTheDocument();
    expect(screen.getByText('Dashboard')).toHaveClass('text-primary');
    expect(screen.getByText('Dashboard').closest('.cds--tag')).toBeNull();
    expect(container.querySelector('[data-page-header-icon]')).toBeNull();
    expect(screen.getByRole('combobox', { name: 'Time range' })).toHaveValue(
      '30d',
    );
    expect(screen.queryByText('Project performance')).not.toBeInTheDocument();
    expect(
      screen.getByText(
        'Monitor workspace performance, conversation activity, assistant health, and component usage across the selected time range.',
      ),
    ).toBeInTheDocument();
    expect(container.querySelector('[data-page-header]')).toHaveClass(
      'border-b',
      'border-border-subtle',
      'bg-layer',
      'px-5',
      'py-6',
    );
    expect(
      screen.getByRole('heading', { name: 'Success Rate', level: 2 }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Average Latency', level: 2 }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Failed Sessions', level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText('94.8%')).toBeInTheDocument();
    expect(screen.getByText('Component Usage')).toBeInTheDocument();
    expect(screen.getByText('Recent Failures')).toBeInTheDocument();
    expect(screen.getByText('Top Assistants')).toBeInTheDocument();
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

  it('uses bordered KPI and detail surfaces', async () => {
    const { container } = render(<HomePage />);

    expect(await screen.findByText('12,486')).toBeInTheDocument();
    expect(container.querySelector('[data-dashboard-page]')).toHaveClass(
      'bg-layer',
    );
    expect(container.querySelector('[data-dashboard-kpi-strip]')).toHaveClass(
      'gap-4',
    );

    const kpis = container.querySelectorAll('[data-dashboard-kpi]');
    expect(kpis).toHaveLength(4);
    kpis.forEach(card => {
      expect(card).toHaveClass(
        'h-[156px]',
        'border',
        'border-border-subtle',
        'bg-surface',
      );
      expect(card.querySelector('[data-dashboard-kpi-header]')).toHaveClass(
        'h-10',
        'border-b',
        'border-border-subtle',
        'px-4',
      );
      expect(card.querySelector('[data-dashboard-kpi-body]')).toHaveClass(
        'h-[116px]',
        'p-4',
      );
    });
    expect(container.querySelector('[data-dashboard-grid]')).toHaveClass(
      'gap-4',
    );
    const cards = container.querySelectorAll('[data-dashboard-card]');
    expect(cards).toHaveLength(5);
    cards.forEach(card => {
      expect(card).toHaveClass('border', 'border-border-subtle', 'bg-surface');
      expect(card.querySelector('header')).toHaveClass('px-6', 'py-4');
    });
  });

  it('explains each important KPI with an accessible toggletip', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<HomePage />);

    await screen.findByText('12,486');

    const informationButtons = [
      'Sessions information',
      'Success Rate information',
      'Average Latency information',
      'Failed Sessions information',
    ].map(name => screen.getByRole('button', { name }));

    expect(informationButtons).toHaveLength(4);

    await user.click(informationButtons[0]);
    expect(
      await screen.findByText(
        'Total conversation sessions handled during the selected time range.',
      ),
    ).toBeVisible();
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
    expect(screen.queryByText('Project performance')).not.toBeInTheDocument();
    expect(
      screen.getByText(
        'Monitor workspace performance, conversation activity, assistant health, and component usage across the selected time range.',
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Conversation Activity' }),
    ).toHaveClass(
      'text-[1rem]',
      'font-[600]',
      'leading-[1.5rem]',
      'tracking-[0]',
    );
    expect(screen.getByText('Sessions and failures over time')).toHaveClass(
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
    document.querySelectorAll('[data-dashboard-card]').forEach(card => {
      expect(
        within(card as HTMLElement).queryByText(
          /Last (24 hours|7 days|30 days)/i,
        ),
      ).not.toBeInTheDocument();
    });
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
    expect(
      screen.getByRole('heading', { name: 'Workspace Overview', level: 1 }),
    ).toBeInTheDocument();
    expect(screen.getByRole('combobox', { name: 'Time range' })).toHaveValue(
      '30d',
    );
    expect(screen.queryByText('Top Assistants')).not.toBeInTheDocument();
    expect(
      container.querySelectorAll('[data-dashboard-loading-card]'),
    ).toHaveLength(8);
    const loadingKpis = container.querySelectorAll(
      '[data-dashboard-loading-kpi]',
    );
    expect(loadingKpis).toHaveLength(4);
    loadingKpis.forEach(card => {
      expect(card.querySelector('[data-dashboard-kpi-header]')).toHaveClass(
        'h-10',
        'border-b',
        'border-border-subtle',
        'px-4',
      );
      expect(card.querySelector('[data-dashboard-kpi-body]')).toHaveClass(
        'h-[116px]',
        'p-4',
      );
    });
  });

  it('keeps the selected time range visible and requests that duration', async () => {
    const user = userEvent.setup({ writeToClipboard: false });

    render(<HomePage />);
    await screen.findByText('12,486');

    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Time range' }),
      '7d',
    );

    await waitFor(() =>
      expect(mockGetProjectDashboardRange).toHaveBeenCalledTimes(2),
    );
    const request = mockGetProjectDashboardRange.mock.calls[1][0];
    expect(request.toDate.getSeconds() - request.fromDate.getSeconds()).toBe(
      7 * 24 * 60 * 60,
    );
    expect(screen.getByRole('combobox', { name: 'Time range' })).toHaveValue(
      '7d',
    );
    document.querySelectorAll('[data-dashboard-card]').forEach(card => {
      expect(
        within(card as HTMLElement).queryByText(/Last 7 days/i),
      ).not.toBeInTheDocument();
    });
  });

  it('moves the time range into the compact sticky header after the title scrolls away', async () => {
    const { container } = render(<HomePage />);
    await screen.findByText('12,486');

    act(() => {
      pageHeaderIntersectionCallback(
        [{ isIntersecting: false } as IntersectionObserverEntry],
        {} as IntersectionObserver,
      );
    });

    const stickyHeader = container.querySelector('[data-page-header-sticky]');
    expect(stickyHeader).toHaveAttribute('data-stuck', 'true');
    expect(stickyHeader).toHaveClass('sticky', 'top-0');
    expect(
      stickyHeader?.querySelector('[data-page-header-sticky-content]'),
    ).toHaveClass(
      'h-10',
      'duration-[240ms]',
      'ease-[cubic-bezier(0.2,0,0.38,0.9)]',
    );
    const stickyTimeRange = within(stickyHeader as HTMLElement).getByRole(
      'combobox',
      { name: 'Time range' },
    );
    expect(stickyTimeRange).toHaveValue('30d');
    expect(stickyTimeRange.closest('.cds--select')).toHaveClass(
      'cds--select--md',
      'cds--layout--size-md',
    );
    expect(
      screen.getAllByRole('combobox', { name: 'Time range' }),
    ).toHaveLength(1);
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
      screen.getByRole('heading', { name: 'Workspace Overview', level: 1 }),
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

    expect(await screen.findByText('Component Usage')).toBeInTheDocument();
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
      screen.getByText('Component Usage is unavailable'),
    ).toBeInTheDocument();
    expect(screen.getByText('Top Assistants')).toBeInTheDocument();
  });

  it('renders a setup checklist when the dashboard has no activity', async () => {
    mockGetProjectDashboardRange.mockResolvedValue(
      dashboardResponse({
        analyticsState: DashboardSectionState.DASHBOARD_SECTION_STATE_EMPTY,
        usageState: DashboardSectionState.DASHBOARD_SECTION_STATE_EMPTY,
      }),
    );

    render(<HomePage />);

    expect(
      await screen.findByRole('heading', {
        name: 'Build your first dashboard insights',
        level: 2,
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Workspace Overview', level: 1 }),
    ).toBeInTheDocument();
    const checklist = screen.getByLabelText('Dashboard setup checklist');
    expect(checklist).toHaveClass('fixed', 'bottom-4', 'right-4');
    expect(
      within(checklist).getByText('Set up your dashboard'),
    ).toBeInTheDocument();
    expect(within(checklist).getByText('25% complete')).toBeInTheDocument();
    expect(within(checklist).getByText('Workspace created')).toBeInTheDocument();
    expect(
      within(checklist).getByRole('button', { name: 'Create an assistant' }),
    ).toBeInTheDocument();
    expect(
      within(checklist).getByRole('button', {
        name: 'Configure and test your assistant',
      }),
    ).toBeInTheDocument();
    expect(
      within(checklist).getByRole('button', {
        name: 'Return here to review performance',
      }),
    ).toBeInTheDocument();
    expect(
      within(checklist).getByRole('button', { name: 'View all assistants' }),
    ).toBeInTheDocument();
    const checklistToggle = within(checklist).getByRole('button', {
      name: 'Collapse or expand dashboard setup',
    });
    expect(checklistToggle).toHaveAttribute(
      'aria-label',
      'Toggle dashboard setup checklist',
    );
    expect(checklistToggle).toHaveAttribute('aria-expanded', 'true');
    await userEvent.click(checklistToggle);
    expect(checklistToggle).toHaveAttribute('aria-expanded', 'false');
    expect(
      screen.getByText(
        'Complete these steps to generate workspace performance data. Metrics will appear after an assistant completes its first test conversation.',
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: 'Open assistants' }),
    ).toHaveAttribute('href', '/deployment/assistant');
    expect(
      screen.getByRole('link', { name: 'Create assistant' }),
    ).toHaveAttribute('href', '/deployment/assistant/create-assistant');
    expect(
      screen.queryByText('Your dashboard is not ready yet'),
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
      name: 'Build your first dashboard insights',
    });
    mockCurrentCredential = {
      ...mockCurrentCredential,
      projectId: 'project-2',
    };
    rerender(<HomePage />);

    expect(await screen.findByText('212')).toBeInTheDocument();
    expect(
      screen.queryByRole('heading', {
        name: 'Build your first dashboard insights',
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
    expect(screen.getByText('Component Usage')).toBeInTheDocument();
    expect(
      screen.queryByRole('heading', {
        name: 'Build your first dashboard insights',
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
        name: 'Conversation Activity showing 12,486 sessions and 649 failures for the selected time range',
      }),
    ).toBeInTheDocument();
    expect(screen.getByTestId('carbon-line-chart')).toBeInTheDocument();
    expect(mockLineChart).toHaveBeenCalledWith(
      expect.objectContaining({
        data: expect.arrayContaining([
          { group: 'Sessions', key: expect.any(String), value: 428 },
          { group: 'Failed', key: expect.any(String), value: 18 },
        ]),
        options: expect.objectContaining({ theme: 'white' }),
      }),
    );
    expect(mockLineChart.mock.calls.at(-1)?.[0].options).not.toHaveProperty(
      'color',
    );
    expect(
      screen.getByRole('table', { name: 'Recent Failures' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('table', { name: 'Top Assistants' }),
    ).toBeInTheDocument();
  });
});
