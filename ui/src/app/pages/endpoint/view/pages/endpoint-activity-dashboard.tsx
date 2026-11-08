import { EndpointLog } from '@rapidaai/react';
import { ArrowRight, Play } from '@carbon/icons-react';
import {
  Button,
  InlineNotification,
  SkeletonPlaceholder,
  SkeletonText,
} from '@carbon/react';
import { useEffect, useMemo, useState } from 'react';
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { listEndpointLogs } from '@/clients/endpoint.client';
import { useCredential } from '@/hooks/use-credential';
import { toDate, toDateString } from '@/utils/date';
import { getTotalTokenMetric } from '@/utils/metadata';
import { Dropdown, Tile } from '@/app/components/ui/primitives';

const RECENT_LOG_LIMIT = 20;
const SUCCESS_STATES = new Set(['SUCCESS', 'COMPLETE', 'COMPLETED']);
const FAILURE_STATES = new Set(['ERROR', 'FAILED']);

const DATE_RANGES = [
  { id: 'last_24_hours', text: 'Last 24 hours' },
  { id: 'last_3_days', text: 'Last 3 days' },
  { id: 'last_7_days', text: 'Last 7 days' },
  { id: 'last_30_days', text: 'Last 30 days' },
];

const AUTO_REFRESH_OPTIONS = [
  { id: '0', text: 'Off' },
  { id: '5', text: 'Every 5 min' },
  { id: '10', text: 'Every 10 min' },
  { id: '30', text: 'Every 30 min' },
];

type DateRangeId =
  | 'last_24_hours'
  | 'last_3_days'
  | 'last_7_days'
  | 'last_30_days';

type DropdownItem = { id: string; text: string };

type EndpointActivityDashboardProps = {
  endpointId: string;
  onOpenLogs: () => void;
  onOpenPlayground: () => void;
};

function MetricTile(props: {
  title: string;
  label: string;
  value: string;
  unit?: string;
  description: string;
  loading: boolean;
}) {
  return (
    <Tile className="h-[156px] !rounded-none !border !border-border-subtle !bg-surface !p-0 ">
      <div className="flex h-10 items-center border-b border-border-subtle px-4 ">
        <h3 className="truncate text-sm font-semibold text-foreground ">
          {props.title}
        </h3>
      </div>
      <div className="flex h-[116px] flex-col justify-between p-4">
        {props.loading ? (
          <>
            <div>
              <SkeletonText width="44%" className="mb-0!" />
              <div className="mt-3 flex items-end gap-2">
                <SkeletonText heading width="96px" className="mb-0!" />
                <SkeletonPlaceholder className="h-4! w-8!" />
              </div>
            </div>
            <SkeletonText width="78%" className="mb-0!" />
          </>
        ) : (
          <>
            <div>
              <p className="truncate text-xs text-muted">{props.label}</p>
              <div className="mt-2 flex items-baseline gap-1">
                <span className="text-4xl font-light leading-none tabular-nums text-foreground ">
                  {props.value}
                </span>
                {props.unit && (
                  <span className="text-sm text-muted">{props.unit}</span>
                )}
              </div>
            </div>
            <p className="truncate text-xs text-muted">{props.description}</p>
          </>
        )}
      </div>
    </Tile>
  );
}

function formatInteger(value: number): string {
  return new Intl.NumberFormat('en-US', {
    maximumFractionDigits: 0,
  }).format(value);
}

function recentSuccessRate(logs: EndpointLog[]): number | undefined {
  if (logs.length === 0) return undefined;
  const successful = logs.filter(log =>
    SUCCESS_STATES.has(log.getStatus().toUpperCase()),
  ).length;
  return Math.round((successful / logs.length) * 100);
}

function recentP50Latency(logs: EndpointLog[]): number | undefined {
  const latencies = logs
    .map(log => Number(log.getTimetaken()))
    .filter(value => Number.isFinite(value) && value > 0)
    .sort((left, right) => left - right);
  if (latencies.length === 0) return undefined;
  const median = latencies[Math.ceil(latencies.length * 0.5) - 1];
  return Math.round(median / 1_000_000);
}

