import type { ReactNode } from 'react';
import { SkeletonText } from '@carbon/react';
import { cn } from '@/utils';

export const typographyClassNames = {
  'code-01':
    'font-mono text-[0.75rem] font-normal leading-[1rem] tracking-[0.32px]',
  'code-02':
    'font-mono text-[0.875rem] font-normal leading-[1.25rem] tracking-[0.32px]',
  'label-01':
    'font-sans text-[0.75rem] font-normal leading-[1rem] tracking-[0.32px]',
  'label-02':
    'font-sans text-[0.875rem] font-normal leading-[1.125rem] tracking-[0.16px]',
  'helper-text-01':
    'font-sans text-[0.75rem] font-normal leading-[1rem] tracking-[0.32px]',
  'helper-text-02':
    'font-sans text-[0.875rem] font-normal leading-[1.125rem] tracking-[0.16px]',
  'legal-01':
    'font-sans text-[0.75rem] font-normal leading-[1rem] tracking-[0.32px]',
  'legal-02':
    'font-sans text-[0.875rem] font-normal leading-[1.125rem] tracking-[0.16px]',
  'body-compact-01':
    'font-sans text-[0.875rem] font-normal leading-[1.125rem] tracking-[0.16px]',
  'body-compact-02':
    'font-sans text-[1rem] font-normal leading-[1.375rem] tracking-[0]',
  'body-01':
    'font-sans text-[0.875rem] font-normal leading-[1.25rem] tracking-[0.16px]',
  'body-02': 'font-sans text-[1rem] font-normal leading-[1.5rem] tracking-[0]',
  'heading-compact-01':
    'font-sans text-[0.875rem] font-[600] leading-[1.125rem] tracking-[0.16px]',
  'heading-compact-02':
    'font-sans text-[1rem] font-[600] leading-[1.375rem] tracking-[0]',
  'heading-01':
    'font-sans text-[0.875rem] font-[600] leading-[1.25rem] tracking-[0.16px]',
  'heading-02':
    'font-sans text-[1rem] font-[600] leading-[1.5rem] tracking-[0]',
  'heading-03':
    'font-sans text-[1.25rem] font-normal leading-[1.75rem] tracking-[0]',
  'heading-04':
    'font-sans text-[1.75rem] font-normal leading-[2.25rem] tracking-[0]',
  'heading-05':
    'font-sans text-[2rem] font-normal leading-[2.5rem] tracking-[0]',
  'heading-06':
    'font-sans text-[2.625rem] font-light leading-[3.125rem] tracking-[0]',
  'heading-07':
    'font-sans text-[3.375rem] font-light leading-[4rem] tracking-[0]',
  pageTitle: 'text-2xl font-semibold leading-8',
  sectionTitle: 'text-lg font-semibold leading-6',
  componentTitle: 'text-sm font-semibold leading-5',
  body: 'text-sm font-normal leading-5',
  label: 'text-sm font-medium leading-5',
  helper: 'text-xs font-normal leading-4',
  metric: 'text-4xl font-light leading-none tracking-tight tabular-nums',
} as const;

export type TypographyVariant = keyof typeof typographyClassNames;

export interface CarbonTextProps {
  children: ReactNode;
  className?: string;
  id?: string;
  isLoading?: boolean;
  /** Show skeleton as a heading block (taller). */
  heading?: boolean;
  /** Width of the skeleton placeholder (e.g. "60%", "200px"). */
  skeletonWidth?: string;
  /** Number of skeleton lines when paragraph mode. */
  lineCount?: number;
  /** Semantic type style shared across product surfaces. */
  variant?: TypographyVariant;
  /** Render as a specific HTML element. Default: span. */
  as?:
    | 'span'
    | 'p'
    | 'h1'
    | 'h2'
    | 'h3'
    | 'h4'
    | 'h5'
    | 'h6'
    | 'div'
    | 'label'
    | 'dt'
    | 'dd';
}

/** Carbon Text: renders children or a SkeletonText placeholder when loading. */
export function Text({
  children,
  className,
  id,
  isLoading = false,
  heading = false,
  skeletonWidth = '75%',
  lineCount = 1,
  variant,
  as: Tag = 'span',
}: CarbonTextProps) {
  const textClassName =
    variant || className
      ? cn(variant ? typographyClassNames[variant] : undefined, className)
      : undefined;

  if (isLoading) {
    return (
      <SkeletonText
        heading={heading}
        width={skeletonWidth}
        lineCount={lineCount}
        paragraph={lineCount > 1}
        className={cn('!mb-0', textClassName)}
      />
    );
  }

  return (
    <Tag id={id} className={textClassName}>
      {children}
    </Tag>
  );
}
