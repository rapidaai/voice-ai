import React, { HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

export function InfoHeading(props: HTMLAttributes<HTMLSpanElement>) {
  return (
    <span className={cn('text-base ml-2', props.className)}>
      {props.children}
    </span>
  );
}
