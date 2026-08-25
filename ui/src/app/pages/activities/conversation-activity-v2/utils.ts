import type {
  MetricSummary,
  TimelineDocument,
  TimelineGroup,
  TimelineItem,
  TraceSummary,
} from './types';
import type {
  ObservabilityEventRecord,
  ObservabilityLogRecord,
  ObservabilityMetricRecord,
  ObservabilityRecord,
} from '@rapidaai/react';
import { toHumanReadableDateTimeFromDate } from '@/utils/date';

const MIN_VISIBLE_WIDTH_PCT = 0.75;

export const COMPONENT_COLORS: Record<string, string> = {
  call: '#0f62fe',
  telephony: '#0f62fe',
  conversation: '#8a3ffc',
  turn: '#4589ff',
  stt: '#198038',
  agent: '#f1c21b',
  tool: '#ff7eb6',
  tts: '#fa4d56',
  vad: '#007d79',
  eos: '#007d79',
  denoise: '#1192e8',
  webhook: '#ee538b',
  analysis: '#a56eff',
  authentication: '#009d9a',
  recording: '#6fdc8c',
  storage: '#d2a106',
  sip: '#ff832b',
  webrtc: '#33b1ff',
  usage: '#6f6f6f',
  log: '#6f6f6f',
  metric: '#6f6f6f',
  metadata: '#6f6f6f',
  error: '#da1e28',
};

const METRIC_COMPONENT_PREFIXES: readonly string[] = [
  'call',
  'sip',
  'transfer',
  'rtp',
  'webrtc',
  'telephony',
  'stt',
  'tts',
  'vad',
  'eos',
  'denoise',
  'agent',
  'storage',
  'analysis',
  'authentication',
  'recording',
  'knowledge',
];

const getTimeMs = (value: string): number => {
  const parsed = new Date(value).getTime();
  return Number.isFinite(parsed) ? parsed : 0;
};

const timestampToIso = (
  timestamp: { toDate: () => Date } | undefined,
): string => {
  const date = timestamp?.toDate();
  return date && Number.isFinite(date.getTime())
    ? date.toISOString()
    : new Date(0).toISOString();
};

const mapToObject = (map: {
  toArray: () => Array<[string, string]>;
}): Record<string, string> => Object.fromEntries(map.toArray());

const firstPresent = (...values: Array<string | undefined>): string =>
  values.find(value => value && value.trim() !== '') || '';

const inferComponentFromName = (name: string, fallback: string): string => {
  const normalizedName = name.trim().toLowerCase();
  if (!normalizedName) return fallback;

  if (normalizedName === 'user_turn' || normalizedName === 'assistant_turn') {
    return 'turn';
  }

  const dottedPrefix = normalizedName.split('.')[0];
  if (METRIC_COMPONENT_PREFIXES.includes(dottedPrefix)) {
    return dottedPrefix;
  }

  const underscoredPrefix = normalizedName.split('_')[0];
  if (METRIC_COMPONENT_PREFIXES.includes(underscoredPrefix)) {
    return underscoredPrefix;
  }

  return fallback;
};

const parseDurationMs = (value: string): number | undefined => {
  if (value.trim() === '') return undefined;
  const durationMs = Number(value);
  return Number.isFinite(durationMs) && durationMs >= 0
    ? Math.round(durationMs)
    : undefined;
};

const inferOutcome = ({
  attributes,
  level,
  name,
}: {
  attributes: Record<string, string>;
  level?: string;
  name?: string;
}): string => {
  const normalized = [
    level,
    name,
    attributes.status,
    attributes.outcome,
    attributes.result,
    attributes.error,
  ]
    .join(' ')
    .toLowerCase();

  if (
    normalized.includes('error') ||
    normalized.includes('fail') ||
    normalized.includes('exception')
  ) {
    return 'failure';
  }

  if (
    normalized.includes('success') ||
    normalized.includes('complete') ||
    normalized.includes('ok')
  ) {
    return 'success';
  }

  return 'unknown';
};

const getDurationFromAttributes = (
  attributes: Record<string, string>,
): number | undefined => {
  const candidate = firstPresent(
    attributes.durationMs,
    attributes.duration_ms,
    attributes.latencyMs,
    attributes.latency_ms,
    attributes.elapsedMs,
    attributes.elapsed_ms,
  );
  return parseDurationMs(candidate);
};

