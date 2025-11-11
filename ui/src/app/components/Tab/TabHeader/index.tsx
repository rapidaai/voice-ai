import { cn } from '@/styles/media';
import { HTMLAttributes } from 'react';

export function TabHeader(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn('border-b dark:border-gray-800', props.className)}>
      {props.children}
    </div>
  );
}
