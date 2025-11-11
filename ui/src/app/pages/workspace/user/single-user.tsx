import React from 'react';
import { UserOption } from './user-options';
import { TextImage } from '@/app/components/Image/TextImage';
import { User } from '@rapidaai/react';
import { StatusIndicator } from '@/app/components/indicators/status';
import { RoleIndicator } from '@/app/components/indicators/role';
import { toHumanReadableDate } from '@/styles/media';
import { TableRow } from '@/app/components/base/tables/table-row';

/**
 * all the user row
 * @param props
 * @returns
 */
export function SingleUser(props: { user: User }) {
  return (
    <TableRow>
      <td>
        <div className="underline underline-offset-2 hover:text-blue-600 text-blue-500 text-[15px]  px-2 md:px-5 py-3">
          {props.user.getId()}
        </div>
      </td>
      <td>
        <div className="whitespace-no-wrap px-2 md:px-5 py-3">
          <div className="flex items-center">
            <div className="shrink-0 mr-3">
              <TextImage size={7} name={props.user.getName()}></TextImage>
            </div>
            <div className="">{props.user.getName()}</div>
          </div>
        </div>
      </td>
      <td>
        <div className="font-normal text-left max-w-[20rem] truncate">
          {props.user.getEmail()}
        </div>
      </td>
      <td>
        <div className="whitespace-no-wrap px-2 md:px-5 py-3">
          <RoleIndicator role={'SUPER_ADMIN'} />
        </div>
      </td>
      <td>
        <div className="font-normal text-left underline decoration-dotted">
          {toHumanReadableDate(props.user.getCreateddate()!)}
        </div>
      </td>
      <td>
        <div className="whitespace-no-wrap px-2 md:px-5 py-3">
          <StatusIndicator state={props.user.getStatus()} />
        </div>
      </td>
      <td>
        <div className="whitespace-no-wrap px-2 md:px-5 py-3">
          <UserOption id={props.user.getId()}></UserOption>
        </div>
      </td>
    </TableRow>
  );
}
