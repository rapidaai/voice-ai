import type { TYPES } from '@carbon/react/es/components/Tag/Tag';
import type {
  ObservabilityEventRecord,
  ObservabilityMetricRecord,
} from '@rapidaai/react';

export type TelemetryTagType = keyof typeof TYPES;

export type SelectOption = {
  id: string;
  label: string;
};

export type CriteriaInput = {
  key: string;
  value: string;
};

export type TelemetryFilterState = {
  searchText: string;
  names: string[];
  messageOrContextId: string;
  eventDataType: string;
  metricScope: string;
};

export type TelemetrySearchDocument = {
  kind: 'event' | 'metric';
  componentType: string;
  typeLabel: string;
  name: string;
  scope: string;
  conversationId: string;
  messageId: string;
  contextId: string;
  eventDataType: string;
  rawText: string;
};

export type TelemetryRow =
  | { kind: 'event'; ts: Date; key: string; record: ObservabilityEventRecord }
  | {
      kind: 'metric';
      ts: Date;
      key: string;
      record: ObservabilityMetricRecord;
    };

export type LatencyMetricName =
  | 'stt.latency_ms'
  | 'tts.latency_ms'
  | 'agent.ttft_ms'
  | 'eos.latency_ms';

export type LatencyMetricDocument = {
  timestampMs: number;
  contextId: string;
  conversationId: string;
  metrics: Array<{ name: string; value: string }>;
};

export type LatencySeriesPoint = {
  key: string;
  sequence: number;
  timestampMs: number;
  timeLabel: string;
  contextId: string;
  conversationId: string;
} & Partial<Record<LatencyMetricName, number>>;

export type EventTelemetryJson = {
  name: string;
  component: string;
  scope: string;
  messageId: string;
  conversationId: string;
  data: Record<string, string>;
  context: Record<string, string>;
  scopeAttributes: Record<string, string>;
};

export type MetricTelemetryJson = {
  scope: string;
  contextId: string;
  conversationId: string;
  metrics: Array<{ name: string; value: string }>;
  context: Record<string, string>;
  scopeAttributes: Record<string, string>;
};

export type TelemetryRowJson = EventTelemetryJson | MetricTelemetryJson;

export type TelemetryRowData = {
  typeLabel: string;
  tagType: TelemetryTagType;
  json: TelemetryRowJson;
};

export const EVENT_TAG_TYPE: Record<string, TelemetryTagType> = {
  session: 'gray',
  telephony: 'teal',
  webrtc: 'cool-gray',
  stt: 'green',
  agent: 'blue',
  tts: 'purple',
  vad: 'warm-gray',
  eos: 'cyan',
  denoise: 'warm-gray',
  recording: 'purple',
  tool: 'magenta',
  knowledge: 'teal',
  metric: 'high-contrast',
};

export const EVENT_NAME_OPTIONS: SelectOption[] = [
  'session',
  'telephony',
  'webrtc',
  'stt',
  'agent',
  'tts',
  'vad',
  'eos',
  'denoise',
  'recording',
  'tool',
  'knowledge',
].map(id => ({
  id,
  label: id,
}));

export const METRIC_SCOPE_OPTIONS: SelectOption[] = [
  'message',
  'conversation',
].map(id => ({
  id,
  label: id.charAt(0) + id.slice(1),
}));

export const LATENCY_METRIC_META: Record<
  LatencyMetricName,
  {
    label: string;
    shortLabel: string;
    seriesColor: string;
    fillOpacity: number;
  }
> = {
  'stt.latency_ms': {
    label: 'STT Latency',
    shortLabel: 'STT',
    seriesColor: 'var(--color-data-3)',
    fillOpacity: 0.16,
  },
  'tts.latency_ms': {
    label: 'TTS Latency',
    shortLabel: 'TTS',
    seriesColor: 'var(--color-data-1)',
    fillOpacity: 0.38,
  },
  'agent.ttft_ms': {
    label: 'Agent Time to First Token',
    shortLabel: 'Agent TTFT',
    seriesColor: 'var(--color-data-4)',
    fillOpacity: 0.3,
  },
  'eos.latency_ms': {
    label: 'EOS Latency',
    shortLabel: 'EOS',
    seriesColor: 'var(--color-data-2)',
    fillOpacity: 0.22,
  },
};

