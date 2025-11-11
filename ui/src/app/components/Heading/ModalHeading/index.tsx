import React, { HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

export function ModalHeading(props: HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h1
      className={cn(
        'dark:text-gray-100 text-base font-semibold',
        props.className,
      )}
    >
      {props.children}
    </h1>
  );
}
