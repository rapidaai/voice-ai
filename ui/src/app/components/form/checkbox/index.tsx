import { cn } from '@/utils';
import { CheckIcon } from 'lucide-react';
import { forwardRef, InputHTMLAttributes } from 'react';
interface InputCheckboxProps extends InputHTMLAttributes<HTMLInputElement> {}

/**
 *
 */
export const InputCheckbox = forwardRef<HTMLInputElement, InputCheckboxProps>(
  (props: InputCheckboxProps, ref) => {
    /**
     * when any request is going disable all the input boxes
     */
    return (
      <label className="cursor-pointer inline-flex items-center">
        <input
          ref={ref}
          {...props}
          type="checkbox"
          className={cn('peer hidden')}
        />
        <div
          className={cn(
            'outline-solid outline-[1.5px] outline-transparent',
            'focus-within:outline-blue-600 focus:outline-blue-600 outline-offset-[-1.5px]',
            'border-[0.5px] border-gray-400 dark:border-gray-600',
            'dark:focus:border-blue-600 focus:border-blue-600',
            'dark:hover:border-blue-600 hover:border-blue-600',
            'transition-all duration-200 ease-in-out',
            'peer-checked:text-white! text-transparent!',
            'peer-checked:bg-blue-600 peer-checked:border-blue-600 peer-focus:ring-2 peer-focus:ring-blue-500',
            'h-4 w-4 rounded-none flex items-center justify-center transition',
            'bg-white dark:bg-gray-950',
            props.className,
          )}
        >
          <CheckIcon className="" />
        </div>
      </label>
    );
  },
);
