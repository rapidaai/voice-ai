import {
  QuerySearch,
  parseQuerySearchFilters,
} from '@/app/components/ui/composites';
import type {
  QuerySearchField,
  QuerySearchOption,
} from '@/app/components/ui/composites';

export type OrganizationProjectSearchFilter = {
  key: 'collaborators' | 'name' | 'role' | 'status';
  logic: string;
  value: string;
};

type ProjectQuerySearchProps = {
  onApply: (filters: OrganizationProjectSearchFilter[]) => void;
  onChange: (value: string) => void;
  value: string;
};

const ROLE_OPTIONS: QuerySearchOption[] = [
  { id: 'super admin', text: 'super admin' },
  { id: 'admin', text: 'admin' },
  { id: 'writer', text: 'writer' },
  { id: 'reader', text: 'reader' },
];

const STATUS_OPTIONS: QuerySearchOption[] = [{ id: 'ACTIVE', text: 'active' }];

const PROJECT_SEARCH_FIELDS: QuerySearchField[] = [
  {
    logicLabel: 'contains',
    logicOptions: [{ label: 'contains', logic: 'contains' }],
    queryKey: 'name',
    text: 'name',
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
    logicOptions: [{ label: 'contains', logic: 'contains' }],
    queryKey: 'collaborators',
    text: 'collaborator',
    type: 'string',
  },
  {
    items: STATUS_OPTIONS,
    queryKey: 'status',
    text: 'status',
    type: 'string',
  },
];

export const getOrganizationProjectSearchFilters = (
  value: string,
): OrganizationProjectSearchFilter[] =>
  parseQuerySearchFilters(PROJECT_SEARCH_FIELDS, value).map(filter => ({
    key: filter.key as OrganizationProjectSearchFilter['key'],
    logic: filter.logic,
    value: filter.value,
  }));

export function ProjectQuerySearch({
  onApply,
  onChange,
  value,
}: ProjectQuerySearchProps) {
  return (
    <QuerySearch
      fields={PROJECT_SEARCH_FIELDS}
      value={value}
      maxOptions={PROJECT_SEARCH_FIELDS.length}
      placeholder="Search for name, role, collaborator, status"
      onChange={onChange}
      onApply={nextValue =>
        onApply(getOrganizationProjectSearchFilters(nextValue))
      }
    />
  );
}
