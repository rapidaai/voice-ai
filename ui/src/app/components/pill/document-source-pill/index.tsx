import { LanguageIcon } from '@/app/components/Icon/Language';
import { ToolProviderPill } from '@/app/components/pill/tool-provider-pill';
import { cn } from '@/utils';
import { FC, HTMLAttributes } from 'react';

interface DocumentSourcePillProps extends HTMLAttributes<HTMLSpanElement> {
  source?: string;
  type?: string;
}

export const DocumentSourcePill: FC<DocumentSourcePillProps> = ({
  source,
  type,
  className,
  onClick,
  children,
}) => {
  if (type === 'tool')
    return <ToolProviderPill className="text-sm" toolProviderId={source} />;
  return (
    <span
      onClick={onClick}
      className={cn(
        'px-2 py-1 truncate',
        'items-center bg-stone-400/10 dark:bg-stone-800/10 rounded-[2px]',
        'ring-stone-500/30 dark:ring-stone-400/20 ring-[0.7px]',
        'flex items-center justify-center w-fit text-sm',
        className,
      )}
    >
      <LanguageIcon className="w-4 h-4 mr-1.5 inline-block" />
      <span className="font-medium opacity-80 ">{type}</span>
    </span>
  );
};
