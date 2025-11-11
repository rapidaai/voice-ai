import React, { HTMLAttributes } from 'react';
import { cn } from '@/styles/media';

export const BetaIcon = React.forwardRef<
  HTMLOrSVGElement,
  HTMLAttributes<HTMLSpanElement>
>((props, ref) => (
  <span
    className={cn(
      'ml-2 text-xs rounded-[2px] from-rose-600 via-pink-600 to-blue-600 bg-linear-to-r bg-clip-text text-transparent font-medium',
      props.className,
    )}
    style={{
      WebkitBackgroundClip: 'text',
    }}
  >
    {props.children ? props.children : 'beta'}
  </span>
));
