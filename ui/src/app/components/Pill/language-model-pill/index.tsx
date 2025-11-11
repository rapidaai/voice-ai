import { LanguageIcon } from '@/app/components/Icon/Language';
import { cn } from '@/styles/media';
import { FC, HTMLAttributes } from 'react';

interface LanguagePillProps extends HTMLAttributes<HTMLSpanElement> {
  language: string;
}

export const LanguagePill: FC<LanguagePillProps> = props => {
  return (
    <span
      onClick={props.onClick}
      className={cn(
        'px-2 py-1 truncate',
        'items-center bg-stone-400/10 dark:bg-stone-800/10 rounded-[2px]',
        'ring-stone-500/30 dark:ring-stone-400/20 ring-[0.7px]',
        'flex items-center justify-center w-fit',
        props.className,
      )}
    >
      <LanguageIcon className="w-4 h-4 mr-1.5 inline-block" />
      <span className="font-medium opacity-80 ">{props.language}</span>
    </span>
  );
};
