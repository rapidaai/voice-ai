import { cn } from '@/styles/media';
import { FC, HTMLAttributes } from 'react';

export const ModalBody: FC<HTMLAttributes<HTMLDivElement>> = props => {
  return (
    <div
      {...props}
      className={cn(
        'space-y-6 shrink',
        'relative px-8 pb-8 pt-4',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
};
