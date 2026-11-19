import { UserAvatar } from '@/app/components/ui/primitives';
import { User } from '@rapidaai/react';
import { toHumanReadableDate } from '@/utils/date';
import { TableCell, Tag } from '@carbon/react';
import { CarbonIconIndicator } from '@/app/components/ui/feedback';
import { CopyButton } from '@/app/components/ui/primitives';

export function SingleUser({ user }: { user: User }) {
  return (
    <>
      <TableCell>{user.getId()}</TableCell>
      <TableCell>
        <div className="flex items-center gap-3">
          <UserAvatar
            aria-label={user.getName().trim() || 'User'}
            name={user.getName().trim()}
            role="img"
            size="sm"
          />
          <span>{user.getName()}</span>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2 min-w-0 max-w-[280px]">
          <span className="truncate">{user.getEmail()}</span>
          <div className="shrink-0" onClick={e => e.stopPropagation()}>
            <CopyButton className="h-6 w-6">{user.getEmail()}</CopyButton>
          </div>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex flex-wrap gap-1">
          {user.getOrganizationrolesList().length > 0
            ? user.getOrganizationrolesList().map(role => (
                <Tag key={role.getRoledefinitionid()} type="gray" size="sm">
                  {role.getName()}
                </Tag>
              ))
            : '-'}
        </div>
      </TableCell>
      <TableCell>
        {user.getCreateddate() && toHumanReadableDate(user.getCreateddate()!)}
      </TableCell>
      <TableCell>
        <CarbonIconIndicator state={user.getStatus?.()} />
      </TableCell>
    </>
  );
}
