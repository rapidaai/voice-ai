import { ProjectRole } from '@rapidaai/react';
import { cn } from '@/utils';
import { Dropdown } from '@/app/components/dropdown';

/**
 *
 * @param props
 * @returns
 */
export function ProjectSelectorDropdown(props: {
  projects: ProjectRole.AsObject[];
  project: ProjectRole.AsObject | undefined;
  setProject: (project: ProjectRole.AsObject) => void;
  placement?: 'top' | 'bottom';
}) {
  return (
    <Dropdown
      className={cn(
        'border-none pl-6 h-12 min-w-[200px]',
        'hover:bg-gray-100 dark:hover:bg-gray-950',
      )}
      placement={props.placement}
      allValue={props.projects}
      currentValue={props.project}
      setValue={props.setProject}
      placeholder="Select a Project"
      label={prj => {
        return (
          <span className={cn('block truncate capitalize font-medium text-sm')}>
            {prj.projectname}
          </span>
        );
      }}
      option={(prj, selected) => {
        return (
          <span
            className={cn(
              'block truncate capitalize text-sm',
              selected ? 'opacity-100 font-medium' : 'opacity-80',
            )}
          >
            {prj.projectname}
          </span>
        );
      }}
    />
  );
}
