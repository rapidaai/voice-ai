import { Spinner } from '@/app/components/loader/spinner';
import React, { FC } from 'react';
import { cn } from '@/utils';
import { ChevronRight, PlusIcon } from 'lucide-react';
/**
 *
 */
export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /**
   *
   */
  children?: any;

  /**
   * if loading represent
   */
  isLoading?: boolean;

  /**
   *
   */
  size?: 'sm' | 'md' | 'lg';
}

export interface LinkProps
  extends React.AnchorHTMLAttributes<HTMLAnchorElement> {
  /**
   *
   */
  children?: any;

  /**
   * if loading represent
   */
  isLoading?: boolean;
}

export function Button(props: ButtonProps) {
  const { isLoading, ...btnProps } = props;
  return (
    <button
      {...btnProps}
      className={cn(
        'rounded-[2px]',
        'flex h-9 leading-7 truncate w-fit justify-center items-center relative',
        'bg-blue-600 text-white hover:bg-blue-500 py-1.5 px-3',
        'button',
        props.disabled && 'opacity-80! cursor-not-allowed!',
        props.className,
      )}
    >
      {isLoading ? (
        <span className="inline-block absolute">
          <Spinner className="border-white" />
        </span>
      ) : (
        props.children
      )}
    </button>
  );
}

export function BlueBorderButton(props: ButtonProps) {
  const { isLoading, ...btnProps } = props;

  return (
    <button
      {...btnProps}
      className={cn(
        'rounded-[2px]',
        'flex h-9 truncate w-fit justify-center items-center',
        'py-1.5 px-3',
        'text-blue-600 dark:text-blue-400',
        'border-[1.5px] border-blue-600 dark:border-blue-600 hover:border-blue-400 dark:hover:border-blue-700',
        'bg-white dark:bg-gray-950/50 dark:hover:bg-blue-700/20 hover:bg-blue-200/20',
        'button',
        props.disabled && 'cursor-not-allowed opacity-70',
        props.className,
      )}
    >
      {isLoading ? (
        <span className="inline-block absolute">
          <Spinner />
        </span>
      ) : (
        props.children
      )}
    </button>
  );
}

