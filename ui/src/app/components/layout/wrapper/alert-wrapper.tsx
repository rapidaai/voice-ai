import type { HTMLAttributes } from 'react';
import { cn } from '@/utils';

const wrapperBaseClassName =
  'flex items-start relative border border-l-4 bg-layer px-2 py-4 pr-8 text-foreground space-x-2';

export function ErrorWrapper({
  className,
  children,
  ...attributes
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      {...attributes}
      className={cn(
        wrapperBaseClassName,
        'border-border-subtle border-l-support-error',
        className,
      )}
    >
      {children}
    </div>
  );
}

export function SuccessWrapper({
  className,
  children,
  ...attributes
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      {...attributes}
      className={cn(
        wrapperBaseClassName,
        'border-border-subtle border-l-support-success',
        className,
      )}
    >
      {children}
    </div>
  );
}

export function InfoWrapper({
  className,
  children,
  ...attributes
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      {...attributes}
      className={cn(
        wrapperBaseClassName,
        'border-border-subtle border-l-support-info',
        className,
      )}
    >
      {children}
    </div>
  );
}

export function WarnWrapper({
  className,
  children,
  ...attributes
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      {...attributes}
      className={cn(
        wrapperBaseClassName,
        'border-border-subtle border-l-support-warning',
        className,
      )}
    >
      {children}
    </div>
  );
}

export function PlainWrapper({
  className,
  children,
  ...attributes
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      {...attributes}
      className={cn(
        'flex items-start relative border-y border-border-subtle bg-layer px-2 py-4 pr-8 text-foreground space-x-2',
        className,
      )}
    >
      {children}
    </div>
  );
}
