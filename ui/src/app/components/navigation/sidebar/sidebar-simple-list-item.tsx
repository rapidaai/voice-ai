import { CustomLink } from '@/app/components/custom-link';
import React, { HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

interface SidebarLinkItemProps extends HTMLAttributes<HTMLDivElement> {
  active?: boolean;
  redirect?: boolean;
  navigate: string;
}

export function SidebarSimpleListItem(props: SidebarLinkItemProps) {
  const { active, redirect, navigate, ...dProps } = props;
  return (
    <CustomLink to={navigate} isExternal={redirect}>
      <div
        {...dProps}
        className={cn(
          'flex items-center mx-2 hover:bg-gray-200 dark:hover:bg-gray-950 cursor-pointer',
          active && 'bg-gray-200 dark:bg-gray-950 text-blue-600',
          props.className,
        )}
      >
        {props.children}
      </div>
    </CustomLink>
  );
}
