import React, { HTMLAttributes } from 'react';
import { cn } from '@/utils';
import { Spinner } from '@/app/components/loader/spinner';

export function SectionLoader(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'flex items-center justify-center relative h-10 w-10',
        props.className,
      )}
      {...props}
    >
      <Spinner className="absolute z-50 top-0 left-0 w-full h-full border-[4.5px] animate-spin" />
    </div>
  );
}
