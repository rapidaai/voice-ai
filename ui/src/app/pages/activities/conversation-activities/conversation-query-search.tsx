import {
  QuerySearch,
  parseQuerySearchFilters,
} from '@/app/components/carbon/query-search';
import type {
  QuerySearchField,
  QuerySearchOption,
} from '@/app/components/carbon/query-search';

type ConversationLogSearchCriteria = {
  k: string;
  logic: string;
  v: string;
};

type ConversationLogQuerySearchProps = {
  onApply: (criteria: ConversationLogSearchCriteria[]) => void;
  onChange: (value: string) => void;
  value: string;
};

const ROLE_OPTIONS: QuerySearchOption[] = [
  { id: 'assistant', text: 'assistant' },
  { id: 'user', text: 'user' },
];

const STATUS_OPTIONS: QuerySearchOption[] = [
  { id: 'IN_PROGRESS', text: 'in progress' },
  { id: 'COMPLETE', text: 'complete' },
  { id: 'FAILED', text: 'failed' },
];

const NUMBER_LOGIC_OPTIONS = [
  { label: 'is', logic: '=' },
  { label: 'is greater than or equal to', logic: '>=' },
  { label: 'is less than or equal to', logic: '<=' },
];

const CONVERSATION_LOG_SEARCH_FIELDS: QuerySearchField[] = [
  {
    queryKey: 'message_id',
    text: 'messageID',
    type: 'string',
  },
  {
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
    queryKey: 'assistant_conversation_id',
    text: 'sessionID',
    type: 'string',
  },
  {
    queryKey: 'assistant_id',
    text: 'assistantID',
    type: 'string',
  },
  {
    queryKey: 'source',
    text: 'source',
    type: 'string',
  },
  {
    items: ROLE_OPTIONS,
    queryKey: 'role',
    text: 'role',
    type: 'string',
  },
  {
    logicLabel: 'contains',
    logicOptions: [
      { label: 'contains', logic: 'contains' },
      { label: 'is', logic: '=' },
    ],
    queryKey: 'body',
    text: 'message',
    type: 'string',
  },
  {
    formatValue: value =>
      STATUS_OPTIONS.find(option => option.id === value)?.text || value,
    items: STATUS_OPTIONS,
    queryKey: 'status',
    text: 'status',
    type: 'string',
  },
  {
    logicLabel: 'is',
    logicOptions: NUMBER_LOGIC_OPTIONS,
    queryKey: 'stt.latency_ms',
    text: 'stt',
    type: 'number',
  },
  {
    logicLabel: 'is',
    logicOptions: NUMBER_LOGIC_OPTIONS,
    queryKey: 'agent.latency_ms',
    text: 'agent',
    type: 'number',
  },
  {
    logicLabel: 'is',
    logicOptions: NUMBER_LOGIC_OPTIONS,
    queryKey: 'agent.ttft_ms',
    text: 'ttft',
    type: 'number',
  },
  {
    logicLabel: 'is',
    logicOptions: NUMBER_LOGIC_OPTIONS,
    queryKey: 'tts.latency_ms',
    text: 'tts',
    type: 'number',
  },
  {
    logicLabel: 'is',
    logicOptions: NUMBER_LOGIC_OPTIONS,
    queryKey: 'eos.latency_ms',
    text: 'eos',
    type: 'number',
  },
  {
    logicLabel: 'is',
    logicOptions: NUMBER_LOGIC_OPTIONS,
    queryKey: 'agent.total_token',
    text: 'tokens',
    type: 'number',
  },
  {
    logicLabel: 'is',
    logicOptions: [
      { label: 'is', logic: '=' },
      { label: 'contains', logic: 'contains' },
    ],
    queryKey: 'language',
    text: 'language',
    type: 'string',
  },
];

const CONVERSATION_LOG_SEARCH_CRITERIA: Record<string, string> = {
  'agent.ttft_ms': 'agent.ttft_ms',
  'agent.total_token': 'agent.total_token',
  assistant_conversation_id: 'assistant_conversation_id',
  assistant_id: 'assistant_id',
  body: 'body',
  'agent.latency_ms': 'agent.latency_ms',
  'eos.latency_ms': 'eos.latency_ms',
  language: 'language',
  message_id: 'message_id',
  role: 'role',
  source: 'source',
  status: 'status',
  'stt.latency_ms': 'stt.latency_ms',
  timestamp: 'created_date',
  'tts.latency_ms': 'tts.latency_ms',
};

export const getConversationLogSearchCriteria = (
  value: string,
): ConversationLogSearchCriteria[] =>
  parseQuerySearchFilters(CONVERSATION_LOG_SEARCH_FIELDS, value)
    .map(filter => {
      const criteria = CONVERSATION_LOG_SEARCH_CRITERIA[filter.key];
      if (!criteria || !filter.value.trim()) return null;

      return {
        k: criteria,
        logic: filter.logic,
        v: filter.value,
      };
    })
    .filter(
      (criteria): criteria is ConversationLogSearchCriteria =>
        criteria !== null,
    );

export const ConversationLogQuerySearch = ({
  onApply,
  onChange,
  value,
}: ConversationLogQuerySearchProps) => (
  <QuerySearch
    dateTimeMode="local-to-utc"
    fields={CONVERSATION_LOG_SEARCH_FIELDS}
    value={value}
    maxOptions={CONVERSATION_LOG_SEARCH_FIELDS.length}
    placeholder="Search for messageID, sessionID, role, latency and more"
    onChange={onChange}
    onApply={nextValue => onApply(getConversationLogSearchCriteria(nextValue))}
  />
);
