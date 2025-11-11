import React from 'react';
import { cn } from '@/styles/media';

export function TextSearchIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      {...props}
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={cn('w-5 h-5 opacity-70', props.className)}
    >
      <path d="M21 6H3" />
      <path d="M10 12H3" />
      <path d="M10 18H3" />
      <circle cx="17" cy="15" r="3" />
      <path d="m21 19-1.9-1.9" />
    </svg>
  );
}
