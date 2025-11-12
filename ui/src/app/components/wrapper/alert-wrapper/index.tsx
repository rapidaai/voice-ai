import React, { HTMLAttributes } from 'react';
import { cn } from '@/utils';

export function ErrorWrapper(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'flex p-2 border-red-600/30 dark:border-red-700/30',
        'bg-red-600/10 dark:bg-red-700/10',
        'items-start',
        'relative',
        'pr-8 py-4 space-x-2',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
}

export function SuccessWrapper(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'flex p-2 border-green-600/30 dark:border-green-700/30',
        'bg-green-600/10 dark:bg-green-700/10',
        'items-start',
        'relative',
        'pr-8 py-4 space-x-2',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
}

export function InfoWrapper(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'flex p-2 border-blue-600/30 dark:border-blue-700/30',
        'bg-blue-600/10 dark:bg-blue-700/10',
        'items-start',
        'relative',
        'pr-8 py-4 space-x-2',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
}

export function WarnWrapper(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'flex p-2 border-yellow-600/30 dark:border-yellow-700/30',
        'bg-yellow-600/10 dark:bg-yellow-700/10',
        'items-start',
        'relative',
        'pr-8 py-4 space-x-2',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
}

export function PlainWrapper(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'flex p-2 border-b border-t dark:border-gray-800',
        'items-start',
        'relative',
        'bg-gray-50 dark:bg-gray-800',
        'pr-8 py-4 space-x-2',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
}
