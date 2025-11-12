import React from 'react';
import { Link } from 'react-router-dom';
import { Popover, PopoverProps } from '@/app/components/popover';

/**
 *
 * @param props
 * @returns
 */
export function HelpPopover(props: PopoverProps) {
  return (
    <Popover {...props}>
      <div className="flex flex-col divide-y">
        <Link
          className="font-medium text-sm flex items-center px-4 py-3 dark:hover:bg-gray-950 hover:bg-white"
          to="https://doc.rapida.ai"
          target="_blank"
        >
          <svg
            className="w-3 h-3 fill-current shrink-0 mr-3"
            fill="currentColor"
            viewBox="0 0 12 12"
          >
            <rect y="3" width="12" height="9" rx="1" />
            <path d="M2 0h8v2H2z" />
          </svg>
          <span>Documentation</span>
        </Link>
        <Link
          className="font-medium text-sm flex items-center px-4 py-3 dark:hover:bg-gray-950 hover:bg-white"
          to="mailto:prashant@rapida.ai"
        >
          <svg
            className="w-3 h-3 fill-current shrink-0 mr-3"
            fill="currentColor"
            viewBox="0 0 12 12"
          >
            <path d="M11.854.146a.5.5 0 00-.525-.116l-11 4a.5.5 0 00-.015.934l4.8 1.921 1.921 4.8A.5.5 0 007.5 12h.008a.5.5 0 00.462-.329l4-11a.5.5 0 00-.116-.525z" />
          </svg>
          <span>Contact us</span>
        </Link>
      </div>
    </Popover>
  );
}
