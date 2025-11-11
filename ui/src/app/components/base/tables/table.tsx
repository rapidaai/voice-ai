import { cn } from '@/styles/media';
import { FC, HTMLAttributes } from 'react';

export const Table: FC<HTMLAttributes<HTMLTableElement>> = props => {
  return (
    <table {...props} className={cn('text-sm', props.className)}>
      {props.children}
    </table>
  );
};