function recentTokenCount(logs: EndpointLog[]): number | undefined {
  const tokenCount = logs.reduce(
    (total, log) => total + getTotalTokenMetric(log.getMetricsList()),
    0,
  );
  return logs.length === 0 ? undefined : tokenCount;
}

function getStartDate(range: DateRangeId): Date {
  const now = new Date();
  switch (range) {
    case 'last_24_hours':
      return new Date(now.getTime() - 24 * 60 * 60 * 1000);
    case 'last_3_days':
      return new Date(now.getTime() - 3 * 24 * 60 * 60 * 1000);
    case 'last_7_days':
      return new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
    default:
      return new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
  }
}

function RequestDetailsCard(props: {
  logs: EndpointLog[];
  totalCount: number;
  loading: boolean;
  error: string;
}) {
  const successful = props.logs.filter(log =>
    SUCCESS_STATES.has(log.getStatus().toUpperCase()),
  ).length;
  const failed = props.logs.filter(log =>
    FAILURE_STATES.has(log.getStatus().toUpperCase()),
  ).length;
  const latencies = props.logs
    .map(log => Number(log.getTimetaken()))
    .filter(value => Number.isFinite(value) && value > 0);
  const averageLatency =
    latencies.length > 0
      ? Math.round(
          latencies.reduce((total, latency) => total + latency, 0) /
            latencies.length /
            1_000_000,
        )
      : undefined;
  const unavailable = props.error ? '--' : undefined;

  return (
    <Tile className="h-[310px] !rounded-none !border !border-border-subtle !bg-surface !p-0 ">
      <div className="flex h-12 items-center border-b border-border-subtle px-4 ">
        <h2 className="truncate text-sm font-semibold text-foreground ">
          Request details
        </h2>
      </div>
      <div className="h-[262px] p-6">
        {props.loading ? (
          <>
            <SkeletonText width="52%" className="mb-2!" />
            <SkeletonText heading width="112px" className="mb-2!" />
            <SkeletonText width="72%" className="mb-0!" />
            <div className="mt-5 space-y-4 border-t border-border-subtle pt-4 ">
              <SkeletonText width="100%" className="mb-0!" />
              <SkeletonText width="100%" className="mb-0!" />
              <SkeletonText width="100%" className="mb-0!" />
            </div>
          </>
        ) : (
          <>
            <p className="text-xs text-muted">Avg request latency</p>
            <div className="mt-2 flex items-baseline gap-1">
              <span className="text-4xl font-light leading-none tabular-nums text-foreground ">
                {unavailable ?? averageLatency ?? '--'}
              </span>
              {!unavailable && averageLatency !== undefined && (
                <span className="text-sm text-muted">ms</span>
              )}
            </div>
            <p className="mt-2 text-xs text-muted">
              Average across the latest {props.logs.length || 0} requests
            </p>
            <div className="mt-5 divide-y divide-border-subtle border-t border-border-subtle ">
              {[
                { label: 'Successful', value: unavailable ?? successful },
                { label: 'Failed', value: unavailable ?? failed },
                {
                  label: 'Total in range',
                  value: unavailable ?? formatInteger(props.totalCount),
                },
              ].map(row => (
                <div
                  key={row.label}
                  className="flex items-center justify-between gap-4 py-2.5"
                >
                  <span className="text-sm text-muted ">{row.label}</span>
                  <span className="text-sm font-semibold tabular-nums text-foreground ">
                    {row.value}
                  </span>
                </div>
              ))}
            </div>
          </>
        )}
      </div>
    </Tile>
  );
}

