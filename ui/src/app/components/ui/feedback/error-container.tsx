import { ErrorEmptyState } from '@carbon/ibm-products/lib/components/EmptyStates/ErrorEmptyState/ErrorEmptyState';
import { NotFoundEmptyState } from '@carbon/ibm-products/lib/components/EmptyStates/NotFoundEmptyState/NotFoundEmptyState';
import { UnauthorizedEmptyState } from '@carbon/ibm-products/lib/components/EmptyStates/UnauthorizedEmptyState/UnauthorizedEmptyState';
import type { EmptyStateAction } from '@carbon/ibm-products/lib/components/EmptyStates/EmptyState';
import { ArrowLeft } from '@carbon/icons-react';
import { cn } from '@/utils';
import { useTheme } from '@/theme/theme-provider';

interface ErrorContainerProps {
  code: string;
  title: string;
  description: string;
  actionLabel: string;
  onAction: () => void;
  actionIcon?: EmptyStateAction['renderIcon'] | null;
  className?: string;
}

export function ErrorContainer({
  code,
  title,
  description,
  actionLabel,
  onAction,
  actionIcon = ArrowLeft,
  className,
}: ErrorContainerProps) {
  const { resolvedMode } = useTheme();
  const InternalErrorState =
    code === '403'
      ? UnauthorizedEmptyState
      : code === '404'
        ? NotFoundEmptyState
        : ErrorEmptyState;

  return (
    <section
      aria-label={title}
      className={cn(
        'flex min-h-0 w-full flex-1 items-center justify-center px-4 py-12 text-foreground sm:px-6 lg:px-8',
        className,
      )}
    >
      <InternalErrorState
        action={{
          text: actionLabel,
          onClick: onAction,
          renderIcon: actionIcon ?? undefined,
          iconDescription: actionIcon ? actionLabel : undefined,
        }}
        headingAs="h1"
        illustrationTheme={resolvedMode === 'dark' ? 'dark' : 'light'}
        subtitle={description}
        title={title}
      />
    </section>
  );
}
