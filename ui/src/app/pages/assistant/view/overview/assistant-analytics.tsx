import {
  Assistant,
  AssistantDashboard,
  GetAssistantDashboard,
  GetAssistantDashboardRequest,
} from '@rapidaai/react';
import { connectionConfig } from '@/configs';
import { toDate, toDateString } from '@/utils/date';
import { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb';
import {
  XAxis,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  Bar,
  BarChart,
  YAxis,
  AreaChart,
  Area,
} from 'recharts';
import {
  NameType,
  ValueType,
} from 'recharts/types/component/DefaultTooltipContent';
import { ContentType } from 'recharts/types/component/Tooltip';
import {
  FC,
  ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { cn } from '@/utils';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { Dropdown } from '@/app/components/carbon/dropdown';
import { Tile } from '@/app/components/carbon/tile';
import toast from 'react-hot-toast/headless';
import {
  Button,
  SkeletonPlaceholder,
  SkeletonText,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import { Information } from '@carbon/icons-react';

const CHART_COLORS = [
  'var(--cds-interactive, #1e40af)',
  '#22d3ee',
  '#f59e0b',
  '#10b981',
  '#f43f5e',
  '#8b5cf6',
];

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

type DistributionData = {
  name: string;
  count: number;
  percentage: string;
};

type BucketData = {
  dateHour: string;
  total: number;
  sttLatency: number;
  eosLatency: number;
  ttsLatency: number;
  agentLatency: number;
  label: string;
};

type ChartTooltipPayload = {
  color?: string;
  dataKey?: string;
  name?: string;
  payload?: { label?: string };
  stroke?: string;
  value?: ReactNode;
};

type DashboardWidgetSkeletonVariant =
  | 'metric-list'
  | 'latency-chart'
  | 'donut'
  | 'progress-list'
  | 'bar-chart';

const DASHBOARD_UNAVAILABLE_VALUE = '--';
const DASHBOARD_LOAD_ERROR = 'Dashboard data is unavailable. Please try again.';

const getStartDate = (range: DateRangeId): Date => {
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
};

const toTimestamp = (date: Date): Timestamp => {
  const timestamp = new Timestamp();
  timestamp.setSeconds(Math.floor(date.getTime() / 1000));
  timestamp.setNanos((date.getTime() % 1000) * 1_000_000);
  return timestamp;
};

const getBucketLabel = (date: Date, range: DateRangeId): string => {
  const hours = date.getHours().toString().padStart(2, '0');
  const minutes = date.getMinutes().toString().padStart(2, '0');

  switch (range) {
    case 'last_24_hours':
      return `${hours}:${minutes}`;
    case 'last_3_days':
    case 'last_7_days':
      return `${toDateString(date)} ${hours}:00`;
    default:
      return toDateString(date);
  }
};

export const AssistantAnalytics: FC<{ assistant: Assistant }> = props => {
  const navigation = useGlobalNavigation();
  const assistantId = props.assistant.getId();
  const dashboardRequestId = useRef(0);
  const dashboardContextKey = useRef('');
  const [autoRefreshInterval, setAutoRefreshInterval] = useState<null | number>(
    null,
  );
  const [selectedRange, setSelectedRange] =
    useState<DateRangeId>('last_30_days');
  const [dashboard, setDashboard] = useState<AssistantDashboard | null>(null);
  const [loading, setLoading] = useState(true);
  const { authId, token, projectId } = useCurrentCredential();

  const fetchDashboard = useCallback(() => {
    dashboardRequestId.current += 1;
    const requestId = dashboardRequestId.current;

    if (!assistantId || !authId || !token || !projectId) {
      dashboardContextKey.current = '';
      setDashboard(null);
      setLoading(false);
      return;
    }

    const requestContextKey = `${assistantId}:${projectId}:${selectedRange}`;
    if (dashboardContextKey.current !== requestContextKey) {
      dashboardContextKey.current = requestContextKey;
      setDashboard(null);
    }

    setLoading(true);
    const fromDate = getStartDate(selectedRange);
    const toDate = new Date();
    const request = new GetAssistantDashboardRequest();
    request.setAssistantid(assistantId);
    request.setFromdate(toTimestamp(fromDate));
    request.setTodate(toTimestamp(toDate));

    GetAssistantDashboard(connectionConfig, request, {
      authorization: token,
      'x-auth-id': authId,
      'x-project-id': projectId,
    })
      .then(response => {
        if (requestId !== dashboardRequestId.current) return;

        const dashboardData = response.getData();
        if (response.getSuccess() && dashboardData) {
          setDashboard(dashboardData);
          return;
        }

        toast.error(DASHBOARD_LOAD_ERROR);
      })
      .catch(() => {
        if (requestId !== dashboardRequestId.current) return;
        toast.error(DASHBOARD_LOAD_ERROR);
      })
      .finally(() => {
        if (requestId !== dashboardRequestId.current) return;
        setLoading(false);
      });
  }, [assistantId, authId, projectId, selectedRange, token]);

  useEffect(() => {
    fetchDashboard();
  }, [fetchDashboard]);

  useEffect(
    () => () => {
      dashboardRequestId.current += 1;
    },
    [],
  );

  useEffect(() => {
    let id: NodeJS.Timeout | null = null;
    if (autoRefreshInterval && autoRefreshInterval > 0)
      id = setInterval(
        () => {
          fetchDashboard();
        },
        autoRefreshInterval * 60 * 1000,
      );
    return () => {
      if (id) clearInterval(id);
    };
  }, [autoRefreshInterval, fetchDashboard]);

  const hasDashboard = dashboard !== null;
  const summary = dashboard?.getSummary();
  const latency = dashboard?.getLatency();
  const usage = dashboard?.getUsage();

  const totalSessions = summary?.getTotalsessions() || 0;
  const activeConversations = summary?.getActivesessions() || 0;
  const completedConversations = summary?.getCompletedsessions() || 0;
  const failedConversations = summary?.getFailedsessions() || 0;
  const totalMessages = summary?.getTotalmessages() || 0;
  const failureRate = summary?.getFailurerate() || 0;
  const avgSessionDuration = summary?.getAveragesessiondurationseconds() || 0;

  const avgLatency = latency?.getAveragems() || 0;
  const avgSttLatency = latency?.getSttms() || 0;
  const avgEosLatency = latency?.getEosms() || 0;
  const avgTtsLatency = latency?.getTtsms() || 0;
  const avgLlmLatency = latency?.getLlmms() || 0;

  const totalTokens = usage?.getTotaltokens() || 0;
  const totalSttDurationSec = usage?.getSttdurationseconds() || 0;
  const totalTtsDurationSec = usage?.getTtsdurationseconds() || 0;
  const totalDuration = usage?.getTotaldurationseconds() || 0;

  const sourceData = useMemo<DistributionData[]>(
    () =>
      (dashboard?.getSourcesList() || []).map(source => ({
        name: source.getName() || 'unknown',
        count: source.getCount(),
        percentage: source.getPercentage().toFixed(1),
      })),
    [dashboard],
  );

  const languageData = useMemo<DistributionData[]>(
    () =>
      (dashboard?.getLanguagesList() || []).map(language => ({
        name: language.getName() || 'unknown',
        count: language.getCount(),
        percentage: language.getPercentage().toFixed(1),
      })),
    [dashboard],
  );

  const activeSessionsData = useMemo<BucketData[]>(
    () =>
      (dashboard?.getBucketsList() || []).map(bucket => {
        const startDate = bucket.getStartdate()
          ? toDate(bucket.getStartdate()!)
          : new Date();

        return {
          dateHour: getBucketLabel(startDate, selectedRange),
          total: bucket.getMessagecount(),
          sttLatency: Math.round(bucket.getSttlatencyms()),
          eosLatency: Math.round(bucket.getEoslatencyms()),
          ttsLatency: Math.round(bucket.getTtslatencyms()),
          agentLatency: Math.round(bucket.getLlmlatencyms()),
          label: `From: ${startDate.toISOString().split('.')[0].replace('T', ' ')}`,
        };
      }),
    [dashboard, selectedRange],
  );

  const emptyDashboardCaption = 'Dashboard data not loaded';

  const sessionsAction = (
    <Toggletip align="bottom-left">
      <ToggletipButton label="Sessions actions" title="Sessions actions">
        <Information size={16} className="text-gray-600 dark:text-gray-300" />
      </ToggletipButton>
      <ToggletipContent>
        <p className="text-xs mb-2">
          Open the full sessions page for this assistant.
        </p>
        <div className="flex justify-end">
          <Button
            kind="primary"
            size="sm"
            onClick={() => navigation.goToAssistantSessionList(assistantId)}
          >
            Go to sessions
          </Button>
        </div>
      </ToggletipContent>
    </Toggletip>
  );

  return (
    <div className="w-full min-h-full bg-gray-100 p-4 dark:bg-[#161616]">
      <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-xs text-gray-500 dark:text-gray-400">Dashboard</p>
          <h2 className="text-2xl font-normal text-gray-900 dark:text-gray-100">
            Assistant activity
          </h2>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <Dropdown
            id="date-range"
            titleText=""
            hideLabel
            label="Date range"
            size="sm"
            items={DATE_RANGES}
            selectedItem={DATE_RANGES.find(r => r.id === selectedRange)}
            itemToString={(item: DropdownItem) => item?.text || ''}
            onChange={({ selectedItem }) => {
              if (selectedItem)
                setSelectedRange(selectedItem.id as DateRangeId);
            }}
            className="min-w-[160px]"
          />
          <Dropdown
            id="auto-refresh"
            titleText=""
            hideLabel
            label="Auto-refresh"
            size="sm"
            items={AUTO_REFRESH_OPTIONS}
            selectedItem={AUTO_REFRESH_OPTIONS.find(
              o => o.id === String(autoRefreshInterval || 0),
            )}
            itemToString={(item: DropdownItem) => item?.text || ''}
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
        <KpiTile
          title="Sessions"
          label="Total sessions"
          value={hasDashboard ? totalSessions : DASHBOARD_UNAVAILABLE_VALUE}
          caption={
            hasDashboard
              ? `${activeConversations.toLocaleString()} active, ${completedConversations.toLocaleString()} completed`
              : emptyDashboardCaption
          }
          action={sessionsAction}
          isLoading={loading}
        />
        <KpiTile
          title="Messages"
          label="Total messages"
          value={hasDashboard ? totalMessages : DASHBOARD_UNAVAILABLE_VALUE}
          caption={
            hasDashboard
              ? `${totalTokens.toLocaleString()} tokens used`
              : emptyDashboardCaption
          }
          isLoading={loading}
        />
        <KpiTile
          title="Avg latency"
          label="Average response latency"
          value={
            hasDashboard ? Math.round(avgLatency) : DASHBOARD_UNAVAILABLE_VALUE
          }
          unit={hasDashboard ? 'ms' : undefined}
          caption={
            hasDashboard
              ? `STT ${Math.round(avgSttLatency).toLocaleString()} ms, Agent ${Math.round(avgLlmLatency).toLocaleString()} ms`
              : emptyDashboardCaption
          }
          isLoading={loading}
        />
        <KpiTile
          title="Failure rate"
          label="Failed sessions"
          value={
            hasDashboard ? failureRate.toFixed(1) : DASHBOARD_UNAVAILABLE_VALUE
          }
          unit={hasDashboard ? '%' : undefined}
          caption={
            hasDashboard
              ? `${failedConversations.toLocaleString()} failed of ${totalSessions.toLocaleString()} sessions`
              : emptyDashboardCaption
          }
          isLoading={loading}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        <DashboardWidget
          title="Session details"
          isLoading={loading}
          skeletonVariant="metric-list"
        >
          <WidgetHeroMetric
            label="Avg session duration"
            value={
              hasDashboard
                ? Math.round(avgSessionDuration)
                : DASHBOARD_UNAVAILABLE_VALUE
            }
            unit={hasDashboard ? 's' : undefined}
            caption={
              hasDashboard
                ? 'Average duration across sessions'
                : emptyDashboardCaption
            }
          />
          <WidgetList
            rows={[
              {
                label: 'Active',
                value: hasDashboard
                  ? activeConversations.toLocaleString()
                  : DASHBOARD_UNAVAILABLE_VALUE,
              },
              {
                label: 'Completed',
                value: hasDashboard
                  ? completedConversations.toLocaleString()
                  : DASHBOARD_UNAVAILABLE_VALUE,
              },
              {
                label: 'Failed',
                value: hasDashboard
                  ? failedConversations.toLocaleString()
                  : DASHBOARD_UNAVAILABLE_VALUE,
              },
            ]}
          />
        </DashboardWidget>

        <DashboardWidget
          title="Latency"
          size="large"
          isLoading={loading}
          skeletonVariant="latency-chart"
        >
          <div className="grid grid-cols-2 gap-x-6 gap-y-4 md:grid-cols-5">
            <InlineMetric
              label="Avg latency"
              value={
                hasDashboard
                  ? Math.round(avgLatency)
                  : DASHBOARD_UNAVAILABLE_VALUE
              }
              unit={hasDashboard ? 'ms' : undefined}
            />
            <InlineMetric
              label="STT"
              value={
                hasDashboard
                  ? Math.round(avgSttLatency)
                  : DASHBOARD_UNAVAILABLE_VALUE
              }
              unit={hasDashboard ? 'ms' : undefined}
            />
            <InlineMetric
              label="EOS"
              value={
                hasDashboard
                  ? Math.round(avgEosLatency)
                  : DASHBOARD_UNAVAILABLE_VALUE
              }
              unit={hasDashboard ? 'ms' : undefined}
            />
            <InlineMetric
              label="TTS"
              value={
                hasDashboard
                  ? Math.round(avgTtsLatency)
                  : DASHBOARD_UNAVAILABLE_VALUE
              }
              unit={hasDashboard ? 'ms' : undefined}
            />
            <InlineMetric
              label="Agent"
              value={
                hasDashboard
                  ? Math.round(avgLlmLatency)
                  : DASHBOARD_UNAVAILABLE_VALUE
              }
              unit={hasDashboard ? 'ms' : undefined}
            />
          </div>
          <div className="h-[166px] pt-4">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart
                data={activeSessionsData}
                margin={{ top: 4, right: 8, left: 0, bottom: 0 }}
              >
                <defs>
                  <linearGradient id="sttGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#ff832b" stopOpacity={0.28} />
                    <stop
                      offset="100%"
                      stopColor="#ff832b"
                      stopOpacity={0.02}
                    />
                  </linearGradient>
                  <linearGradient id="eosGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#1192e8" stopOpacity={0.28} />
                    <stop
                      offset="100%"
                      stopColor="#1192e8"
                      stopOpacity={0.02}
                    />
                  </linearGradient>
                  <linearGradient id="ttsGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop
                      offset="0%"
                      stopColor="var(--cds-interactive, #0f62fe)"
                      stopOpacity={0.28}
                    />
                    <stop
                      offset="100%"
                      stopColor="var(--cds-interactive, #0f62fe)"
                      stopOpacity={0.02}
                    />
                  </linearGradient>
                  <linearGradient
                    id="agentGradient"
                    x1="0"
                    y1="0"
                    x2="0"
                    y2="1"
                  >
                    <stop offset="0%" stopColor="#24a148" stopOpacity={0.28} />
                    <stop
                      offset="100%"
                      stopColor="#24a148"
                      stopOpacity={0.02}
                    />
                  </linearGradient>
                </defs>
                <Area
                  type="monotone"
                  dataKey="sttLatency"
                  stroke="#ff832b"
                  strokeWidth={1.5}
                  fill="url(#sttGradient)"
                  dot={false}
                  activeDot={{ r: 3 }}
                />
                <Area
                  type="monotone"
                  dataKey="eosLatency"
                  stroke="#1192e8"
                  strokeWidth={1.5}
                  fill="url(#eosGradient)"
                  dot={false}
                  activeDot={{ r: 3 }}
                />
                <Area
                  type="monotone"
                  dataKey="ttsLatency"
                  stroke="var(--cds-interactive, #0f62fe)"
                  strokeWidth={1.5}
                  fill="url(#ttsGradient)"
                  dot={false}
                  activeDot={{ r: 3 }}
                />
                <Area
                  type="monotone"
                  dataKey="agentLatency"
                  stroke="#24a148"
                  strokeWidth={1.5}
                  fill="url(#agentGradient)"
                  dot={false}
                  activeDot={{ r: 3 }}
                />
                <Tooltip
                  content={
                    (({ active, payload }) => {
                      if (!active || !payload?.length) return null;
                      const labelMap: Record<string, string> = {
                        sttLatency: 'STT',
                        eosLatency: 'EOS',
                        ttsLatency: 'TTS',
                        agentLatency: 'Agent',
                      };
                      return (
                        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 shadow-lg px-3 py-2 text-sm min-w-[140px]">
                          <p className="text-gray-400 text-xs mb-1.5">
                            {payload[0]?.payload?.label}
                          </p>
                          {(payload as ChartTooltipPayload[]).map(p => (
                            <div
                              key={p.dataKey}
                              className="flex items-center gap-2"
                            >
                              <div
                                className="w-2 h-2"
                                style={{ backgroundColor: p.stroke }}
                              />
                              <span className="text-gray-600 dark:text-gray-300 uppercase text-xs">
                                {labelMap[p.dataKey] || p.dataKey}
                              </span>
                              <span className="ml-auto font-semibold tabular-nums">
                                {p.value} ms
                              </span>
                            </div>
                          ))}
                        </div>
                      );
                    }) as ContentType<ValueType, NameType>
                  }
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
          <div className="flex flex-wrap gap-4 px-4 pb-4 text-xs">
            <LegendItem color="#ff832b" label="STT" />
            <LegendItem color="#1192e8" label="EOS" />
            <LegendItem color="var(--cds-interactive, #0f62fe)" label="TTS" />
            <LegendItem color="#24a148" label="Agent" />
          </div>
        </DashboardWidget>

        <DashboardWidget
          title="Usage totals"
          isLoading={loading}
          skeletonVariant="metric-list"
        >
          <WidgetHeroMetric
            label="Tokens"
            value={hasDashboard ? totalTokens : DASHBOARD_UNAVAILABLE_VALUE}
            caption={
              hasDashboard
                ? `${totalMessages.toLocaleString()} messages processed`
                : emptyDashboardCaption
            }
          />
          <WidgetList
            rows={[
              {
                label: 'STT duration',
                value: hasDashboard
                  ? `${Math.round(totalSttDurationSec).toLocaleString()} s`
                  : DASHBOARD_UNAVAILABLE_VALUE,
              },
              {
                label: 'TTS duration',
                value: hasDashboard
                  ? `${Math.round(totalTtsDurationSec).toLocaleString()} s`
                  : DASHBOARD_UNAVAILABLE_VALUE,
              },
              {
                label: 'Total duration',
                value: hasDashboard
                  ? `${Math.round(totalDuration).toLocaleString()} s`
                  : DASHBOARD_UNAVAILABLE_VALUE,
              },
            ]}
          />
        </DashboardWidget>

        <DashboardWidget
          title="Sources"
          isLoading={loading}
          skeletonVariant="donut"
        >
          <DonutContent
            data={sourceData}
            dataKey="count"
            nameKey="name"
            total={totalMessages}
          />
        </DashboardWidget>

        <DashboardWidget
          title="Languages"
          isLoading={loading}
          skeletonVariant="progress-list"
        >
          <LanguageContent data={languageData} />
        </DashboardWidget>

        <DashboardWidget
          title="Reliability"
          isLoading={loading}
          skeletonVariant="metric-list"
        >
          <WidgetHeroMetric
            label="Completed sessions"
            value={
              hasDashboard
                ? completedConversations
                : DASHBOARD_UNAVAILABLE_VALUE
            }
            caption={
              hasDashboard
                ? `${failedConversations.toLocaleString()} failed sessions`
                : emptyDashboardCaption
            }
          />
          <WidgetList
            rows={[
              {
                label: 'Completed',
                value: hasDashboard
                  ? completedConversations.toLocaleString()
                  : DASHBOARD_UNAVAILABLE_VALUE,
              },
              {
                label: 'Active',
                value: hasDashboard
                  ? activeConversations.toLocaleString()
                  : DASHBOARD_UNAVAILABLE_VALUE,
              },
              {
                label: 'Sessions tracked',
                value: hasDashboard
                  ? totalSessions.toLocaleString()
                  : DASHBOARD_UNAVAILABLE_VALUE,
              },
            ]}
          />
        </DashboardWidget>

        <DashboardWidget
          title="Message activity"
          size="large"
          isLoading={loading}
          skeletonVariant="bar-chart"
          bodyClassName="pt-4"
        >
          <p className="mb-3 text-xs text-gray-500 dark:text-gray-400">
            Messages over selected range
          </p>
          <div className="h-[206px]">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart
                data={activeSessionsData}
                margin={{ top: 0, right: 16, left: 0, bottom: 0 }}
              >
                <YAxis
                  dataKey="total"
                  tickLine={false}
                  axisLine={false}
                  tick={{ fontSize: 11, fill: '#9ca3af' }}
                  width={36}
                />
                <XAxis
                  dataKey="dateHour"
                  tickLine={false}
                  axisLine={false}
                  tick={{ fontSize: 11, fill: '#9ca3af' }}
                  interval="preserveStartEnd"
                />
                <Tooltip
                  cursor={{
                    fill: 'var(--cds-interactive, #0f62fe)',
                    fillOpacity: 0.08,
                  }}
                  content={
                    (({ active, payload }) => {
                      if (!active || !payload?.length) return null;
                      return (
                        <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 shadow-lg px-3 py-2.5 text-sm min-w-[140px]">
                          <p className="text-gray-400 text-xs mb-1.5">
                            {payload[0]?.payload?.label}
                          </p>
                          <div className="flex items-center gap-2">
                            <div
                              className="w-2 h-2"
                              style={{
                                backgroundColor:
                                  'var(--cds-interactive, #0f62fe)',
                              }}
                            />
                            <span className="text-gray-600 dark:text-gray-300">
                              Messages
                            </span>
                            <span className="ml-auto font-semibold tabular-nums">
                              {payload[0]?.value}
                            </span>
                          </div>
                        </div>
                      );
                    }) as ContentType<ValueType, NameType>
                  }
                />
                <Bar
                  dataKey="total"
                  fill="var(--cds-interactive, #0f62fe)"
                  fillOpacity={0.9}
                  maxBarSize={32}
                />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </DashboardWidget>
      </div>
    </div>
  );
};

const DashboardWidget: FC<{
  title: string;
  size?: 'small' | 'large';
  action?: ReactNode;
  className?: string;
  bodyClassName?: string;
  isLoading?: boolean;
  skeletonVariant?: DashboardWidgetSkeletonVariant;
  children: ReactNode;
}> = ({
  title,
  size = 'small',
  action,
  className,
  bodyClassName,
  isLoading = false,
  skeletonVariant = 'metric-list',
  children,
}) => (
  <Tile
    className={cn(
      '!rounded-none !p-0 !bg-white dark:!bg-[#262626] border border-gray-200 dark:border-gray-800 h-[310px]',
      size === 'large' && 'xl:col-span-2',
      className,
    )}
  >
    <div className="flex h-12 items-center justify-between border-b border-gray-200 px-4 dark:border-gray-800">
      <div className="min-w-0">
        <h3 className="truncate text-sm font-semibold text-gray-900 dark:text-gray-100">
          {title}
        </h3>
      </div>
      {action && <div className="ml-4 shrink-0">{action}</div>}
    </div>
    <div className={cn('h-[262px] p-6', bodyClassName)}>
      {isLoading ? (
        <DashboardWidgetSkeleton variant={skeletonVariant} />
      ) : (
        children
      )}
    </div>
  </Tile>
);

const KpiTile: FC<{
  title: string;
  label: string;
  value: number | string;
  unit?: string;
  caption?: string;
  action?: ReactNode;
  isLoading?: boolean;
}> = ({ title, label, value, unit, caption, action, isLoading = false }) => (
  <Tile className="!rounded-none !p-0 !bg-white dark:!bg-[#262626] border border-gray-200 dark:border-gray-800 h-[156px]">
    <div className="flex h-10 items-center justify-between border-b border-gray-200 px-4 dark:border-gray-800">
      <h3 className="truncate text-sm font-semibold text-gray-900 dark:text-gray-100">
        {title}
      </h3>
      {action && <div className="ml-3 shrink-0">{action}</div>}
    </div>
    <div className="flex h-[116px] flex-col justify-between p-4">
      {isLoading ? (
        <KpiTileSkeleton />
      ) : (
        <>
          <div>
            <p className="truncate text-xs text-gray-500 dark:text-gray-400">
              {label}
            </p>
            <div className="mt-2 flex items-baseline gap-1">
              <span className="text-4xl font-light leading-none tabular-nums text-gray-900 dark:text-gray-100">
                {typeof value === 'number' ? value.toLocaleString() : value}
              </span>
              {unit && (
                <span className="text-sm text-gray-500 dark:text-gray-400">
                  {unit}
                </span>
              )}
            </div>
          </div>
          {caption && (
            <p className="truncate text-xs text-gray-500 dark:text-gray-400">
              {caption}
            </p>
          )}
        </>
      )}
    </div>
  </Tile>
);

const KpiTileSkeleton = () => (
  <>
    <div>
      <SkeletonText width="44%" className="!mb-0" />
      <div className="mt-3 flex items-end gap-2">
        <SkeletonText heading width="96px" className="!mb-0" />
        <SkeletonPlaceholder className="!h-4 !w-8" />
      </div>
    </div>
    <SkeletonText width="78%" className="!mb-0" />
  </>
);

const DashboardWidgetSkeleton: FC<{
  variant: DashboardWidgetSkeletonVariant;
}> = ({ variant }) => {
  if (variant === 'latency-chart') {
    return (
      <div className="flex h-full flex-col">
        <div className="grid grid-cols-2 gap-x-6 gap-y-4 md:grid-cols-5">
          {Array.from({ length: 5 }).map((_, index) => (
            <div key={index}>
              <SkeletonText width="52px" className="!mb-2" />
              <SkeletonText heading width="56px" className="!mb-0" />
            </div>
          ))}
        </div>
        <SkeletonPlaceholder className="!mt-5 !h-[126px] !w-full" />
        <div className="mt-4 flex flex-wrap gap-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <SkeletonPlaceholder key={index} className="!h-3 !w-12" />
          ))}
        </div>
      </div>
    );
  }

  if (variant === 'donut') {
    return (
      <div className="flex h-full flex-col">
        <div className="flex h-[140px] items-center justify-center">
          <SkeletonPlaceholder className="!h-[116px] !w-[116px] !rounded-full" />
        </div>
        <div className="mt-4 space-y-2">
          {Array.from({ length: 4 }).map((_, index) => (
            <div key={index} className="flex items-center gap-2">
              <SkeletonPlaceholder className="!h-2.5 !w-2.5 shrink-0" />
              <SkeletonText width="100%" className="!mb-0 flex-1" />
              <SkeletonPlaceholder className="!h-3 !w-10 shrink-0" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (variant === 'progress-list') {
    return (
      <div className="space-y-4">
        {Array.from({ length: 5 }).map((_, index) => (
          <div key={index}>
            <div className="mb-2 flex items-center justify-between gap-4">
              <SkeletonText width="96px" className="!mb-0" />
              <SkeletonPlaceholder className="!h-4 !w-10 shrink-0" />
            </div>
            <SkeletonPlaceholder className="!h-2 !w-full" />
            <SkeletonText width="120px" className="!mt-2 !mb-0" />
          </div>
        ))}
      </div>
    );
  }

  if (variant === 'bar-chart') {
    return (
      <div className="flex h-full flex-col">
        <SkeletonText width="160px" className="!mb-3" />
        <SkeletonPlaceholder className="!h-[206px] !w-full" />
      </div>
    );
  }

  return (
    <div>
      <SkeletonText width="44%" className="!mb-2" />
      <SkeletonText heading width="112px" className="!mb-2" />
      <SkeletonText width="72%" className="!mb-0" />
      <div className="mt-5 divide-y divide-gray-200 border-t border-gray-200 dark:divide-gray-800 dark:border-gray-800">
        {Array.from({ length: 3 }).map((_, index) => (
          <div
            key={index}
            className="flex items-center justify-between gap-4 py-2.5"
          >
            <SkeletonText width="92px" className="!mb-0" />
            <SkeletonPlaceholder className="!h-4 !w-14 shrink-0" />
          </div>
        ))}
      </div>
    </div>
  );
};

const WidgetHeroMetric: FC<{
  label: string;
  value: number | string;
  unit?: string;
  caption?: string;
}> = ({ label, value, unit, caption }) => (
  <div>
    <p className="text-xs text-gray-500 dark:text-gray-400">{label}</p>
    <div className="mt-2 flex items-baseline gap-1">
      <span className="text-4xl font-light leading-none tabular-nums text-gray-900 dark:text-gray-100">
        {typeof value === 'number' ? value.toLocaleString() : value}
      </span>
      {unit && (
        <span className="text-sm text-gray-500 dark:text-gray-400">{unit}</span>
      )}
    </div>
    {caption && (
      <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">{caption}</p>
    )}
  </div>
);

const InlineMetric: FC<{
  label: string;
  value: number | string;
  unit?: string;
}> = ({ label, value, unit }) => (
  <div>
    <p className="text-xs text-gray-500 dark:text-gray-400">{label}</p>
    <div className="mt-1 flex items-baseline gap-1">
      <span className="text-2xl font-light leading-none tabular-nums text-gray-900 dark:text-gray-100">
        {typeof value === 'number' ? value.toLocaleString() : value}
      </span>
      {unit && (
        <span className="text-xs text-gray-500 dark:text-gray-400">{unit}</span>
      )}
    </div>
  </div>
);

const WidgetList: FC<{
  rows: Array<{ label: string; value: ReactNode }>;
}> = ({ rows }) => (
  <div className="mt-5 divide-y divide-gray-200 border-t border-gray-200 dark:divide-gray-800 dark:border-gray-800">
    {rows.map(row => (
      <div
        key={row.label}
        className="flex items-center justify-between gap-4 py-2.5"
      >
        <span className="min-w-0 truncate text-sm text-gray-600 dark:text-gray-300">
          {row.label}
        </span>
        <span className="shrink-0 text-sm font-semibold tabular-nums text-gray-900 dark:text-gray-100">
          {row.value}
        </span>
      </div>
    ))}
  </div>
);

const LegendItem: FC<{ color: string; label: string }> = ({ color, label }) => (
  <div className="flex items-center gap-1.5">
    <div className="h-0.5 w-3" style={{ backgroundColor: color }} />
    <span>{label}</span>
  </div>
);

// ─── Donut chart content ────────────────────────────────────────────────────

const DonutContent: FC<{
  data: DistributionData[];
  dataKey: string;
  nameKey: keyof DistributionData;
  total: number;
}> = ({ data, dataKey, nameKey, total }) => {
  if (data.length === 0)
    return (
      <div className="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400">
        No source data
      </div>
    );

  return (
    <>
      <div className="relative h-[140px]">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={data}
              cx="50%"
              cy="50%"
              labelLine={false}
              outerRadius={58}
              innerRadius={36}
              dataKey={dataKey}
              nameKey={nameKey}
              stroke="none"
            >
              {data.map((_, i) => (
                <Cell key={i} fill={CHART_COLORS[i % CHART_COLORS.length]} />
              ))}
            </Pie>
            <Tooltip
              content={
                (({ active, payload }) => {
                  if (!active || !payload?.length) return null;
                  const item = payload[0];
                  return (
                    <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 shadow-lg px-3 py-2 text-sm">
                      <div className="flex items-center gap-2">
                        <div
                          className="w-2.5 h-2.5 shrink-0"
                          style={{ backgroundColor: item.color || '#6366f1' }}
                        />
                        <span className="capitalize">
                          {item.name || 'Unknown'}
                        </span>
                        <span className="ml-3 font-semibold">{item.value}</span>
                      </div>
                    </div>
                  );
                }) as ContentType<ValueType, NameType>
              }
            />
          </PieChart>
        </ResponsiveContainer>
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
          <div className="text-center">
            <p className="text-lg font-bold tabular-nums">{total}</p>
            <p className="text-[10px] text-gray-400 uppercase">Total</p>
          </div>
        </div>
      </div>
      <div className="mt-4 space-y-2">
        {data.slice(0, 4).map((item, i) => (
          <div
            key={item[nameKey] || i}
            className="flex items-center gap-2 text-xs"
          >
            <div
              className="w-2.5 h-2.5 shrink-0"
              style={{ backgroundColor: CHART_COLORS[i % CHART_COLORS.length] }}
            />
            <span className="text-gray-600 dark:text-gray-400 truncate flex-1 capitalize">
              {item[nameKey] || 'Unknown'}
            </span>
            <span className="font-semibold tabular-nums">
              {item.percentage}%
            </span>
            <span className="text-gray-400 tabular-nums">({item.count})</span>
          </div>
        ))}
      </div>
    </>
  );
};

const LanguageContent: FC<{
  data: DistributionData[];
}> = ({ data }) => {
  if (data.length === 0)
    return (
      <div className="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400">
        No language data
      </div>
    );

  return (
    <div className="space-y-4">
      {data.slice(0, 5).map((item, i) => (
        <div key={item.name || i}>
          <div className="mb-1.5 flex items-center justify-between gap-4 text-sm">
            <span className="truncate capitalize text-gray-700 dark:text-gray-200">
              {item.name || 'Unknown'}
            </span>
            <span className="shrink-0 font-semibold tabular-nums text-gray-900 dark:text-gray-100">
              {item.percentage}%
            </span>
          </div>
          <div className="h-2 bg-gray-200 dark:bg-gray-800">
            <div
              className="h-2"
              style={{
                width: `${item.percentage}%`,
                backgroundColor: CHART_COLORS[i % CHART_COLORS.length],
              }}
            />
          </div>
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {item.count.toLocaleString()} messages
          </p>
        </div>
      ))}
    </div>
  );
};