export function EndpointActivityDashboard({
  endpointId,
  onOpenLogs,
  onOpenPlayground,
}: EndpointActivityDashboardProps) {
  const [userId, token, projectId] = useCredential();
  const [logs, setLogs] = useState<EndpointLog[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [requestNumber, setRequestNumber] = useState(0);
  const [selectedRange, setSelectedRange] =
    useState<DateRangeId>('last_30_days');
  const [autoRefreshInterval, setAutoRefreshInterval] = useState<number | null>(
    null,
  );

  useEffect(() => {
    let isActive = true;
    setLoading(true);
    setError('');

    listEndpointLogs({
      endpointId,
      page: 1,
      pageSize: RECENT_LOG_LIMIT,
      criteria: [
        {
          key: 'created_date',
          value: toDateString(new Date()),
          logic: '<=',
        },
        {
          key: 'created_date',
          value: toDateString(getStartDate(selectedRange)),
          logic: '>=',
        },
      ],
      auth: { userId, token, projectId },
      callback: (requestError, response) => {
        if (!isActive) return;
        setLoading(false);

        if (requestError || !response?.getSuccess()) {
          setError(
            response?.getError()?.getHumanmessage() ||
              requestError?.message ||
              'Unable to load endpoint activity.',
          );
          return;
        }

        const recentLogs = response.getDataList();
        setLogs(recentLogs);
        setTotalCount(
          response.getPaginated()?.getTotalitem() ?? recentLogs.length,
        );
      },
    });

    return () => {
      isActive = false;
    };
  }, [endpointId, projectId, requestNumber, selectedRange, token, userId]);

  useEffect(() => {
    if (!autoRefreshInterval) return;
    const intervalId = window.setInterval(
      () => setRequestNumber(value => value + 1),
      autoRefreshInterval * 60 * 1000,
    );
    return () => window.clearInterval(intervalId);
  }, [autoRefreshInterval]);

  const metrics = useMemo(() => {
    const successRate = recentSuccessRate(logs);
    const p50Latency = recentP50Latency(logs);
    const tokenCount = recentTokenCount(logs);
    const unavailable = error ? '--' : undefined;

    return [
      {
        title: 'Requests',
        label: 'Total requests',
        value: unavailable ?? formatInteger(totalCount),
        description: 'Selected date range',
      },
      {
        title: 'Reliability',
        label: 'Success rate',
        value: unavailable ?? (successRate?.toString() || '--'),
        unit: successRate === undefined || error ? undefined : '%',
        description: `Latest ${logs.length || RECENT_LOG_LIMIT} requests`,
      },
      {
        title: 'Latency',
        label: 'Recent p50',
        value: unavailable ?? (p50Latency?.toString() || '--'),
        unit: p50Latency === undefined || error ? undefined : 'ms',
        description: `Latest ${logs.length || RECENT_LOG_LIMIT} requests`,
      },
      {
        title: 'Usage',
        label: 'Recent tokens',
        value:
          unavailable ??
          (tokenCount === undefined ? '--' : formatInteger(tokenCount)),
        description: `Latest ${logs.length || RECENT_LOG_LIMIT} requests`,
      },
    ];
  }, [error, logs, totalCount]);

  const chartData = useMemo(
    () =>
      [...logs].reverse().map((log, index) => ({
        label: log.getCreateddate()
          ? toDate(log.getCreateddate()!).toLocaleTimeString([], {
              hour: '2-digit',
              minute: '2-digit',
            })
          : `Request ${index + 1}`,
        latency: Math.round(Number(log.getTimetaken()) / 1_000_000),
      })),
    [logs],
  );

  return (
    <section
      className="min-h-full w-full bg-layer p-4 "
      aria-labelledby="endpoint-activity-title"
    >
      <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-xs text-muted">Dashboard</p>
          <h1
            id="endpoint-activity-title"
            className="text-2xl font-normal text-foreground "
          >
            Endpoint activity
          </h1>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <Dropdown
            id="endpoint-date-range"
            titleText=""
            hideLabel
            label="Date range"
            size="sm"
            items={DATE_RANGES}
            selectedItem={DATE_RANGES.find(range => range.id === selectedRange)}
            itemToString={(item: DropdownItem | null) => item?.text || ''}
            onChange={({ selectedItem }) => {
              if (selectedItem)
                setSelectedRange(selectedItem.id as DateRangeId);
            }}
            className="min-w-[160px]"
          />
          <Dropdown
            id="endpoint-auto-refresh"
            titleText=""
            hideLabel
            label="Auto-refresh"
            size="sm"
            items={AUTO_REFRESH_OPTIONS}
            selectedItem={AUTO_REFRESH_OPTIONS.find(
              option => option.id === String(autoRefreshInterval || 0),
            )}
            itemToString={(item: DropdownItem | null) => item?.text || ''}
            onChange={({ selectedItem }) => {
              if (selectedItem)
                setAutoRefreshInterval(
                  selectedItem.id === '0' ? null : Number(selectedItem.id),
                );
            }}
            className="min-w-[140px]"
          />
        </div>
      </div>

      <div className="mb-4 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        {metrics.map(metric => (
          <MetricTile key={metric.label} loading={loading} {...metric} />
        ))}
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        <RequestDetailsCard
          logs={logs}
          totalCount={totalCount}
          loading={loading}
          error={error}
        />
        <Tile className="h-[310px] !rounded-none !border !border-border-subtle !bg-surface !p-0 md:col-span-2 ">
          <div className="flex h-12 items-center justify-between border-b border-border-subtle px-4 ">
            <h2
              id="recent-requests-title"
              className="text-sm font-semibold text-foreground "
            >
              Request latency
            </h2>
            <Button
              kind="ghost"
              size="sm"
              renderIcon={ArrowRight}
              onClick={onOpenLogs}
            >
              View all logs
            </Button>
          </div>

          {error ? (
            <div className="p-4">
              <InlineNotification
                kind="error"
                lowContrast
                hideCloseButton
                title="Endpoint activity is unavailable"
                subtitle={error}
                className="m-0! max-w-full!"
              />
              <Button
                kind="tertiary"
                size="sm"
                className="mt-4"
                onClick={() => setRequestNumber(value => value + 1)}
              >
                Retry
              </Button>
            </div>
          ) : loading ? (
            <div className="h-[262px] p-6">
              <SkeletonPlaceholder className="h-full! w-full!" />
            </div>
          ) : logs.length === 0 ? (
            <div className="flex h-[262px] flex-col items-start justify-center px-4 py-6">
              <h3 className="text-base font-semibold text-foreground">
                No requests yet
              </h3>
              <p className="mt-1 max-w-lg text-sm text-muted">
                Run this endpoint in the playground. Its status, latency, and
                token usage will appear here.
              </p>
              <div className="mt-6">
                <Button
                  kind="tertiary"
                  size="sm"
                  renderIcon={Play}
                  onClick={onOpenPlayground}
                >
                  Open playground
                </Button>
              </div>
            </div>
          ) : (
            <div
              className="h-[262px] px-4 pb-4 pt-6"
              role="img"
              aria-label={`Latency for ${chartData.length} recent endpoint requests`}
              data-testid="endpoint-request-performance-chart"
            >
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart
                  data={chartData}
                  margin={{ top: 4, right: 16, left: 0, bottom: 0 }}
                >
                  <defs>
                    <linearGradient
                      id="endpointLatencyGradient"
                      x1="0"
                      y1="0"
                      x2="0"
                      y2="1"
                    >
                      <stop
                        offset="0%"
                        stopColor="var(--color-primary)"
                        stopOpacity={0.28}
                      />
                      <stop
                        offset="100%"
                        stopColor="var(--color-primary)"
                        stopOpacity={0.02}
                      />
                    </linearGradient>
                  </defs>
                  <CartesianGrid
                    stroke="var(--color-border-subtle)"
                    vertical={false}
                  />
                  <XAxis
                    dataKey="label"
                    tick={{ fontSize: 11, fill: 'var(--color-muted)' }}
                    tickLine={false}
                    axisLine={false}
                  />
                  <YAxis
                    width={48}
                    tick={{ fontSize: 11, fill: 'var(--color-muted)' }}
                    tickLine={false}
                    axisLine={false}
                    unit=" ms"
                  />
                  <Tooltip
                    formatter={value => [`${value} ms`, 'Latency']}
                    contentStyle={{
                      border: '1px solid var(--color-border-subtle)',
                      borderRadius: 0,
                      background: 'var(--color-layer)',
                      color: 'var(--color-foreground)',
                      fontSize: 12,
                    }}
                  />
                  <Area
                    type="monotone"
                    dataKey="latency"
                    name="Latency"
                    stroke="var(--color-primary)"
                    fill="url(#endpointLatencyGradient)"
                    strokeWidth={2}
                    activeDot={{ r: 4 }}
                    isAnimationActive={false}
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          )}
        </Tile>
      </div>
    </section>
  );
}
