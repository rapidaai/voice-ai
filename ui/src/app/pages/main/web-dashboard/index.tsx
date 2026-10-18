import {
  ArrowRight,
  CheckmarkFilled,
  Renew,
  WarningAltFilled,
} from '@carbon/icons-react';
import {
  Button,
  DataTable,
  InlineNotification,
  SkeletonPlaceholder,
  SkeletonText,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableHeader,
  TableRow,
  Tag,
  Tile,
} from '@carbon/react';
import {
  DashboardAssistantHealth,
  DashboardFailureCode,
  DashboardSectionState,
  DashboardUsageComponent,
} from '@rapidaai/react';
import type {
  DashboardRecentFailure,
  DashboardUsageSection,
  ProjectDashboard,
  ProjectDashboardAnalytics,
} from '@rapidaai/react';
import { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb';
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import {
  Area,
  AreaChart,
  CartesianGrid,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { EmptyState } from '@/app/components/ui/feedback';
import { Dropdown } from '@/app/components/ui/primitives';
import { getProjectDashboardRange } from '@/clients/dashboard.client';
import { useCurrentCredential } from '@/hooks/use-credential';

type DashboardRange = '30d' | '7d' | '24h';

type DashboardMetric = {
  label: string;
  value: string;
  unit?: string;
  comparison: string;
  comparisonTone?: 'danger';
};

type DashboardBreakdown = {
  label: string;
  value: string;
  width: number;
};

type DashboardRangeOption = {
  id: DashboardRange;
  text: string;
};

type AutoRefreshOption = {
  id: string;
  text: string;
};

const dashboardRangeOptions: DashboardRangeOption[] = [
  { id: '30d', text: 'Last 30 days' },
  { id: '7d', text: 'Last 7 days' },
  { id: '24h', text: 'Last 24 hours' },
];

const autoRefreshOptions: AutoRefreshOption[] = [
  { id: '0', text: 'Off' },
  { id: '5', text: 'Every 5 min' },
  { id: '10', text: 'Every 10 min' },
  { id: '30', text: 'Every 30 min' },
];

const assistantTableHeaders = [
  { key: 'assistant', header: 'Assistant' },
  { key: 'conversations', header: 'Conversations' },
  { key: 'successRate', header: 'Success rate' },
  { key: 'latency', header: 'Average latency' },
  { key: 'health', header: 'Health' },
  { key: 'lastActive', header: 'Last active' },
];

const failureTableHeaders = [
  { key: 'assistant', header: 'Assistant' },
  { key: 'reason', header: 'Reason' },
  { key: 'details', header: 'Details' },
  { key: 'occurredAt', header: 'Occurred at' },
];

const numberFormatter = new Intl.NumberFormat();
const dateFormatter = new Intl.DateTimeFormat(undefined, {
  month: 'short',
  day: 'numeric',
  hour: 'numeric',
  minute: '2-digit',
});

function toTimestamp(date: Date) {
  const timestamp = new Timestamp();
  timestamp.setSeconds(Math.floor(date.getTime() / 1000));
  timestamp.setNanos((date.getTime() % 1000) * 1_000_000);
  return timestamp;
}

export function getDashboardStartDate(range: DashboardRange, endDate: Date) {
  const durationMs = {
    '24h': 24 * 60 * 60 * 1000,
    '7d': 7 * 24 * 60 * 60 * 1000,
    '30d': 30 * 24 * 60 * 60 * 1000,
  }[range];
  return new Date(endDate.getTime() - durationMs);
}

function timestampDate(timestamp?: Timestamp) {
  return timestamp ? timestamp.toDate() : null;
}

function signedValue(value: number, suffix: string) {
  const sign = value > 0 ? '+' : '';
  return `${sign}${value.toFixed(1)}${suffix}`;
}

export function formatDashboardDuration(durationMs: number) {
  const totalSeconds = Math.round(durationMs / 1000);
  if (totalSeconds < 60) return `${totalSeconds} sec`;

  const totalMinutes = Math.round(durationMs / 60_000);
  if (totalMinutes < 60) return `${totalMinutes} min`;

  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  return minutes ? `${hours} hr ${minutes} min` : `${hours} hr`;
}

function formatDate(timestamp?: Timestamp) {
  const date = timestampDate(timestamp);
  return date ? dateFormatter.format(date) : 'Not available';
}

export function dashboardActivityLabel(
  timestamp: Timestamp | undefined,
  range: DashboardRange,
) {
  const date = timestampDate(timestamp);
  if (!date) return '';
  if (range === '24h')
    return new Intl.DateTimeFormat(undefined, {
      hour: 'numeric',
      minute: '2-digit',
    }).format(date);
  if (range === '7d')
    return new Intl.DateTimeFormat(undefined, {
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
    }).format(date);
  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
  }).format(date);
}

