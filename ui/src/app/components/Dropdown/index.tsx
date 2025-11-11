import {
  Listbox,
  ListboxButton,
  ListboxOption,
  ListboxOptions,
  Transition,
} from '@headlessui/react';
import { Spinner } from '@/app/components/Loader/Spinner';
import React, { ChangeEvent, Fragment, HTMLAttributes } from 'react';
import { cn } from '@/styles/media';
import { Float } from '@headlessui-float/react';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { Check, ChevronDown } from 'lucide-react';
/**
 *
 */
export interface DropdownProps<T> extends HTMLAttributes<HTMLDivElement> {
  currentValue?: T | null;
  setValue: (value: T) => void;
  allValue: T[];
  option?: (value: T, isSelected: boolean) => React.ReactElement;
  label?: (value: T) => React.ReactElement;
  placeholder?: string;
  multiple?: boolean;
  disable?: boolean;
  placement?: 'bottom' | 'top';
  searchable?: boolean;
  onSearching?: (qry: ChangeEvent<HTMLInputElement>) => void;
}
/**
 *
 * @param props
 * @returns
 */
export function Dropdown(props: DropdownProps<any>) {
  return (
    <div className="relative flex flex-1">
      <Listbox
        value={props.currentValue || null}
        onChange={props.setValue}
        multiple={props.multiple}
        disabled={props.disable}
      >
        {({ open }) => (
          <Float
            floatingAs={Fragment}
            placement={props.placement ? props.placement : 'bottom'}
            flip
            shift
            offset={4}
          >
            <ListboxButton
              aria-label={props.placeholder}
              onClick={() => {
                if (props.disable) return;
              }}
              className={cn(
                'w-full',
                'h-10 cursor-default relative',
                'py-2 pl-3 pr-10 text-left',
                'outline-solid outline-transparent border-collapse',
                'focus-within:outline-blue-600 focus:outline-blue-600 ',
                'border-b border-gray-400 dark:border-gray-600',
                'dark:focus:border-blue-600 focus:border-blue-600',
                'transition-all duration-200 ease-in-out',
                'dark:text-gray-300 text-gray-600',
                'focus:ring-0',
                'flex items-center',
                props.disable && 'cursor-not-allowed!',
                props.className,
              )}
              type="button"
            >
              {props.disable ? (
                <Spinner className="mr-2" />
              ) : props.allValue.length === 0 ? (
                <span className="inline-flex items-center gap-1.5 sm:gap-2 max-w-full">
                  <span className="truncate sm:max-w-[300px] max-w-[120px] form-input  dark:text-gray-600 text-gray-400">
                    {props.placeholder}
                  </span>
                </span>
              ) : props.currentValue ? (
                props.label?.(props.currentValue) ?? props.currentValue
              ) : (
                <span className="inline-flex items-center gap-1.5 sm:gap-2 max-w-full">
                  <span className="truncate sm:max-w-[300px] max-w-[120px] form-input  dark:text-gray-600 text-gray-400">
                    {props.placeholder}
                  </span>
                </span>
              )}
              <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
                <ChevronDown
                  className={cn(
                    'w-4 h-4 ml-2 opacity-50 shrink-0 transition-all delay-200',
                    open && 'rotate-180',
                  )}
                />
              </span>
            </ListboxButton>
            <Transition
              as={Fragment}
              leave="transition ease-in duration-100"
              leaveFrom="opacity-100"
              leaveTo="opacity-0"
            >
              <ListboxOptions
                className={cn(
                  'shadow-lg relative',
                  'z-50 max-h-96 w-full border overflow-y-scroll',
                  'bg-light-background dark:bg-gray-900 dark:border-gray-700',
                  'dark:text-gray-300 text-gray-600',
                  'divide-y divide-gray-200 dark:divide-gray-800',
                  'outline-hidden',
                )}
              >
                {props.searchable && (
                  <div className="px-3 py-3 sticky top-0 bg-light-background dark:bg-gray-900 z-10  border-b">
                    <SearchIconInput
                      wrapperClassName="w-full!"
                      onChange={props.onSearching}
                    />
                  </div>
                )}

                {props.allValue.map((mp, idx) => {
                  return (
                    <ListboxOption
                      as="div"
                      key={idx}
                      value={mp}
                      className={cn(
                        'inline-flex py-2 px-3 w-full relative',
                        'items-center leading-6',
                        'transition-colors ease justify-between truncate',
                        'dark:hover:bg-gray-950 hover:bg-white',
                      )}
                    >
                      {({ selected }) => (
                        <>
                          {props.option && props.option(mp, selected)}
                          {selected && (
                            <span className="h-4 w-4 rounded-[2px] bg-blue-600 p-[2px] ml-auto flex items-center justify-center">
                              <Check className="text-white" />
                            </span>
                          )}
                        </>
                      )}
                    </ListboxOption>
                  );
                })}
              </ListboxOptions>
            </Transition>
          </Float>
        )}
      </Listbox>
    </div>
  );
}