const isDurationMetricName = (name: string): boolean => {
  const normalizedName = name.trim().toLowerCase();
  if (!normalizedName) return false;

  return (
    ['duration', 'duration_ms', 'latency', 'latency_ms', 'elapsed_ms'].includes(
      normalizedName,
    ) ||
    normalizedName.endsWith('_duration_ms') ||
    normalizedName.endsWith('.duration_ms') ||
    normalizedName.endsWith('_latency_ms') ||
    normalizedName.endsWith('.latency_ms') ||
    normalizedName.endsWith('_elapsed_ms') ||
    normalizedName.endsWith('.elapsed_ms')
  );
};

const getDurationFromMetric = (
  metric: ObservabilityMetricRecord,
): number | undefined => {
  if (!isDurationMetricName(metric.getName())) return undefined;
  return parseDurationMs(metric.getValue());
};

const getScopeAttributes = (
  record:
    | ObservabilityLogRecord
    | ObservabilityEventRecord
    | ObservabilityMetricRecord,
): Record<string, string> => mapToObject(record.getScopeattributesMap());

const getContext = (
  record:
    | ObservabilityLogRecord
    | ObservabilityEventRecord
    | ObservabilityMetricRecord,
): Record<string, string> => mapToObject(record.getContextMap());

const getTraceId = (
  context: Record<string, string>,
  attributes: Record<string, string>,
): string =>
  firstPresent(
    context.traceId,
    context.traceID,
    context.trace_id,
    attributes.traceId,
    attributes.traceID,
    attributes.trace_id,
  );

const getScopeContext = (scopeAttributes: Record<string, string>) => ({
  assistantConversationId: firstPresent(
    scopeAttributes.assistantConversationId,
    scopeAttributes.conversationId,
  ),
  assistantId: scopeAttributes.assistantId || '',
  contextId: firstPresent(
    scopeAttributes.contextId,
    scopeAttributes.messageId,
    scopeAttributes.assistantConversationId,
    scopeAttributes.conversationId,
  ),
  messageId: scopeAttributes.messageId || '',
  messageRole: scopeAttributes.messageRole || '',
});

const buildBaseDocument = ({
  id,
  kind,
  name,
  title,
  level,
  component,
  organizationId,
  projectId,
  assistantId,
  assistantConversationId,
  scope,
  messageId,
  messageRole,
  traceId,
  contextId,
  occurredAt,
  attributes,
  data,
  durationMs,
}: {
  id: string;
  kind: TimelineDocument['kind'];
  name: string;
  title: string;
  level: string;
  component: string;
  organizationId: string;
  projectId: string;
  assistantId: string;
  assistantConversationId: string;
  scope: string;
  messageId: string;
  messageRole: string;
  traceId: string;
  contextId: string;
  occurredAt: string;
  attributes: Record<string, string>;
  data: Record<string, unknown>;
  durationMs?: number;
}): TimelineDocument => ({
  id,
  kind,
  name,
  component: component || undefined,
  category: component || name.split('.')[0] || kind,
  level,
  outcome: inferOutcome({ attributes, level, name }),
  title,
  organizationId,
  projectId,
  assistantId,
  assistantConversationId,
  scope: scope || 'unknown',
  messageId: messageId || undefined,
  messageRole: messageRole || undefined,
  traceId: traceId || undefined,
  contextId:
    contextId ||
    messageId ||
    (assistantConversationId
      ? `conversation-${assistantConversationId}`
      : assistantId
        ? `assistant-${assistantId}`
        : `project-${projectId || 'unknown'}`),
  occurredAt,
  receivedAt: occurredAt,
  durationMs,
  attributes,
  data,
});

const eventToTimelineDocument = (
  event: ObservabilityEventRecord,
  index: number,
): TimelineDocument => {
  const attributes = mapToObject(event.getAttributesMap());
  const context = getContext(event);
  const scopeAttributes = getScopeAttributes(event);
  const scopeContext = getScopeContext(scopeAttributes);
  const name = event.getEvent() || 'event';
  return buildBaseDocument({
    id: event.getId() || `event-${index}`,
    kind: 'event',
    name,
    title: name,
    level: attributes.level || 'info',
    component:
      event.getComponent() || attributes.component || name.split('.')[0],
    organizationId: event.getOrganizationid(),
    projectId: event.getProjectid(),
    assistantId: scopeContext.assistantId,
    assistantConversationId: scopeContext.assistantConversationId,
    scope: event.getScope(),
    messageId: scopeContext.messageId,
    messageRole: scopeContext.messageRole,
    traceId: getTraceId(context, attributes),
    contextId: scopeContext.contextId,
    occurredAt: timestampToIso(event.getOccurredat()),
    attributes,
    data: { context, scopeAttributes },
    durationMs: getDurationFromAttributes(attributes),
  });
};

