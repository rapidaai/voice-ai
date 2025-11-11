import type { FC, HtmlHTMLAttributes } from 'react';
import React, { useRef } from 'react';
import cn from 'classnames';
import { useBoolean } from 'ahooks';
import { IButton, IconButton } from '@/app/components/Form/Button';
import { CircleCheck, Clipboard, MaximizeIcon, XIcon } from 'lucide-react';
import { useToggleExpend } from '@/hooks/use-toggle-expend';
import { TickIcon } from '@/app/components/Icon/Tick';
import { CopyIcon } from '@/app/components/Icon/Copy';
import { CloseIcon } from '@/app/components/Icon/Close';
import { ExpandIcon } from '@/app/components/Icon/Expand';

interface BaseEditorProps extends HtmlHTMLAttributes<HTMLDivElement> {
  minHeight?: number;
  isFocus: boolean;
  leftAction?: (boolean) => JSX.Element;
  rightAction?: (boolean) => JSX.Element;
  placeholder?: string;
  value: string;
}

const BaseEditor: FC<BaseEditorProps> = ({
  className,
  children,
  minHeight,
  value,
  isFocus,
}) => {
  const ref = useRef<HTMLDivElement>(null);
  const { isExpand, setIsExpand } = useToggleExpend(ref);

  const [isChecked, { setTrue: setChecked, setFalse: setUnCheck }] =
    useBoolean(false);

  const copyItem = (item: string) => {
    setChecked();
    navigator.clipboard.writeText(item);
    setTimeout(() => {
      setUnCheck();
    }, 4000); // Reset back after 2 seconds
  };

  return (
    <div
      ref={ref}
      className={cn(
        'group',
        'outline-solid outline-[1.5px] outline-transparent',
        'focus-within:outline-blue-600 focus:outline-blue-600 outline-offset-[-1.5px]',
        'border-b border-gray-400 dark:border-gray-600',
        'dark:focus:border-blue-600 focus:border-blue-600',
        'transition-all duration-200 ease-in-out',
        'relative',
        'bg-white dark:bg-gray-950',
        isFocus && 'border-blue-600! outline-blue-600! ',
        isExpand
          ? 'fixed top-0 bottom-0 right-0 left-0 h-full z-50 m-0! p-0!'
          : '',
      )}
    >
      <div
        className={cn(
          'flex items-center absolute z-20 bg-light-background dark:bg-gray-950 top-0 right-0 border divide-x',
        )}
      >
        <IButton
          className="h-8"
          tabIndex={-1}
          onClick={() => {
            copyItem(value);
          }}
        >
          {isChecked ? (
            <TickIcon className="h-3.5 w-3.5 text-green-600" />
          ) : (
            <CopyIcon className="h-3.5 w-3.5" />
          )}
        </IButton>
        <IButton
          className="h-8"
          tabIndex={-1}
          onClick={() => {
            setIsExpand(!isExpand);
          }}
        >
          {isExpand ? (
            <CloseIcon className="h-3.5 w-3.5" />
          ) : (
            <ExpandIcon className="h-3.5 w-3.5" />
          )}
        </IButton>
      </div>
      {children}
    </div>
  );
};
export default React.memo(BaseEditor);
