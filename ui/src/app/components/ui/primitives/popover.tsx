import type { ElementType } from 'react';
import {
  Popover as CarbonPopover,
  PopoverContent,
  type PopoverProps as CarbonPopoverProps,
} from '@carbon/react';
import { cn } from '@/utils';

export type PopoverProps<E extends ElementType = 'span'> = Omit<
  CarbonPopoverProps<E>,
  'caret'
>;

/** Carbon Popover with the caretless behavior that becomes standard in v12. */
export function Popover<E extends ElementType = 'span'>({
  className,
  ...props
}: PopoverProps<E>) {
  return (
    <CarbonPopover
      {...(props as CarbonPopoverProps<E>)}
      className={cn(
        'app-popover--without-caret [--cds-popover-offset:0.25rem]! [&>.cds--popover>.cds--popover-caret]:!hidden [&>.cds--popover>.cds--popover-content>.cds--popover-caret]:!hidden',
        className,
      )}
    />
  );
}

export { PopoverContent };
