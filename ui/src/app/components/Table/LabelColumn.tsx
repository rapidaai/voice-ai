import { TD } from '@/app/components/Table/TD';
import React, { HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

/**
 *
 * @param props
 * @returns
 */
export function LabelColumn(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <TD>
      <span
        className={cn(
          'text-center max-w-[20rem] truncate rounded-[2px] px-1.5 py-0.5 font-medium',
          props.className,
        )}
      >
        {props.children}
      </span>
    </TD>
  );
}
