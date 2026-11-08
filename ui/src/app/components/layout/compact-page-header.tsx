import type { HTMLAttributes } from 'react';
import { cn } from '@/utils';

export function CompactPageHeader({
  className,
  children,
  ...attributes
}: HTMLAttributes<HTMLElement>) {
  return (
    <header
      {...attributes}
      className={cn(
        'flex h-12 shrink-0 items-center justify-between border-b border-border-subtle bg-shell pl-4 pr-0',
        className,
      )}
    >
      {children}
    </header>
  );
}
