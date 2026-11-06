import { TextImage } from '@/app/components/ui/primitives';

interface ProjectUserGroupAvatarProps {
  size?: 7 | 8 | 9;
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
        <li key={`${member.name}-${index}`} aria-label={member.name}>
          <div className="block rounded-[2px] border border-border-subtle hover:border-border-strong ">
            <TextImage size={size} name={member.name} />
          </div>
        </li>
      ))}
    </ul>
  );
}
