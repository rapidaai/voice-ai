import { cn } from '@/styles/media';
import { forwardRef, InputHTMLAttributes } from 'react';
interface InputProps extends InputHTMLAttributes<HTMLInputElement> {}

/**
 *
 */
export const Input = forwardRef<HTMLInputElement, InputProps>(
  (props: InputProps, ref) => {
    /**
     * when any request is going disable all the input boxes
     */
    return (
      <input
        ref={ref}
        id={props.name}
        {...props}
        disabled={props.disabled}
        className={cn(
          'w-full',
          'form-input',
          'h-10',
          'dark:placeholder-gray-600 placeholder-gray-400',
          'dark:text-gray-300 text-gray-600',
          'outline-solid outline-[1.5px] outline-transparent',
          'focus-within:outline-blue-600 focus:outline-blue-600 outline-offset-[-1.5px]',
          'border-b border-gray-400 dark:border-gray-600',
          'dark:focus:border-blue-600 focus:border-blue-600',
          'transition-all duration-200 ease-in-out',

          'bg-white dark:bg-gray-950',
          'px-2 py-1.5 pl-3',
          props.className,
        )}
      />
    );
  },
);
