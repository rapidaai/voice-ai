import type { ElementType, ReactNode } from 'react';
import { Button } from '@carbon/react';
import { cn } from '@/utils';
import { useId } from 'react';

export interface EmptyStateProps {
  icon?: ElementType;
  title: string;
  subtitle?: string;
  action?: string;
  actionIcon?: ElementType;
  onAction?: () => void;
  actionComponent?: ReactNode;
  className?: string;
}

export function EmptyState({
  icon: Icon,
  title,
  subtitle,
  action,
  actionIcon,
  onAction,
  actionComponent,
  className,
}: EmptyStateProps) {
  const headingId = useId();

  return (
    <div
      aria-labelledby={headingId}
      className={cn(
        'flex flex-1 flex-col items-center justify-center px-8 py-12 text-center text-foreground',
        className,
      )}
    >
      {Icon && (
        <div className="mb-5 flex h-16 w-16 items-center justify-center bg-layer-accent">
          <Icon size={32} className="text-icon-secondary" />
        </div>
      )}
      <h3 id={headingId} className="mb-2 text-base font-semibold">
        {title}
      </h3>
      {subtitle && (
        <p className="mb-5 max-w-md text-sm leading-5 text-muted">{subtitle}</p>
      )}
      {actionComponent}
      {action && onAction && (
        <Button
          size="md"
          kind="tertiary"
          renderIcon={actionIcon}
          onClick={onAction}
        >
          {action}
        </Button>
      )}
    </div>
  );
}
