import { RedNoticeBlock } from '@/app/components/layout/container/message/notice-block';
import { WarningAlt } from '@carbon/icons-react';
import type { HTMLAttributes } from 'react';
import { cn } from '@/utils';

interface PageActionButtonBlockProps extends HTMLAttributes<HTMLDivElement> {
  errorMessage?: string;
}

export function PageActionButtonBlock({
  errorMessage,
  children,
  className,
  ...attributes
}: PageActionButtonBlockProps) {
  return (
    <div {...attributes} className={cn('w-full shrink-0', className)}>
      {errorMessage && (
        <RedNoticeBlock className="flex items-center space-x-2">
          <WarningAlt
            aria-hidden="true"
            className="h-4 w-4 text-support-error"
          />
          <span role="alert">{errorMessage}</span>
        </RedNoticeBlock>
      )}
      <div className="flex h-12 border-t border-border-subtle">{children}</div>
    </div>
  );
}
