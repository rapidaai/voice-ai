import { cn } from '@/utils';
import { forwardRef, InputHTMLAttributes } from 'react';

/**
 *
 */
interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  max?: number | string;
  min?: number | string;
  step?: number | string;
  onSlide: (number) => void;
}

/**
 *
 */
export const Slider = forwardRef<HTMLInputElement, InputProps>(
  (props: InputProps, ref) => {
    /**
     * when any request is going disable all the input boxes
     */

    const { onSlide, ...atr } = props;
    return (
      <input
        ref={ref}
        id={props.name}
        type="range"
        {...atr}
        disabled={props.disabled}
        max={props.max}
        min={props.min}
        step={props.step}
        onChange={t => {
          onSlide(t.target.valueAsNumber);
        }}
        className={cn(
          'w-full',
          'h-1 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700',
          //   'bg-white dark:bg-gray-950',
          props.className,
        )}
      />
    );
  },
);
