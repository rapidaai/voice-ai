import { SidePanelClose, SidePanelOpen } from '@carbon/icons-react';
import { Button } from '@carbon/react';
import { forwardRef, type HTMLAttributes, type ReactNode } from 'react';
import { typographyClassNames } from '@/app/components/ui/primitives/text';
import { cn } from '@/utils';

interface SideNavigationPanelProps extends HTMLAttributes<HTMLDivElement> {
  expanded: boolean;
}

export function SideNavigationPanel({
  expanded,
  className,
  children,
  style,
  ...attributes
}: SideNavigationPanelProps) {
  return (
    <div
      {...attributes}
      className={cn(
        'relative flex h-full shrink-0 flex-col',
        'border-r border-border-subtle bg-shell text-foreground',
        'transition-[width] duration-200 ease-in-out',
        className,
      )}
      style={{ ...style, width: expanded ? '13.5rem' : '3rem' }}
    >
      {children}
    </div>
  );
}

interface SideNavigationSectionLabelProps {
  as?: 'div' | 'li';
  children: ReactNode;
  className?: string;
  expanded: boolean;
}

export function SideNavigationSectionLabel({
  as: Component = 'li',
  children,
  className,
  expanded,
}: SideNavigationSectionLabelProps) {
  return (
    <Component
      className={cn(
        'flex min-h-8 items-end px-4 pb-1 pt-3',
        typographyClassNames['label-01'],
        'text-helper',
        'transition-all duration-200',
        !expanded && 'h-0 min-h-0 overflow-hidden border-none p-0 opacity-0',
        className,
      )}
    >
      {children}
    </Component>
  );
}

interface SideNavigationCollapseControlProps {
  'aria-label': string;
  expanded: boolean;
  onClick: () => void;
  showLabel?: boolean;
}

export const SideNavigationCollapseControl = forwardRef<
  HTMLButtonElement,
  SideNavigationCollapseControlProps
>(function SideNavigationCollapseControl(
  { expanded, onClick, showLabel = expanded, ...attributes },
  ref,
) {
  return (
    <Button
      {...attributes}
      ref={ref}
      type="button"
      kind="ghost"
      size="sm"
      onClick={onClick}
      className={cn(
        'app-side-navigation-collapse-control',
        'flex! h-8! min-h-8! w-full! max-w-none! items-center! justify-start! px-4!',
        'text-muted! hover:bg-layer-hover! hover:text-foreground!',
        'transition-colors! duration-100!',
      )}
    >
      <span className="flex h-4 w-4 shrink-0 items-center justify-center">
        {expanded ? <SidePanelClose size={16} /> : <SidePanelOpen size={16} />}
      </span>
      <span
        className={cn(
          typographyClassNames['label-01'],
          'ml-3 truncate transition-all duration-200',
          showLabel ? 'opacity-100' : 'ml-0 w-0 overflow-hidden opacity-0',
        )}
      >
        {expanded ? 'Collapse' : 'Expand'}
      </span>
    </Button>
  );
});
