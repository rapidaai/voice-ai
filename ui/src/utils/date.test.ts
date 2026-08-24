import { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb';
import {
  toHumanReadableDateTime,
  toHumanReadableDateTimeFromDate,
} from './date';

describe('toHumanReadableDateTime', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('formats timestamps with the table date-time display contract', () => {
    jest
      .spyOn(Intl.DateTimeFormat.prototype, 'formatToParts')
      .mockReturnValue([
        { type: 'weekday', value: 'Mon' },
        { type: 'literal', value: ', ' },
        { type: 'day', value: '24' },
        { type: 'literal', value: ' ' },
        { type: 'month', value: 'Aug' },
        { type: 'literal', value: ' ' },
        { type: 'year', value: '2026' },
        { type: 'literal', value: ', ' },
        { type: 'hour', value: '16' },
        { type: 'literal', value: ':' },
        { type: 'minute', value: '24' },
        { type: 'literal', value: ':' },
        { type: 'second', value: '11' },
      ]);
    const timestamp = new Timestamp();
    timestamp.setSeconds(1_725_000_000);

    expect(toHumanReadableDateTime(timestamp)).toBe(
      'Mon, 24 Aug 2026 16:24:11',
    );
  });

  it('preserves fractional timestamp precision before formatting', () => {
    let formattedTime = 0;
    jest
      .spyOn(Intl.DateTimeFormat.prototype, 'formatToParts')
      .mockImplementation(date => {
        formattedTime = (date as Date).getTime();
        return [];
      });
    const timestamp = new Timestamp();
    timestamp.setSeconds(1);
    timestamp.setNanos(500_000_000);

    toHumanReadableDateTime(timestamp);

    expect(formattedTime).toBe(1_500);
  });

  it('uses the same display contract for Date values', () => {
    jest
      .spyOn(Intl.DateTimeFormat.prototype, 'formatToParts')
      .mockReturnValue([
        { type: 'weekday', value: 'Mon' },
        { type: 'day', value: '24' },
        { type: 'month', value: 'Aug' },
        { type: 'year', value: '2026' },
        { type: 'hour', value: '16' },
        { type: 'minute', value: '24' },
        { type: 'second', value: '11' },
      ]);

    expect(toHumanReadableDateTimeFromDate(new Date())).toBe(
      'Mon, 24 Aug 2026 16:24:11',
    );
  });
});
