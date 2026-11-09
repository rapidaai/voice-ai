import {
  QuerySearch,
  parseQuerySearchFilters,
} from '@/app/components/ui/composites';
import type {
  QuerySearchField,
  QuerySearchOption,
} from '@/app/components/ui/composites';

export type OrganizationUserSearchFilter = {
  key: 'email' | 'name' | 'role' | 'status';
  logic: string;
  value: string;
};

type UserQuerySearchProps = {
  onApply: (filters: OrganizationUserSearchFilter[]) => void;
  onChange: (value: string) => void;
  value: string;
};

const ROLE_OPTIONS: QuerySearchOption[] = [
  { id: 'owner', text: 'owner' },
  { id: 'admin', text: 'admin' },
  { id: 'member', text: 'member' },
];

const STATUS_OPTIONS: QuerySearchOption[] = [
  { id: 'ACTIVE', text: 'active' },
  { id: 'INVITED', text: 'invited' },
];

const USER_SEARCH_FIELDS: QuerySearchField[] = [
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
    queryKey: 'email',
    text: 'email',
    type: 'string',
  },
  {
    items: ROLE_OPTIONS,
    queryKey: 'role',
    text: 'role',
    type: 'string',
  },
  {
    items: STATUS_OPTIONS,
    queryKey: 'status',
    text: 'status',
    type: 'string',
  },
];

export const getOrganizationUserSearchFilters = (
  value: string,
): OrganizationUserSearchFilter[] =>
  parseQuerySearchFilters(USER_SEARCH_FIELDS, value).map(filter => ({
    key: filter.key as OrganizationUserSearchFilter['key'],
    logic: filter.logic,
    value: filter.value,
  }));

export function UserQuerySearch({
  onApply,
  onChange,
  value,
}: UserQuerySearchProps) {
  return (
    <QuerySearch
      fields={USER_SEARCH_FIELDS}
      value={value}
      maxOptions={USER_SEARCH_FIELDS.length}
      placeholder="Search for name, email, role, status"
      onChange={onChange}
      onApply={nextValue =>
        onApply(getOrganizationUserSearchFilters(nextValue))
      }
    />
  );
}
