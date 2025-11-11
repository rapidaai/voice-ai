import { cn } from '@/styles/media';
import { Input } from '@/app/components/Form/Input';
import { SearchIcon } from '@/app/components/Icon/Search';
import { forwardRef, InputHTMLAttributes } from 'react';

interface IconInputProps extends InputHTMLAttributes<HTMLInputElement> {
  wrapperClassName?: string;
  iconClassName?: string;
  placeholder?: string;
}

export const SearchIconInput = forwardRef<HTMLInputElement, IconInputProps>(
  (props: IconInputProps, ref) => {
    const { wrapperClassName, className, iconClassName, ...atr } = props;
    return (
      <div
        className={cn(
          'relative w-160 max-w-full h-10 flex items-center',
          wrapperClassName,
        )}
      >
        <label htmlFor="search-input" className="sr-only">
          Search
        </label>
        <Input
          {...atr}
          id="search-input"
          name="search-input"
          ref={ref}
          className={cn('w-full py-2 pl-9!', className)}
          type="search"
          placeholder={
            props.placeholder ? props.placeholder : 'Find resources..'
          }
        />
        <button
          className="absolute inset-0 right-auto group flex items-center"
          type="submit"
          aria-label="Search"
        >
          <SearchIcon
            className={cn(
              'w-3 h-3 shrink-0 fill-current mx-3 opacity-70',
              iconClassName,
            )}
          />
        </button>
      </div>
    );
  },
);
