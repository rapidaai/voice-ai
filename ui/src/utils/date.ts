import moment from 'moment';
import { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb';

export function formatDateWithMillisecond(date: Date): string {
  const pad = (n: number, z = 2) => n.toString().padStart(z, '0');
  const day = pad(date.getDate());
  const month = pad(date.getMonth() + 1); // Month is 0-indexed
  const year = date.getFullYear();
  const hours = pad(date.getHours());
  const minutes = pad(date.getMinutes());
  const seconds = pad(date.getSeconds());
  const milliseconds = pad(date.getMilliseconds(), 3);

  return `${day}/${month}/${year}, ${hours}:${minutes}:${seconds}.${milliseconds}`;
}
export function toDateWithMicroseconds(timestamp: Timestamp): {
  date: Date;
  microseconds: number;
} {
  const seconds = timestamp.getSeconds();
  const nanos = timestamp.getNanos();

  const totalMilliseconds = seconds * 1000 + Math.floor(nanos / 1e6);
  const utcDate = new Date(totalMilliseconds);

  // safer: avoid rounding into the next millisecond
  const microseconds = Math.floor((nanos % 1e6) / 1e3);

  return { date: utcDate, microseconds };
}
export function toDate(timestamp: Timestamp): Date {
  // Extract seconds and nanos from gRPC Timestamp
  const seconds = timestamp.getSeconds();
  const nanos = timestamp.getNanos();

  // Calculate milliseconds since Unix epoch
  // const milliseconds = seconds * 1000 + Math.round(nanos / 1e6);

  // // Create Moment.js object from milliseconds
  // return moment.utc(milliseconds).toDate();

  // // Extract seconds and nanos from the gRPC timestamp
  // const { seconds, nanos } = timestamp;

  // Convert seconds to milliseconds
  const millisecondsFromSeconds = seconds * 1000;

  // Convert nanos to milliseconds
  const millisecondsFromNanos = nanos / 1000000;

  // Combine the two to get the total milliseconds
  const totalMilliseconds = millisecondsFromSeconds + millisecondsFromNanos;

  // Create a new Date object using the total milliseconds (interpreted as UTC)
  const utcDate = new Date(totalMilliseconds);

  // The Date object automatically handles conversion to local time
  return utcDate;
}

export function toHumanReadableDate(timestamp: Timestamp): string {
  return toHumanReadableDateFromDate(toDate(timestamp));
}

export function toHumanReadableDateFromDate(date: Date): string {
  const options = {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  };

  return date.toLocaleDateString(
    'en-US',
    options as Intl.DateTimeFormatOptions,
  );
}

export function toHumanReadableDateTime(timestamp: Timestamp): string {
  return toHumanReadableDateTimeFromDate(toDate(timestamp));
}

export function toHumanReadableDateTimeFromDate(date: Date): string {
  const parts = new Intl.DateTimeFormat('en-GB', {
    weekday: 'short',
    year: 'numeric',
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).formatToParts(date);
  const value = (type: Intl.DateTimeFormatPartTypes) =>
    parts.find(part => part.type === type)?.value ?? '';

  return `${value('weekday')}, ${value('day')} ${value('month')} ${value(
    'year',
  )} ${value('hour')}:${value('minute')}:${value('second')}`;
}

export function toHumanReadableRelativeTime(timestamp: Timestamp): string {
  return moment(toDate(timestamp).toUTCString()).fromNow();
}

export function toHumanReadableRelativeTimeFromDate(date: Date): string {
  return moment(date.toUTCString()).fromNow();
}

export function daysAgoFromTimestamp(timestamp: Timestamp): number {
  const givenDate = moment(toDate(timestamp).toUTCString());
  const today = moment().utc();
  return today.diff(givenDate, 'days');
}

export function toHumanReadableRelativeDay(timestamp: Timestamp): string {
  const daysAgo = daysAgoFromTimestamp(timestamp);
  if (daysAgo === 0) {
    return 'today';
  } else if (daysAgo === 1) {
    return 'yesterday';
  } else {
    return `${daysAgo} days ago`;
  }
}

export function getTimeFromDate(timestamp: Timestamp): string {
  const hours = toDate(timestamp).getHours().toString().padStart(2, '0');
  const minutes = toDate(timestamp).getMinutes().toString().padStart(2, '0');
  return `${hours}:${minutes}`;
}

/**
 * Converts nanoseconds to milliseconds.
 * @param nano - The number of nanoseconds or undefined.
 * @returns The equivalent number of milliseconds, or undefined if input is undefined.
 */
export function nanoToMilli(
  nano: number | string | undefined,
): number | undefined {
  if (nano === undefined) return undefined;
  const nanoNumber = typeof nano === 'string' ? parseFloat(nano) : nano;
  return Number((nanoNumber / 1_000_000).toFixed(2));
}

export function nanoToMinute(
  nano: number | string | undefined,
): number | undefined {
  if (nano === undefined) return undefined;
  const nanoNumber = typeof nano === 'string' ? parseFloat(nano) : nano;
  return Number((nanoNumber / 60_000_000_000).toFixed(2));
}

export function formatNanoToReadableMinute(
  nano: number | string | undefined,
): string {
  if (nano === undefined || isNaN(Number(nano))) {
    return 'n/a';
  }

  const nanoNumber = typeof nano === 'string' ? parseFloat(nano) : nano;
  const totalSeconds = Math.floor(nanoNumber / 1_000_000_000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  if (totalSeconds <= 0) {
    return 'n/a';
  }

  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  } else {
    return `${seconds}s`;
  }
}
export function formatNanoToReadableMilli(
  nano: number | string | undefined,
  fraction = 2,
): string {
  if (nano === undefined || isNaN(Number(nano))) {
    return 'n/a';
  }

  const nanoNumber = typeof nano === 'string' ? parseFloat(nano) : nano;
  const totalMilliSeconds = nanoNumber / 1_000_000; // No Math.floor, keep fractions

  if (totalMilliSeconds <= 0) {
    return 'n/a';
  }

  return `${totalMilliSeconds.toFixed(fraction)} ms`; // ToFixed ensures readability for fractions
}

export const toDateString = (d: Date) => {
  const postgresDateString = d.toLocaleDateString('en-CA');
  return postgresDateString;
};
