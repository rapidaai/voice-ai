import { PrimaryButton } from '@/app/components/ui/primitives/button';
import { Text } from '@/app/components/ui/primitives/text';
import { ArrowLeft } from '@carbon/icons-react';
import { useId } from 'react';
import type { ComponentProps } from 'react';
import { cn } from '@/utils';

interface ErrorContainerProps {
  code: string;
  title: string;
  description: string;
  actionLabel: string;
  onAction: () => void;
  actionIcon?: ComponentProps<typeof PrimaryButton>['renderIcon'] | null;
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
  const titleId = useId();

  return (
    <section
      aria-labelledby={titleId}
      className={cn(
        'flex min-h-0 w-full flex-1 items-center justify-center px-4 py-12 text-[var(--cds-text-primary)] sm:px-6 lg:px-8',
        className,
      )}
    >
      <div className="flex w-full max-w-xl flex-col items-center text-center">
        <Text
          as="span"
          variant="heading-06"
          className="[color:var(--cds-text-secondary)]"
        >
          {code}
        </Text>
        <Text
          as="h1"
          id={titleId}
          variant="heading-04"
          className="mt-4"
        >
          {title}
        </Text>
        <Text
          as="p"
          variant="body-01"
          className="mt-3 max-w-lg [color:var(--cds-text-secondary)]"
        >
          {description}
        </Text>
        <div className="mt-8">
          <PrimaryButton
            size="md"
            onClick={onAction}
            renderIcon={actionIcon ?? undefined}
          >
            {actionLabel}
          </PrimaryButton>
        </div>
      </div>
    </section>
  );
}
