import { cn } from '@/styles/media';
import { FC } from 'react';
import Datepicker from 'react-tailwindcss-datepicker';

export const DateSelector: FC<{
  className?: string;
  value: {
    startDate: Date | null;
    endDate: Date | null;
  };
  onChangeValue: (to: Date | null, from: Date | null) => void;
}> = ({ className, value, onChangeValue }) => {
  return (
    <Datepicker
      asSingle={true}
      dateLooking={'middle'}
      showShortcuts={false}
      maxDate={new Date(new Date().setDate(new Date().getDate() + 1))}
      popupClassName={v => {
        return cn('rounded-none!', v);
      }}
      configs={{}}
      classNames={{
        container: props => {
          return cn('rounded-none');
        },
        input: props => {
          return cn(
            'w-full',
            'h-10',
            'px-4',
            'dark:placeholder-gray-600 placeholder-gray-400',
            'dark:text-gray-300 text-gray-600',
            'outline-solid outline-[1.5px] outline-transparent',
            'focus-within:outline-blue-600 focus:outline-blue-600 outline-offset-[-1.5px]',
            'border-b border-gray-400 dark:border-gray-600',
            'dark:focus:border-blue-600 focus:border-blue-600',
            'transition-all duration-200 ease-in-out',
            'bg-white dark:bg-gray-950',
            'justify-center w-full',
            className,
          );
        },
      }}
      separator="to"
      showFooter={true}
      value={value}
      onChange={newValue => {
        if (newValue && newValue.startDate && newValue.endDate) {
          onChangeValue(newValue.startDate, newValue.endDate);
        }
      }}
    />
  );
};
