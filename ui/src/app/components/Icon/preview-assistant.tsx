import React from 'react';
import { cn } from '@/styles/media';

export function AssistantPreviewIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      {...props}
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={cn('w-5 h-5', props.className)}
    >
      <path d="M10 9.5 8 12l2 2.5" />
      <path d="m14 9.5 2 2.5-2 2.5" />
      <path d="M7.9 20A9 9 0 1 0 4 16.1L2 22z" />
    </svg>
  );
}
