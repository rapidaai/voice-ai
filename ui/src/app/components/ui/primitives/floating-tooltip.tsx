import type { ReactNode } from 'react';
import { useState } from 'react';
import {
  PortalToFollowElem,
  PortalToFollowElemContent,
  PortalToFollowElemTrigger,
} from '@/app/components/ui/primitives/portal-to-follow-elem';
import { Placement } from '@floating-ui/react';
import { cn } from '@/utils';

type TooltipProps = {
  className?: string;
  content: ReactNode;
  placement?: Placement;
  children?: ReactNode;
};

export function Tooltip({
  className,
  content,
  children,
  placement,
}: TooltipProps) {
  const [open, setOpen] = useState(false);

  return (
    <PortalToFollowElem
      open={open}
      onOpenChange={setOpen}
      placement={placement ? placement : 'top-start'}
    >
      <PortalToFollowElemTrigger
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
      >
        <div className="flex items-center">{children}</div>
      </PortalToFollowElemTrigger>
      <PortalToFollowElemContent
        style={{ zIndex: 1001 }}
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
      >
        <div
          className={cn(
            'p-3 text-xs font-medium shadow-lg bg-surface border-[0.05px]',
            className,
          )}
        >
          {content}
        </div>
      </PortalToFollowElemContent>
    </PortalToFollowElem>
  );
}
