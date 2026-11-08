import type { HTMLAttributes, ReactNode } from 'react';
import { Information } from '@carbon/icons-react';
import { Link } from '@carbon/react';
import { useDocumentationUrl } from '@/theme/documentation-url';
import { cn } from '@/utils';

interface DocNoticeBlockProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
  docPath?: string;
  docUrl?: string;
  linkText?: string;
}

export function DocNoticeBlock({
  children,
  className,
  docPath = '',
  docUrl,
  linkText = 'Read documentation',
  ...attributes
}: DocNoticeBlockProps) {
  const configuredDocUrl = useDocumentationUrl(docPath);

  return (
    <div
      {...attributes}
      className={cn(
        'flex w-full items-center gap-3 border-l-[3px] border-support-info bg-support-info/10 px-4 py-3 text-foreground ',
        className,
      )}
      role="note"
    >
      <Information
        aria-hidden="true"
        size={20}
        className="shrink-0 text-support-info "
      />
      <span className="flex-1 text-sm text-foreground">{children}</span>
      <Link
        href={docUrl ?? configuredDocUrl}
        target="_blank"
        rel="noopener noreferrer"
        className="!font-semibold shrink-0"
      >
        {linkText}
      </Link>
    </div>
  );
}
