export type TimelineDocument = {
  id: string;
  kind: 'log' | 'event' | 'metric';
  name: string;
  component?: string;
  category: string;
  level: string;
  outcome: string;
  title: string;
  projectId: number | string;
  organizationId: number | string;
  scope: string;
  assistantId: number | string;
  assistantConversationId: number | string;
  messageId?: string;
  messageRole?: string;
  traceId?: string;
  contextId: string;
  occurredAt: string;
  receivedAt: string;
  durationMs?: number;
  attributes?: Record<string, string>;
  data?: Record<string, unknown>;
};

export type TimelineItem = TimelineDocument & {
  endMs: number;
  offsetPct: number;
  startMs: number;
  widthPct: number;
};

export type TimelineGroup = {
  contextId: string;
  durationMs: number;
  endMs: number;
  items: TimelineItem[];
  startMs: number;
  title: string;
};

export type TraceSummary = {
  assistantConversationId: number | string;
  assistantId: number | string;
  components: string[];
  contextId: string;
  durationMs: number;
  endMs: number;
  failureCount: number;
  level: string;
  outcome: string;
  spanCount: number;
  startMs: number;
  title: string;
};

export type MetricSummary = {
  averageDurationMs: number;
  component: string;
  count: number;
  failureCount: number;
  p95DurationMs: number;
  slowestContextId: string;
  slowestDurationMs: number;
};
