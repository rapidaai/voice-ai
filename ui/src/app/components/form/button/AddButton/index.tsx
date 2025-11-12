import {
  BorderButton,
  Button,
  ButtonProps,
} from '@/app/components/form/button';
import { useRapidaStore } from '@/hooks';
import React from 'react';
import { cn } from '@/utils';
import { PlusIcon } from '@/app/components/Icon/plus';
import { Spinner } from '@/app/components/loader/spinner';

interface AddButtonProp extends ButtonProps {
  label?: string;
}
/**
 *
 * @returns
 */
export function AddButton(props: AddButtonProp) {
  const { isBlocking } = useRapidaStore();
  return (
    <Button
      {...props}
      disabled={isBlocking() || props.disabled}
      className={cn(
        isBlocking() ? 'opacity-80' : 'opacity-100',
        props.className,
      )}
    >
      {isBlocking() ? (
        <div role="status" className="mr-1">
          <Spinner size="sm" className="border-white" />
          <span className="sr-only">loading...</span>
        </div>
      ) : (
        <div role="status" className="mr-1">
          <PlusIcon className="w-4 h-4" />
          <span className="sr-only">Create...</span>
        </div>
      )}
      {props.label && <span className="font-medium">{props.label}</span>}
      {props.children}
    </Button>
  );
}

/**
 *
 * @returns
 */
export function AddBorderButton(props: AddButtonProp) {
  const { isBlocking } = useRapidaStore();
  return (
    <BorderButton
      {...props}
      disabled={isBlocking()}
      className={cn(
        isBlocking() ? 'opacity-80' : 'opacity-100',
        props.className,
      )}
    >
      {isBlocking() ? (
        <Spinner className="border-white" />
      ) : (
        <div role="status" className="mr-1">
          <PlusIcon className="h-4 w-4" />
          <span className="sr-only">Create...</span>
        </div>
      )}

      {props.label && <span className="font-medium">{props.label}</span>}
      {props.children}
    </BorderButton>
  );
}
