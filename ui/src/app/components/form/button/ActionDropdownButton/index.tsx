import { Transition, Menu } from '@headlessui/react';
import { BorderButton, ButtonProps } from '@/app/components/form/button';
import React, { Fragment } from 'react';
import { cn } from '@/utils';

/**
 *
 */
interface ActionDropdownButtonProps extends ButtonProps {
  /**
   * Primary Action
   */

  action?: React.ReactElement | string;

  /**
   * When someone click on action button
   * @returns
   */
  onActionClick?: () => void;

  /**
   *
   */
  dropdownActions: { action: string; onActionClick?: () => void }[];
}

export function ActionDropdownButton(props: ActionDropdownButtonProps) {
  const { action, dropdownActions, ...btnProps } = props;
  return (
    <div className="flex space-x-1">
      <BorderButton type="button" className="px-6" {...btnProps}>
        {props.action}
      </BorderButton>

      {props.dropdownActions.length > 0 && (
        <Menu as="div" className="relative inline-block text-left">
          {({ open }) => (
            <>
              <Menu.Button as="div" className="">
                <BorderButton
                  className={cn('w-8', btnProps.className)}
                  type="button"
                >
                  {open ? (
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      viewBox="0 0 24 24"
                      fill="currentColor"
                      stroke="currentColor"
                      strokeWidth={1.5}
                      className="w-5 h-5"
                    >
                      <path
                        fillRule="evenodd"
                        d="M11.47 7.72a.75.75 0 0 1 1.06 0l7.5 7.5a.75.75 0 1 1-1.06 1.06L12 9.31l-6.97 6.97a.75.75 0 0 1-1.06-1.06l7.5-7.5Z"
                        clipRule="evenodd"
                      />
                    </svg>
                  ) : (
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      viewBox="0 0 24 24"
                      fill="currentColor"
                      stroke="currentColor"
                      strokeWidth={1.5}
                      className="w-5 h-5"
                    >
                      <path d="M12.53 16.28a.75.75 0 0 1-1.06 0l-7.5-7.5a.75.75 0 0 1 1.06-1.06L12 14.69l6.97-6.97a.75.75 0 1 1 1.06 1.06l-7.5 7.5Z" />
                    </svg>
                  )}
                </BorderButton>
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
                <Menu.Items className="absolute z-20 right-0 mt-2 w-56 origin-top-right divide-y divide-gray-100 rounded-[2px] bg-white dark:bg-gray-700 shadow-lg ring-1 ring-black/5 focus:outline-hidden">
                  <div className="px-1 py-1 ">
                    {props.dropdownActions.map((act, idx) => {
                      return (
                        <Menu.Item key={idx}>
                          {({ active }) => (
                            <button
                              className={cn(
                                'group flex w-full items-center rounded-[2px] px-2 py-2 text-sm',
                                'dark:text-white',
                                active
                                  ? 'dark:bg-gray-800 bg-gray-200 '
                                  : 'text-gray-900 dark:text-white',
                              )}
                              onClick={act.onActionClick}
                            >
                              {act.action}
                            </button>
                          )}
                        </Menu.Item>
                      );
                    })}
                  </div>
                </Menu.Items>
              </Transition>
            </>
          )}
        </Menu>
      )}
    </div>
  );
}

ActionDropdownButton.defaultProps = {
  dropdownActions: [],
};
