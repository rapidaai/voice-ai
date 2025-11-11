import React, { FC, HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

export const TableSection: FC<HTMLAttributes<HTMLDivElement>> = props => {
  return (
    <div
      className={cn(
        'bg-light-background dark:bg-gray-900 flex-1 flex flex-col',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
};