const metricToTimelineDocument = (
  metric: ObservabilityMetricRecord,
  index: number,
): TimelineDocument => {
  const attributes = mapToObject(metric.getAttributesMap());
  const context = getContext(metric);
  const scopeAttributes = getScopeAttributes(metric);
  const scopeContext = getScopeContext(scopeAttributes);
  const name = metric.getName() || attributes.name || 'metric';
  const metrics = [
    {
      description: metric.getDescription(),
      name,
      value: metric.getValue(),
    },
  ];
  return buildBaseDocument({
    id: metric.getId() || `metric-${scopeContext.contextId || index}`,
    kind: 'metric',
    name,
    title: `Metric: ${name}`,
    level: attributes.level || 'info',
    component:
      attributes.component || inferComponentFromName(name, name.split('.')[0]),
    organizationId: metric.getOrganizationid(),
    projectId: metric.getProjectid(),
    assistantId: scopeContext.assistantId,
    assistantConversationId: scopeContext.assistantConversationId,
    scope: metric.getScope(),
    messageId: scopeContext.messageId,
    messageRole: scopeContext.messageRole,
    traceId: getTraceId(context, attributes),
    contextId: scopeContext.contextId,
    occurredAt: timestampToIso(metric.getOccurredat()),
    attributes,
    data: {
      context,
      description: metric.getDescription(),
      metrics,
      scopeAttributes,
    },
    durationMs:
      getDurationFromMetric(metric) || getDurationFromAttributes(attributes),
  });
};

const logToTimelineDocument = (
  log: ObservabilityLogRecord,
  index: number,
): TimelineDocument => {
  const attributes = mapToObject(log.getAttributesMap());
  const context = getContext(log);
  const scopeAttributes = getScopeAttributes(log);
  const scopeContext = getScopeContext(scopeAttributes);
  const message = log.getMessage() || 'Log record';
  return buildBaseDocument({
    id: log.getId() || `log-${index}`,
    kind: 'log',
    name: message,
    title: message,
    level: log.getLevel() || attributes.level || 'info',
    component: attributes.component || attributes.provider || 'log',
    organizationId: log.getOrganizationid(),
    projectId: log.getProjectid(),
    assistantId: scopeContext.assistantId,
    assistantConversationId: scopeContext.assistantConversationId,
    scope: log.getScope(),
    messageId: scopeContext.messageId,
    messageRole: scopeContext.messageRole,
    traceId: getTraceId(context, attributes),
    contextId: scopeContext.contextId,
    occurredAt: timestampToIso(log.getOccurredat()),
    attributes,
    data: { context, message, scopeAttributes },
    durationMs: getDurationFromAttributes(attributes),
  });
};

export const telemetryRecordToTimelineDocument = (
  record: ObservabilityRecord,
  index: number,
): TimelineDocument | null => {
  const log = record.getLog();
  if (log) return logToTimelineDocument(log, index);

  const event = record.getEvent();
  if (event) return eventToTimelineDocument(event, index);

  const metric = record.getMetric();
  if (metric) return metricToTimelineDocument(metric, index);

  return null;
};

export const getDocumentComponent = (doc: TimelineDocument): string => {
  const attributeComponent =
    doc.attributes?.component || doc.attributes?.provider;
  const fallback = doc.category || doc.name.split('.')[0] || 'conversation';
  return (
    doc.component ||
    attributeComponent ||
    inferComponentFromName(doc.name, fallback)
  );
};

export type TraceFilterSource = 'facet' | 'query';

export type TraceFilterToken = {
  criteriaKey: string;
  fieldKey: string;
  label: string;
  logic: '=' | '>=' | '<=';
  rawText?: string;
  source: TraceFilterSource;
  value: string;
};

