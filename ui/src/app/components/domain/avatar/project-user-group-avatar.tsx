import {
  UserAvatar,
  type UserAvatarSize,
} from '@/app/components/ui/primitives';

interface ProjectUserGroupAvatarProps {
  size?: UserAvatarSize;
  members: { name: string }[];
  label?: string;
}

export function ProjectUserGroupAvatar({
  size,
  members,
  label = 'Project members',
}: ProjectUserGroupAvatarProps) {
  return (
    <ul
      aria-label={label}
      className="flex flex-wrap justify-center sm:justify-start mb-8 sm:mb-0 -space-x-2 -ml-px"
    >
      {members.map((member, index) => (
        <li key={`${member.name}-${index}`}>
          <div className="block rounded-full border border-border-subtle hover:border-border-strong">
            <UserAvatar
              aria-label={member.name.trim() || 'User'}
              name={member.name.trim()}
              role="img"
              size={size}
            />
          </div>
        </li>
      ))}
    </ul>
  );
}
