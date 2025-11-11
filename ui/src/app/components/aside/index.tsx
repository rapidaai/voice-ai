import { cn } from '@/styles/media';
import { FC, HTMLAttributes } from 'react';

interface AsideProps extends HTMLAttributes<HTMLDivElement> {}
export const Aside: FC<AsideProps> = (props: AsideProps) => {
  return (
    <div
      className={cn(
        'flex flex-col absolute top-0 bottom-0 z-12',
        'no-scrollbar',
        'backdrop-blur-2xl',
        'group',
        'overflow-y-scroll',
        'w-14 h-full duration-200 hover:w-80 pb-10 hover:border-r hover:bg-white dark:hover:bg-gray-900',
        props.className,
      )}
    >
      {props.children}
    </div>
  );
};
