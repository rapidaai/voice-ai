import { cn } from '@/utils';
import type { HTMLAttributes, ReactNode } from 'react';

interface NoticeBlockFrameProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
  toneClassName: string;
}

const NoticeBlockFrame = ({
  className,
  children,
  toneClassName,
  role = 'status',
  ...attributes
}: NoticeBlockFrameProps) => (
  <div
    {...attributes}
    role={role}
    className={cn(
      'border-0 border-l-4 px-4 py-3 text-sm text-foreground',
      toneClassName,
      className,
    )}
  >
    {children}
  </div>
);

export function BlueNoticeBlock({
  className,
  children,
  ...attributes
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <NoticeBlockFrame
      {...attributes}
      className={className}
      toneClassName="border-l-support-info bg-support-info/10"
    >
      {children}
    </NoticeBlockFrame>
  );
}

export function GreenNoticeBlock({
  className,
  children,
  ...attributes
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <NoticeBlockFrame
      {...attributes}
      className={className}
      toneClassName="border-l-support-success bg-support-success/10"
    >
      {children}
    </NoticeBlockFrame>
  );
}

export function RedNoticeBlock({
  className,
  children,
  ...attributes
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <NoticeBlockFrame
      {...attributes}
      className={className}
      role="alert"
      toneClassName="border-l-support-error bg-support-error/10"
    >
      {children}
    </NoticeBlockFrame>
  );
}

export function YellowNoticeBlock({
  className,
  children,
  ...attributes
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <NoticeBlockFrame
      {...attributes}
      className={className}
      toneClassName="border-l-support-warning bg-support-warning/10"
    >
      {children}
    </NoticeBlockFrame>
  );
}