type TraceFilterField = {
  criteriaKey: string;
  getDocumentValue?: (doc: TimelineDocument) => string | number | undefined;
  key: string;
  label: string;
  logic?: TraceFilterToken['logic'];
  match?: (
    doc: TimelineDocument,
    value: string,
    logic: TraceFilterToken['logic'],
  ) => boolean;
};

export type ParsedTraceFilterQuery = {
  filters: TraceFilterToken[];
  freeText: string;
};

export const getTraceFilterValues = (filter: TraceFilterToken): string[] =>
  Array.from(
    new Set(
      filter.value
        .split(',')
        .map(value => value.trim())
        .filter(Boolean),
    ),
  );

export const getTraceFilterRequestValue = (filter: TraceFilterToken): string =>
  getTraceFilterValues(filter).join(',');

const getMetricNames = (doc: TimelineDocument): string[] => {
  const metrics = doc.data?.metrics as
    | Array<{ name?: string; value?: string | number }>
    | undefined;
  return [doc.name, ...(metrics || []).map(metric => metric.name || '')].filter(
    Boolean,
  );
};

const getScopeAttribute = (
  doc: TimelineDocument,
  attributeKey: string,
): string | number | undefined => {
  if (attributeKey === 'assistantId') return doc.assistantId;
  if (attributeKey === 'assistantConversationId') {
    return doc.assistantConversationId;
  }
  if (attributeKey === 'messageId') return doc.messageId;
  if (attributeKey === 'messageRole') return doc.messageRole;

  return (doc.data?.scopeAttributes as Record<string, string> | undefined)?.[
    attributeKey
  ];
};

const getDateMs = (value: string): number | null => {
  const ms = new Date(value).getTime();
  return Number.isFinite(ms) ? ms : null;
};

const TRACE_FILTER_FIELD_DEFINITIONS: TraceFilterField[] = [
  {
    key: 'id',
    label: 'Record ID',
    criteriaKey: 'id',
    getDocumentValue: doc => doc.id,
  },
  {
    key: 'trace',
    label: 'traceID',
    criteriaKey: 'traceId',
    getDocumentValue: doc => doc.traceId,
  },
  {
    key: 'kind',
    label: 'Record type',
    criteriaKey: 'kind',
    getDocumentValue: doc => doc.kind,
  },
  {
    key: 'level',
    label: 'Level',
    criteriaKey: 'level',
    getDocumentValue: doc => doc.level,
  },
  {
    key: 'scope',
    label: 'Record scope',
    criteriaKey: 'scope',
    getDocumentValue: doc => doc.scope,
  },
  {
    key: 'assistant',
    label: 'Assistant ID',
    criteriaKey: 'assistantId',
    getDocumentValue: doc => doc.assistantId,
  },
  {
    key: 'conversation',
    label: 'Conversation ID',
    criteriaKey: 'assistantConversationId',
    getDocumentValue: doc => doc.assistantConversationId,
  },
  {
    key: 'message',
    label: 'Message ID',
    criteriaKey: 'messageId',
    getDocumentValue: doc => doc.messageId,
  },
  {
    key: 'role',
    label: 'Message role',
    criteriaKey: 'messageRole',
    getDocumentValue: doc => doc.messageRole,
  },
  {
    key: 'event',
    label: 'Event',
    criteriaKey: 'event',
    getDocumentValue: doc => doc.name,
  },
  {
    key: 'name',
    label: 'Name',
    criteriaKey: 'name',
    getDocumentValue: doc => doc.name,
  },
  {
    key: 'component',
    label: 'Component',
    criteriaKey: 'component',
    getDocumentValue: getDocumentComponent,
  },
  {
    key: 'metric',
    label: 'Metric name',
    criteriaKey: 'name',
    match: (doc, value) => getMetricNames(doc).includes(value),
  },
  {
    key: 'timestamp',
    label: 'Timestamp',
    criteriaKey: 'timestamp',
    logic: '>=',
    match: (doc, value, logic) => {
      const docMs = getDateMs(doc.occurredAt);
      const valueMs = getDateMs(value);
      if (docMs === null || valueMs === null) return false;
      if (logic === '<=') return docMs <= valueMs;
      if (logic === '=') return docMs >= valueMs && docMs < valueMs + 60 * 1000;
      return docMs >= valueMs;
    },
  },
];

const TRACE_FILTER_FIELD_LOOKUP = TRACE_FILTER_FIELD_DEFINITIONS.reduce(
  (lookup, field) => {
    lookup.set(field.key.toLowerCase(), field);
    return lookup;
  },
  new Map<string, TraceFilterField>(),
);

