import { useRapidaStore } from '@/hooks';
import { FC, HTMLAttributes, ReactElement } from 'react';
import { motion } from 'framer-motion';
import { cn } from '@/styles/media';
import { Loader } from '@/app/components/Loader';
import { RapidaIcon } from '@/app/components/Icon/Rapida';
import { RapidaTextIcon } from '@/app/components/Icon/RapidaText';
import { RightArrowIcon } from '@/app/components/Icon/RightArrow';

interface WithActionToolbarProps extends HTMLAttributes<HTMLDivElement> {
  logo?: ReactElement;
  action?: ReactElement;
}
export const WithActionToolbar: FC<WithActionToolbarProps> = ({
  children,
  logo = (
    <div className="flex items-center align-middle text-blue-700">
      <RapidaIcon className="h-7 w-8 shrink-0" />
      <RapidaTextIcon className="h-6 shrink-0 w-20" strokeWidth="4" />
    </div>
  ),
  action = (
    <motion.a
      whileHover={{ scale: 1.05 }}
      onHoverStart={e => {}}
      onHoverEnd={e => {}}
      className={cn(
        'flex items-center justify-center w-fit p-2 md:py-1 md:px-4 bg-blue-800 font-medium rounded-[2px] text-white space-x-1',
      )}
      target="_blank"
      href="https://calendly.com/rapida-ai/30min"
      rel="noreferrer"
    >
      <span className="hidden sm:block">Book a demo</span>
      <span className="block sm:hidden">
        <RightArrowIcon />
      </span>
    </motion.a>
  ),
}) => {
  const {} = useRapidaStore();
  return (
    <main className="relative flex flex-col justify-center items-center w-full h-[calc(100dvh)] bg-white dark:bg-slate-950">
      <Loader />
      <div className="flex flex-col h-full w-full">
        <div
          className="flex gap-4 md:py-4 justify-between items-center shrink-0 md:px-4 border-b dark:border-b-[0.5px]"
          style={{ height: 56 }}
        >
          <div className="flex items-center">{logo}</div>
          <div className="flex basis-1/3 justify-end items-center gap-2">
            {action}
          </div>
        </div>
        <div className="py-4 flex flex-col grow relative">{children}</div>
      </div>
    </main>
  );
};
