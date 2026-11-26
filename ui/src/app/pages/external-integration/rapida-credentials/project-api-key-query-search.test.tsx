import { getProjectAPIKeySearchCriteria } from './project-api-key-query-search';

describe('getProjectAPIKeySearchCriteria', () => {
  it('maps name and role filters to criteria', () => {
    expect(getProjectAPIKeySearchCriteria('name:widget role:runtime')).toEqual([
      { key: 'name', logic: 'contains', value: 'widget' },
      { key: 'role', logic: 'contains', value: 'runtime' },
    ]);
  });

  it('ignores unsupported filters', () => {
    expect(getProjectAPIKeySearchCriteria('key:secret')).toEqual([]);
  });
});
