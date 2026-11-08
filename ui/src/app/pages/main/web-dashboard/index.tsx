import {
  ArrowRight,
  Bot,
  CheckmarkFilled,
  Information,
  Renew,
  WarningAltFilled,
} from '@carbon/icons-react';
import {
  Button,
  DataTable,
  InlineNotification,
  Select,
  SelectItem,
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
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import {
  Alignments,
  ChartTheme,
  LegendPositions,
  LineChart,
  ScaleTypes,
  type ChartTabularData,
  type LineChartOptions,
} from '@carbon/charts-react';
import '@carbon/charts-react/styles.css';
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
import { EmptyState } from '@/app/components/ui/feedback';
import { PageHeader } from '@/app/components/layout/page-header';
import {
  PrimaryButton,
  Text,
  typographyClassNames,
} from '@/app/components/ui/primitives';
import { getProjectDashboardRange } from '@/clients/dashboard.client';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useTheme } from '@/theme/theme-provider';

type DashboardRange = '30d' | '7d' | '24h';

type DashboardMetric = {
  label: string;
  description: string;
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
    <header className="flex min-h-[76px] flex-col items-start gap-4 border-b border-border-subtle px-6 py-4 sm:flex-row sm:justify-between ">
      <div>
        <Text as="h2" variant="heading-02" className="text-foreground">
          {title}
        </Text>
        <Text as="p" variant="body-compact-01" className="mt-1 text-muted">
          {subtitle}
        </Text>
      </div>
      {action}
    </header>
  );
}

function DashboardCard({
  title,
  subtitle,
  action,
  children,
}: {
  title: string;
  subtitle: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section
      data-dashboard-card
      className="h-full min-w-0 border border-border-subtle bg-surface"
    >
      <PanelHeader title={title} subtitle={subtitle} action={action} />
      {children}
    </section>
  );
}

function DashboardNoActivity() {
  return (
    <section
      data-dashboard-empty
      aria-labelledby="dashboard-getting-started-title"
      className="flex min-h-[420px] items-center bg-layer"
    >
      <div className="mx-auto w-full max-w-[640px] px-6 py-12 md:px-12 md:py-16">
        <div className="flex h-12 w-12 items-center justify-center bg-primary text-on-primary">
          <Bot aria-hidden="true" size={24} />
        </div>

        <Text as="p" variant="label-01" className="mt-8 text-muted">
          Getting started
        </Text>
        <Text
          as="h2"
          id="dashboard-getting-started-title"
          variant="heading-05"
          className="mt-2 text-foreground"
        >
          Your dashboard is not ready yet
        </Text>
        <Text
          as="p"
          variant="body-02"
          className="mt-4 max-w-[520px] text-muted"
        >
          Create an assistant and run a test conversation. Sessions, latency,
          and health will appear here after the first session.
        </Text>

        <div className="mt-6">
          <PrimaryButton
            size="lg"
            href="/deployment/assistant/create-assistant"
            renderIcon={ArrowRight}
            className="w-full max-w-[320px] justify-between"
          >
            Create assistant
          </PrimaryButton>
        </div>
      </div>
    </section>
  );
}

function MetricCard({ metric }: { metric: DashboardMetric }) {
  return (
    <article
      data-dashboard-kpi
      className="flex h-[156px] flex-col border border-border-subtle bg-surface"
    >
      <header
        data-dashboard-kpi-header
        className="flex h-10 shrink-0 items-center justify-between border-b border-border-subtle px-4"
      >
        <Text
          as="h2"
          variant="heading-compact-01"
          className="truncate text-foreground"
        >
          {metric.label}
        </Text>
        <Toggletip align="bottom-end">
          <ToggletipButton label={`${metric.label} information`}>
            <Information aria-hidden="true" className="text-muted" size={16} />
          </ToggletipButton>
          <ToggletipContent>
            <Text as="p" variant="body-compact-01">
              {metric.description}
            </Text>
          </ToggletipContent>
        </Toggletip>
      </header>
      <div
        data-dashboard-kpi-body
        className="flex h-[116px] flex-col justify-between p-4"
      >
        <Text
          as="p"
          variant="heading-05"
          className="tabular-nums text-foreground"
        >
          {metric.value}
          {metric.unit && (
            <Text variant="body-compact-02" className="ml-1 text-muted">
              {metric.unit}
            </Text>
          )}
        </Text>
        <Text
          as="p"
          variant="label-01"
          className={
            metric.comparisonTone === 'danger'
              ? 'text-support-error'
              : 'text-muted'
          }
        >
          {metric.comparison}
        </Text>
      </div>
    </article>
  );
}

