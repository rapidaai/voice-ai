import React, { HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

export function LoadingPill(props: HTMLAttributes<HTMLSpanElement>) {
  return (
    <div className="card pill-box-animator relative overflow-hidden flex items-center justify-center rounded-[2px]">
      <div className="inner relative bg-gray-100 dark:bg-gray-950  rounded-[2px] m-[2px]">
        {props.children}
      </div>
    </div>
  );
}
