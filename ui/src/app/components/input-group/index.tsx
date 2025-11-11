import { cn } from '@/styles/media';
import { AnimatePresence, motion } from 'framer-motion';
import { ChevronDown } from 'lucide-react';
import { FC, HTMLAttributes, useState } from 'react';

interface InputGroupProps extends HTMLAttributes<HTMLDivElement> {
  title?: any;
  initiallyExpanded?: boolean;
}
export const InputGroup: FC<InputGroupProps> = ({
  initiallyExpanded = true,
  ...props
}) => {
  const [isExpanded, setIsExpanded] = useState(initiallyExpanded);

  return (
    <section
      {...props}
      className={cn('border-t last:border-b', props.className)}
    >
      <div
        onClick={() => {
          setIsExpanded(!isExpanded);
        }}
        className={cn(
          'cursor-pointer',
          'outline-solid outline-[1.5px] outline-transparent',
          'focus-within:outline-blue-600 focus:outline-blue-600 outline-offset-[-1.5px]',
          'px-4 group flex justify-between w-full items-center py-3 text-left text-base leading-tight hover:bg-white dark:hover:bg-gray-950',
        )}
      >
        <div className="mr-3.5 flex items-center">
          <div className={cn('flex-none font-semibold text-base')}>
            {props.title}
          </div>
        </div>
        <span className="h-6 w-6 flex items-center justify-center rounded-[2px] hover:bg-gray-300 dark:hover:bg-gray-800">
          <ChevronDown
            strokeWidth={1.5}
            className={cn(
              'h-full w-full transition-all',
              isExpanded && 'rotate-180',
            )}
          />
        </span>
      </div>
      <AnimatePresence>
        <motion.div
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: 'auto' }}
          exit={{ opacity: 0, height: 0 }}
          transition={{ duration: 0.3, ease: 'easeInOut' }}
          style={{ display: isExpanded ? 'block' : 'none' }}
        >
          {props.children}
        </motion.div>
      </AnimatePresence>
    </section>
  );
};
