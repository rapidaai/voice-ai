import type { ElementType } from 'react';
import { Text } from '@/app/components/ui/primitives/text';
import { cn } from '@/utils';

interface SessionEmptyStateProps {
  icon: ElementType;
  title: string;
  description: string;
  className?: string;
}

export function SessionEmptyState({
  icon: Icon,
  title,
  description,
  className,
}: SessionEmptyStateProps) {
  return (
    <section
      className={cn(
        'flex min-h-0 w-full flex-1 items-center justify-center px-4 py-8 text-center',
        className,
      )}
    >
      <div className="flex max-w-md flex-col items-center">
        <Icon
          aria-hidden="true"
          size={32}
          className="text-[var(--cds-icon-secondary)]"
        />
        <Text as="h2" variant="heading-03" className="mt-4">
          {title}
        </Text>
        <Text
          as="p"
          variant="body-01"
          className="mt-2 text-[var(--cds-text-secondary)]"
        >
          {description}
        </Text>
      </div>
    </section>
  );
}
