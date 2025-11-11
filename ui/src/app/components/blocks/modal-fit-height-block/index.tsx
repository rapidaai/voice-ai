import { cn } from '@/styles/media';
import { FC, HTMLAttributes } from 'react';

export const ModalFitHeightBlock: FC<
  HTMLAttributes<HTMLDivElement>
> = props => {
  return (
    <div
      {...props}
      className={cn(
        'w-[750px] max-w-full bg-light-background dark:bg-gray-900 relative items-start max-h-full',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
};
