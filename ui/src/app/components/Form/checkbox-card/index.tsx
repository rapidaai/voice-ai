import { cn } from '@/styles/media';
import { Check } from 'lucide-react';
import React, { FC, InputHTMLAttributes } from 'react';

interface CheckboxCardProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string | JSX.Element;
  wrapperClassNames?: string;
  selectedClassNames?: string;
}

const CheckboxCard: FC<CheckboxCardProps> = ({
  id,
  label,
  wrapperClassNames,
  disabled,
  children,
  ...atr
}) => {
  return (
    <label
      htmlFor={id}
      className={cn(
        'relative h-fit',
        wrapperClassNames,
        !disabled && 'cursor-pointer',
      )}
    >
      {label}
      {children}
      <input
        id={id}
        className="hidden appearance-none peer"
        {...atr}
        disabled={disabled}
      />
      <span
        className={cn(
          'hidden peer-checked:block absolute inset-0 border-b-2 border-blue-600',
        )}
      >
        <span className="absolute top-4 right-4 h-5 w-5 inline-flex items-center justify-center rounded-[2px] bg-blue-600 p-[2px]">
          <Check className="text-white" />
        </span>
      </span>
    </label>
  );
};
export default React.memo(CheckboxCard);
