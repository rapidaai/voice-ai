import { useSidebar } from '@/context/sidebar-context';
import { cn } from '@/utils';
import type { HTMLAttributes, MouseEvent } from 'react';

export interface AsideProps extends HTMLAttributes<HTMLElement> {}

export function Aside({
  className,
  children,
  onMouseEnter,
  onMouseLeave,
  ...attributes
}: AsideProps) {
  const { open, setOpen } = useSidebar();

  const handleMouseEnter = (event: MouseEvent<HTMLElement>) => {
    onMouseEnter?.(event);
    setOpen(true);
  };

  const handleMouseLeave = (event: MouseEvent<HTMLElement>) => {
    onMouseLeave?.(event);
    setOpen(false);
  };

  return (
    <aside
      {...attributes}
      className={cn(
        'flex flex-col shrink-0 z-12',
        'bg-shell text-foreground',
        'border-r border-border-subtle',
        'no-scrollbar overflow-y-auto overflow-x-hidden',
        'group',
        open
          ? 'absolute inset-y-0 left-0 w-64 shadow-lg md:relative md:inset-auto md:shadow-none'
          : 'relative w-12',
        'h-full transition-[width] duration-200 ease-in-out',
        className,
      )}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      {children}
    </aside>
  );
}
