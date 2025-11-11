import { cn } from '@/styles/media';
import React, { HTMLAttributes } from 'react';

export function TabBody(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'bg-transparent flex-1 grow h-full w-full',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
}
