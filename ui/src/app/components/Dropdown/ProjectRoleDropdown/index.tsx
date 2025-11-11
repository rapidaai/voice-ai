import React from 'react';
import { cn } from '@/styles/media';
import { Dropdown } from '@/app/components/Dropdown';

const Roles = ['super admin', 'admin', 'writer', 'reader'];

export function ProjectRoleDropdown(props: {
  projectRole: string;
  setProjectRoleId: (string) => void;
}) {
  return (
    <Dropdown
      allValue={Roles}
      currentValue={props.projectRole}
      setValue={props.setProjectRoleId}
      className="bg-white dark:bg-gray-950"
      placeholder="Select a project role"
      label={prj => {
        return (
          <span className={cn('block truncate capitalize font-medium text-sm')}>
            {prj}
          </span>
        );
      }}
      option={(prj, selected) => {
        return (
          <span
            className={cn(
              'block truncate text-sm capitalize',
              selected ? 'opacity-100 font-medium' : 'opacity-80',
            )}
          >
            {prj}
          </span>
        );
      }}
    />
  );
}
