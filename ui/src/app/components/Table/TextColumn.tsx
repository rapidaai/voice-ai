import { TD } from '@/app/components/Table/TD';
import React, { HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

/**
 *
 * @param props
 * @returns
 */
export function TextColumn(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <TD>
      <span
        className={cn(
          'font-normal text-left max-w-[20rem] truncate',
          props.className,
        )}
      >
        {props.children}
      </span>
    </TD>
  );
}
