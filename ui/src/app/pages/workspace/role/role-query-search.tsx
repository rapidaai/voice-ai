import { RoleDefinitionType, RoleScope, RoleSubject } from '@rapidaai/react';
import {
  QuerySearch,
  parseQuerySearchFilters,
} from '@/app/components/ui/composites';
import type {
  QuerySearchField,
  QuerySearchOption,
} from '@/app/components/ui/composites';

export type RoleSearchState = {
  definitionType: keyof typeof RoleDefinitionType;
  query: string;
  scope: keyof typeof RoleScope;
  subject: keyof typeof RoleSubject;
};

type RoleQuerySearchProps = {
  onApply: (filters: RoleSearchState) => void;
  onChange: (value: string) => void;
  value: string;
};

const SCOPE_OPTIONS: QuerySearchOption[] = [
  { id: 'ROLE_SCOPE_ORGANIZATION', text: 'organization' },
  { id: 'ROLE_SCOPE_PROJECT', text: 'project' },
];

const SUBJECT_OPTIONS: QuerySearchOption[] = [
  { id: 'ROLE_SUBJECT_USER', text: 'user' },
  { id: 'ROLE_SUBJECT_PROJECT_TOKEN', text: 'project token' },
];

const TYPE_OPTIONS: QuerySearchOption[] = [
  { id: 'ROLE_DEFINITION_TYPE_BUILT_IN', text: 'built-in' },
  { id: 'ROLE_DEFINITION_TYPE_CUSTOM', text: 'custom' },
];

const ROLE_SEARCH_FIELDS: QuerySearchField[] = [
  {
    logicLabel: 'contains',
    logicOptions: [{ label: 'contains', logic: 'contains' }],
    queryKey: 'query',
    text: 'name',
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

export const getRoleSearchState = (value: string): RoleSearchState => {
  const state: RoleSearchState = {
    definitionType: 'ROLE_DEFINITION_TYPE_UNSPECIFIED',
    query: '',
    scope: 'ROLE_SCOPE_UNSPECIFIED',
    subject: 'ROLE_SUBJECT_UNSPECIFIED',
  };

  parseQuerySearchFilters(ROLE_SEARCH_FIELDS, value).forEach(filter => {
    if (filter.key === 'query') state.query = filter.value;
    if (
      filter.key === 'scope' &&
      SCOPE_OPTIONS.some(option => option.id === filter.value)
    ) {
      state.scope = filter.value as keyof typeof RoleScope;
    }
    if (
      filter.key === 'subject' &&
      SUBJECT_OPTIONS.some(option => option.id === filter.value)
    ) {
      state.subject = filter.value as keyof typeof RoleSubject;
    }
    if (
      filter.key === 'type' &&
      TYPE_OPTIONS.some(option => option.id === filter.value)
    ) {
      state.definitionType = filter.value as keyof typeof RoleDefinitionType;
    }
  });

  return state;
};

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
      placeholder="Search for name, scope, subject, type"
      onChange={onChange}
      onApply={nextValue => onApply(getRoleSearchState(nextValue))}
    />
  );
}
