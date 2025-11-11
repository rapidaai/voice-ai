import React, { FC, HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

export const PageTitleBlock: FC<HTMLAttributes<HTMLDivElement>> = props => {
  return (
    <div className={cn('font-medium', props.className)}>{props.children}</div>
  );
};
