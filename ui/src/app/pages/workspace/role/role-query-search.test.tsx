import { getRoleSearchCriteria } from './role-query-search';

describe('getRoleSearchCriteria', () => {
  it('maps committed query filters to criteria', () => {
    expect(
      getRoleSearchCriteria(
        'name:support key:manager scope:PROJECT subject:USER type:CUSTOM',
      ),
    ).toEqual([
      { key: 'name', logic: 'contains', value: 'support' },
      { key: 'key', logic: 'contains', value: 'manager' },
      { key: 'scope', logic: '=', value: 'PROJECT' },
      { key: 'subject', logic: '=', value: 'USER' },
      { key: 'type', logic: '=', value: 'CUSTOM' },
    ]);
  });

  it('ignores unsupported enum values', () => {
    expect(
      getRoleSearchCriteria('scope:invalid subject:invalid type:invalid'),
    ).toEqual([]);
  });
});
