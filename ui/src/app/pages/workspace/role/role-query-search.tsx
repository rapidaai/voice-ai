import {
  QuerySearch,
  parseQuerySearchFilters,
} from '@/app/components/ui/composites';
import type {
  QuerySearchField,
  QuerySearchOption,
} from '@/app/components/ui/composites';

export type RoleSearchCriteria = {
  key: 'key' | 'name' | 'scope' | 'subject' | 'type';
  logic: string;
  value: string;
};

type RoleQuerySearchProps = {
  onApply: (filters: RoleSearchCriteria[]) => void;
  onChange: (value: string) => void;
  value: string;
};

const SCOPE_OPTIONS: QuerySearchOption[] = [
  { id: 'ORGANIZATION', text: 'organization' },
  { id: 'PROJECT', text: 'project' },
];

const SUBJECT_OPTIONS: QuerySearchOption[] = [
  { id: 'USER', text: 'user' },
  { id: 'PROJECT_TOKEN', text: 'project token' },
];

const TYPE_OPTIONS: QuerySearchOption[] = [
  { id: 'BUILT_IN', text: 'built-in' },
  { id: 'CUSTOM', text: 'custom' },
];

const ROLE_SEARCH_FIELDS: QuerySearchField[] = [
  {
    logicLabel: 'contains',
    logicOptions: [{ label: 'contains', logic: 'contains' }],
    queryKey: 'name',
    text: 'name',
    type: 'string',
  },
  {
    logicLabel: 'contains',
    logicOptions: [{ label: 'contains', logic: 'contains' }],
    queryKey: 'key',
    text: 'key',
    type: 'string',
  },
  {
    items: SCOPE_OPTIONS,
    queryKey: 'scope',
    text: 'scope',
    type: 'string',
  },
  {
    items: SUBJECT_OPTIONS,
    queryKey: 'subject',
    text: 'subject',
    type: 'string',
  },
  {
    items: TYPE_OPTIONS,
    queryKey: 'type',
    text: 'type',
    type: 'string',
  },
];

export const getRoleSearchCriteria = (value: string): RoleSearchCriteria[] =>
  parseQuerySearchFilters(ROLE_SEARCH_FIELDS, value)
    .filter(filter => {
      if (filter.key === 'name' || filter.key === 'key') return true;
      if (filter.key === 'scope')
        return SCOPE_OPTIONS.some(option => option.id === filter.value);
      if (filter.key === 'subject')
        return SUBJECT_OPTIONS.some(option => option.id === filter.value);
      if (filter.key === 'type')
        return TYPE_OPTIONS.some(option => option.id === filter.value);
      return false;
    })
    .map(filter => ({
      key: filter.key as RoleSearchCriteria['key'],
      logic: filter.logic,
      value: filter.value,
    }));

export function RoleQuerySearch({
  onApply,
  onChange,
  value,
}: RoleQuerySearchProps) {
  return (
    <QuerySearch
      fields={ROLE_SEARCH_FIELDS}
      value={value}
      maxOptions={ROLE_SEARCH_FIELDS.length}
      placeholder="Search for name, key, scope, subject, type"
      onChange={onChange}
      onApply={nextValue => onApply(getRoleSearchCriteria(nextValue))}
    />
  );
}
