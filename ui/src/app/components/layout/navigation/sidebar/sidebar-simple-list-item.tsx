import { CustomLink } from '@/app/components/ui/primitives';
import type { HTMLAttributes } from 'react';
import { cn } from '@/utils';
import { SkeletonIcon, SkeletonText } from '@carbon/react';
import { useSidebar } from '@/context/sidebar-context';

interface SidebarLinkItemProps extends HTMLAttributes<HTMLDivElement> {
  active?: boolean;
  redirect?: boolean;
  navigate: string;
  loading?: boolean;
}

export function SidebarSimpleListItem(props: SidebarLinkItemProps) {
  const {
    active,
    redirect,
    navigate,
    loading,
    className,
    children,
    ...dProps
  } = props;
  const { open } = useSidebar();

  const isLoading = loading;

  if (isLoading) {
    return (
      <div className={cn('flex h-8 w-full items-center px-1', className)}>
        <div className="flex h-8 w-12 flex-shrink-0 items-center justify-center">
          <SkeletonIcon className="!h-4 !w-4" />
        </div>
        {open && <SkeletonText className="!mb-0 flex-1" width="70%" />}
      </div>
    );
  }

  return (
    <CustomLink to={navigate} isExternal={redirect}>
      <div
        {...dProps}
        className={cn(
          'relative flex h-8 w-full cursor-pointer items-center',
          'text-muted',
          'transition-colors duration-100',
          'hover:bg-layer-hover hover:text-foreground',
          active && [
            'bg-layer-selected text-foreground',
            'before:absolute before:inset-y-0 before:left-0 before:w-[3px] before:bg-primary before:content-[""]',
          ],
          className,
        )}
      >
        {children}
      </div>
    </CustomLink>
  );
}
