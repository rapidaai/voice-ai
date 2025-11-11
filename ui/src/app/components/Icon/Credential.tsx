import React from 'react';
import { cn } from '@/styles/media';

export function CredentialIcon(props: React.SVGProps<SVGSVGElement>) {
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
      <rect width="18" height="18" x="3" y="3" rx="2" />
      <path d="M12 8v8" />
      <path d="m8.5 14 7-4" />
      <path d="m8.5 10 7 4" />
    </svg>
  );
}
