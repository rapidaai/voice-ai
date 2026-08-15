import {
  QuerySearch,
  parseQuerySearchFilters,
} from '@/app/components/carbon/query-search';
import type {
  QuerySearchField,
  QuerySearchOption,
} from '@/app/components/carbon/query-search';

type SessionSearchCriteria = {
  k: string;
  logic: string;
  v: string;
};

type SessionQuerySearchProps = {
  onApply: (criteria: SessionSearchCriteria[]) => void;
  onChange: (value: string) => void;
  value: string;
};

const STATUS_OPTIONS: QuerySearchOption[] = [
  { id: 'ACTIVE', text: 'active' },
  { id: 'in_progress', text: 'in progress' },
  { id: 'complete', text: 'complete' },
  { id: 'error', text: 'error' },
  { id: 'FAILED', text: 'failed' },
];

const CALL_STATUS_OPTIONS: QuerySearchOption[] = [
  { id: 'INPROGRESS', text: 'in progress' },
  { id: 'RINGING', text: 'ringing' },
  { id: 'COMPLETE', text: 'complete' },
  { id: 'FAILED', text: 'failed' },
  { id: 'CANCELLED', text: 'cancelled' },
];

const DIRECTION_OPTIONS: QuerySearchOption[] = [
  { id: 'inbound', text: 'inbound' },
  { id: 'outbound', text: 'outbound' },
];

const SOURCE_OPTIONS: QuerySearchOption[] = [
  { id: 'web-plugin', text: 'web plugin' },
  { id: 'debugger', text: 'debugger' },
  { id: 'sdk', text: 'sdk' },
  { id: 'phone-call', text: 'phone call' },
  { id: 'whatsapp', text: 'whatsapp' },
];

const CHANNEL_OPTIONS: QuerySearchOption[] = [
  { id: 'webrtc', text: 'webrtc' },
  { id: 'phone', text: 'phone' },
  { id: 'sip', text: 'sip' },
  { id: 'twilio', text: 'twilio' },
  { id: 'exotel', text: 'exotel' },
  { id: 'telnyx', text: 'telnyx' },
  { id: 'asterisk', text: 'asterisk' },
  { id: 'vobiz', text: 'vobiz' },
  { id: 'vonage', text: 'vonage' },
  { id: 'web', text: 'web' },
];

const CODEC_OPTIONS: QuerySearchOption[] = [
  { id: 'PCMU', text: 'PCMU' },
  { id: 'PCMA', text: 'PCMA' },
  { id: 'OPUS', text: 'OPUS' },
];

const DISCONNECT_REASON_OPTIONS: QuerySearchOption[] = [
  { id: 'DISCONNECTION_TYPE_USER', text: 'user' },
  { id: 'DISCONNECTION_TYPE_ERROR', text: 'error' },
  { id: 'remote_hangup', text: 'remote hangup' },
  { id: 'normal_clearing', text: 'normal clearing' },
  { id: 'busy', text: 'busy' },
  { id: 'no_answer', text: 'no answer' },
  { id: 'rejected', text: 'rejected' },
  { id: 'cancelled', text: 'cancelled' },
  { id: 'network_failure', text: 'network failure' },
  { id: 'remote_error', text: 'remote error' },
  { id: 'websocket_closed', text: 'websocket closed' },
  { id: 'provider_stop', text: 'provider stop' },
  { id: 'provider_hangup', text: 'provider hangup' },
  { id: 'reader_closed', text: 'reader closed' },
  { id: 'bye_received', text: 'bye received' },
  { id: 'server_side_disconnect', text: 'server side disconnect' },
  { id: 'tool_end_conversation', text: 'tool end conversation' },
  { id: 'outbound_setup_failed', text: 'outbound setup failed' },
  { id: 'outbound_health_gate_failed', text: 'outbound health gate failed' },
  { id: 'outbound_request_cancelled', text: 'outbound request cancelled' },
  { id: 'outbound_no_answer', text: 'outbound no answer' },
  {
    id: 'outbound_cancelled_before_answer',
    text: 'outbound cancelled before answer',
  },
  { id: 'outbound_setup_failure', text: 'outbound setup failure' },
  { id: 'outbound_wait_answer_failed', text: 'outbound wait answer failed' },
  { id: 'outbound_auth_failed', text: 'outbound auth failed' },
  { id: 'outbound_unavailable', text: 'outbound unavailable' },
  { id: 'outbound_rejected', text: 'outbound rejected' },
  { id: 'outbound_media_rejected', text: 'outbound media rejected' },
  { id: 'outbound_media_timeout', text: 'outbound media timeout' },
  { id: 'outbound_upstream_failure', text: 'outbound upstream failure' },
  { id: 'outbound_trunk_capacity', text: 'outbound trunk capacity' },
  { id: 'outbound_network_failure', text: 'outbound network failure' },
  { id: 'outbound_answer_sdp_failed', text: 'outbound answer sdp failed' },
  { id: 'outbound_ack_failed', text: 'outbound ack failed' },
  { id: 'outbound_max_duration', text: 'outbound max duration' },
  { id: 'outbound_teardown_timeout', text: 'outbound teardown timeout' },
  { id: 'outbound_failed', text: 'outbound failed' },
  { id: 'transfer_outbound_ended', text: 'transfer outbound ended' },
];

