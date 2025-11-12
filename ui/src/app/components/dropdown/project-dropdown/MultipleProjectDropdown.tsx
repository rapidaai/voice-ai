import { Fragment, useContext } from 'react';
import React from 'react';
import { Listbox, Transition } from '@headlessui/react';
import { cn } from '@/utils';
import { Float } from '@headlessui-float/react';
import { AuthContext } from '@/context/auth-context';
import { Check } from 'lucide-react';

/**
 *
 * @param props
 * @returns
 */
export function MultipleProjectDropdown(props: {
  projectIds?: string[];
  setProjectIds: (arg: string[]) => void;
}) {
  /**
   *
   */
  const { projectRoles } = useContext(AuthContext);

  /**
   *
   */
  return (
    <Listbox value={props.projectIds} onChange={props.setProjectIds} multiple>
      <Float
        as="div"
        className="relative"
        placement={'bottom'}
        offset={4}
        portal
        adaptiveWidth
      >
        <Listbox.Button
          className={cn(
            'bg-white dark:bg-gray-950',
            'w-full',
            'h-10 cursor-default',
            'py-2 pl-3 pr-10 text-left',
            'outline-solid outline-transparent',
            'focus-within:outline-blue-600 focus:outline-blue-600 ',
            'border-b border-gray-400 dark:border-gray-600',
            'dark:focus:border-blue-600 focus:border-blue-600',
            'transition-all duration-200 ease-in-out',
            'dark:text-gray-300 text-gray-600',
            'focus:ring-0',
            'flex items-center',
          )}
        >
          {props.projectIds ? (
            <>
              <span className="block truncate">
                {props.projectIds.length} projects
              </span>
            </>
          ) : (
            <span className="block truncate dark:text-gray-400">
              Select the projects
            </span>
          )}
          <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
              stroke="currentColor"
              className="h-5 w-5 dark:text-gray-400"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M8.25 15L12 18.75 15.75 15m-7.5-6L12 5.25 15.75 9"
              />
            </svg>
          </span>
        </Listbox.Button>
        <Transition
          as={Fragment}
          leave="transition ease-in duration-100"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <Listbox.Options
            className={cn(
              'shadow-lg',
              'z-50 max-h-96 w-full border overflow-y-scroll p-0.5 rounded-[2px]',
              'bg-white dark:bg-gray-800 dark:border-gray-700',
              'dark:text-gray-300 text-gray-600',
              'divide-y divide-gray-200 dark:divide-gray-700',
              'outline-hidden',
            )}
          >
            {projectRoles &&
              projectRoles.map((project, idx) => (
                <Listbox.Option
                  as="div"
                  key={idx}
                  value={project.projectid}
                  className={cn(
                    'inline-flex py-2 px-3 w-full relative',
                    'items-center leading-6',
                    'transition-colors ease justify-between truncate h-10',
                    'dark:hover:bg-gray-950 hover:bg-white',
                  )}
                >
                  {({ selected }) => (
                    <>
                      <span
                        className={`block truncate text-sm ${
                          selected ? 'font-semibold' : 'font-medium opacity-80'
                        }`}
                      >
                        {project.projectname}
                      </span>
                      {selected && (
                        <span className="h-5 w-5 rounded-[2px] bg-blue-600 p-[2px] ml-auto flex items-center justify-center">
                          <Check className="text-white" />
                        </span>
                      )}
                    </>
                  )}
                </Listbox.Option>
              ))}
          </Listbox.Options>
        </Transition>
      </Float>
    </Listbox>
  );
}
