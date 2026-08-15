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
  it('maps millisecond duration metric filters', () => {
    expect(
      getSessionSearchCriteria('conversation.duration_ms~>=:1000'),
    ).toEqual([
      {
        k: 'conversation.duration_ms',
        logic: '>=',
        v: '1000',
      },
    ]);
    expect(getSessionSearchCriteria('stt.duration_ms~<=:250')).toEqual([
      {
        k: 'stt.duration_ms',
        logic: '<=',
        v: '250',
      },
    ]);
    expect(getSessionSearchCriteria('tts.duration_ms:500')).toEqual([
      {
        k: 'tts.duration_ms',
        logic: '=',
        v: '500',
      },
    ]);
  });

  it('maps recording init metric filter', () => {
    expect(getSessionSearchCriteria('recording.init_ms~>=:120')).toEqual([
      {
        k: 'recording.init_ms',
        logic: '>=',
        v: '120',
      },
    ]);

    expect(getSessionSearchCriteria('authentication.init_ms~<=:80')).toEqual([
      {
        k: 'authentication.init_ms',
        logic: '<=',
        v: '80',
      },
    ]);
    expect(
      getSessionSearchCriteria('authentication.latency_ms~>=:120'),
    ).toEqual([
      {
        k: 'authentication.latency_ms',
        logic: '>=',
        v: '120',
      },
    ]);
    expect(getSessionSearchCriteria('analysis.init_ms~<=:40')).toEqual([
      {
        k: 'analysis.init_ms',
        logic: '<=',
        v: '40',
      },
    ]);
    expect(getSessionSearchCriteria('storage.init_ms~>=:60')).toEqual([
      {
        k: 'storage.init_ms',
        logic: '>=',
        v: '60',
      },
    ]);
    expect(getSessionSearchCriteria('denoise.init_ms~<=:25')).toEqual([
      {
        k: 'denoise.init_ms',
        logic: '<=',
        v: '25',
      },
    ]);
    expect(getSessionSearchCriteria('eos.init_ms~<=:30')).toEqual([
      {
        k: 'eos.init_ms',
        logic: '<=',
        v: '30',
      },
    ]);
    expect(getSessionSearchCriteria('vad.init_ms~>=:35')).toEqual([
      {
        k: 'vad.init_ms',
        logic: '>=',
        v: '35',
      },
    ]);
  });

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
