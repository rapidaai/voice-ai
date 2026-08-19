import { Metadata, Metric } from '@rapidaai/react';
import { Struct, Value } from 'google-protobuf/google/protobuf/struct_pb';

// Metric name constants
const TIME_TAKEN = 'time_taken';
const STATUS = 'status';
const TOTAL_TOKEN = 'agent_total_token';
const CONVERSATION_DURATION_MS = 'conversation.duration_ms';

// ── Universal metric finder ─────────────────────────────────────────────────
// Proto metrics may expose the name via getName() or getKey() depending on the
// message type (conversation-level vs message-level). This helper checks both.

export function findMetricByName(metrics: any[], name: string): string {
  const m = metrics.find((x: any) => (x.getName?.() || x.getKey?.()) === name);
  return m ? m.getValue() : '';
}

// ── Conversation status helpers ─────────────────────────────────────────────

export function isConversationCompleted(metrics: any[]): boolean {
  const s = findMetricByName(metrics, STATUS).toUpperCase();
  return s === 'COMPLETE' || s === 'COMPLETED';
}

export function isConversationActive(metrics: any[]): boolean {
  return !isConversationCompleted(metrics);
}

// ── Duration formatting ─────────────────────────────────────────────────────

/** Format seconds into human-readable duration (e.g. "2m 30s", "45s") */
export function formatDurationSeconds(secs: number): string {
  if (isNaN(secs) || secs <= 0) return '–';
  if (secs >= 3600) {
    const h = Math.floor(secs / 3600);
    const m = Math.floor((secs % 3600) / 60);
    return m > 0 ? `${h}h ${m}m` : `${h}h`;
  }
  if (secs >= 60) {
    const m = Math.floor(secs / 60);
    const s = Math.round(secs % 60);
    return s > 0 ? `${m}m ${s}s` : `${m}m`;
  }
  return `${Math.round(secs)}s`;
}

/** Read conversation duration in milliseconds. */
export function getConversationDuration(metrics: any[]): string {
  const conversationDuration = findMetricByName(
    metrics,
    CONVERSATION_DURATION_MS,
  );
  if (conversationDuration)
    return formatDurationSeconds(Number(conversationDuration) / 1_000);
  return '–';
}

/**
 *
 * @param metrics
 * @returns
 */
export const getTotalTokenMetric = (metrics: Array<Metric>): number => {
  let ttl = metrics.find(x => x.getName() === TOTAL_TOKEN);
  return ttl ? +ttl.getValue() : 0;
};

/**
 *
 * @param metrics
 * @returns
 */
export const getTimeTakenMetric = (metrics: Array<Metric>): number => {
  let ttl = metrics.find(x => x.getName() === TIME_TAKEN);
  return ttl ? +ttl.getValue() : 0;
};

/**
 *
 * @param metrics
 * @returns
 */
export const getStatusMetric = (metrics?: Array<Metric>): string => {
  let ttl = metrics?.find(x => x.getName() === STATUS);
  return ttl ? ttl.getValue() : 'ACTIVE';
};

/**
 *
 * @param metrics
 * @param k
 * @returns
 */
export function getMetricValue(metrics: Metric[], k: string): string {
  let ttl = metrics.find(x => x.getName() === k);
  return ttl ? ttl.getValue() : '';
}

/**
 *
 * @param metrics
 * @param k
 * @param vl
 * @returns
 */
export function getMetricValueOrDefault(
  metrics: Metric[],
  k: string,
  vl: string,
): string {
  let ttl = getMetricValue(metrics, k);
  return ttl ? ttl : vl;
}

/**
 *
 * @param mt
 * @param k
 * @returns
 */
export function getMetadataValue(mt: Metadata[], k: string) {
  let _mt = mt.find(m => {
    return m.getKey() === k;
  });
  return _mt?.getValue();
}

/**
 *
 * @param mt
 * @param k
 * @param df
 * @returns
 */
export function getMetadataValueOrDefault(
  mt: Metadata[],
  k: string,
  df: string,
) {
  let _mt = mt.find(m => {
    return m.getKey() === k;
  });

  return _mt ? _mt?.getValue() : df;
}

// Function to extract a string value
export function getStringFromProtoStruct(
  struct?: Struct,
  key?: string,
): string | null {
  if (!struct || !key) {
    return null;
  }
  const fields = struct.getFieldsMap();
  const value = fields.get(key);

  if (value && value.getKindCase() === Value.KindCase.STRING_VALUE) {
    return value.getStringValue();
  }

  return null; // Return null if the key doesn't exist or isn't a string
}

export function getJsonFromProtoStruct(
  struct?: Struct,
  key?: string,
): Record<string, any> | null {
  if (!struct || !key) {
    return null;
  }
  const fields = struct.getFieldsMap();
  const value = fields.get(key);

  if (value && value.getKindCase() === Value.KindCase.STRING_VALUE) {
    return JSON.parse(value.getStringValue());
  }

  if (value && value.getKindCase() === Value.KindCase.STRUCT_VALUE) {
    const result = value.getStructValue()?.toJavaScript();
    return result ?? {};
  }

  return null; // Return null if the key doesn't exist or isn't a string
}

export const SetMetadata = (
  existings: Metadata[],
  key: string,
  defaultValue?: string,
  validationFn?: (value: string) => boolean,
): Metadata | undefined => {
  const existingMetadata = existings.find(m => m.getKey() === key);
  let valueToSet: string | undefined;

  if (existingMetadata) {
    const existingValue = existingMetadata.getValue();
    if (!validationFn || validationFn(existingValue)) {
      valueToSet = existingValue;
    }
  }

  if (valueToSet === undefined && defaultValue !== undefined) {
    if (!validationFn || validationFn(defaultValue)) {
      valueToSet = defaultValue;
    }
  }

  if (valueToSet !== undefined) {
    const metadata = new Metadata();
    metadata.setKey(key);
    metadata.setValue(valueToSet);
    return metadata;
  }

  return undefined;
};
