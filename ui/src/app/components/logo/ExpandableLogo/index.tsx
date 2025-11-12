import React, { HTMLAttributes } from 'react';
import { cn } from '@/utils';

export function ExpandableLogo(porps: HTMLAttributes<HTMLDivElement>) {
  return (
    <div {...porps}>
      <div className="grid place-items-center items-center">
        <img
          src="/images/logos/icon-01.png"
          className="inline-block"
          alt="rapida_logo"
        />
      </div>
      <span
        className="ml-3 text-dark-500 text-3xl font-medium"
        sidebar-toggle-item=""
      >
        {/* <img
          src="/images/logos/logo-06.png"
          className={cn('h-10 ')}
          alt="rapida_logo"
        />
        <img
          src="/images/logos/logo-04.png"
          className="h-10 hidden dark:group-hover:inline-block p-1"
          alt="rapida_logo"
        /> */}
      </span>
    </div>
  );
}