function componentLabel(component: number) {
  const labels: Record<number, string> = {
    [DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_STT]: 'Speech to text',
    [DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_TTS]: 'Text to speech',
    [DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_LLM]: 'Language model',
    [DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_VAD]:
      'Voice activity detection',
    [DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_EOS]: 'End of speech',
    [DashboardUsageComponent.DASHBOARD_USAGE_COMPONENT_DENOISING]:
      'Noise reduction',
  };
  return labels[component] || 'Other';
}

function failureLabel(code: number) {
  const labels: Record<number, string> = {
    [DashboardFailureCode.DASHBOARD_FAILURE_CODE_AUTHENTICATION]:
      'Authentication',
    [DashboardFailureCode.DASHBOARD_FAILURE_CODE_CONFIGURATION]:
      'Configuration',
    [DashboardFailureCode.DASHBOARD_FAILURE_CODE_PROVIDER]: 'Provider',
    [DashboardFailureCode.DASHBOARD_FAILURE_CODE_NETWORK]: 'Network',
    [DashboardFailureCode.DASHBOARD_FAILURE_CODE_CANCELLED]: 'Cancelled',
    [DashboardFailureCode.DASHBOARD_FAILURE_CODE_NO_ANSWER]: 'No answer',
    [DashboardFailureCode.DASHBOARD_FAILURE_CODE_REJECTED]: 'Rejected',
    [DashboardFailureCode.DASHBOARD_FAILURE_CODE_UNKNOWN]: 'Unknown',
  };
  return labels[code] || 'Unknown';
}

function assistantTagType(health: number) {
  if (
    health ===
    DashboardAssistantHealth.DASHBOARD_ASSISTANT_HEALTH_NEEDS_ATTENTION
  )
    return 'red' as const;
  if (health === DashboardAssistantHealth.DASHBOARD_ASSISTANT_HEALTH_WATCH)
    return 'warm-gray' as const;
  if (health === DashboardAssistantHealth.DASHBOARD_ASSISTANT_HEALTH_HEALTHY)
    return 'green' as const;
  return 'gray' as const;
}

function assistantHealthLabel(health: number) {
  if (
    health ===
    DashboardAssistantHealth.DASHBOARD_ASSISTANT_HEALTH_NEEDS_ATTENTION
  )
    return 'Needs attention';
  if (health === DashboardAssistantHealth.DASHBOARD_ASSISTANT_HEALTH_WATCH)
    return 'Watch';
  if (health === DashboardAssistantHealth.DASHBOARD_ASSISTANT_HEALTH_HEALTHY)
    return 'Healthy';
  return 'No data';
}

function PanelHeader({
  title,
  subtitle,
  action,
}: {
  title: string;
  subtitle: string;
  action?: ReactNode;
}) {
  return (
    <header className="flex min-h-[76px] items-start justify-between gap-4 border-b border-gray-200 px-6 py-4 dark:border-gray-800">
      <div>
        <h2 className="text-lg font-semibold text-[var(--cds-text-primary)]">
          {title}
        </h2>
        <p className="mt-1 text-xs text-[var(--cds-text-secondary)]">
          {subtitle}
        </p>
      </div>
      {action}
    </header>
  );
}

function MetricCard({ metric }: { metric: DashboardMetric }) {
  return (
    <article
      data-dashboard-kpi
      className="flex h-[156px] flex-col border border-gray-200 bg-white dark:border-gray-800 dark:bg-[#262626]"
    >
      <div className="flex h-10 shrink-0 items-center border-b border-gray-200 px-4 dark:border-gray-800">
        <h2 className="truncate text-sm font-semibold text-[var(--cds-text-primary)]">
          {metric.label}
        </h2>
      </div>
      <div className="flex min-h-0 flex-1 flex-col justify-between p-4">
        <p className="text-4xl font-light leading-none tracking-tight tabular-nums text-[var(--cds-text-primary)]">
          {metric.value}
          {metric.unit && (
            <span className="ml-1 text-base font-normal tracking-normal text-[var(--cds-text-secondary)]">
              {metric.unit}
            </span>
          )}
        </p>
        <p
          className={`text-xs font-semibold ${
            metric.comparisonTone === 'danger'
              ? 'text-[var(--cds-support-error)]'
              : 'text-[var(--cds-text-secondary)]'
          }`}
        >
          {metric.comparison}
        </p>
      </div>
    </article>
  );
}

