import type { ReactNode } from 'react';
import { SkeletonText } from '@carbon/react';
import { cn } from '@/utils';

export const typographyClassNames = {
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
  as?: 'span' | 'p' | 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6' | 'div' | 'label';
}

/** Carbon Text: renders children or a SkeletonText placeholder when loading. */
export function Text({
  children,
  className,
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

  return <Tag className={textClassName}>{children}</Tag>;
}
