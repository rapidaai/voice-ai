import { Metadata } from '@rapidaai/react';

export type HttpMethod = 'POST' | 'GET';
export type FailBehavior = 'block' | 'do_nothing';

export const AUTH_OPTION_METHOD = 'http_method';
export const AUTH_OPTION_ENDPOINT = 'http_url';
export const AUTH_OPTION_HEADERS = 'http_headers';
export const AUTH_OPTION_BODY = 'http_body';
export const AUTH_OPTION_CONDITION = 'authentication.condition';
export const AUTH_OPTION_FAIL_BEHAVIOR = 'fail_behavior';
export const AUTH_OPTION_TIMEOUT_MS = 'timeout_ms';
export const FAIL_BEHAVIOR_BLOCK = 'BLOCK';
export const FAIL_BEHAVIOR_DO_NOTHING = 'DO_NOTHING';

export const DEFAULT_HEADERS = '{}';
export const DEFAULT_BODY =
  '{"assistant.id":"assistantId","client.phone":"clientPhone"}';
export const DEFAULT_TIMEOUT_MS = 5000;
export const DEFAULT_SOURCE_CONDITIONS = [
  {
    key: 'source',
    condition: '=',
    value: 'all',
  },
];

export const AUTH_PARAMETER_TYPE_OPTIONS = [
  { value: 'client', name: 'Client' },
  { value: 'assistant', name: 'Assistant' },
  { value: 'conversation', name: 'Conversation' },
  { value: 'argument', name: 'Argument' },
  { value: 'metadata', name: 'Metadata' },
  { value: 'option', name: 'Option' },
  { value: 'custom', name: 'Custom' },
];

export const AUTH_KEY_OPTIONS_BY_TYPE = {
  assistant: [
    { value: 'id', name: 'ID' },
    { value: 'name', name: 'Name' },
    { value: 'prompt', name: 'Prompt' },
  ],
  client: [
    { value: 'phone', name: 'Phone' },
    { value: 'assistant_phone', name: 'Assistant Phone' },
    { value: 'direction', name: 'Direction' },
    { value: 'channel', name: 'Channel' },
    { value: 'provider_call_id', name: 'Provider Call ID' },
  ],
  conversation: [
    { value: 'messages', name: 'Messages' },
    { value: 'id', name: 'ID' },
  ],
};

export const fromApiFailBehavior = (value?: string): FailBehavior => {
  const normalized = (value || '').trim().toLowerCase();
  if (
    normalized === 'do_nothing' ||
    normalized === 'do-nothing' ||
    normalized === 'none'
  ) {
    return 'do_nothing';
  }
  return 'block';
};

export const toApiFailBehavior = (value: FailBehavior): string =>
  value === 'do_nothing' ? FAIL_BEHAVIOR_DO_NOTHING : FAIL_BEHAVIOR_BLOCK;

const LEGACY_CLIENT_KEYS: Record<string, string> = {
  'client.assistantPhone': 'client.assistant_phone',
  'client.providerCallId': 'client.provider_call_id',
};

export const normalizeAuthenticationBody = (value: string): string => {
  try {
    const parsed = JSON.parse(value);
    if (
      typeof parsed !== 'object' ||
      parsed === null ||
      Array.isArray(parsed)
    ) {
      return value;
    }

    const normalized = { ...(parsed as Record<string, unknown>) };
    Object.entries(LEGACY_CLIENT_KEYS).forEach(([legacyKey, canonicalKey]) => {
      if (!(canonicalKey in normalized) && legacyKey in normalized) {
        normalized[canonicalKey] = normalized[legacyKey];
      }
      delete normalized[legacyKey];
    });
    return JSON.stringify(normalized);
  } catch {
    return value;
  }
};

export const toOptionMap = (options: Metadata[] = []) =>
  options.reduce(
    (acc, opt) => {
      acc[opt.getKey()] = opt.getValue();
      return acc;
    },
    {} as Record<string, string>,
  );