function ActivityPanel({
  analytics,
  range,
  rangeLabel,
}: {
  analytics: ProjectDashboardAnalytics;
  range: DashboardRange;
  rangeLabel: string;
}) {
  const summary = analytics.getSummary();
  const sessions = summary?.getSessions()?.getValue() || 0;
  const failures = summary?.getFailedsessions()?.getValue() || 0;
  const activity = analytics.getActivityList().map(bucket => ({
    label: dashboardActivityLabel(bucket.getFromdate(), range),
    sessions: bucket.getSessions(),
    failed: bucket.getFailedsessions(),
  }));

  return (
    <section
      data-dashboard-panel
      className="h-full border border-gray-200 bg-white dark:border-gray-800 dark:bg-[#262626]"
    >
      <PanelHeader
        title="Conversation activity"
        subtitle="Sessions and failures during the selected period"
        action={
          <div className="flex shrink-0 flex-col gap-1 text-xs text-[var(--cds-text-secondary)] sm:flex-row sm:gap-4">
            <span className="flex items-center gap-1.5">
              <span className="h-2.5 w-2.5 bg-[var(--cds-interactive)]" />
              Sessions
            </span>
            <span className="flex items-center gap-1.5">
              <span className="h-2.5 w-2.5 bg-[var(--cds-support-error)]" />
              Failed
            </span>
          </div>
        }
      />
      {activity.length ? (
        <div
          className="h-[320px] min-w-0 px-6 pb-6 pt-6"
          role="img"
          aria-label={`Conversation activity showing ${numberFormatter.format(
            sessions,
          )} sessions and ${numberFormatter.format(
            failures,
          )} failures over ${rangeLabel}`}
        >
          <ResponsiveContainer width="100%" height="100%" minWidth={0}>
            <AreaChart
              data={activity}
              margin={{ top: 16, right: 24, left: 0, bottom: 0 }}
            >
              <defs>
                <linearGradient
                  id="projectActivityGradient"
                  x1="0"
                  y1="0"
                  x2="0"
                  y2="1"
                >
                  <stop
                    offset="0%"
                    stopColor="var(--cds-interactive)"
                    stopOpacity={0.24}
                  />
                  <stop
                    offset="100%"
                    stopColor="var(--cds-interactive)"
                    stopOpacity={0.02}
                  />
                </linearGradient>
              </defs>
              <CartesianGrid
                stroke="var(--cds-border-subtle-01)"
                vertical={false}
              />
              <XAxis
                dataKey="label"
                tick={{ fontSize: 11, fill: 'var(--cds-text-secondary)' }}
                tickLine={false}
                axisLine={false}
              />
              <YAxis
                width={42}
                tick={{ fontSize: 11, fill: 'var(--cds-text-secondary)' }}
                tickLine={false}
                axisLine={false}
              />
              <Tooltip
                contentStyle={{
                  border: '1px solid var(--cds-border-subtle-01)',
                  background: 'var(--cds-layer-01)',
                  color: 'var(--cds-text-primary)',
                  fontSize: 12,
                }}
              />
              <Area
                type="monotone"
                dataKey="sessions"
                name="Sessions"
                stroke="var(--cds-interactive)"
                fill="url(#projectActivityGradient)"
                strokeWidth={2}
                activeDot={{ r: 4 }}
                isAnimationActive={false}
              />
              <Line
                type="monotone"
                dataKey="failed"
                name="Failed"
                stroke="var(--cds-support-error)"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4 }}
                isAnimationActive={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      ) : (
        <EmptyContent
          title="No activity yet"
          description="Conversation activity will appear after this project handles sessions."
        />
      )}
    </section>
  );
}

