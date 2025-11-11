import { cn } from '@/styles/media';
import React, { HtmlHTMLAttributes } from 'react';
interface SingleRowWrapperProps extends HtmlHTMLAttributes<HTMLDivElement> {}
function SingleRowWrapper(props: SingleRowWrapperProps) {
  return (
    <div
      className={cn(
        'flex items-center justify-between bg-gray-100 space-x-1 p-1 rounded-[2px] border dark:border-gray-800 dark:bg-gray-900',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
}

export default React.memo(SingleRowWrapper);
