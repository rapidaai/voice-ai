import React from 'react';
import { UserAvatar } from '@/app/components/ui/primitives';
import { User } from '@rapidaai/react';
import { OrganizationRoleIndicator } from '@/app/components/domain/indicators/organization-role';
import { toHumanReadableDate } from '@/utils/date';
import { RadioButton, TableRow, TableCell } from '@carbon/react';
import { CarbonIconIndicator } from '@/app/components/ui/feedback';
import { CopyButton } from '@/app/components/ui/primitives';

export function SingleUser(props: {
  user: User;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <TableRow
      isSelected={props.selected}
      onClick={props.onSelect}
      className="cursor-pointer"
    >
      <TableCell className="!w-12 !pr-0" onClick={e => e.stopPropagation()}>
        <RadioButton
          id={`user-select-${props.user.getId()}`}
          name="user-select"
          labelText=""
          hideLabel
          checked={props.selected}
          onChange={props.onSelect}
        />
      </TableCell>
      <TableCell>{props.user.getId()}</TableCell>
      <TableCell>
        <div className="flex items-center gap-3">
          <UserAvatar
            aria-label={props.user.getName().trim() || 'User'}
            name={props.user.getName().trim()}
            role="img"
            size="sm"
          />
          <span>{props.user.getName()}</span>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2 min-w-0 max-w-[280px]">
          <span className="truncate">{props.user.getEmail()}</span>
          <div className="shrink-0" onClick={e => e.stopPropagation()}>
            <CopyButton className="h-6 w-6">{props.user.getEmail()}</CopyButton>
          </div>
        </div>
      </TableCell>
      <TableCell>
        <OrganizationRoleIndicator role={props.user.getRole()} />
      </TableCell>
      <TableCell>
        {props.user.getCreateddate() &&
          toHumanReadableDate(props.user.getCreateddate()!)}
      </TableCell>
      <TableCell>
        <CarbonIconIndicator state={props.user.getStatus?.()} />
      </TableCell>
    </TableRow>
  );
}
