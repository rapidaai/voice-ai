import { TD } from '@/app/components/Table/Body';
import React from 'react';
import { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb';
import {
  toHumanReadableDate,
  toHumanReadableDateTime,
  toHumanReadableRelativeTime,
} from '@/styles/media';

/**
 *
 * @param props
 * @returns
 */
export function RelativeDateColumn(props: { date?: Timestamp }) {
  return (
    <TD>
      <div className="font-normal text-left underline decoration-dotted">
        {props.date && toHumanReadableRelativeTime(props.date)}
      </div>
    </TD>
  );
}

/**
 *
 * @param props
 * @returns
 */
export function DateColumn(props: { date?: Timestamp }) {
  return (
    <TD>
      <div className="font-normal text-left underline decoration-dotted">
        {props.date && toHumanReadableDate(props.date)}
      </div>
    </TD>
  );
}

/**
 *
 * @param props
 * @returns
 */
export function DateTimeColumn(props: { date?: Timestamp }) {
  return (
    <div className="font-normal text-left underline decoration-dotted">
      {props.date && toHumanReadableDateTime(props.date)}
    </div>
  );
}
