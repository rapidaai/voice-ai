import React, { FC, Fragment } from 'react';
import { Menu, Transition } from '@headlessui/react';
import { cn } from '@/styles/media';
import { Float } from '@headlessui-float/react';
import { DotIcon } from '@/app/components/Icon/Dot';
import { ChevronDown } from 'lucide-react';

interface OptionMenuProps {
  /*
  options that will be listed
   */
  options: { option: any; onActionClick: () => void }[];
  classNames?: string;
  activeClassName?: string;
}

export const OptionMenu: FC<OptionMenuProps> = props => {
  return (
    <Menu as="div" className="inline-block text-left w-fit relative">
      {({ open }) => (
        <Float placement="bottom-end" portal>
          <Menu.Button
            className={cn(
              'bg-gray-100 dark:bg-gray-950/40 dark:hover:bg-gray-950 hover:bg-gray-200 focus:outline-hidden hover:shadow-sm',
              'p-1.5',
              open && 'dark:bg-gray-950 bg-gray-200',
            )}
          >
            <span
              className={cn(
                'absolute w-px h-px p-0 -m-px overflow-hidden whitespace-no-wrap border-0',
              )}
            >
              Menu
            </span>
            <DotIcon className="h-5 w-5 opacity-50" />
          </Menu.Button>
          <Transition
            as={Fragment}
            enter="transition ease-out duration-100"
            enterFrom="transform opacity-0 scale-95"
            enterTo="transform opacity-100 scale-100"
            leave="transition ease-in duration-75"
            leaveFrom="transform opacity-100 scale-100"
            leaveTo="transform opacity-0 scale-95"
          >
            <Menu.Items className="p-0.5 absolute right-0 mt-2 w-48 z-10 origin-top-right divide-y divide-gray-200 dark:divide-gray-700 bg-white dark:bg-gray-800 shadow-lg ring-1 ring-black/5 focus:outline-hidden">
              {props.options.map((opt, idx) => {
                return (
                  <Menu.Item key={`opt-menu-${idx}`}>
                    {({ active }) => (
                      <button
                        onClick={opt.onActionClick}
                        className={cn(
                          'group flex w-full items-center px-2 py-2 text-sm font-medium dark:text-white',
                          active
                            ? 'bg-gray-100 dark:bg-gray-900/80 opacity-100'
                            : 'opacity-80',
                        )}
                      >
                        {opt.option}
                      </button>
                    )}
                  </Menu.Item>
                );
              })}
            </Menu.Items>
          </Transition>
        </Float>
      )}
    </Menu>
  );
};

export const CardOptionMenu: FC<OptionMenuProps> = props => {
  return (
    <Menu as="div" className="inline-block text-left w-fit relative">
      {({ open }) => (
        <Float placement="bottom-end" portal>
          <Menu.Button
            className={cn(
              'flex h-9 truncate w-fit justify-center items-center',
              'dark:text-gray-400 font-medium',
              'py-1.5 px-2.5',
              'bg-white dark:bg-gray-950/50 dark:hover:bg-gray-700/50 hover:bg-gray-200',
              'button',
              props.classNames,
              open && 'dark:bg-gray-900/50 bg-gray-100',
              open && props.activeClassName,
            )}
          >
            <span
              className={cn(
                'absolute w-px h-px p-0 -m-px overflow-hidden whitespace-no-wrap border-0',
              )}
            >
              Menu
            </span>

            <ChevronDown
              className={cn(
                'w-4 h-4 transition-all delay-100',
                open && 'rotate-180',
              )}
              strokeWidth="2"
            />
          </Menu.Button>
          <Transition
            as={Fragment}
            enter="transition ease-out duration-100"
            enterFrom="transform opacity-0 scale-95"
            enterTo="transform opacity-100 scale-100"
            leave="transition ease-in duration-75"
            leaveFrom="transform opacity-100 scale-100"
            leaveTo="transform opacity-0 scale-95"
          >
            <Menu.Items className="p-1 absolute right-0 mt-2 w-max z-10 origin-top-right bg-white dark:bg-gray-800 shadow-lg ring-1 ring-black/5 focus:outline-hidden divide-y divide-gray-200 dark:divide-gray-700">
              {props.options.map((opt, idx) => {
                return (
                  <Menu.Item key={`opt-menu-${idx}`}>
                    {({ active }) => (
                      <button
                        onClick={opt.onActionClick}
                        className={cn(
                          'group flex w-full items-center px-3 py-2.5 text-sm font-medium dark:text-white',
                          active
                            ? 'bg-gray-100 dark:bg-gray-900/80 opacity-100'
                            : 'opacity-80',
                        )}
                      >
                        {opt.option}
                      </button>
                    )}
                  </Menu.Item>
                );
              })}
            </Menu.Items>
          </Transition>
        </Float>
      )}
    </Menu>
  );
};

export function OptionMenuItem(props: {
  type: 'danger' | 'info';
  children?: any;
}) {
  return props.type === 'danger' ? (
    <span className="text-rose-600 dark:text-rose-500">{props.children}</span>
  ) : (
    <></>
  );
}
