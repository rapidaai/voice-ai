import type { HTMLAttributes } from 'react';
import { cn } from '@/utils';

interface CenterBoxProps extends HTMLAttributes<HTMLDivElement> {
  mainLabel?: string;
}

export function CenterBox({
  children,
  className,
  mainLabel,
  ...attributes
}: CenterBoxProps) {
  return (
    <div
      {...attributes}
      className={cn(
        'grid min-h-dvh grid-cols-[0_0_minmax(0,var(--container-2xl))_0_0] grid-rows-[1fr_auto_1fr] overflow-clip bg-surface text-foreground sm:grid-cols-[1fr_2.5rem_minmax(0,var(--container-2xl))_2.5rem_1fr]',
        className,
      )}
    >
      <div className="col-start-2 row-span-full row-start-1 border-x border-border-subtle" />
      <div className="col-start-4 row-span-full border-x border-border-subtle" />
      <main
        aria-label={mainLabel}
        className="col-start-3 row-start-2 grid grid-cols-1 border-y border-border-subtle"
      >
        <div className="grid! grid-cols-1! items-center! bg-surface p-4! sm:p-10!">
          <div className="grid w-full grid-cols-1 gap-10">{children}</div>
        </div>
      </main>
    </div>
  );
}