const getDynamicTraceFilterField = (
  key: string,
): TraceFilterField | undefined => {
  if (key.startsWith('attributes.')) {
    const attributeKey = key.slice('attributes.'.length);
    return {
      key,
      label: key,
      criteriaKey: key,
      getDocumentValue: doc => doc.attributes?.[attributeKey],
    };
  }

  if (key.startsWith('scopeAttributes.')) {
    const attributeKey = key.slice('scopeAttributes.'.length);
    return {
      key,
      label: key,
      criteriaKey: key,
      getDocumentValue: doc => getScopeAttribute(doc, attributeKey),
    };
  }

  if (key.startsWith('context.')) {
    const contextKey = key.slice('context.'.length);
    return {
      key,
      label: key,
      criteriaKey: key,
      getDocumentValue: doc =>
        (doc.data?.context as Record<string, string> | undefined)?.[contextKey],
    };
  }

  return undefined;
};

export const getTraceFilterField = (
  key: string,
): TraceFilterField | undefined =>
  TRACE_FILTER_FIELD_LOOKUP.get(key.toLowerCase()) ||
  getDynamicTraceFilterField(key);

export const createTraceFilter = (
  key: string,
  value: string | number | undefined,
  source: TraceFilterSource,
  logic?: TraceFilterToken['logic'],
): TraceFilterToken | null => {
  const normalizedValue = String(value ?? '').trim();
  if (!normalizedValue) return null;

  const field = getTraceFilterField(key);
  if (!field) return null;

  return {
    criteriaKey: field.criteriaKey,
    fieldKey: field.key,
    label: field.label,
    logic: logic || field.logic || '=',
    source,
    value: normalizedValue,
  };
};

const unquoteTraceFilterValue = (value: string): string => {
  const trimmed = value.trim();
  if (
    (trimmed.startsWith('"') && trimmed.endsWith('"')) ||
    (trimmed.startsWith("'") && trimmed.endsWith("'"))
  ) {
    return trimmed.slice(1, -1);
  }
  return trimmed;
};

const TRACE_FILTER_PATTERN =
  /(^|\s)([A-Za-z][\w.]*)(?:~(>=|<=|=))?(:|=)("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^\s]+)/g;

export const parseTraceFilterQuery = (
  query: string,
): ParsedTraceFilterQuery => {
  const filters: TraceFilterToken[] = [];
  const freeTextParts: string[] = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = TRACE_FILTER_PATTERN.exec(query)) !== null) {
    const [rawMatch, prefix, key, logic, , rawValue] = match;
    const matchStart = match.index + prefix.length;
    const before = query.slice(lastIndex, matchStart).trim();
    if (before) freeTextParts.push(before);

    const filter = createTraceFilter(
      key,
      unquoteTraceFilterValue(rawValue),
      'query',
      logic as TraceFilterToken['logic'] | undefined,
    );

    if (filter) {
      filters.push({ ...filter, rawText: rawMatch.trim() });
    } else {
      freeTextParts.push(rawMatch.trim());
    }

    lastIndex = match.index + rawMatch.length;
  }

  const tail = query.slice(lastIndex).trim();
  if (tail) freeTextParts.push(tail);

  return {
    filters: dedupeTraceFilters(filters),
    freeText: freeTextParts.join(' ').trim(),
  };
};

