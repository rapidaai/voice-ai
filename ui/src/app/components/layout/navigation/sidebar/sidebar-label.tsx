import type { HTMLAttributes } from 'react';
import { cn } from '@/utils';
import { useSidebar } from '@/context/sidebar-context';
import { Text } from '@/app/components/ui/primitives';

interface SidebarLabelProps extends HTMLAttributes<HTMLSpanElement> {
  isLoading?: boolean;
}

export function SidebarLabel({
  isLoading,
  className,
  children,
  ...attributes
}: SidebarLabelProps) {
  const { open } = useSidebar();

  return (
    <span
      {...attributes}
      className={cn(
        'flex-1 truncate transition-all duration-200',
        open ? 'opacity-100' : 'opacity-0 w-0',
        className,
      )}
    >
      <Text isLoading={isLoading} skeletonWidth="70%" variant="body-compact-01">
        {children}
      </Text>
    </span>
  );
}
