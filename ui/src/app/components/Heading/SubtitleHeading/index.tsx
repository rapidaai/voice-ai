import React, { HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

export function SubtitleHeading(props: HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h1 className={cn('text-base', props.className)} {...props}>
      {props.children}
    </h1>
  );
}
