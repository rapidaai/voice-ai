import React, { FC } from 'react';
import { Switch as SW } from '@headlessui/react';
import { cn } from '@/styles/media';

/**
 *
 * @param props
 * @returns
 */

export const Switch: FC<{
  enable: boolean;
  setEnable: (e: boolean) => void;
  id?: string;
  name?: string;
}> = React.memo(({ enable, setEnable, id, name }) => {
  //
  return (
    <SW
      id={id}
      name={name}
      checked={enable}
      onChange={setEnable}
      className={cn(
        enable ? 'bg-blue-600 justify-end' : 'bg-gray-500 justify-start',
        'relative inline-flex shrink-0 cursor-pointer rounded-[2px] items-center border-2 border-transparent transition-all duration-200 ease-in-out focus:outline-hidden focus-visible:ring-2  focus-visible:ring-white/75',
        'w-8 h-5',
      )}
    >
      <span className="sr-only">Switch</span>
      <span
        aria-hidden="true"
        className={cn(
          'pointer-events-none inline-block h-4 w-4 transform rounded-[2px] bg-white shadow-lg ring-0',
        )}
      />
    </SW>
  );
});

export const SwitchWithLabel: FC<{
  enable: boolean;
  setEnable: (e: boolean) => void;
  id?: string;
  label?: string;
  className?: string;
}> = React.memo(({ enable, setEnable, id, label, className }) => {
  return (
    <div
      className={cn(
        'w-full',
        'form-input',
        'h-10',
        'dark:placeholder-gray-600 placeholder-gray-400',
        'dark:text-gray-300 text-gray-600',

        'outline-solid outline-[1.5px] outline-transparent',
        'hover:outline-blue-600 focus:outline-blue-600 outline-offset-[-1.5px]',
        'border-b border-gray-400 dark:border-gray-600',
        'dark:hover:border-blue-600 hover:border-blue-600',
        'transition-all duration-200 ease-in-out',

        'bg-white dark:bg-gray-950',
        'px-2 py-1.5 pl-3',
        'h-10 px-4 py-2 flex justify-between items-center w-full',
        'cursor-pointer',
        className,
      )}
      onClick={() => {
        setEnable(!enable);
      }}
    >
      <span>{label}</span>
      <Switch enable={enable} setEnable={setEnable} />
    </div>
  );
});
