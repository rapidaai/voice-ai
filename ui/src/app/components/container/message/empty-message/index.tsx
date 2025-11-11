import { HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

export function EmptyMessage(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'bg-gray-300/20 rounded-lg dark:bg-gray-400/20 backdrop-blur-sm px-4 py-4 m-4 space-y-0.5',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
}
