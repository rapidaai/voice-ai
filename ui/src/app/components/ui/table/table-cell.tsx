import { cn } from '@/utils';
import { TableCell as CarbonTableCell } from '@carbon/react';
import type { ComponentProps } from 'react';

export function TableCell(props: ComponentProps<typeof CarbonTableCell>) {
  return (
    <CarbonTableCell
      {...props}
      className={cn(
        'whitespace-no-wrap p-0 m-0 px-2 md:px-5 py-2',
        props.className,
      )}
    >
      {props.children}
    </CarbonTableCell>
  );
}
