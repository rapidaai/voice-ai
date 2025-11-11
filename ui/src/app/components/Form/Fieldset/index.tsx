import React, { FieldsetHTMLAttributes } from 'react';
import { cn } from '@/styles/media';

/**
 *
 * @param props
 * @returns
 */
export function FieldSet(props: FieldsetHTMLAttributes<HTMLElement>) {
  return (
    <fieldset
      {...props}
      className={cn(
        'space-y-2 dark:space-y-2 flex flex-col min-w-0',
        props.className,
      )}
    >
      {props.children}
    </fieldset>
  );
}
