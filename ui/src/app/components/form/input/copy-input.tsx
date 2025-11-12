import { forwardRef, InputHTMLAttributes, useState } from 'react';
import { cn } from '@/utils';
import { Copy } from 'lucide-react';
interface InputProps extends InputHTMLAttributes<HTMLInputElement> {}

export const CopyInput = forwardRef<HTMLInputElement, InputProps>(
  (props: InputProps, ref) => {
    const [copied, setCopied] = useState(false);

    const handleCopy = () => {
      if (props.value && typeof props.value === 'string') {
        navigator.clipboard.writeText(props.value);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      }
    };

    return (
      <div className="relative w-full">
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
        <button
          type="button"
          onClick={handleCopy}
          className="absolute right-2 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:text-gray-600 dark:hover:text-gray-400 transition-colors"
        >
          <Copy className="h-4 w-4" strokeWidth={1.5} />
          {copied && (
            <span className="absolute bottom-full left-1/2 transform -translate-x-1/2 bg-gray-800 text-white text-xs rounded-sm py-1 px-2">
              Copied!
            </span>
          )}
        </button>
      </div>
    );
  },
);
