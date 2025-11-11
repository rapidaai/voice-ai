import { ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';
import moment from 'moment';
import { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb';

/*
 * Media queries utility
 */

/*
 * Inspired by https://github.com/DefinitelyTyped/DefinitelyTyped/issues/32914
 */

// Update your breakpoints if you want
export const sizes = {
  small: 600,
  medium: 1024,
  large: 1440,
  xlarge: 1920,
};

// Iterate through the sizes and create min-width media queries
export const media = (Object.keys(sizes) as Array<keyof typeof sizes>).reduce(
  (acc, size) => {
    acc[size] = () => `@media (min-width:${sizes[size]}px)`;
    return acc;
  },
  {} as { [key in keyof typeof sizes]: () => string },
);

/* Example
const SomeDiv = styled.div`
  display: flex;
  ....
  ${media.medium} {
    display: block
  }
`;
*/

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

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
  return toDate(timestamp).toUTCString();
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