export function BorderButton(props: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        'rounded-[2px]',
        'flex h-9 truncate w-fit justify-center items-center',
        'dark:text-gray-400',
        'py-2 px-2.5',
        'border-[1.5px] border-gray-300/50 dark:border-gray-600/50',
        'bg-white dark:bg-gray-950/50 dark:hover:bg-gray-700/50 hover:bg-gray-200',
        'button',
        'focus:outline-hidden focus:outline-2 focus:outline-solid focus:outline-offset-1 outline-gray-100/10',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

export const ILinkBorderButton: FC<LinkProps> = props => {
  const { isLoading, ...btnProps } = props;
  return (
    <a
      {...btnProps}
      className={cn(
        'rounded-[2px]',
        'flex h-10 truncate w-fit justify-center items-center',
        'dark:hover:text-gray-400 dark:hover:text-gray-300',
        'py-1.5 px-3',
        'hover:bg-white dark:hover:bg-gray-950',
        'button',
        'focus:outline-hidden',

        'outline-solid outline-transparent hover:outline-blue-600 -outline-offset-[1.5px]',
        'transition-all delay-200',
        'focus:outline-hidden',
        props.className,
      )}
    >
      {isLoading ? (
        <span className="inline-block absolute">
          <Spinner />
        </span>
      ) : (
        props.children
      )}
    </a>
  );
};

export function SimpleButton(props: ButtonProps) {
  return (
    <button
      {...props}
      className={cn(
        'rounded-[2px]',
        'flex h-10 truncate w-fit justify-center items-center',
        'dark:hover:text-gray-300 hover:text-gray-900',
        'py-3 px-3 ',
        'hover:bg-gray-100 dark:hover:bg-gray-900',
        'button',
        'focus:outline-hidden',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

export function IconButton(props: ButtonProps) {
  return (
    <SimpleButton
      className={cn('rounded-[2px]', 'h-6! w-6! p-[4px]!', props.className)}
      onClick={props.onClick}
      {...props}
    >
      {props.children}
    </SimpleButton>
  );
}

export function ILinkButton(props: LinkProps) {
  return (
    <a
      {...props}
      className={cn(
        'rounded-[2px]',
        'flex h-10 truncate w-fit justify-center items-center',
        'text-white',
        'py-2.5 px-3',
        'font-medium',
        'bg-blue-600 hover:bg-blue-700',
        'button',
        'border-y border-transparent',
        'focus:outline-hidden',
        props.className,
      )}
    >
      {props.children}
    </a>
  );
}

export function HoverButton(props: ButtonProps) {
  const { isLoading, ...btnProps } = props;
  return (
    <button
      {...btnProps}
      className={cn(
        'rounded-[2px]',
        'flex h-9 leading-7 truncate w-fit justify-center items-center relative border-none',
        'py-3 px-3 dark:hover:bg-gray-900 hover:bg-gray-100',
        'button',
        'hover:shadow-sm',
        'focus:outline-solid focus:outline-offset-1 focus:outline-gray-300 dark:focus:outline-gray-700',
        props.className,
      )}
    >
      {isLoading ? (
        <span className="inline-block absolute">
          <Spinner />
        </span>
      ) : (
        props.children
      )}
    </button>
  );
}

export const OutlineButton = (props: ButtonProps) => {
  const { isLoading, className, ...btnProps } = props;
  return (
    <Button
      className={cn(
        'rounded-[2px]',
        'rounded-[2px] px-4 capitalize font-medium',
        className,
        props.disabled && 'opacity-60',
      )}
      type="submit"
      {...btnProps}
    >
      {props.children}
      {props.isLoading && <Spinner className="h-4 w-4 ml-2 border-white" />}
    </Button>
  );
};

//

export function IButton(props: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        'rounded-[2px]',
        'flex h-10 truncate w-fit justify-center items-center',
        'dark:hover:text-gray-400 dark:hover:text-gray-300',
        'py-1.5 px-3',
        'hover:bg-white dark:hover:bg-gray-950',
        'button',
        'focus:outline-hidden',
        !props.disabled &&
          'outline-solid outline-transparent hover:outline-blue-600 -outline-offset-[1.5px]',
        'transition-all delay-200',
        'focus:outline-hidden',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

export function IBlueButton(props: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        'rounded-[2px]',
        'flex h-10 truncate w-fit justify-center items-center',
        'text-blue-500 hover:text-blue-600',
        'py-1.5 px-3',
        'bg-white hover:bg-light-background dark:bg-gray-900 dark:hover:bg-gray-950',
        'button',
        !props.disabled &&
          'outline-solid outline-transparent hover:outline-blue-600 -outline-offset-[1.5px]',
        'transition-all delay-200',
        'focus:outline-hidden',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

export function IBlueBorderPlusButton(props: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        'rounded-[2px]',
        'flex h-10 truncate w-fit justify-start items-center',
        'bg-light-background dark:bg-gray-900',
        'text-blue-600 hover:text-blue-600 border border-blue-600',
        'px-4',
        'text-sm',
        'hover:bg-blue-600 hover:text-white dark:hover:bg-blue-600',
        'button',
        'transition-all delay-200',
        'focus:outline-hidden',
        props.className,
      )}
    >
      {props.children}
      {props.isLoading ? (
        <Spinner className="ml-16 border-white animate-spin" size="xs" />
      ) : (
        <PlusIcon className="ml-12 w-5 h-5" strokeWidth={1.5} />
      )}
    </button>
  );
}

export function IBlueBorderButton(props: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        'rounded-[2px] cursor-pointer',
        'flex h-10 truncate w-fit justify-start items-center',
        'bg-light-background dark:bg-gray-900',
        'text-blue-600 hover:text-blue-600 border border-blue-600',
        'px-4 space-x-16',
        'text-sm',
        'hover:bg-blue-600 dark:hover:bg-blue-600 hover:text-white',
        'button',
        'transition-all delay-200',
        'focus:outline-hidden',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

export function IBorderButton(props: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        'rounded-[2px]',
        'flex h-10 truncate w-fit justify-start items-center',
        'bg-light-background dark:bg-gray-900',
        'text-gray-600 hover:text-gray-600 border border-gray-300 dark:border-gray-600',
        'px-4',
        'text-sm',
        'hover:bg-gray-600 dark:hover:bg-gray-600 hover:text-white',
        'button',
        'transition-all delay-200',
        'focus:outline-hidden',
        props.className,
      )}
    >
      {!props.isLoading ? (
        props.children
      ) : (
        <Spinner className="w-4 h-4 border-white" />
      )}
    </button>
  );
}

export function IBlueBGButton(props: ButtonProps) {
  const { isLoading, ...alt } = props;
  return (
    <button
      type="button"
      {...alt}
      className={cn(
        'rounded-[2px]',
        'flex h-10 dark:border-[1.4px] border-gray-900 truncate w-fit justify-center items-center cursor-pointer',
        'text-white',
        'py-2.5 px-3',
        'font-medium',
        'bg-blue-600 hover:bg-blue-700',
        'button',
        'focus:outline-hidden',
        props.isLoading && 'cursor-wait',
        props.className,
      )}
    >
      {!props.isLoading ? (
        props.children
      ) : (
        <Spinner className="w-4 h-4 border-white" />
      )}
    </button>
  );
}

export function ICancelButton(props: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        'rounded-[2px]',
        'flex h-10 truncate w-fit justify-center items-center',
        'py-2.5 px-3',
        'font-medium',
        'border border-gray-300 hover:bg-gray-200',
        'bg-white dark:bg-gray-800',
        'cursor-pointer',
        'dark:border-gray-800 dark:text-gray-400 dark:hover:bg-gray-900',
        'button',
        'focus:outline-hidden',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

export function IRedBGButton(props: ButtonProps) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        'rounded-[2px]',
        'flex h-10 truncate w-fit justify-center items-center',
        'text-white',
        'py-3 px-3 ',
        'disabled:opacity-80',
        'bg-red-600 group-hover:bg-red-600',
        'button',
        'focus:outline-hidden',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

export function IRedBorderButton(props: ButtonProps) {
  const { isLoading, ...btnProps } = props;
  return (
    <button
      type="button"
      {...btnProps}
      className={cn(
        'rounded-[2px]',
        'flex h-10 truncate w-fit justify-center items-center border',
        'text-red-600 hover:text-white',
        'py-3 px-3 ',
        'bg-red-600/5 transition-all delay-200',
        'border-red-600 hover:bg-red-600',
        'button',
        'focus:outline-hidden',
        props.className,
      )}
    >
      {props.children}
    </button>
  );
}

export function IBlueBGArrowButton(props: ButtonProps) {
  const { isLoading, ...btnProps } = props;
  return (
    <IBlueBGButton
      {...btnProps}
      className={cn('rounded-[2px]', props.className)}
      disabled={isLoading}
    >
      {props.children}
      {props.isLoading ? (
        <Spinner className="w-5 h-5 ml-3 border-white" />
      ) : (
        <ChevronRight className="w-5 h-5 ml-3" strokeWidth={1.5} />
      )}
    </IBlueBGButton>
  );
}
