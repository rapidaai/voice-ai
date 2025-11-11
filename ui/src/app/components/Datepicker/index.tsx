import React, { FC } from 'react';
import Flatpickr from 'react-flatpickr';
import { cn } from '@/styles/media';
import { Calendar } from 'lucide-react';

export const Datepicker: FC<{
  align?: string;
  className?: string;
  defaultDate?: {
    from: Date;
    to: Date;
  };
  onDateSelect?: (to: Date, from: Date) => void;
}> = ({
  className,
  align,
  onDateSelect,
  defaultDate = {
    from: new Date().setDate(new Date().getDate() - 30),
    to: new Date().setDate(new Date().getDate() + 1),
  },
}) => {
  const options = {
    mode: 'range',
    position: 'right',
    dateFormat: 'M j, Y',
    defaultDate: [defaultDate.to, defaultDate.from],
    prevArrow:
      '<svg class="fill-current" width="7" height="11" viewBox="0 0 7 11"><path d="M5.4 10.8l1.4-1.4-4-4 4-4L5.4 0 0 5.4z" /></svg>',
    nextArrow:
      '<svg class="fill-current" width="7" height="11" viewBox="0 0 7 11"><path d="M1.4 10.8L0 9.4l4-4-4-4L1.4 0l5.4 5.4z" /></svg>',
    onReady: (selectedDates, dateStr, instance) => {
      instance.element.value = dateStr.replace('to', '-');
      const customClass = align ? align : '';
      instance.calendarContainer.classList.add(`flatpickr-${customClass}`);
      if (onDateSelect) {
        onDateSelect(
          new Date(instance.selectedDates[1]),
          new Date(instance.selectedDates[0]),
        );
      }
    },
    onChange: (selectedDates, dateStr, instance) => {
      instance.element.value = dateStr.replace('to', '-');
    },
    onClose: (selectedDates, dateStr, instance) => {
      instance.element.value = dateStr.replace('to', '-');
      if (onDateSelect) {
        onDateSelect(
          new Date(instance.selectedDates[1]),
          new Date(instance.selectedDates[0]),
        );
      }
    },
  };

  return (
    <div className="relative min-w-64">
      <Flatpickr
        className={cn(
          'w-full',
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
          'px-4',
          'justify-center w-full text-sm',
          className,
        )}
        options={options}
      />
      <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-gray-700">
        <Calendar className="w-4 h-4" strokeWidth={1.5} />
      </div>
    </div>
  );
};
