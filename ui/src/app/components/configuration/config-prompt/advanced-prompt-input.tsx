import type { FC } from 'react';
import React, { useRef } from 'react';
import { useBoolean } from 'ahooks';
import MessageTypeSelector from './message-type-selector';
import type { PromptRole } from '@/models/debug';
import { DeleteIcon } from '@/app/components/Icon/delete';
import { IButton } from '@/app/components/Form/Button';
import { CloseIcon } from '@/app/components/Icon/Close';
import { ExpandIcon } from '@/app/components/Icon/Expand';
import { TickIcon } from '@/app/components/Icon/Tick';
import { CopyIcon } from '@/app/components/Icon/Copy';
import { cn } from '@/styles/media';
import { useToggleExpend } from '@/hooks/use-toggle-expend';
import PromptEditor from '@/app/components/prompt-editor';
import { Maximize2, Minimize2 } from 'lucide-react';
type PromptEditorProps = {
  type?: PromptRole;
  isChatMode: boolean;
  value: string;
  onTypeChange: (value: PromptRole) => void;
  onChange: (value: string) => void;
  canDelete: boolean;
  onDelete: () => void;
  className?: string;
  instanceId?: string;
};

const AdvancedPromptInput: FC<PromptEditorProps> = ({
  type,
  value,
  onChange,
  onTypeChange,
  canDelete,
  onDelete,
  className,
}) => {
  // expand feature
  const ref = useRef<HTMLDivElement>(null);
  const { wrapClassName, isExpand, setIsExpand } = useToggleExpend(ref);

  //   focus
  const [isFocus, { setTrue: setFocus, setFalse: setBlur }] = useBoolean(false);

  //   is checked when copy
  const [isChecked, { setTrue: setChecked, setFalse: setUnCheck }] =
    useBoolean(false);

  const handlePromptChange = (newValue: string) => {
    if (value === newValue) return;
    onChange(newValue);
  };

  const copyItem = (item: string) => {
    setChecked();
    navigator.clipboard.writeText(item);
    setTimeout(() => {
      setUnCheck();
    }, 4000); // Reset back after 2 seconds
  };

  return (
    <div className={cn(wrapClassName)}>
      <div
        ref={ref}
        className={cn(
          'outline-solid outline-[1.5px] outline-transparent',
          'focus-within:outline-blue-600 focus:outline-blue-600 outline-offset-[-1.5px]',
          'border-b border-gray-400 dark:border-gray-600',
          'dark:focus:border-blue-600 focus:border-blue-600',
          'transition-all duration-200 ease-in-out',
          'relative',
          'bg-white dark:bg-gray-950',
          isFocus && 'border-blue-600! outline-blue-600! ',
          isExpand ? 'h-full  z-50' : '',
          className,
        )}
      >
        <div
          className={cn(
            'flex justify-between items-center rounded-t-lg',
            'border-b dark:border-gray-800',
          )}
        >
          <MessageTypeSelector value={type} onChange={onTypeChange} />
          {canDelete && (
            <IButton
              onClick={onDelete}
              tabIndex={-1}
              className="hover:border-red-600  dark:hover:border-red-600 transition-colors border border-transparent border-l-gray-200 dark:border-l-gray-800"
            >
              <DeleteIcon className="w-4 h-4 text-red-600" />
            </IButton>
          )}
          <IButton
            tabIndex={-1}
            onClick={() => {
              copyItem(value);
            }}
            className="hover:border-blue-600  dark:hover:border-blue-600  transition-colors border border-transparent border-l-gray-200 dark:border-l-gray-800"
          >
            {isChecked ? (
              <TickIcon className="h-4 w-4 text-green-600" />
            ) : (
              <CopyIcon className="h-4 w-4" />
            )}
          </IButton>
          <IButton
            tabIndex={-1}
            onClick={() => {
              setIsExpand(!isExpand);
            }}
            className="hover:border-blue-600 dark:hover:border-blue-600  transition-colors border border-transparent border-l-gray-200 dark:border-l-gray-800"
          >
            {isExpand ? (
              <Minimize2 className="h-4 w-4" strokeWidth={1.5} />
            ) : (
              <Maximize2 className="h-4 w-4" strokeWidth={1.5} />
            )}
          </IButton>
        </div>

        <PromptEditor
          className={cn(
            'min-h-[200px]',
            isExpand ? 'py-2 px-4 h-screen' : 'py-1 px-2',
          )}
          placeholder={`Write your prompt here. enter {{variable}} to insert a variable.`}
          value={value}
          onFocus={setFocus}
          onChange={handlePromptChange}
          onBlur={setBlur}
        />
      </div>
    </div>
  );
};
export default React.memo(AdvancedPromptInput);
