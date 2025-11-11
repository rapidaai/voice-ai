import { TD } from '@/app/components/Table/TD';
import React, { HTMLAttributes } from 'react';

/**
 *
 * @param props
 * @returns
 */
export function LatencyColumn(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <TD className="text-center">
      <span className="font-medium text-left max-w-[20rem] truncate">
        {props.children} ms
      </span>
    </TD>
  );
}
