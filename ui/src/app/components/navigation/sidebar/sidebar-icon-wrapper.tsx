import React, { HTMLAttributes } from 'react';

export function SidebarIconWrapper(props: HTMLAttributes<HTMLDivElement>) {
  return <div className="px-2.5 py-2.5">{props.children}</div>;
}
