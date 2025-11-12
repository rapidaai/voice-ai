import { cn } from '@/utils';
import { forwardRef } from 'react';

export type SelectionOption = {
  name: string;
  value: string | number;
};

interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  autocomplete?: string;
  placeholder?: string;
  options: SelectionOption[];
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  (props: SelectProps, ref) => {
    return (
      <div className="relative w-full">
        <select
          ref={ref}
          id={props.name}
          name={props.name}
          {...props}
          required={props.required}
          value={props.value}
          onChange={props.onChange}
          className={cn(
            'block appearance-none',
            'w-full',
            'h-10',
            'form-input',
            'dark:disabled:placeholder-gray-600 disabled:placeholder-gray-400',
            'dark:text-gray-300 text-gray-600',
            'outline-solid outline-[1.5px] outline-transparent',
            'focus-within:outline-blue-600 focus:outline-blue-600 outline-offset-[-1.5px]',
            'border-b border-gray-400 dark:border-gray-600',
            'dark:focus:border-blue-600 focus:border-blue-600',
            'transition-all duration-200 ease-in-out',
            'relative',
            'bg-white dark:bg-gray-950',
            'ring-0',
            'px-2 py-1.5 pl-3',
            props.className,
          )}
        >
          <option value="" disabled>
            {props.placeholder}
          </option>
          {props.options.map((e, idx) => {
            return (
              <option value={e.value} key={idx}>
                {e.name}
              </option>
            );
          })}
        </select>
        <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-gray-700">
          <svg
            className="fill-current h-4 w-4"
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 20 20"
            strokeWidth={0.5}
          >
            <path d="M9.293 12.95l.707.707L15.657 8l-1.414-1.414L10 10.828 5.757 6.586 4.343 8z" />
          </svg>
        </div>
      </div>
    );
  },
);