export const LATENCY_STACK_ORDER: LatencyMetricName[] = [
  'stt.latency_ms',
  'eos.latency_ms',
  'agent.ttft_ms',
  'tts.latency_ms',
];

export const mapComponentType = (nameKey: string): string =>
  nameKey === 'sip' ? 'telephony' : nameKey;

const mapToObject = (map: {
  toArray: () => Array<[string, string]>;
}): Record<string, string> => Object.fromEntries(map.toArray());

const firstPresent = (...values: Array<string | undefined>): string =>
  values.find(value => value && value.trim() !== '') || '';

export const getTelemetryScopeAttributes = (
  record: ObservabilityEventRecord | ObservabilityMetricRecord,
): Record<string, string> => mapToObject(record.getScopeattributesMap());

const getContext = (
  record: ObservabilityEventRecord | ObservabilityMetricRecord,
): Record<string, string> => mapToObject(record.getContextMap());

export const getTelemetryConversationId = (
  scopeAttributes: Record<string, string>,
): string =>
  firstPresent(
    scopeAttributes.assistantConversationId,
    scopeAttributes.conversationId,
  );

export const getTelemetryContextId = (
  scopeAttributes: Record<string, string>,
): string =>
  firstPresent(
    scopeAttributes.contextId,
    scopeAttributes.messageId,
    scopeAttributes.assistantConversationId,
    scopeAttributes.conversationId,
  );

export const splitStructuredTelemetryCriteria = (
  criteriaInputs: CriteriaInput[],
): {
  conversationId: string;
  messageId: string;
  remaining: CriteriaInput[];
} => {
  let conversationId = '';
  let messageId = '';
  const remaining: CriteriaInput[] = [];

  criteriaInputs.forEach(c => {
    if (c.key === 'conversationId') {
      conversationId = c.value;
      return;
    }
    if (c.key === 'messageId' || c.key === 'contextId') {
      messageId = c.value;
      return;
    }
    remaining.push(c);
  });

  return { conversationId, messageId, remaining };
};

export const buildTelemetryCriteriaInputs = (
  remaining: CriteriaInput[],
  conversationId: string,
  messageId: string,
): CriteriaInput[] => {
  const out = [...remaining];
  if (conversationId)
    out.push({ key: 'conversationId', value: conversationId });
  if (messageId) out.push({ key: 'messageId', value: messageId });
  return out;
};

