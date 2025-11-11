import React, { HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

export function SidebarLabel(props: HTMLAttributes<HTMLSpanElement>) {
  return (
    <span
      className={cn(
        'text-sm truncate inline-block opacity-0 group-hover:opacity-100 transition-all duration-200',
        props.className,
      )}
      {...props}
    >
      {props.children}
    </span>
  );
}
