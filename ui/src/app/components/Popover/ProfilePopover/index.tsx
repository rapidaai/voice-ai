import React from 'react';
import { Popover, PopoverProps } from '@/app/components/Popover';
import { CustomLink } from '@/app/components/custom-link';
import { Link } from 'react-router-dom';

interface ProfilePopoverProps extends PopoverProps {
  account: { email: string };
}
/**
 *
 * @param props
 * @returns
 */
export function ProfilePopover(props: ProfilePopoverProps) {
  const { account, ...atr } = props;
  return (
    <Popover {...atr}>
      <div className="">
        <div className="px-4 py-3 border-b dark:border-gray-700">
          <p className="text-sm leading-5 font-medium">Signed in as</p>
          <p className="text-sm font-semibold leading-5 truncate">
            {props.account.email}
          </p>
        </div>
        <div className="flex flex-col divide-y">
          <Link
            to="/account/personal-settings"
            className="px-4 py-3 text-sm dark:hover:bg-gray-950 hover:bg-white"
          >
            Account Settings
          </Link>
          <Link
            to="/auth/signin"
            className="px-4 py-3 text-sm dark:hover:bg-gray-950  hover:bg-white"
          >
            Signout
          </Link>
        </div>
      </div>
    </Popover>
  );
}
