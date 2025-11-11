import React from 'react';
import { cn } from '@/styles/media';

export function TokenIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 256 256"
      fill="none"
      stroke="currentColor"
      className={cn('w-5 h-5 opacity-75', props.className)}
      {...props}
    >
      <path
        fill="currentColor"
        d="M208 56v32a8 8 0 0 1-16 0V64h-56v128h24a8 8 0 0 1 0 16H96a8 8 0 0 1 0-16h24V64H64v24a8 8 0 0 1-16 0V56a8 8 0 0 1 8-8h144a8 8 0 0 1 8 8"
      ></path>
    </svg>
  );
}
