import React from 'react';
import { cn } from '@/styles/media';

export function TD(props: React.TdHTMLAttributes<HTMLTableCellElement>) {
  return (
    <td
      {...props}
      className={cn('whitespace-no-wrap px-2 md:px-5 py-3', props.className)}
    >
      {props.children}
    </td>
  );
}
