import React, { HTMLAttributes } from 'react';
import { cn } from '@/utils';

export function BluredWrapper(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'flex justify-between items-center border-b border-t dark:border-gray-800 bg-white dark:bg-gray-900',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
}
