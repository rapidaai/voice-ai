import { AssistantConversation } from '@rapidaai/react';
import { getMetadataValueOrDefault, getMetricValue } from '@/utils/metadata';

const DISCONNECT_REASON_KEY = 'disconnect_reason';
const UNKNOWN_DISCONNECT_REASON = 'unknown';
const CHANNEL_KEY = 'client.channel';
const DEFAULT_CHANNEL = 'webrtc';

export const UNKNOWN_DURATION_VALUE = 'unknown';

const DURATION_BREAKDOWN_METRICS = [
  {
    key: 'conversation.duration_ms',
  },
  {
    key: 'call.duration_ms',
  },
  {
    key: 'tts.duration_ms',
  },
  {
    key: 'stt.duration_ms',
  },
] as const;

export type DurationBreakdownRow = {
  key: string;
  value: string;
};

const getSessionMetadataValue = (
  conversation: AssistantConversation,
  key: string,
  fallback: string,
): string => {
  const value = getMetadataValueOrDefault(
    conversation.getMetadataList(),
    key,
    fallback,
  );
  return value?.trim() ? value : fallback;
};

export const getDisconnectReasonValue = (
  conversation: AssistantConversation,
): string => {
  return getSessionMetadataValue(
    conversation,
    DISCONNECT_REASON_KEY,
    UNKNOWN_DISCONNECT_REASON,
  );
};

export const getChannelValue = (conversation: AssistantConversation): string =>
  getSessionMetadataValue(conversation, CHANNEL_KEY, DEFAULT_CHANNEL);

const formatDurationMetricSeconds = (rawValue: string): string => {
  if (!rawValue?.trim()) return UNKNOWN_DURATION_VALUE;

  const numericValue = Number(rawValue);
  if (!Number.isFinite(numericValue)) return UNKNOWN_DURATION_VALUE;

  return (numericValue / 1_000).toFixed(2);
};

export const getDurationBreakdownRows = (
  conversation: AssistantConversation,
): DurationBreakdownRow[] => {
  const metrics = conversation.getMetricsList();
  return DURATION_BREAKDOWN_METRICS.map(metric => ({
    key: metric.key,
    value: formatDurationMetricSeconds(getMetricValue(metrics, metric.key)),
  }));
};