function ActivityPanel({
  analytics,
  range,
}: {
  analytics: ProjectDashboardAnalytics;
  range: DashboardRange;
}) {
  const { resolvedMode } = useTheme();
  const summary = analytics.getSummary();
  const sessions = summary?.getSessions()?.getValue() || 0;
  const failures = summary?.getFailedsessions()?.getValue() || 0;
  const activity: ChartTabularData = analytics
    .getActivityList()
    .flatMap(bucket => {
      const key = dashboardActivityLabel(bucket.getFromdate(), range);
      return [
        { group: 'Sessions', key, value: bucket.getSessions() },
        { group: 'Failed', key, value: bucket.getFailedsessions() },
      ];
    });
  const chartLabel = `Conversation Activity showing ${numberFormatter.format(
    sessions,
  )} sessions and ${numberFormatter.format(
    failures,
  )} failures for the selected time range`;
  const chartOptions: LineChartOptions = {
    accessibility: { svgAriaLabel: chartLabel },
    animations: false,
    axes: {
      bottom: { mapsTo: 'key', scaleType: ScaleTypes.LABELS },
      left: {
        includeZero: true,
        mapsTo: 'value',
        scaleType: ScaleTypes.LINEAR,
      },
    },
    curve: 'curveMonotoneX',
    grid: { x: { enabled: false }, y: { enabled: true } },
    height: '280px',
    legend: {
      alignment: Alignments.RIGHT,
      position: LegendPositions.TOP,
    },
    points: { enabled: false },
    theme: resolvedMode === 'dark' ? ChartTheme.G100 : ChartTheme.WHITE,
    toolbar: { enabled: false },
  };

  return (
    <DashboardCard
      title="Conversation Activity"
      subtitle="Sessions and failures over time"
    >
      {activity.length ? (
        <div className="min-w-0 px-6 pb-6">
          <LineChart data={activity} options={chartOptions} />
        </div>
      ) : (
        <EmptyContent
          title="No activity yet"
          description="Conversation activity will appear after this project handles sessions."
        />
      )}
    </DashboardCard>
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
    <DashboardCard
      title="Health Summary"
      subtitle="Reliability signals requiring attention"
    >
      <ul className="divide-y divide-border-subtle px-6 ">
        <li className="flex min-h-[106px] items-start gap-4 py-5">
          <WarningAltFilled
            size={24}
            aria-hidden="true"
            className="mt-0.5 shrink-0 text-support-error"
          />
          <div className="min-w-0 flex-1">
            <Text as="p" variant="heading-compact-01">
              Failed sessions
            </Text>
            <Text as="p" variant="label-01" className="mt-2 text-muted">
              {numberFormatter.format(recentFailures)} recent{' '}
              {recentFailures === 1 ? 'failure' : 'failures'}
            </Text>
          </div>
          <Text variant="heading-04" className="tabular-nums">
            {numberFormatter.format(health?.getFailedsessions() || 0)}
          </Text>
        </li>
        <li className="flex min-h-[106px] items-start gap-4 py-5">
          <WarningAltFilled
            size={24}
            aria-hidden="true"
            className="mt-0.5 shrink-0 text-support-warning"
          />
          <div className="min-w-0 flex-1">
            <Text as="p" variant="heading-compact-01">
              Slow sessions
            </Text>
            <Text as="p" variant="label-01" className="mt-2 text-muted">
              Response latency above {numberFormatter.format(threshold)} ms
            </Text>
          </div>
          <Text variant="heading-04" className="tabular-nums">
            {numberFormatter.format(health?.getSlowsessions() || 0)}
          </Text>
        </li>
        <li className="flex min-h-[106px] items-start gap-4 py-5">
          <CheckmarkFilled
            size={24}
            aria-hidden="true"
            className="mt-0.5 shrink-0 text-support-success"
          />
          <div className="min-w-0 flex-1">
            <Text as="p" variant="heading-compact-01">
              Healthy assistants
            </Text>
            <Text as="p" variant="label-01" className="mt-2 text-muted">
              Based on success rate and latency
            </Text>
          </div>
          <Text variant="heading-04" className="tabular-nums">
            {healthyAssistants} of {assistantCount}
          </Text>
        </li>
      </ul>
    </DashboardCard>
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
        className="text-disabled"
      />
      <Text as="h3" variant="heading-02" className="mt-4 text-foreground">
        {title}
      </Text>
      <Text
        as="p"
        variant="body-compact-01"
        className="mt-1 max-w-sm text-muted"
      >
        {description}
      </Text>
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
    <DashboardCard title={title} subtitle={subtitle}>
      <EmptyContent
        title={isEmpty ? `No ${title.toLowerCase()} yet` : unavailableTitle}
        description={
          isEmpty
            ? 'Data will appear here after the project starts handling sessions.'
            : 'This section could not be loaded. Other available dashboard data is still shown.'
        }
      />
    </DashboardCard>
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
    <DashboardCard title={title} subtitle={subtitle}>
      {values.length ? (
        <div className="grid gap-6 p-6 sm:grid-cols-2">
          {values.map(value => (
            <div key={value.label}>
              <div className="mb-2 flex items-center justify-between gap-4">
                <Text variant="body-compact-01">{value.label}</Text>
                <Text variant="heading-compact-01" className="tabular-nums">
                  {value.value}
                </Text>
              </div>
              <div className="h-1 bg-layer ">
                <div
                  data-dashboard-bar
                  className="h-full bg-primary"
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
    </DashboardCard>
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
    <DashboardCard
      title="Recent Failures"
      subtitle="Latest sessions requiring investigation"
    >
      {rows.length ? (
        <DataTable rows={rows} headers={failureTableHeaders}>
          {({ rows, headers, getHeaderProps, getRowProps, getTableProps }) => (
            <TableContainer>
              <div className="overflow-x-auto">
                <Table
                  {...getTableProps()}
                  aria-label="Recent Failures"
                  className="min-w-[440px]"
                  size="md"
                  tabIndex={0}
                >
                  <caption
                    className={`${typographyClassNames['label-01']} sr-only`}
                  >
                    Recent failed sessions for the selected time range
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
                                className="text-link hover:underline focus:underline"
                              >
                                <Text variant="heading-compact-01">
                                  {cell.value}
                                </Text>
                              </a>
                            ) : cell.info.header === 'reason' ? (
                              <Tag size="sm" type="red">
                                <Text variant="label-01">{cell.value}</Text>
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
          description="No failed sessions were returned for the selected time range."
        />
      )}
    </DashboardCard>
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
    <DashboardCard
      title="Top Assistants"
      subtitle="Assistants ranked by conversation volume"
      action={
        <Button
          kind="ghost"
          size="sm"
          renderIcon={ArrowRight}
          href="/deployment/assistant"
        >
          <Text variant="body-compact-01">View all assistants</Text>
        </Button>
      }
    >
      {rows.length ? (
        <DataTable rows={rows} headers={assistantTableHeaders}>
          {({ rows, headers, getHeaderProps, getRowProps, getTableProps }) => (
            <TableContainer>
              <div className="overflow-x-auto">
                <Table
                  {...getTableProps()}
                  aria-label="Top Assistants"
                  className="min-w-[760px]"
                  size="md"
                  tabIndex={0}
                >
                  <caption
                    className={`${typographyClassNames['label-01']} sr-only`}
                  >
                    Top Assistants ranked by conversation volume for the
                    selected time range
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
                                    ? 'bg-support-warning! text-foreground!'
                                    : undefined
                                }
                              >
                                <Text variant="label-01">{cell.value}</Text>
                              </Tag>
                            ) : cell.info.header === 'assistant' ? (
                              <a
                                href={`/deployment/assistant/${encodeURIComponent(
                                  row.id,
                                )}`}
                                className="text-link hover:underline focus:underline"
                              >
                                <Text variant="heading-compact-01">
                                  {cell.value}
                                </Text>
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
    </DashboardCard>
  );
}

function DashboardLoading() {
  return (
    <div
      className="col-span-full"
      role="status"
      aria-label="Loading project dashboard"
    >
      <Text variant="label-01" className="sr-only">
        Loading project dashboard
      </Text>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        {[0, 1, 2, 3].map(item => (
          <div
            data-dashboard-loading-card
            data-dashboard-loading-kpi
            className="flex h-[156px] flex-col border border-border-subtle bg-surface"
            key={item}
            aria-hidden="true"
          >
            <div
              data-dashboard-kpi-header
              className="flex h-10 shrink-0 items-center border-b border-border-subtle px-4"
            >
              <SkeletonText className="!mb-0" width="45%" />
            </div>
            <div
              data-dashboard-kpi-body
              className="flex h-[116px] flex-col justify-between p-4"
            >
              <SkeletonText className="!mb-0" width="70%" heading />
              <SkeletonText className="!mb-0" width="80%" />
            </div>
          </div>
        ))}
      </div>
      <div className="mt-4 grid gap-4 xl:grid-cols-16">
        <div
          data-dashboard-loading-card
          className="h-[396px] border border-border-subtle bg-surface p-6 xl:col-span-11"
        >
          <SkeletonText width="30%" heading />
          <SkeletonText width="45%" />
          <SkeletonPlaceholder className="mt-8 h-[260px]! w-full!" />
        </div>
        <div
          data-dashboard-loading-card
          className="h-[396px] border border-border-subtle bg-surface p-6 xl:col-span-5"
        >
          <SkeletonText width="55%" heading />
          <SkeletonText width="75%" />
          <SkeletonText className="mt-10" width="100%" />
          <SkeletonText className="mt-12" width="100%" />
          <SkeletonText className="mt-12" width="100%" />
        </div>
      </div>
      <div className="mt-4 grid gap-4 lg:grid-cols-2">
        {[0, 1].map(item => (
          <div
            data-dashboard-loading-card
            className="h-[260px] border border-border-subtle bg-surface p-6 "
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
      description:
        'Total conversation sessions handled during the selected time range.',
      value: numberFormatter.format(sessions?.getValue() || 0),
      comparison: sessionComparison,
    },
    {
      label: 'Success Rate',
      description:
        'Percentage of project sessions completed successfully during the selected time range.',
      value: successRate?.hasValuepercent()
        ? `${successRate.getValuepercent().toFixed(1)}%`
        : 'Not available',
      comparison: successComparison,
    },
    {
      label: 'Average Latency',
      description:
        'Average end-to-end response latency across project sessions during the selected time range.',
      value: latency?.hasValuems()
        ? numberFormatter.format(latency.getValuems())
        : 'Not available',
      unit: latency?.hasValuems() ? 'ms' : undefined,
      comparison: latencyComparison,
    },
    {
      label: 'Failed Sessions',
      description:
        'Project sessions that failed during the selected time range.',
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
  const [dashboardRange, setDashboardRange] = useState<DashboardRange>('30d');
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
  const currentContextKey = `${projectId}:${dashboardRange}`;
  const isContextPending = dashboardContextKey.current !== currentContextKey;
  const dashboard =
    dashboardState?.contextKey === currentContextKey
      ? dashboardState.data
      : null;

  const fetchDashboard = useCallback(() => {
    requestId.current += 1;
    const currentRequestId = requestId.current;
    const contextKey = `${projectId}:${dashboardRange}`;
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
      fromDate: toTimestamp(getDashboardStartDate(dashboardRange, toDate)),
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
  }, [authId, dashboardRange, projectId, token]);

  useEffect(() => {
    fetchDashboard();
  }, [fetchDashboard]);

  useEffect(
    () => () => {
      requestId.current += 1;
    },
    [],
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
  const isDashboardActivityEmpty =
    analyticsSection?.getState() ===
      DashboardSectionState.DASHBOARD_SECTION_STATE_EMPTY &&
    usage?.getState() === DashboardSectionState.DASHBOARD_SECTION_STATE_EMPTY;
  const metrics = useMemo(
    () => (analytics ? buildMetrics(analytics) : []),
    [analytics],
  );
  const componentUsage = useMemo(
    () => (usage && isUsageReady ? buildUsage(usage) : []),
    [isUsageReady, usage],
  );

  return (
    <main
      data-dashboard-page
      className="min-h-0 flex-1 overflow-auto bg-layer text-foreground "
    >
      <div data-dashboard-content className="flex min-h-full flex-col">
        <PageHeader
          eyebrow={
            <Text variant="label-01" className="text-primary">
              Dashboard
            </Text>
          }
          title="Workspace Overview"
          description="Monitor workspace performance, conversation activity, assistant health, and component usage across the selected time range."
          contextualActions={
            dashboard?.getPartial() ? (
              <Tag size="sm" type="warm-gray">
                <Text variant="label-01">Partial data</Text>
              </Tag>
            ) : undefined
          }
          pageActions={({ isCollapsed }) => (
            <div className="w-full md:w-56">
              <Select
                id="dashboard-time-range"
                hideLabel={isCollapsed}
                labelText="Time range"
                size={isCollapsed ? 'md' : 'sm'}
                value={dashboardRange}
                onChange={event =>
                  setDashboardRange(event.target.value as DashboardRange)
                }
              >
                <SelectItem value="24h" text="Last 24 hours" />
                <SelectItem value="7d" text="Last 7 days" />
                <SelectItem value="30d" text="Last 30 days" />
              </Select>
            </div>
          )}
        />

        <div
          data-dashboard-grid
          className="mx-auto grid w-full max-w-[1584px] flex-1 grid-cols-4 gap-4 p-4 md:grid-cols-8 md:px-8 lg:grid-cols-16"
        >
          {loadError && !dashboard ? (
            <div className="col-span-full flex min-h-[420px]">
              <EmptyState
                icon={WarningAltFilled}
                title="Project dashboard is unavailable"
                subtitle="Check your connection and try again."
                action="Try again"
                actionIcon={Renew}
                onAction={fetchDashboard}
              />
            </div>
          ) : (
            <>
              {refreshError && !isContextPending && (
                <div className="col-span-full">
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
              ) : dashboard && isDashboardActivityEmpty ? (
                <div className="col-span-full">
                  <DashboardNoActivity />
                </div>
              ) : dashboard ? (
                <>
                  {analytics ? (
                    <>
                      <div
                        data-dashboard-kpi-strip
                        className="col-span-full grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4"
                      >
                        {metrics.map(metric => (
                          <MetricCard key={metric.label} metric={metric} />
                        ))}
                      </div>

                      <div className="col-span-full grid gap-4 xl:grid-cols-16">
                        <div className="min-w-0 xl:col-span-11">
                          <ActivityPanel
                            analytics={analytics}
                            range={dashboardRange}
                          />
                        </div>
                        <div className="xl:col-span-5">
                          <HealthPanel analytics={analytics} />
                        </div>
                      </div>
                    </>
                  ) : (
                    <div className="col-span-full">
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

                  <div className="col-span-full min-w-0 xl:col-span-8">
                    {usage && isUsageReady ? (
                      <BreakdownPanel
                        title="Component Usage"
                        subtitle="Processing time by component"
                        values={componentUsage}
                      />
                    ) : (
                      <StatePanel
                        title="Component Usage"
                        subtitle="Processing time by component"
                        state={
                          usage?.getState() ||
                          DashboardSectionState.DASHBOARD_SECTION_STATE_UNAVAILABLE
                        }
                      />
                    )}
                  </div>

                  <div className="col-span-full min-w-0 xl:col-span-8">
                    {analytics ? (
                      <RecentFailuresTable
                        failures={analytics.getRecentfailuresList()}
                      />
                    ) : (
                      <StatePanel
                        title="Recent Failures"
                        subtitle="Latest sessions requiring investigation"
                        state={
                          analyticsSection?.getState() ||
                          DashboardSectionState.DASHBOARD_SECTION_STATE_UNAVAILABLE
                        }
                      />
                    )}
                  </div>

                  {analytics && (
                    <div className="col-span-full">
                      <AssistantsTable analytics={analytics} />
                    </div>
                  )}
                </>
              ) : null}
            </>
          )}
        </div>
      </div>
    </main>
  );
};
