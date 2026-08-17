import { QuerySearch } from '@/app/components/carbon/query-search';
import type {
  QuerySearchField,
  QuerySearchOption,
  QuerySearchTab,
} from '@/app/components/carbon/query-search';
import {
  ALL_EVENT_OPTIONS,
  COMPONENT_OPTIONS,
  KIND_OPTIONS,
  LEVEL_OPTIONS,
  METRIC_NAME_OPTIONS,
  ROLE_OPTIONS,
  SCOPE_OPTIONS,
} from '../constants';

type QueryFilterCategory = 'attributes' | 'date' | 'event' | 'record' | 'scope';

type TraceQuerySearchProps = {
  onApply: (value: string) => void;
  onChange: (value: string) => void;
  value: string;
};

const QUERY_FILTER_TABS: QuerySearchTab[] = [
  { id: 'all', text: 'All' },
  { id: 'scope', text: 'Scope' },
  { id: 'attributes', text: 'Attributes' },
  { id: 'record', text: 'Record' },
];

const field = (
  category: QueryFilterCategory,
  queryKey: string,
  text: string,
  type: QuerySearchField['type'],
  items?: QuerySearchOption[],
  logicLabel?: string,
  logicOptions?: QuerySearchField['logicOptions'],
): QuerySearchField => ({
  category,
  items: items?.filter(option => option.id !== 'all'),
  logicLabel,
  logicOptions,
  queryKey,
  text,
  type,
});

const QUERY_FILTER_FIELDS: QuerySearchField[] = [
  field('scope', 'conversation', 'conversation', 'number'),
  field(
    'attributes',
    'component',
    'component',
    'multi-select',
    COMPONENT_OPTIONS,
  ),
  field('event', 'event', 'event', 'string', ALL_EVENT_OPTIONS),
  field('scope', 'scope', 'scope', 'string', SCOPE_OPTIONS),
  field('scope', 'role', 'role', 'string', ROLE_OPTIONS),
  field('record', 'kind', 'kind', 'string', KIND_OPTIONS),
  field('record', 'level', 'level', 'string', LEVEL_OPTIONS),
  field('record', 'metric', 'metric', 'string', METRIC_NAME_OPTIONS),
  field('record', 'trace', 'trace', 'string'),
  field('scope', 'assistant', 'assistant', 'number'),
  field('scope', 'message', 'message', 'string'),
  field('attributes', 'attributes.component', 'attributes.component', 'string'),
  field('attributes', 'attributes.provider', 'attributes.provider', 'string'),
  field('attributes', 'context.traceId', 'context.traceId', 'string'),
  field('record', 'timestamp', 'timestamp', 'date', undefined, 'is after', [
    { label: 'is after', logic: '>=' },
    { label: 'is before', logic: '<=' },
    { label: 'is', logic: '=' },
  ]),
];

export const TraceQuerySearch = ({
  onApply,
  onChange,
  value,
}: TraceQuerySearchProps) => (
  <QuerySearch
    dateTimeMode="local-to-utc"
    fields={QUERY_FILTER_FIELDS}
    tabs={QUERY_FILTER_TABS}
    value={value}
    maxOptions={QUERY_FILTER_FIELDS.length}
    placeholder="Search for event, attribute, log and more"
    onChange={onChange}
    onApply={onApply}
  />
);
