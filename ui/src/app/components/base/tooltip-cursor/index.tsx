import { cn } from '@/styles/media';
import React, { useState, useEffect, useRef } from 'react';

interface TooltipCursorProps {
  content: any;
  children: React.ReactNode;
  delay: number;
}

export const TooltipCursor: React.FC<TooltipCursorProps> = ({
  children,
  content,
  delay,
}) => {
  const [isTooltipVisible, setTooltipVisible] = useState(false);
  const [tooltipPosition, setTooltipPosition] = useState({ x: 0, y: 0 });
  const [showTooltipContent, setShowTooltipContent] = useState(false);

  const tooltipRef = useRef<HTMLDivElement>(document.createElement('div'));

  // ... component logic

  const handleMouseMove = (event: React.MouseEvent) => {
    const { clientX, clientY } = event;

    const tooltipWidth = tooltipRef.current?.offsetWidth || 0;
    const tooltipHeight = tooltipRef.current?.offsetHeight || 0;
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;

    //+12 is added to give a spice between cursor and tooltip
    let tooltipX = clientX + 12;
    let tooltipY = clientY + 12;

    // Check if tooltip exceeds the right side of the viewport
    if (tooltipX + tooltipWidth > viewportWidth) {
      tooltipX = clientX - tooltipWidth - 10;
    }

    // Check if tooltip exceeds the bottom of the viewport
    if (tooltipY + tooltipHeight > viewportHeight) {
      tooltipY = viewportHeight - tooltipHeight - 10;
    }

    setTooltipPosition({ x: tooltipX, y: tooltipY });
  };

  const handleMouseEnter = () => {
    setTooltipVisible(true);
    setShowTooltipContent(false);
  };

  const handleMouseLeave = () => {
    setTooltipVisible(false);
  };

  return (
    <div
      className="min-w-min tooltip-cursor-wrapper"
      onMouseMove={handleMouseMove}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      {isTooltipVisible && (
        <div
          ref={tooltipRef}
          className={cn(
            'fixed px-3 py-1.5 backdrop-blur-md rounded-[2px] bg-white/80 dark:bg-gray-800/80 dark:text-gray-400 shadow-sm z-50 text-sm font-medium max-w-md',
          )}
          style={{
            top: tooltipPosition.y,
            left: tooltipPosition.x,
            zIndex: '2147483647',
          }}
        >
          {content}
        </div>
      )}
      {children}
    </div>
  );
};