function HealthPanel({ analytics }: { analytics: ProjectDashboardAnalytics }) {
  const health = analytics.getHealth();
  const projectState = analytics.getProjectstate();
  const recentFailures = analytics.getRecentfailuresList().length;
  const assistantCount = projectState?.getAssistantcount() || 0;
  const healthyAssistants = health?.getHealthyassistants() || 0;
  const threshold = health?.getSlowsessionthresholdms() || 0;

  return (
    <section
      data-dashboard-panel
      className="h-full border border-gray-200 bg-white dark:border-gray-800 dark:bg-[#262626]"
    >
      <PanelHeader
        title="Health summary"
        subtitle="Signals that may need attention"
      />
      <ul className="divide-y divide-gray-200 px-6 dark:divide-gray-800">
        <li className="flex min-h-[106px] items-start gap-4 py-5">
          <WarningAltFilled
            size={24}
            aria-hidden="true"
            className="mt-0.5 shrink-0 text-[var(--cds-support-error)]"
          />
          <div className="min-w-0 flex-1">
            <p className="text-sm font-semibold">Failed sessions</p>
            <p className="mt-2 text-xs text-[var(--cds-text-secondary)]">
              {numberFormatter.format(recentFailures)} recent{' '}
              {recentFailures === 1 ? 'failure' : 'failures'}
            </p>
          </div>
          <span className="text-2xl font-light tabular-nums">
            {numberFormatter.format(health?.getFailedsessions() || 0)}
          </span>
        </li>
        <li className="flex min-h-[106px] items-start gap-4 py-5">
          <WarningAltFilled
            size={24}
            aria-hidden="true"
            className="mt-0.5 shrink-0 text-[var(--cds-support-warning)]"
          />
          <div className="min-w-0 flex-1">
            <p className="text-sm font-semibold">Slow sessions</p>
            <p className="mt-2 text-xs text-[var(--cds-text-secondary)]">
              Response latency above {numberFormatter.format(threshold)} ms
            </p>
          </div>
          <span className="text-2xl font-light tabular-nums">
            {numberFormatter.format(health?.getSlowsessions() || 0)}
          </span>
        </li>
        <li className="flex min-h-[106px] items-start gap-4 py-5">
          <CheckmarkFilled
            size={24}
            aria-hidden="true"
            className="mt-0.5 shrink-0 text-[var(--cds-support-success)]"
          />
          <div className="min-w-0 flex-1">
            <p className="text-sm font-semibold">Healthy assistants</p>
            <p className="mt-2 text-xs text-[var(--cds-text-secondary)]">
              Based on success rate and latency
            </p>
          </div>
          <span className="text-2xl font-light tabular-nums">
            {healthyAssistants} of {assistantCount}
          </span>
        </li>
      </ul>
    </section>
  );
}

function EmptyContent({
  title,
  description,
  action,
}: {
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex min-h-[220px] flex-col items-center justify-center px-6 py-12 text-center">
      <WarningAltFilled
        size={40}
        aria-hidden="true"
        className="text-[var(--cds-text-disabled)]"
      />
      <h3 className="mt-4 text-base font-semibold text-[var(--cds-text-primary)]">
        {title}
      </h3>
      <p className="mt-1 max-w-sm text-sm text-[var(--cds-text-secondary)]">
        {description}
      </p>
      {action}
    </div>
  );
}

function StatePanel({
  title,
  subtitle,
  state,
}: {
  title: string;
  subtitle: string;
  state: number;
}) {
  const isEmpty = state === DashboardSectionState.DASHBOARD_SECTION_STATE_EMPTY;
  const unavailableTitle =
    title === 'Analytics'
      ? 'Analytics are unavailable'
      : `${title} is unavailable`;
  return (
    <Tile className="h-full border! border-gray-200! bg-white! p-0! dark:border-gray-800! dark:bg-[#262626]!">
      <PanelHeader title={title} subtitle={subtitle} />
      <EmptyContent
        title={isEmpty ? `No ${title.toLowerCase()} yet` : unavailableTitle}
        description={
          isEmpty
            ? 'Data will appear here after the project starts handling sessions.'
            : 'This section could not be loaded. Other available dashboard data is still shown.'
        }
      />
    </Tile>
  );
}

function BreakdownPanel({
  title,
  subtitle,
  values,
}: {
  title: string;
  subtitle: string;
  values: DashboardBreakdown[];
}) {
  return (
    <Tile className="h-full border! border-gray-200! bg-white! p-0! dark:border-gray-800! dark:bg-[#262626]!">
      <PanelHeader title={title} subtitle={subtitle} />
      {values.length ? (
        <div className="grid gap-6 p-6 sm:grid-cols-2">
          {values.map(value => (
            <div key={value.label}>
              <div className="mb-2 flex items-center justify-between gap-4 text-sm">
                <span>{value.label}</span>
                <span className="font-semibold tabular-nums">
                  {value.value}
                </span>
              </div>
              <div className="h-1 bg-gray-200 dark:bg-gray-800">
                <div
                  data-dashboard-bar
                  className="h-full bg-[var(--cds-interactive)]"
                  style={{ width: `${value.width}%` }}
                />
              </div>
            </div>
          ))}
        </div>
      ) : (
        <EmptyContent
          title="No component usage yet"
          description="Usage will appear after assistants process voice sessions."
        />
      )}
    </Tile>
  );
}

