import React from 'react';
import { cn } from '@/styles/media';

export function DashboardIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={cn('w-5 h-5 opacity-70', props.className)}
      {...props}
    >
      <path d="M15.6 2.7a10 10 0 1 0 5.7 5.7"></path>
      <circle cx="12" cy="12" r="2"></circle>
      <path d="M13.4 10.6 19 5"></path>
    </svg>
  );
}
