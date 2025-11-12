import React from 'react';
import { cn } from '@/utils';
import { Button } from '@/app/components/form/button';
import { useRapidaStore } from '@/hooks';
import { Spinner } from '@/app/components/loader/spinner';
import { ArrowUpIcon } from '@heroicons/react/24/solid';
import { UpArrowIcon } from '@/app/components/Icon/up-arrow';
/**
 *
 */
interface SendButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  isLoading?: boolean;
}

/**
 *
 * @param props
 * @returns
 */
export function SendButton(props: SendButtonProps) {
  /**
   * for disabling and loading
   */
  const { isBlocking } = useRapidaStore();

  return (
    <Button
      {...props}
      disabled={props.isLoading || props.disabled || isBlocking()}
      className={cn(
        'h-9! w-9!',
        isBlocking() ? 'opacity-80' : 'opacity-100',
        props.className,
      )}
    >
      {/* <span className="font-medium text-sm">{props.children}</span> */}
      {props.isLoading || isBlocking() ? (
        <Spinner className="border-white" />
      ) : (
        // <div role="status" className="ml-1">
        <UpArrowIcon strokeWidth={1.5} className="text-white" />
        // </div>
      )}
    </Button>
  );
}