const formatOptionValue =
  (options: QuerySearchOption[]) =>
  (value: string): string =>
    options.find(option => option.id === value)?.text || value;

const SESSION_SEARCH_FIELDS: QuerySearchField[] = [
  {
    category: 'session',
    queryKey: 'id',
    text: 'sessionID',
    type: 'number',
  },
  {
    category: 'session',
    logicLabel: 'is after',
    logicOptions: [
      { label: 'is after', logic: '>=' },
      { label: 'is before', logic: '<=' },
      { label: 'is', logic: '=' },
    ],
    queryKey: 'timestamp',
    text: 'timestamp',
    type: 'date',
  },
  {
    category: 'session',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'contains', logic: 'contains' },
    ],
    queryKey: 'identifier',
    text: 'identifier',
    type: 'string',
  },
  {
    category: 'metrics',
    formatValue: formatOptionValue(STATUS_OPTIONS),
    items: STATUS_OPTIONS,
    logicLabel: 'is',
    logicOptions: [{ label: 'is', logic: '=' }],
    queryKey: 'status',
    text: 'status',
    type: 'string',
  },
  {
    category: 'metrics',
    formatValue: formatOptionValue(CALL_STATUS_OPTIONS),
    items: CALL_STATUS_OPTIONS,
    logicLabel: 'is',
    logicOptions: [{ label: 'is', logic: '=' }],
    queryKey: 'call.status',
    text: 'call.status',
    type: 'string',
  },
  {
    category: 'session',
    formatValue: formatOptionValue(DISCONNECT_REASON_OPTIONS),
    items: DISCONNECT_REASON_OPTIONS,
    logicLabel: 'is',
    logicOptions: [{ label: 'is', logic: '=' }],
    queryKey: 'disconnect_reason',
    text: 'disconnectReason',
    type: 'string',
  },
  {
    category: 'session',
    formatValue: formatOptionValue(DIRECTION_OPTIONS),
    items: DIRECTION_OPTIONS,
    queryKey: 'direction',
    text: 'direction',
    type: 'string',
  },
  {
    category: 'client',
    formatValue: formatOptionValue(DIRECTION_OPTIONS),
    items: DIRECTION_OPTIONS,
    queryKey: 'client.direction',
    text: 'client.direction',
    type: 'string',
  },
  {
    category: 'client',
    formatValue: formatOptionValue(CHANNEL_OPTIONS),
    items: CHANNEL_OPTIONS,
    queryKey: 'client.channel',
    text: 'client.channel',
    type: 'string',
  },
  {
    category: 'client',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'contains', logic: 'contains' },
    ],
    queryKey: 'client.provider_call_id',
    text: 'client.provider_call_id',
    type: 'string',
  },
  {
    category: 'client',
    formatValue: formatOptionValue(CODEC_OPTIONS),
    items: CODEC_OPTIONS,
    queryKey: 'client.codec',
    text: 'client.codec',
    type: 'string',
  },
  {
    category: 'client',
    queryKey: 'client.sample_rate',
    text: 'client.sample_rate',
    type: 'number',
  },
  {
    category: 'client',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'contains', logic: 'contains' },
    ],
    queryKey: 'client.context_id',
    text: 'client.context_id',
    type: 'string',
  },
  {
    category: 'client',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'contains', logic: 'contains' },
    ],
    queryKey: 'client.phone',
    text: 'client.phone',
    type: 'string',
  },
  {
    category: 'client',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'contains', logic: 'contains' },
    ],
    queryKey: 'client.assistant_phone',
    text: 'client.assistant_phone',
    type: 'string',
  },
  {
    category: 'session',
    formatValue: formatOptionValue(SOURCE_OPTIONS),
    items: SOURCE_OPTIONS,
    queryKey: 'source',
    text: 'source',
    type: 'string',
  },
  {
    category: 'metrics',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'is greater than or equal to', logic: '>=' },
      { label: 'is less than or equal to', logic: '<=' },
    ],
    queryKey: 'recording_init_ms',
    text: 'recording_init_ms',
    type: 'number',
  },
  {
    category: 'metrics',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'is greater than or equal to', logic: '>=' },
      { label: 'is less than or equal to', logic: '<=' },
    ],
    queryKey: 'stt_init_ms',
    text: 'stt_init_ms',
    type: 'number',
  },
  {
    category: 'metrics',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'is greater than or equal to', logic: '>=' },
      { label: 'is less than or equal to', logic: '<=' },
    ],
    queryKey: 'tts_init_ms',
    text: 'tts_init_ms',
    type: 'number',
  },
  {
    category: 'metrics',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'is greater than or equal to', logic: '>=' },
      { label: 'is less than or equal to', logic: '<=' },
    ],
    queryKey: 'llm_init_ms',
    text: 'llm_init_ms',
    type: 'number',
  },
  {
    category: 'metrics',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'is greater than or equal to', logic: '>=' },
      { label: 'is less than or equal to', logic: '<=' },
    ],
    queryKey: 'denoise_init_ms',
    text: 'denoise_init_ms',
    type: 'number',
  },
  {
    category: 'metrics',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'is greater than or equal to', logic: '>=' },
      { label: 'is less than or equal to', logic: '<=' },
    ],
    queryKey: 'eos_init_ms',
    text: 'eos_init_ms',
    type: 'number',
  },
  {
    category: 'metrics',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'is greater than or equal to', logic: '>=' },
      { label: 'is less than or equal to', logic: '<=' },
    ],
    queryKey: 'vad_init_ms',
    text: 'vad_init_ms',
    type: 'number',
  },
  {
    category: 'duration',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'is greater than or equal to', logic: '>=' },
      { label: 'is less than or equal to', logic: '<=' },
    ],
    queryKey: 'conversation.duration_ms',
    text: 'conversation.duration_ms',
    type: 'number',
  },
  {
    category: 'duration',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'is greater than or equal to', logic: '>=' },
      { label: 'is less than or equal to', logic: '<=' },
    ],
    queryKey: 'call.duration_ms',
    text: 'call.duration_ms',
    type: 'number',
  },
  {
    category: 'duration',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'is greater than or equal to', logic: '>=' },
      { label: 'is less than or equal to', logic: '<=' },
    ],
    queryKey: 'tts.duration_ms',
    text: 'tts.duration_ms',
    type: 'number',
  },
  {
    category: 'duration',
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'is greater than or equal to', logic: '>=' },
      { label: 'is less than or equal to', logic: '<=' },
    ],
    queryKey: 'stt.duration_ms',
    text: 'stt.duration_ms',
    type: 'number',
  },
  {
    category: 'session',
    queryKey: 'assistant_provider_model_id',
    text: 'versionID',
    type: 'number',
  },
];

