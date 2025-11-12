import { PrivateIcon } from '@/app/components/Icon/Private';
import { cn } from '@/utils';
import { FC, HTMLAttributes } from 'react';

interface VisibilityPillProps extends HTMLAttributes<HTMLSpanElement> {
  visibility?: string;
}

export const VisibilityPill: FC<VisibilityPillProps> = props => {
  return (
    <span
      onClick={props.onClick}
      className={cn(
        'px-2 py-1 truncate',
        'items-center rounded-[2px]',
        'bg-blue-400/20 dark:bg-blue-100/10 text-blue-600',
        'flex items-center justify-center w-fit text-sm',
        props.className,
      )}
    >
      <PrivateIcon className="w-4 h-4 mr-1.5 inline-block" />
      <span className="font-medium opacity-80 ">{props.visibility}</span>
    </span>
  );
};
