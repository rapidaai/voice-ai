import React, { HTMLAttributes } from 'react';
import { cn } from '@/utils';
import { Spinner } from '@/app/components/loader/spinner';

export function PageLoader(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'fixed top-0 bottom-0 right-0 left-0 flex justify-center items-center z-999999 backdrop-blur-xs',
        props.className,
      )}
      {...props}
    >
      <div className="p-2 flex items-center justify-center">
        <div className="relative p-4">
          <Spinner className="absolute z-50 w-9 h-9 border-[3.5px]" />
        </div>
      </div>
    </div>
  );
}
