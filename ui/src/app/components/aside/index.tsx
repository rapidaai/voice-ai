import { useSidebar } from '@/context/sidebar-context';
import { cn } from '@/utils';
import { FC, HTMLAttributes } from 'react';

interface AsideProps extends HTMLAttributes<HTMLDivElement> {}
export const Aside: FC<AsideProps> = (props: AsideProps) => {
  const { open, setOpen } = useSidebar();
  return (
    <div
      className={cn(
        'flex flex-col shrink-0 z-12',
        'bg-shell text-foreground',
        'border-r border-border-subtle',
        'no-scrollbar overflow-y-auto',
        'group',
        open ? 'w-64' : 'w-12',
        'h-full transition-[width] duration-200 ease-in-out',
        props.className,
      )}
      onMouseEnter={() => {
        setOpen(true);
      }}
      onMouseLeave={() => {
        setOpen(false);
      }}
    >
      {props.children}
    </div>
  );
};
