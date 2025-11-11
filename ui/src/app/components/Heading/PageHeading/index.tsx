import React, { HTMLAttributes } from 'react';

export function PageHeading(props: HTMLAttributes<HTMLDivElement>) {
  return <div {...props}>{props.children}</div>;
}
