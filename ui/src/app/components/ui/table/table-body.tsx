import { cn } from '@/utils';
import { TableBody as CarbonTableBody } from '@carbon/react';
import type { ComponentProps } from 'react';

export function TableBody(props: ComponentProps<typeof CarbonTableBody>) {
  return (
    <CarbonTableBody {...props} className={cn('text-[15px]', props.className)}>
      {props.children}
    </CarbonTableBody>
  );
}
