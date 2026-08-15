import { getSessionSearchCriteria } from './session-query-search';

const localDateBoundaryIso = (
  dateValue: string,
  boundary: 'start' | 'end',
): string => {
  const [year, month, day] = dateValue.split('-').map(Number);

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

describe('session query search criteria', () => {
  it('maps date-only timestamp is to the local day range', () => {
    expect(getSessionSearchCriteria('timestamp~=:2026-08-13')).toEqual([
      {
        k: 'created_date',
        logic: '>=',
        v: localDateBoundaryIso('2026-08-13', 'start'),
      },
      {
        k: 'created_date',
        logic: '<=',
        v: localDateBoundaryIso('2026-08-13', 'end'),
      },
    ]);
  });

  it('maps date-only timestamp boundaries to local day edges', () => {
    expect(getSessionSearchCriteria('timestamp:2026-08-13')).toEqual([
      {
        k: 'created_date',
        logic: '>=',
        v: localDateBoundaryIso('2026-08-13', 'start'),
      },
    ]);
    expect(getSessionSearchCriteria('timestamp~<=:2026-08-13')).toEqual([
      {
        k: 'created_date',
        logic: '<=',
        v: localDateBoundaryIso('2026-08-13', 'end'),
      },
    ]);
  });
});
