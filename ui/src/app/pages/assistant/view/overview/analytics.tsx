import { AssistantConversationMessage } from '@rapidaai/react';
import { toDate } from '@/styles/media';
import {
  getStatusMetric,
  getTimeTakenMetric,
  getTotalTokenMetric,
} from '@/utils/metadata';
import {
  XAxis,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  Legend,
  Bar,
  BarChart,
} from 'recharts';
import {
  NameType,
  ValueType,
} from 'recharts/types/component/DefaultTooltipContent';
import { ContentType } from 'recharts/types/component/Tooltip';
import {
  Activity,
  BarChart3,
  BookOpen,
  Cpu,
  Globe2,
  PhoneCall,
  Waves,
  Webhook,
} from 'lucide-react';

interface AnalyticsProps {
  data: AssistantConversationMessage[];
}

export const Analytics = ({ data }: AnalyticsProps) => {
  const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8'];
  // Group messages by conversation
  const conversationsMap = data.reduce((acc, message) => {
    const conversationId = message.getAssistantconversationid();
    if (!acc.has(conversationId)) {
      acc.set(conversationId, []);
    }
    acc.get(conversationId)!.push(message);
    return acc;
  }, new Map<string, AssistantConversationMessage[]>());

  const conversations = Array.from(conversationsMap.values());

  // Calculate total sessions (unique conversations)
  const totalSessions = conversations.length;

  // Calculate total messages
  const totalMessages = data.length;

  // Calculate average duration
  const avgDuration =
    conversations.reduce((sum, conversation) => {
      const sortedMessages = conversation.sort(
        (a, b) =>
          a.getCreateddate()!.getSeconds() - b.getCreateddate()!.getSeconds(),
      );
      const duration =
        sortedMessages[sortedMessages.length - 1]
          .getCreateddate()!
          .getSeconds() - sortedMessages[0].getCreateddate()!.getSeconds();
      return sum + duration;
    }, 0) / totalSessions;

  // Calculate average latency
  const avgLatency =
    data.reduce(
      (sum, message) => sum + getTimeTakenMetric(message.getMetricsList()),
      0,
    ) / totalMessages;

  // Active sessions simulation (hourly distribution)
  const uniqueSources = Array.from(new Set(data.map(item => item.getSource())));

  const activeSessionsData = Array.from({ length: 24 }, (_, hour) => {
    const itemsAtHour = data.filter(item => {
      const itemHour = toDate(item.getCreateddate()!).getHours();
      return itemHour === hour;
    });

    const normalizedSourceCounts = uniqueSources.reduce(
      (acc, source) => {
        acc[source] = itemsAtHour.filter(
          item => item.getSource() === source,
        ).length;
        return acc;
      },
      {} as Record<string, number>,
    );

    return {
      hour: `${hour.toString().padStart(2, '0')}:00`,
      total: itemsAtHour.length,
      ...normalizedSourceCounts,
      latency: Math.round(
        itemsAtHour.reduce(
          (sum, item) =>
            sum + getTimeTakenMetric(item.getMetricsList()) / 1000000,
          0,
        ) / Math.max(1, itemsAtHour.length),
      ),
    };
  });

  // Language distribution (assuming language is stored in metadata)
  const languageData = Object.entries(
    data.reduce(
      (acc, item) => {
        const language =
          item
            .getMetadataList()
            .find(m => m.getKey() === 'language')
            ?.getValue() || 'Unknown';
        acc[language] = (acc[language] || 0) + 1;
        return acc;
      },
      {} as Record<string, number>,
    ),
  ).map(([lang, count]) => ({
    language: lang,
    count,
    percentage: ((count / data.length) * 100).toFixed(1),
  }));

  // Source distribution for pie chart
  const sourceData = Object.entries(
    data.reduce(
      (acc, item) => {
        const source = item.getSource();
        acc[source] = (acc[source] || 0) + 1;
        return acc;
      },
      {} as Record<string, number>,
    ),
  ).map(([source, count]) => ({
    source,
    count,
    percentage: ((count / data.length) * 100).toFixed(1),
  }));
  const metricsData = [
    {
      title: 'Total Sessions',
      value: totalSessions.toLocaleString(),
      trend: `${((totalSessions / data.length) * 100).toFixed(1)}% of total interactions`,
    },
    {
      title: 'Total Messages',
      value: totalMessages.toLocaleString(),
      trend: `${(totalMessages / totalSessions).toFixed(1)} messages per session`,
    },
    {
      title: 'Avg Duration',
      value: `${Math.round(avgDuration)}s`,
      trend: `${(avgDuration / 60).toFixed(1)} minutes per session`,
    },
    {
      title: 'Avg Latency',
      value: `${Math.round(avgLatency / 1000000)}ms`,
      trend:
        avgLatency / 1000000 > 2000
          ? 'High latency, optimization needed'
          : 'Good response time',
    },
    {
      title: 'Token Efficiency',
      value: `${(data.reduce((sum, item) => sum + getTotalTokenMetric(item.getMetricsList()), 0) / totalMessages).toFixed(1)}`,
      trend: 'Tokens per message',
    },
    {
      title: 'Success Rate',
      value: `${((data.filter(item => getStatusMetric(item.getMetricsList()) === 'SUCCESS').length / totalMessages) * 100).toFixed(1)}%`,
      trend: 'Completed interactions',
    },
  ];
  const sources = Array.from(new Set(data.map(item => item.getSource())));

  return (
    <div className="w-full">
      {/* <section className=""> */}
      <section className="bg-white dark:bg-gray-950 border-y relative grid grid-cols-1  md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 ">
        {metricsData.map((metric, index) => (
          <div
            key={index}
            className="grid grid-rows-[1fr_auto] md:border-r md:border-gray-200 dark:border-gray-800"
          >
            <div className="grid grid-cols-1 items-center">
              <div className="px-4 py-2 sm:px-2">
                <div className="text-2xl font-semibold mt-4">
                  {metric.value}
                </div>
                <div className="flex items-center gap-2 mt-4">
                  <h3 className="text-base/7 font-semibold">
                    <div className="absolute inset-0" />
                    {metric.title}
                  </h3>
                </div>
              </div>
            </div>
            <div className="border-t border-gray-200 dark:border-gray-800 px-4 py-2 max-md:border-y sm:px-2">
              <p className="text-sm/6 text-gray-600 dark:text-gray-400 opacity-70">
                {metric.trend}
              </p>
            </div>
          </div>
        ))}
      </section>

      <div className="m-4 rounded-xl border bg-white dark:bg-gray-950">
        <div className="border-b px-4 py-4 opacity-80">
          <h2 className="text-base font-semibold">
            Session Activity & Hourly Latency Metrics
          </h2>
          <p className="max-w-2xl mt-1 text-sm text-gray-600 dark:text-gray-400">
            Analyze concurrent session trends and hourly latency data for
            efficient capacity management.
          </p>
        </div>
        <div className="h-[300px]">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart
              data={activeSessionsData}
              margin={{ top: 5, right: 0, left: 0, bottom: 5 }}
            >
              <XAxis
                dataKey="hour"
                axisLine={false}
                tickLine={false}
                tick={{ fontSize: 12 }}
              />
              <Tooltip
                content={
                  (({ active, payload, label }) => {
                    if (active && payload && payload.length) {
                      return (
                        <div className="bg-white dark:bg-gray-950 border-[0.5px] rounded-[2px] px-0 py-0 w-64">
                          <div className="divide-y text-sm dark:text-gray-400 text-gray-700">
                            <div className="px-3 py-3 space-y-1.5">
                              {payload.map((entry, index) => (
                                <div
                                  className="flex items-center justify-between"
                                  key={index}
                                >
                                  <div className="flex items-center space-x-1.5">
                                    <div
                                      className="w-2 h-2"
                                      style={{
                                        backgroundColor: entry.color || '#ccc',
                                        borderRadius: '2px',
                                      }}
                                    ></div>
                                    <span className="font-medium capitalize">
                                      {entry.name}
                                    </span>
                                  </div>
                                  <span className="font-medium">
                                    {entry.value}
                                  </span>
                                </div>
                              ))}
                            </div>
                            <div className="px-3 py-2">Hour : {label}</div>
                          </div>
                        </div>
                      );
                    }
                    return null;
                  }) as ContentType<ValueType, NameType>
                }
              />
              <Legend />
              {sources.map((source, index) => (
                <Bar
                  key={`bar-${source}`}
                  dataKey={source}
                  stackId="a"
                  fill={COLORS[index % COLORS.length]}
                />
              ))}
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 px-4">
        <div className="rounded-xl border bg-white dark:bg-gray-950">
          <div className="border-b px-4 py-4 opacity-80">
            <h2 className="text-base font-semibold">Source Distribution</h2>
            <p className="max-w-2xl text-sm mt-1 text-gray-600 dark:text-gray-400">
              Request distribution across different sources
            </p>
          </div>

          <div className="p-4 pt-0">
            <div className="h-[300px]">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={sourceData}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={({ source, percentage }) =>
                      `${source} (${percentage}%)`
                    }
                    outerRadius={80}
                    innerRadius={40} // Added for donut chart
                    fill="#8884d8"
                    dataKey="count"
                    stroke="none"
                  >
                    {sourceData.map((entry, index) => (
                      <Cell
                        key={`cell-${index}`}
                        fill={COLORS[index % COLORS.length]}
                      />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </div>
        </div>
        <div className="rounded-xl border bg-white dark:bg-gray-950">
          <div className="border-b px-4 py-4 opacity-80">
            <h2 className="text-base font-semibold">Language Distribution</h2>
            <p className="max-w-2xl text-sm mt-1 text-gray-600 dark:text-gray-400">
              User language preferences for localization planning
            </p>
          </div>

          <div className="p-4 pt-0">
            <div className="h-[300px]">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={languageData}
                    cx="50%"
                    cy="50%"
                    stroke="none"
                    labelLine={false}
                    label={({ language, percentage }) =>
                      `${language} (${percentage}%)`
                    }
                    outerRadius={80}
                    innerRadius={40} // Added for donut chart
                    fill="#8884d8"
                    dataKey="count"
                  >
                    {languageData.map((entry, index) => (
                      <Cell
                        key={`cell-${index}`}
                        fill={COLORS[index % COLORS.length]}
                      />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