export const matchesTelemetryFilters = (
  document: TelemetrySearchDocument,
  filters: TelemetryFilterState,
): boolean => {
  const normalizeSearchValue = (value?: string) =>
    String(value || '')
      .toLowerCase()
      .replace(/\s+/g, ' ')
      .trim();
  const compactSearchValue = (value?: string) =>
    String(value || '')
      .toLowerCase()
      .replace(/[\s"'`]+/g, '');
  const contains = (source: string, term: string) =>
    normalizeSearchValue(source).includes(normalizeSearchValue(term)) ||
    compactSearchValue(source).includes(compactSearchValue(term));
  const searchTerm = filters.searchText.trim();

  if (
    searchTerm &&
    !contains(document.typeLabel, searchTerm) &&
    !contains(document.rawText, searchTerm)
  ) {
    return false;
  }

  if (
    filters.names.length > 0 &&
    !filters.names.some(name => contains(document.name, name))
  ) {
    return false;
  }

  if (
    filters.messageOrContextId &&
    !contains(document.messageId, filters.messageOrContextId) &&
    !contains(document.contextId, filters.messageOrContextId)
  ) {
    return false;
  }

  if (
    filters.eventDataType &&
    !contains(document.eventDataType, filters.eventDataType)
  ) {
    return false;
  }

  if (filters.metricScope && !contains(document.scope, filters.metricScope))
    return false;

  return true;
};

export function formatDateTime(d: Date): string {
  const pad = (n: number, w = 2) => String(n).padStart(w, '0');
  return (
    `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ` +
    `${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())}.${pad(d.getUTCMilliseconds(), 3)}`
  );
}

const isLatencyMetricName = (name: string): name is LatencyMetricName =>
  name === 'stt.latency_ms' ||
  name === 'tts.latency_ms' ||
  name === 'agent.ttft_ms' ||
  name === 'eos.latency_ms';

export const buildLatencySeries = (
  documents: LatencyMetricDocument[],
): LatencySeriesPoint[] => {
  const merged = new Map<string, LatencySeriesPoint>();

  documents.forEach(document => {
    const contextKey = document.contextId || `ts-${document.timestampMs}`;
    const key = `${document.conversationId || 'unknown'}::${contextKey}`;
    const existing = merged.get(key);
    const point: LatencySeriesPoint =
      existing ??
      ({
        key,
        sequence: 0,
        timestampMs: document.timestampMs,
        timeLabel: formatDateTime(new Date(document.timestampMs)),
        contextId: document.contextId,
        conversationId: document.conversationId,
      } as LatencySeriesPoint);

    const hasExisting = !!existing;
    let hasLatencyValue = false;

    document.metrics.forEach(metric => {
      if (!isLatencyMetricName(metric.name)) return;
      const parsed = Number(metric.value);
      if (!Number.isFinite(parsed)) return;
      point[metric.name] = parsed;
      hasLatencyValue = true;
    });

    if (!hasExisting && !hasLatencyValue) return;

    if (document.timestampMs < point.timestampMs) {
      point.timestampMs = document.timestampMs;
      point.timeLabel = formatDateTime(new Date(document.timestampMs));
    }

    if (!point.contextId && document.contextId) {
      point.contextId = document.contextId;
    }
    if (!point.conversationId && document.conversationId) {
      point.conversationId = document.conversationId;
    }

    merged.set(key, point);
  });

  return Array.from(merged.values())
    .sort((a, b) => a.timestampMs - b.timestampMs)
    .map((point, index) => ({
      ...point,
      sequence: index + 1,
    }));
};

export function eventToJson(
  event: ObservabilityEventRecord,
): EventTelemetryJson {
  const data = mapToObject(event.getAttributesMap());
  const context = getContext(event);
  const scopeAttributes = getTelemetryScopeAttributes(event);
  return {
    name: event.getEvent(),
    component: event.getComponent(),
    scope: event.getScope(),
    messageId: scopeAttributes.messageId || '',
    conversationId: getTelemetryConversationId(scopeAttributes),
    data,
    context,
    scopeAttributes,
  };
}

export function metricToJson(
  metric: ObservabilityMetricRecord,
): MetricTelemetryJson {
  const scopeAttributes = getTelemetryScopeAttributes(metric);
  return {
    scope: metric.getScope(),
    contextId: getTelemetryContextId(scopeAttributes),
    conversationId: getTelemetryConversationId(scopeAttributes),
    metrics: [{ name: metric.getName(), value: metric.getValue() }],
    context: getContext(metric),
    scopeAttributes,
  };
}

export function getTelemetrySearchDocument(
  row: TelemetryRow,
  typeLabel: string,
  json: TelemetryRowJson,
): TelemetrySearchDocument {
  if (row.kind === 'event') {
    const eventJson = json as EventTelemetryJson;
    return {
      kind: 'event',
      componentType: mapComponentType(
        row.record.getComponent() || row.record.getEvent().split('.')[0],
      ),
      typeLabel,
      name: row.record.getEvent(),
      scope: row.record.getScope(),
      conversationId: eventJson.conversationId,
      messageId: eventJson.messageId,
      contextId: '',
      eventDataType: eventJson.data.type || '',
      rawText: `${JSON.stringify(json)}\n${JSON.stringify(json, null, 2)}`,
    };
  }

  const metricJson = json as MetricTelemetryJson;
  return {
    kind: 'metric',
    componentType: 'metric',
    typeLabel,
    name: row.record.getName(),
    scope: row.record.getScope(),
    conversationId: metricJson.conversationId,
    messageId: '',
    contextId: metricJson.contextId,
    eventDataType: '',
    rawText: `${JSON.stringify(json)}\n${JSON.stringify(json, null, 2)}`,
  };
}

export function getTelemetryRowData(row: TelemetryRow): TelemetryRowData {
  if (row.kind === 'event') {
    const eventName = row.record.getEvent();
    const nameKey = mapComponentType(
      row.record.getComponent() || eventName.split('.')[0],
    );
    return {
      typeLabel: eventName,
      tagType: EVENT_TAG_TYPE[nameKey] ?? 'gray',
      json: eventToJson(row.record),
    };
  }
  return {
    typeLabel: `metric·${row.record.getScope()}`,
    tagType: 'high-contrast',
    json: metricToJson(row.record),
  };
}