const SESSION_SEARCH_CRITERIA: Record<string, string> = {
  assistant_provider_model_id: 'assistant_provider_model_id',
  'call.status': 'call.status',
  'call.duration_ms': 'call.duration_ms',
  'conversation.duration_ms': 'conversation.duration_ms',
  'client.assistant_phone': 'client.assistant_phone',
  'client.channel': 'client.channel',
  'client.codec': 'client.codec',
  'client.context_id': 'client.context_id',
  'client.direction': 'client.direction',
  'client.phone': 'client.phone',
  'client.provider_call_id': 'client.provider_call_id',
  'client.sample_rate': 'client.sample_rate',
  denoise_init_ms: 'denoise_init_ms',
  direction: 'direction',
  disconnect_reason: 'disconnect_reason',
  eos_init_ms: 'eos_init_ms',
  id: 'id',
  identifier: 'identifier',
  llm_init_ms: 'llm_init_ms',
  recording_init_ms: 'recording_init_ms',
  source: 'source',
  status: 'status',
  'stt.duration_ms': 'stt.duration_ms',
  stt_init_ms: 'stt_init_ms',
  timestamp: 'created_date',
  'tts.duration_ms': 'tts.duration_ms',
  tts_init_ms: 'tts_init_ms',
  vad_init_ms: 'vad_init_ms',
};

