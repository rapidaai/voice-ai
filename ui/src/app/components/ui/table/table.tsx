import { cn } from '@/utils';
import { Table as CarbonTable } from '@carbon/react';
import type { ComponentProps } from 'react';

export function Table(props: ComponentProps<typeof CarbonTable>) {
  return (
    <CarbonTable {...props} className={cn('text-sm', props.className)}>
      {props.children}
    </CarbonTable>
  );
}
