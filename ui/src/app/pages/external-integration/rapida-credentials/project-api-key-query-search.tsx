import {
  QuerySearch,
  parseQuerySearchFilters,
} from '@/app/components/ui/composites';
import type { QuerySearchField } from '@/app/components/ui/composites';

export type ProjectAPIKeySearchCriteria = {
  key: 'name' | 'role';
  logic: 'contains';
  value: string;
};

type ProjectAPIKeyQuerySearchProps = {
  onApply: (criteria: ProjectAPIKeySearchCriteria[]) => void;
  onChange: (value: string) => void;
  value: string;
};

const PROJECT_API_KEY_SEARCH_FIELDS: QuerySearchField[] = [
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
    queryKey: 'role',
    text: 'role',
    type: 'string',
  },
];

export const getProjectAPIKeySearchCriteria = (
  value: string,
): ProjectAPIKeySearchCriteria[] =>
  parseQuerySearchFilters(PROJECT_API_KEY_SEARCH_FIELDS, value).map(filter => ({
    key: filter.key as ProjectAPIKeySearchCriteria['key'],
    logic: 'contains',
    value: filter.value,
  }));

export function ProjectAPIKeyQuerySearch({
  onApply,
  onChange,
  value,
}: ProjectAPIKeyQuerySearchProps) {
  return (
    <QuerySearch
      fields={PROJECT_API_KEY_SEARCH_FIELDS}
      value={value}
      maxOptions={PROJECT_API_KEY_SEARCH_FIELDS.length}
      placeholder="Search for name or role"
      onChange={onChange}
      onApply={nextValue => onApply(getProjectAPIKeySearchCriteria(nextValue))}
    />
  );
}
