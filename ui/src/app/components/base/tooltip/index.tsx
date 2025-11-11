import React, { useState } from 'react';
import type { FC } from 'react';
import {
  PortalToFollowElem,
  PortalToFollowElemContent,
  PortalToFollowElemTrigger,
} from '@/app/components/portal-to-follow-elem';
import { Placement } from '@floating-ui/react';
import { cn } from '@/styles/media';

type TooltipProps = {
  className?: string;
  content: React.ReactNode;
  placement?: Placement;
  children?: any;
};

export const Tooltip: FC<TooltipProps> = ({
  className,
  content,
  children,
  placement,
}) => {
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
            'p-3 text-xs font-medium shadow-lg bg-white dark:bg-slate-800 border-[0.05px]',
            className,
          )}
        >
          {content}
        </div>
      </PortalToFollowElemContent>
    </PortalToFollowElem>
  );
};
