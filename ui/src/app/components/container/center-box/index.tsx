import { cn } from '@/styles/media';
import { FC } from 'react';

/**
 * Flex box usually a parent container for all the page objects
 */
interface CenterBoxProps extends React.HTMLAttributes<HTMLElement> {}

export const CenterBox: FC<CenterBoxProps> = props => {
  return (
    <div className="m-auto w-full px-4 md:py-20 flex flex-col items-center">
      <div
        className={cn(
          'mt-10 sm:w-120 w-full p-4 md:p-8 dark:bg-gray-900 bg-white md:ring-1 ring-gray-900/5 md:shadow-lg',
          props.className,
        )}
      >
        {props.children}
      </div>
    </div>
  );
};
