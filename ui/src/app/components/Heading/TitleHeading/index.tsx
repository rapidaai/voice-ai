import React, { HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

export function TitleHeading(props: HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h1 className={cn('text-xl font-semibold', props.className)}>
      {props.children}
    </h1>
  );
}
