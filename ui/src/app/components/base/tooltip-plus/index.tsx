import type { FC, HTMLAttributes } from 'react';
import React, { useState } from 'react';
import {
  PortalToFollowElem,
  PortalToFollowElemContent,
  PortalToFollowElemTrigger,
} from '@/app/components/portal-to-follow-elem';
import { cn } from '@/styles/media';
interface TooltipProps extends HTMLAttributes<HTMLDivElement> {
  position?: 'top' | 'right' | 'bottom' | 'left';
  triggerMethod?: 'hover' | 'click';
  popupContent: React.ReactNode;
  children: React.ReactNode;
  hideArrow?: boolean;
}

const arrow = (
  <svg
    className="absolute text-white dark:text-gray-950 h-2 w-full left-0 top-full"
    x="0px"
    y="0px"
    viewBox="0 0 255 255"
  >
    <polygon className="fill-current" points="0,0 127.5,127.5 255,0"></polygon>
  </svg>
);

const Tooltip: FC<TooltipProps> = ({
  position = 'top',
  triggerMethod = 'hover',
  popupContent,
  children,
  hideArrow,
  className,
}) => {
  const [open, setOpen] = useState(false);

  return (
    <PortalToFollowElem
      open={open}
      onOpenChange={setOpen}
      placement={position}
      offset={10}
    >
      <PortalToFollowElemTrigger
        onClick={() => triggerMethod === 'click' && setOpen(v => !v)}
        onMouseEnter={() => triggerMethod === 'hover' && setOpen(true)}
        onMouseLeave={() => triggerMethod === 'hover' && setOpen(false)}
      >
        {children}
      </PortalToFollowElemTrigger>
      <PortalToFollowElemContent className="z-9999">
        <div
          className={cn(
            'relative px-3 py-2 text-xs font-normal rounded-[2px] shadow-lg border dark:border-gray-800',
            'bg-white dark:bg-gray-700',
            'dark:bg-gray-900/90 dark:text-gray-300',
            className,
          )}
        >
          {popupContent}
          {!hideArrow && arrow}
        </div>
      </PortalToFollowElemContent>
    </PortalToFollowElem>
  );
};

export default React.memo(Tooltip);
