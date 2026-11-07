import { cn } from '@/utils';
import { TableRow as CarbonTableRow } from '@carbon/react';
import type { ComponentProps } from 'react';

export function TableRow(props: ComponentProps<typeof CarbonTableRow>) {
  return (
    <CarbonTableRow
      {...props}
      className={cn(
        'divide-border-subtle border-border-subtle border-b-[0.5px] divide-x hover:bg-layer text-sm/6 text-pretty hover:text-muted',
        props.className,
      )}
    >
      {props.children}
    </CarbonTableRow>
  );
}