export const dedupeTraceFilters = (
  filters: TraceFilterToken[],
): TraceFilterToken[] => {
  const seen = new Set<string>();
  return filters.filter(filter => {
    const key = `${filter.criteriaKey}\u0000${filter.logic}\u0000${filter.value}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
};

export const formatTraceFilterToken = (filter: TraceFilterToken): string => {
  const field = getTraceFilterField(filter.fieldKey);
  const defaultLogic = field?.logic || '=';
  const key =
    filter.logic === defaultLogic
      ? filter.fieldKey
      : `${filter.fieldKey}~${filter.logic}`;
  return `${key}:${filter.value}`;
};

export const matchesTraceFilters = (
  doc: TimelineDocument,
  filters: TraceFilterToken[],
): boolean =>
  filters.every(filter => {
    const field = getTraceFilterField(filter.fieldKey);
    if (!field) return true;
    const filterValues = getTraceFilterValues(filter);
    return filterValues.some(filterValue => {
      if (field.match) return field.match(doc, filterValue, filter.logic);
      const value = field.getDocumentValue?.(doc);
      return String(value ?? '') === filterValue;
    });
  });

export const getDocumentColor = (doc: TimelineDocument): string => {
  if (doc.level === 'error' || doc.outcome === 'failure') {
    return COMPONENT_COLORS.error;
  }

  const component = getDocumentComponent(doc);
  return COMPONENT_COLORS[component] || '#6f6f6f';
};

export const buildTimelineItems = (
  documents: TimelineDocument[],
): TimelineItem[] => {
  if (documents.length === 0) return [];

  const starts = documents.map(doc => getTimeMs(doc.occurredAt));
  const ends = documents.map(doc => {
    const start = getTimeMs(doc.occurredAt);
    return start + Math.max(doc.durationMs || 0, 1);
  });
  const rangeStart = Math.min(...starts);
  const rangeEnd = Math.max(...ends);
  const range = Math.max(rangeEnd - rangeStart, 1);

  return documents
    .map(doc => {
      const startMs = getTimeMs(doc.occurredAt);
      const endMs = startMs + Math.max(doc.durationMs || 0, 1);
      const offsetPct = ((startMs - rangeStart) / range) * 100;
      const widthPct = Math.max(
        ((endMs - startMs) / range) * 100,
        MIN_VISIBLE_WIDTH_PCT,
      );

      return {
        ...doc,
        startMs,
        endMs,
        offsetPct,
        widthPct,
      };
    })
    .sort((a, b) => a.startMs - b.startMs || a.name.localeCompare(b.name));
};

export const groupTimelineItems = (
  documents: TimelineDocument[],
): TimelineGroup[] => {
  const items = buildTimelineItems(documents);
  const groups = new Map<string, TimelineItem[]>();

  items.forEach(item => {
    const key =
      item.contextId || `conversation-${item.assistantConversationId}`;
    const current = groups.get(key) || [];
    current.push(item);
    groups.set(key, current);
  });

  return Array.from(groups.entries())
    .map(([contextId, groupItems], index) => {
      const startMs = Math.min(...groupItems.map(item => item.startMs));
      const endMs = Math.max(...groupItems.map(item => item.endMs));
      const title =
        groupItems.find(item => item.kind === 'event')?.title ||
        groupItems[0]?.title ||
        `Turn ${index + 1}`;

      return {
        contextId,
        title,
        items: groupItems,
        startMs,
        endMs,
        durationMs: endMs - startMs,
      };
    })
    .sort((a, b) => a.startMs - b.startMs);
};

export const getTraceSummaries = (
  documents: TimelineDocument[],
): TraceSummary[] =>
  groupTimelineItems(documents).map(group => {
    const failureCount = group.items.filter(
      item => item.outcome === 'failure' || item.level === 'error',
    ).length;
    const components = Array.from(
      new Set(group.items.map(item => getDocumentComponent(item))),
    );
    const firstItem = group.items[0];

    return {
      assistantConversationId: firstItem?.assistantConversationId || 0,
      assistantId: firstItem?.assistantId || 0,
      components,
      contextId: group.contextId,
      durationMs: group.durationMs,
      endMs: group.endMs,
      failureCount,
      level: failureCount > 0 ? 'error' : firstItem?.level || 'info',
      outcome: failureCount > 0 ? 'failure' : 'success',
      spanCount: group.items.length,
      startMs: group.startMs,
      title: group.title,
    };
  });

const percentile = (values: number[], pct: number): number => {
  if (values.length === 0) return 0;
  const sorted = [...values].sort((a, b) => a - b);
  const index = Math.min(
    sorted.length - 1,
    Math.ceil((pct / 100) * sorted.length) - 1,
  );
  return sorted[index] || 0;
};

export const getMetricSummaries = (
  documents: TimelineDocument[],
): MetricSummary[] => {
  const byComponent = new Map<string, TimelineDocument[]>();

  documents.forEach(document => {
    const component = getDocumentComponent(document);
    const current = byComponent.get(component) || [];
    current.push(document);
    byComponent.set(component, current);
  });

  return Array.from(byComponent.entries())
    .map(([component, componentDocuments]) => {
      const durations = componentDocuments.map(document =>
        Math.max(document.durationMs || 0, 0),
      );
      const totalDuration = durations.reduce(
        (sum, duration) => sum + duration,
        0,
      );
      const slowest = componentDocuments.reduce(
        (current, document) =>
          (document.durationMs || 0) > (current.durationMs || 0)
            ? document
            : current,
        componentDocuments[0],
      );

      return {
        averageDurationMs:
          componentDocuments.length > 0
            ? Math.round(totalDuration / componentDocuments.length)
            : 0,
        component,
        count: componentDocuments.length,
        failureCount: componentDocuments.filter(
          document =>
            document.outcome === 'failure' || document.level === 'error',
        ).length,
        p95DurationMs: percentile(durations, 95),
        slowestContextId:
          slowest?.contextId ||
          `conversation-${slowest?.assistantConversationId || 'unknown'}`,
        slowestDurationMs: slowest?.durationMs || 0,
      };
    })
    .sort(
      (a, b) => b.count - a.count || a.component.localeCompare(b.component),
    );
};

export const matchesTimelineSearch = (
  doc: TimelineDocument,
  searchText: string,
): boolean => {
  const query = searchText.trim().toLowerCase();
  if (!query) return true;

  return [
    doc.id,
    doc.kind,
    doc.name,
    doc.category,
    doc.level,
    doc.outcome,
    doc.scope,
    doc.title,
    doc.messageId,
    doc.messageRole,
    doc.traceId,
    doc.contextId,
    JSON.stringify(doc.attributes || {}),
    JSON.stringify(doc.data || {}),
  ]
    .join('\n')
    .toLowerCase()
    .includes(query);
};

export const formatDurationMs = (durationMs?: number): string => {
  if (!durationMs || durationMs <= 0) return '<1 ms';
  if (durationMs < 1000) return `${durationMs} ms`;
  return `${(durationMs / 1000).toFixed(2)} s`;
};

export const formatTime = (value: string): string => {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return value;
  return date.toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    fractionalSecondDigits: 3,
  });
};

export const formatDateTime = (value: string | number): string => {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return String(value);
  return toHumanReadableDateTimeFromDate(date);
};

export const sampleTimelineDocuments: TimelineDocument[] = [
  {
    id: 'evt-call-started',
    kind: 'event',
    name: 'call.started',
    category: 'call',
    level: 'info',
    outcome: 'success',
    title: 'Inbound call started',
    projectId: 2,
    organizationId: 1,
    scope: 'conversation',
    assistantId: 1001,
    assistantConversationId: 80001,
    contextId: 'turn-1',
    attributes: { component: 'telephony', provider: 'twilio' },
    data: { from: '+15551234567', to: '+15557654321' },
    occurredAt: '2026-06-04T03:10:00.000Z',
    receivedAt: '2026-06-04T03:10:00.090Z',
    durationMs: 280,
  },
  {
    id: 'evt-stt-final',
    kind: 'event',
    name: 'stt.final_transcript',
    category: 'stt',
    level: 'info',
    outcome: 'success',
    title: 'User speech transcribed',
    projectId: 2,
    organizationId: 1,
    scope: 'message',
    assistantId: 1001,
    assistantConversationId: 80001,
    messageId: 'user-turn-1',
    messageRole: 'user',
    contextId: 'turn-1',
    attributes: { component: 'stt', provider: 'deepgram' },
    data: { text: 'I need to reschedule my appointment.' },
    occurredAt: '2026-06-04T03:10:01.050Z',
    receivedAt: '2026-06-04T03:10:01.270Z',
    durationMs: 620,
  },
  {
    id: 'evt-agent-response',
    kind: 'event',
    name: 'agent.completed',
    category: 'agent',
    level: 'info',
    outcome: 'success',
    title: 'Assistant response generated',
    projectId: 2,
    organizationId: 1,
    scope: 'message',
    assistantId: 1001,
    assistantConversationId: 80001,
    messageId: 'assistant-turn-1',
    messageRole: 'assistant',
    contextId: 'turn-1',
    attributes: { component: 'agent', provider: 'openai', model: 'gpt-4.1' },
    data: { promptTokens: 542, completionTokens: 68 },
    occurredAt: '2026-06-04T03:10:01.760Z',
    receivedAt: '2026-06-04T03:10:02.650Z',
    durationMs: 890,
  },
  {
    id: 'evt-tool-calendar',
    kind: 'event',
    name: 'tool.calendar.lookup',
    category: 'tool',
    level: 'info',
    outcome: 'success',
    title: 'Calendar availability checked',
    projectId: 2,
    organizationId: 1,
    scope: 'message',
    assistantId: 1001,
    assistantConversationId: 80001,
    messageId: 'assistant-turn-1',
    messageRole: 'assistant',
    contextId: 'turn-1',
    attributes: { component: 'tool', tool: 'calendar.lookup' },
    data: { slots: 3 },
    occurredAt: '2026-06-04T03:10:02.720Z',
    receivedAt: '2026-06-04T03:10:03.080Z',
    durationMs: 360,
  },
  {
    id: 'evt-tts-started',
    kind: 'event',
    name: 'tts.audio.started',
    category: 'tts',
    level: 'info',
    outcome: 'success',
    title: 'Assistant audio started',
    projectId: 2,
    organizationId: 1,
    scope: 'message',
    assistantId: 1001,
    assistantConversationId: 80001,
    messageId: 'assistant-turn-1',
    messageRole: 'assistant',
    contextId: 'turn-1',
    attributes: { component: 'tts', provider: 'cartesia' },
    data: { voice: 'sonic' },
    occurredAt: '2026-06-04T03:10:03.190Z',
    receivedAt: '2026-06-04T03:10:03.300Z',
    durationMs: 480,
  },
  {
    id: 'evt-stt-final-2',
    kind: 'event',
    name: 'stt.final_transcript',
    category: 'stt',
    level: 'info',
    outcome: 'success',
    title: 'User speech transcribed',
    projectId: 2,
    organizationId: 1,
    scope: 'message',
    assistantId: 1001,
    assistantConversationId: 80001,
    messageId: 'user-turn-2',
    messageRole: 'user',
    contextId: 'turn-2',
    attributes: { component: 'stt', provider: 'deepgram' },
    data: { text: 'Tomorrow afternoon works.' },
    occurredAt: '2026-06-04T03:10:08.250Z',
    receivedAt: '2026-06-04T03:10:08.420Z',
    durationMs: 510,
  },
  {
    id: 'evt-agent-response-2',
    kind: 'event',
    name: 'agent.completed',
    category: 'agent',
    level: 'info',
    outcome: 'success',
    title: 'Assistant response generated',
    projectId: 2,
    organizationId: 1,
    scope: 'message',
    assistantId: 1001,
    assistantConversationId: 80001,
    messageId: 'assistant-turn-2',
    messageRole: 'assistant',
    contextId: 'turn-2',
    attributes: { component: 'agent', provider: 'openai', model: 'gpt-4.1' },
    data: { promptTokens: 610, completionTokens: 42 },
    occurredAt: '2026-06-04T03:10:08.950Z',
    receivedAt: '2026-06-04T03:10:09.580Z',
    durationMs: 630,
  },
  {
    id: 'evt-tool-calendar-write',
    kind: 'event',
    name: 'tool.calendar.reschedule',
    category: 'tool',
    level: 'info',
    outcome: 'success',
    title: 'Appointment rescheduled',
    projectId: 2,
    organizationId: 1,
    scope: 'message',
    assistantId: 1001,
    assistantConversationId: 80001,
    messageId: 'assistant-turn-2',
    messageRole: 'assistant',
    contextId: 'turn-2',
    attributes: { component: 'tool', tool: 'calendar.reschedule' },
    data: { appointmentId: 'apt_0912', status: 'confirmed' },
    occurredAt: '2026-06-04T03:10:09.640Z',
    receivedAt: '2026-06-04T03:10:10.320Z',
    durationMs: 680,
  },
  {
    id: 'evt-tts-started-2',
    kind: 'event',
    name: 'tts.audio.started',
    category: 'tts',
    level: 'info',
    outcome: 'success',
    title: 'Confirmation audio started',
    projectId: 2,
    organizationId: 1,
    scope: 'message',
    assistantId: 1001,
    assistantConversationId: 80001,
    messageId: 'assistant-turn-2',
    messageRole: 'assistant',
    contextId: 'turn-2',
    attributes: { component: 'tts', provider: 'cartesia' },
    data: { voice: 'sonic' },
    occurredAt: '2026-06-04T03:10:10.390Z',
    receivedAt: '2026-06-04T03:10:10.500Z',
    durationMs: 430,
  },
];