const SESSION_SEARCH_TABS = [
  { id: 'all', text: 'All' },
  { id: 'session', text: 'Session' },
  { id: 'client', text: 'Client' },
  { id: 'duration', text: 'Duration' },
  { id: 'metrics', text: 'Metrics' },
];

const DATE_ONLY_VALUE_PATTERN = /^\d{4}-\d{2}-\d{2}$/;

const getLocalDateBoundaryIso = (
  dateValue: string,
  boundary: 'start' | 'end',
): string => {
  const [year, month, day] = dateValue.split('-').map(Number);
  if (!year || !month || !day) return dateValue;

  return new Date(
    year,
    month - 1,
    day,
    boundary === 'end' ? 23 : 0,
    boundary === 'end' ? 59 : 0,
    boundary === 'end' ? 59 : 0,
    boundary === 'end' ? 999 : 0,
  ).toISOString();
};

const getTimestampCriteria = (
  value: string,
  logic: string,
): SessionSearchCriteria[] => {
  if (!DATE_ONLY_VALUE_PATTERN.test(value)) {
    return [{ k: 'created_date', logic, v: value }];
  }

  if (logic === '=') {
    return [
      {
        k: 'created_date',
        logic: '>=',
        v: getLocalDateBoundaryIso(value, 'start'),
      },
      {
        k: 'created_date',
        logic: '<=',
        v: getLocalDateBoundaryIso(value, 'end'),
      },
    ];
  }

  return [
    {
      k: 'created_date',
      logic,
      v: getLocalDateBoundaryIso(value, logic === '<=' ? 'end' : 'start'),
    },
  ];
};

export const getSessionSearchCriteria = (
  value: string,
): SessionSearchCriteria[] =>
  parseQuerySearchFilters(SESSION_SEARCH_FIELDS, value)
    .flatMap(filter => {
      const criteria = SESSION_SEARCH_CRITERIA[filter.key];
      if (!criteria || !filter.value.trim()) return [];

      if (filter.key === 'timestamp') {
        return getTimestampCriteria(filter.value, filter.logic);
      }

      return [
        {
          k: criteria,
          logic: filter.logic,
          v: filter.value,
        },
      ];
    })
    .filter((criteria): criteria is SessionSearchCriteria => Boolean(criteria));

export const SessionQuerySearch = ({
  onApply,
  onChange,
  value,
}: SessionQuerySearchProps) => (
  <QuerySearch
    dateTimeMode="local-to-utc"
    fields={SESSION_SEARCH_FIELDS}
    tabs={SESSION_SEARCH_TABS}
    value={value}
    maxOptions={SESSION_SEARCH_FIELDS.length}
    placeholder="Search for sessionID, client, metrics and more"
    preserveDateOnly
    onChange={onChange}
    onApply={nextValue => onApply(getSessionSearchCriteria(nextValue))}
  />
);
