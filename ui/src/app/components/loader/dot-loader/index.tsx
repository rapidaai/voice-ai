import React, { HTMLAttributes } from 'react';

export function DotLoader(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className="flex flex-row gap-0.5 items-center animate-pulse">
      <div className="w-1 h-1 rounded-[2px] bg-blue-500 animate-bounce-custom [animation-delay:-0.3s]"></div>
      <div className="w-1 h-1 rounded-[2px] bg-blue-500 animate-bounce-custom [animation-delay:-0.15s]"></div>
      <div className="w-1 h-1 rounded-[2px] bg-blue-500 animate-bounce-custom"></div>
    </div>
  );
}
