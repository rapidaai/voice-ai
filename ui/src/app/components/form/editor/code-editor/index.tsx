import type { FC } from 'react';
import { useRef } from 'react';
import { useBoolean } from 'ahooks';
import { IButton } from '@/app/components/form/button';
import { cn } from '@/utils';
import { useToggleExpend } from '@/hooks/use-toggle-expend';
import { JsonEditor } from '@/app/components/json-editor';
import { Check, Copy, Maximize2, Minimize2 } from 'lucide-react';

// Prommpt editor //
type CodeEditorProps = {
  placeholder: string;
  value: string;
  onChange: (value: string) => void;
  className?: string;
};

export const CodeEditor: FC<CodeEditorProps> = ({
  placeholder,
  value,
  onChange,
  className,
}) => {
  // expand feature
  const ref = useRef<HTMLDivElement>(null);
  const { isExpand, setIsExpand } = useToggleExpend(ref);
  const [isFocus, { setTrue: setFocus, setFalse: setBlur }] = useBoolean(false);
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
          'flex items-center absolute right-1 top-1 z-20 invisible group-hover:visible bg-light-background dark:bg-gray-900',
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
            <Check className="h-3.5 w-3.5 text-green-600" strokeWidth={1.5} />
          ) : (
            <Copy className="h-3.5 w-3.5" />
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
            <Minimize2 className="h-3.5 w-3.5" strokeWidth={1.5} />
          ) : (
            <Maximize2 className="h-3.5 w-3.5" strokeWidth={1.5} />
          )}
        </IButton>
      </div>

      <JsonEditor
        className={cn(
          'min-h-52 overflow-auto p-2',
          className,
          isExpand && 'h-screen p-4',
        )}
        placeholder={placeholder}
        value={value}
        onFocus={setFocus}
        onChange={handlePromptChange}
        onBlur={setBlur}
      />
    </div>
  );
};
