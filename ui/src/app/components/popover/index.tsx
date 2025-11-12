import React, { Fragment, useRef, useEffect, HTMLAttributes } from 'react';
import { Transition } from '@headlessui/react';
import { cn } from '@/utils';
import {
  offset,
  useDismiss,
  useFloating,
  useInteractions,
} from '@floating-ui/react';
import { FloatingArrow, arrow } from '@floating-ui/react';
import { Placement } from '@floating-ui/utils';
/**
 *
 */
export interface PopoverProps extends HTMLAttributes<HTMLDivElement> {
  align: Placement;
  open: boolean;
  setOpen: (boolean) => void;
}
/**
 *
 * @param props
 * @returns
 */
export function Popover(props: PopoverProps) {
  useEffect(() => {
    const keyHandler = ({ keyCode }) => {
      if (!props.setOpen || keyCode !== 27) return;
      props.setOpen(false);
    };
    document.addEventListener('keydown', keyHandler);
    return () => document.removeEventListener('keydown', keyHandler);
  });

  const arrowRef = useRef(null);

  const { refs, floatingStyles, context } = useFloating({
    open: props.open,
    onOpenChange: props.setOpen,
    placement: props.align,
    transform: false,
    middleware: [
      offset(10),
      arrow({
        element: arrowRef,
      }),
    ],
  });

  const dismiss = useDismiss(context);
  const { getFloatingProps } = useInteractions([dismiss]);

  return (
    <div ref={refs.setReference}>
      <Transition
        show={props.open}
        as={Fragment}
        enter="transition ease-out duration-100"
        enterFrom="transform opacity-0 scale-95"
        enterTo="transform opacity-100 scale-100"
        leave="transition ease-in duration-75"
        leaveFrom="transform opacity-100 scale-100"
        leaveTo="transform opacity-0 scale-95"
      >
        <div
          ref={refs.setFloating}
          style={floatingStyles}
          {...getFloatingProps()}
          className={cn(
            'z-12 w-80 bg-light-background dark:bg-gray-900 border shadow-lg border-t',
            props.className,
          )}
          onFocus={() => props.setOpen(true)}
          onBlur={() => props.setOpen(false)}
        >
          <FloatingArrow
            ref={arrowRef}
            context={context}
            staticOffset={'5%'}
            className="w-5 h-5 fill-gray-300 dark:fill-gray-600"
          />
          {props.children}
        </div>
      </Transition>
    </div>
  );
}