function RecentFailuresTable({
  failures,
}: {
  failures: DashboardRecentFailure[];
}) {
  const rows = failures.map((failure, index) => ({
    id: failure.getConversationid() || `failure-${index}`,
    assistant: failure.getAssistantname() || 'Unknown assistant',
    reason: failureLabel(failure.getFailurecode()),
    details: `${failure.getChannel() || 'Unknown'} · ${formatDashboardDuration(
      failure.getDurationms(),
    )}`,
    occurredAt: formatDate(failure.getOccurredat()),
  }));
  const failureRoutes = new Map(
    failures.map(failure => [
      failure.getConversationid(),
      failure.getAssistantid() && failure.getConversationid()
        ? `/deployment/assistant/${encodeURIComponent(
            failure.getAssistantid(),
          )}/sessions/${encodeURIComponent(failure.getConversationid())}`
        : '',
    ]),
  );

  return (
    <Tile className="h-full border! border-gray-200! bg-white! p-0! dark:border-gray-800! dark:bg-[#262626]!">
      <PanelHeader
        title="Recent failures"
        subtitle="Latest sessions that need investigation"
      />
      {rows.length ? (
        <DataTable rows={rows} headers={failureTableHeaders}>
          {({ rows, headers, getHeaderProps, getRowProps, getTableProps }) => (
            <TableContainer>
              <div className="overflow-x-auto">
                <Table
                  {...getTableProps()}
                  aria-label="Recent failures"
                  className="min-w-[440px]"
                  size="md"
                  tabIndex={0}
                >
                  <caption className="sr-only">
                    Recent failed sessions for the selected period
                  </caption>
                  <TableHead>
                    <TableRow>
                      {headers.map(header => (
                        <TableHeader
                          {...getHeaderProps({ header })}
                          key={header.key}
                        >
                          {header.header}
                        </TableHeader>
                      ))}
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {rows.map(row => (
                      <TableRow {...getRowProps({ row })} key={row.id}>
                        {row.cells.map(cell => (
                          <TableCell key={cell.id}>
                            {cell.info.header === 'assistant' &&
                            failureRoutes.get(row.id) ? (
                              <a
                                href={failureRoutes.get(row.id)}
                                className="font-semibold text-[var(--cds-link-primary)] hover:underline focus:underline"
                              >
                                {cell.value}
                              </a>
                            ) : cell.info.header === 'reason' ? (
                              <Tag size="sm" type="red">
                                {cell.value}
                              </Tag>
                            ) : (
                              cell.value
                            )}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </TableContainer>
          )}
        </DataTable>
      ) : (
        <EmptyContent
          title="No recent failures"
          description="No failed sessions were returned for this period."
        />
      )}
    </Tile>
  );
}

function AssistantsTable({
  analytics,
}: {
  analytics: ProjectDashboardAnalytics;
}) {
  const rows = analytics.getAssistantsList().map(assistant => ({
    id: assistant.getAssistantid(),
    assistant: assistant.getName() || 'Unnamed assistant',
    conversations: numberFormatter.format(assistant.getSessions()),
    successRate: assistant.hasSuccessratepercent()
      ? `${assistant.getSuccessratepercent().toFixed(1)}%`
      : 'Not available',
    latency: assistant.hasResponselatencyms()
      ? `${numberFormatter.format(assistant.getResponselatencyms())} ms`
      : 'Not available',
    health: assistantHealthLabel(assistant.getHealth()),
    lastActive: formatDate(assistant.getLastactivityat()),
  }));
  const successTones = new Map(
    analytics
      .getAssistantsList()
      .map(assistant => [
        assistant.getAssistantid(),
        assistantTagType(assistant.getHealth()),
      ]),
  );

  return (
    <Tile className="border! border-gray-200! bg-white! p-0! dark:border-gray-800! dark:bg-[#262626]!">
      <PanelHeader
        title="Top assistants"
        subtitle="Assistants ranked by conversation volume"
        action={
          <Button
            kind="ghost"
            size="sm"
            renderIcon={ArrowRight}
            href="/deployment/assistant"
          >
            View all assistants
          </Button>
        }
      />
      {rows.length ? (
        <DataTable rows={rows} headers={assistantTableHeaders}>
          {({ rows, headers, getHeaderProps, getRowProps, getTableProps }) => (
            <TableContainer>
              <div className="overflow-x-auto">
                <Table
                  {...getTableProps()}
                  aria-label="Top assistants"
                  className="min-w-[760px]"
                  size="md"
                  tabIndex={0}
                >
                  <caption className="sr-only">
                    Top assistants ranked by conversation volume for the
                    selected period
                  </caption>
                  <TableHead>
                    <TableRow>
                      {headers.map(header => (
                        <TableHeader
                          {...getHeaderProps({ header })}
                          key={header.key}
                        >
                          {header.header}
                        </TableHeader>
                      ))}
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {rows.map(row => (
                      <TableRow {...getRowProps({ row })} key={row.id}>
                        {row.cells.map(cell => (
                          <TableCell key={cell.id}>
                            {cell.info.header === 'health' ? (
                              <Tag
                                size="sm"
                                type={successTones.get(row.id) || 'gray'}
                                className={
                                  cell.value === 'Watch'
                                    ? 'bg-[var(--cds-support-warning)]! text-[var(--cds-text-primary)]!'
                                    : undefined
                                }
                              >
                                {cell.value}
                              </Tag>
                            ) : cell.info.header === 'assistant' ? (
                              <a
                                href={`/deployment/assistant/${encodeURIComponent(
                                  row.id,
                                )}`}
                                className="font-semibold text-[var(--cds-link-primary)] hover:underline focus:underline"
                              >
                                {cell.value}
                              </a>
                            ) : (
                              cell.value
                            )}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </TableContainer>
          )}
        </DataTable>
      ) : (
        <EmptyContent
          title="No assistant activity yet"
          description="Assistant rankings will appear after sessions are recorded."
        />
      )}
    </Tile>
  );
}

function DashboardLoading() {
  return (
    <div
      className="col-span-full"
      role="status"
      aria-label="Loading project dashboard"
    >
      <span className="sr-only">Loading project dashboard</span>
      <div className="mb-4 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        {[0, 1, 2, 3].map(item => (
          <div
            data-dashboard-loading-card
            className="h-[156px] border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-[#262626]"
            key={item}
            aria-hidden="true"
          >
            <SkeletonText width="45%" />
            <SkeletonText className="mt-6" width="70%" heading />
            <SkeletonText className="mt-6" width="80%" />
          </div>
        ))}
      </div>
      <div className="mb-4 grid gap-4 lg:grid-cols-16">
        <div
          data-dashboard-loading-card
          className="h-[396px] border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-[#262626] lg:col-span-11"
        >
          <SkeletonText width="30%" heading />
          <SkeletonText width="45%" />
          <SkeletonPlaceholder className="mt-8 h-[260px]! w-full!" />
        </div>
        <div
          data-dashboard-loading-card
          className="h-[396px] border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-[#262626] lg:col-span-5"
        >
          <SkeletonText width="55%" heading />
          <SkeletonText width="75%" />
          <SkeletonText className="mt-10" width="100%" />
          <SkeletonText className="mt-12" width="100%" />
          <SkeletonText className="mt-12" width="100%" />
        </div>
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        {[0, 1].map(item => (
          <div
            data-dashboard-loading-card
            className="h-[260px] border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-[#262626]"
            key={item}
            aria-hidden="true"
          >
            <SkeletonText width="35%" heading />
            <SkeletonText width="55%" />
            <SkeletonPlaceholder className="mt-8 h-[140px]! w-full!" />
          </div>
        ))}
      </div>
    </div>
  );
}

function buildMetrics(analytics: ProjectDashboardAnalytics) {
  const summary = analytics.getSummary();
  const sessions = summary?.getSessions();
  const successRate = summary?.getSuccessrate();
  const latency = summary?.getResponselatency();
  const failed = summary?.getFailedsessions();

  const sessionComparison = sessions?.hasChangepercent()
    ? `${signedValue(sessions.getChangepercent(), '%')} from previous period`
    : 'No comparison available';
  const successComparison = successRate?.hasChangepoints()
    ? `${signedValue(successRate.getChangepoints(), ' points')} from previous period`
    : 'No comparison available';
  const latencyChange = latency?.hasChangems() ? latency.getChangems() : null;
  const latencyComparison =
    latencyChange === null
      ? 'No comparison available'
      : latencyChange < 0
        ? `${numberFormatter.format(Math.abs(latencyChange))} ms faster than previous period`
        : latencyChange > 0
          ? `${numberFormatter.format(latencyChange)} ms slower than previous period`
          : 'No change from previous period';
  const failureComparison = failed?.hasSharepercent()
    ? `${failed.getSharepercent().toFixed(1)}% of project sessions`
    : 'Share unavailable';

  return [
    {
      label: 'Sessions',
      value: numberFormatter.format(sessions?.getValue() || 0),
      comparison: sessionComparison,
    },
    {
      label: 'Success rate',
      value: successRate?.hasValuepercent()
        ? `${successRate.getValuepercent().toFixed(1)}%`
        : 'Not available',
      comparison: successComparison,
    },
    {
      label: 'Avg latency',
      value: latency?.hasValuems()
        ? numberFormatter.format(latency.getValuems())
        : 'Not available',
      unit: latency?.hasValuems() ? 'ms' : undefined,
      comparison: latencyComparison,
    },
    {
      label: 'Failed sessions',
      value: numberFormatter.format(failed?.getValue() || 0),
      comparison: failureComparison,
      comparisonTone: 'danger' as const,
    },
  ];
}

function buildUsage(usage: DashboardUsageSection) {
  const components = usage.getComponentsList();
  const maxDuration = Math.max(
    ...components.map(component => component.getDurationms()),
    1,
  );
  return components.map(component => ({
    label: componentLabel(component.getComponent()),
    value: formatDashboardDuration(component.getDurationms()),
    width:
      component.getDurationms() === 0
        ? 0
        : Math.max(4, (component.getDurationms() / maxDuration) * 100),
  }));
}

export const HomePage = () => {
  const [selectedRange, setSelectedRange] = useState<DashboardRange>('30d');
  const [selectedAutoRefresh, setSelectedAutoRefresh] =
    useState<AutoRefreshOption>(autoRefreshOptions[0]);
  const [dashboardState, setDashboardState] = useState<{
    contextKey: string;
    data: ProjectDashboard;
  } | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [refreshError, setRefreshError] = useState(false);
  const requestId = useRef(0);
  const dashboardContextKey = useRef('');
  const dashboardRef = useRef<{
    contextKey: string;
    data: ProjectDashboard;
  } | null>(null);
  const { authId, projectId, token } = useCurrentCredential();
  const currentContextKey = `${projectId}:${selectedRange}`;
  const isContextPending = dashboardContextKey.current !== currentContextKey;
  const dashboard =
    dashboardState?.contextKey === currentContextKey
      ? dashboardState.data
      : null;

  const fetchDashboard = useCallback(() => {
    requestId.current += 1;
    const currentRequestId = requestId.current;
    const contextKey = `${projectId}:${selectedRange}`;
    const isNewContext = dashboardContextKey.current !== contextKey;

    if (isNewContext) {
      dashboardContextKey.current = contextKey;
      dashboardRef.current = null;
      setDashboardState(null);
    }
    if (isNewContext || !dashboardRef.current) setLoading(true);
    setLoadError(false);
    setRefreshError(false);

    if (!authId || !projectId || !token) {
      setLoading(false);
      setLoadError(true);
      return;
    }

    const toDate = new Date();
    getProjectDashboardRange({
      fromDate: toTimestamp(getDashboardStartDate(selectedRange, toDate)),
      toDate: toTimestamp(toDate),
      assistantLimit: 3,
      recentFailureLimit: 5,
      auth: { projectId, token, userId: authId },
    })
      .then(response => {
        if (currentRequestId !== requestId.current) return;
        const data = response.getData();
        if (!response.getSuccess() || !data)
          throw new Error('Dashboard response did not include data');
        const nextDashboard = { contextKey, data };
        dashboardRef.current = nextDashboard;
        setDashboardState(nextDashboard);
      })
      .catch(() => {
        if (currentRequestId !== requestId.current) return;
        if (dashboardRef.current?.contextKey === contextKey)
          setRefreshError(true);
        else setLoadError(true);
      })
      .finally(() => {
        if (currentRequestId === requestId.current) setLoading(false);
      });
  }, [authId, projectId, selectedRange, token]);

  useEffect(() => {
    fetchDashboard();
  }, [fetchDashboard]);

  useEffect(
    () => () => {
      requestId.current += 1;
    },
    [],
  );

  useEffect(() => {
    const minutes = Number(selectedAutoRefresh.id);
    if (!minutes) return undefined;
    const interval = window.setInterval(fetchDashboard, minutes * 60 * 1000);
    return () => window.clearInterval(interval);
  }, [fetchDashboard, selectedAutoRefresh]);

  const selectedRangeOption = dashboardRangeOptions.find(
    option => option.id === selectedRange,
  );
  const analyticsSection = dashboard?.getAnalytics();
  const analytics =
    analyticsSection?.getState() ===
    DashboardSectionState.DASHBOARD_SECTION_STATE_READY
      ? analyticsSection.getData()
      : undefined;
  const usage = dashboard?.getUsage();
  const isUsageReady =
    usage?.getState() === DashboardSectionState.DASHBOARD_SECTION_STATE_READY;
  const metrics = useMemo(
    () => (analytics ? buildMetrics(analytics) : []),
    [analytics],
  );
  const componentUsage = useMemo(
    () => (usage && isUsageReady ? buildUsage(usage) : []),
    [isUsageReady, usage],
  );

  if (loadError && !dashboard) {
    return (
      <main
        data-dashboard-page
        className="flex min-h-0 flex-1 overflow-auto bg-gray-100 text-[var(--cds-text-primary)] dark:bg-[#161616]"
      >
        <div data-dashboard-content className="flex min-h-full flex-1">
          <EmptyState
            icon={WarningAltFilled}
            title="Project dashboard is unavailable"
            subtitle="Check your connection and try again."
            action="Try again"
            actionIcon={Renew}
            onAction={fetchDashboard}
          />
        </div>
      </main>
    );
  }

  return (
    <main
      data-dashboard-page
      className="min-h-0 flex-1 overflow-auto bg-gray-100 text-[var(--cds-text-primary)] dark:bg-[#161616]"
    >
      <div
        data-dashboard-content
        className="mx-auto grid max-w-[1584px] grid-cols-4 gap-x-4 p-4 md:grid-cols-8 lg:grid-cols-16"
      >
        <div className="col-span-full mb-4">
          <section className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <div className="flex items-center gap-2">
                <p className="text-xs text-[var(--cds-text-secondary)]">
                  Dashboard
                </p>
                {dashboard?.getPartial() && (
                  <Tag size="sm" type="warm-gray">
                    Partial data
                  </Tag>
                )}
              </div>
              <h1 className="text-2xl font-semibold text-[var(--cds-text-primary)]">
                Project overview
              </h1>
              <p className="mt-1 text-sm text-[var(--cds-text-secondary)]">
                Monitor sessions, reliability, latency, and assistant health.
              </p>
            </div>

            <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
              <Dropdown<DashboardRangeOption>
                id="dashboard-range"
                aria-label="Date range"
                titleText=""
                hideLabel
                label="Date range"
                items={dashboardRangeOptions}
                selectedItem={selectedRangeOption}
                itemToString={item => item?.text || ''}
                onChange={({ selectedItem }) => {
                  if (selectedItem) setSelectedRange(selectedItem.id);
                }}
                size="sm"
                className="min-w-[160px]"
              />
              <Dropdown<AutoRefreshOption>
                id="dashboard-auto-refresh"
                aria-label="Auto-refresh"
                titleText=""
                hideLabel
                label="Auto-refresh"
                items={autoRefreshOptions}
                selectedItem={selectedAutoRefresh}
                itemToString={item => item?.text || ''}
                onChange={({ selectedItem }) => {
                  if (selectedItem) setSelectedAutoRefresh(selectedItem);
                }}
                size="sm"
                className="min-w-[140px]"
              />
            </div>
          </section>
        </div>

        {refreshError && !isContextPending && (
          <div className="col-span-full mb-4">
            <InlineNotification
              kind="warning"
              lowContrast
              hideCloseButton
              title="Refresh failed"
              subtitle="Showing the last successfully loaded dashboard data."
            />
          </div>
        )}

        {(loading || isContextPending) && !dashboard ? (
          <DashboardLoading />
        ) : dashboard ? (
          <>
            {analytics ? (
              <>
                <div
                  data-dashboard-kpi-strip
                  className="col-span-full mb-4 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4"
                >
                  {metrics.map(metric => (
                    <MetricCard key={metric.label} metric={metric} />
                  ))}
                </div>

                <div className="col-span-full mb-4 grid gap-4 lg:grid-cols-16">
                  <div className="min-w-0 lg:col-span-11">
                    <ActivityPanel
                      analytics={analytics}
                      range={selectedRange}
                      rangeLabel={
                        selectedRangeOption?.text || 'the selected period'
                      }
                    />
                  </div>
                  <div className="lg:col-span-5">
                    <HealthPanel analytics={analytics} />
                  </div>
                </div>
              </>
            ) : (
              <div className="col-span-full mb-4">
                <StatePanel
                  title="Analytics"
                  subtitle="Sessions, reliability, and assistant health"
                  state={
                    analyticsSection?.getState() ||
                    DashboardSectionState.DASHBOARD_SECTION_STATE_UNAVAILABLE
                  }
                />
              </div>
            )}

            <div className="col-span-full mb-4 min-w-0 xl:col-span-8">
              {usage && isUsageReady ? (
                <BreakdownPanel
                  title="Component usage"
                  subtitle="Processing time used by voice and AI components"
                  values={componentUsage}
                />
              ) : (
                <StatePanel
                  title="Component usage"
                  subtitle="Processing time used by voice and AI components"
                  state={
                    usage?.getState() ||
                    DashboardSectionState.DASHBOARD_SECTION_STATE_UNAVAILABLE
                  }
                />
              )}
            </div>

            <div className="col-span-full mb-4 min-w-0 xl:col-span-8">
              {analytics ? (
                <RecentFailuresTable
                  failures={analytics.getRecentfailuresList()}
                />
              ) : (
                <StatePanel
                  title="Recent failures"
                  subtitle="Latest sessions that need investigation"
                  state={
                    analyticsSection?.getState() ||
                    DashboardSectionState.DASHBOARD_SECTION_STATE_UNAVAILABLE
                  }
                />
              )}
            </div>

            {analytics && (
              <div className="col-span-full mb-4">
                <AssistantsTable analytics={analytics} />
              </div>
            )}
          </>
        ) : null}
      </div>
    </main>
  );
};
