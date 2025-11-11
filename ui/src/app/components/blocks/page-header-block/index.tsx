import React, { FC, HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

export const PageHeaderBlock: FC<HTMLAttributes<HTMLDivElement>> = props => {
  return (
    <div
      className={cn(
        'flex justify-between pl-4 bg-white dark:bg-gray-900 items-center min-h-10',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
};
