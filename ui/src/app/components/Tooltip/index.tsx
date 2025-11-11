import React from 'react';
import { Tooltip as TP } from '@material-tailwind/react';
import { cn } from '@/styles/media';

export function Tooltip(props: { children: any; icon: React.ReactElement }) {
  const colorClasses = () => {
    return 'bg-white border-gray-200 dark:bg-gray-700 dark:text-gray-100 dark:border-gray-600';
  };
  return (
    <TP
      className={cn(
        colorClasses(),
        'border overflow-hidden shadow-lg shrink-0',
        'z-50',
      )}
      content={props.children}
    >
      {props.icon}
    </TP>
  );
}
