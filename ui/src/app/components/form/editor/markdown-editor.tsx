import React, { useCallback } from 'react';
import { useBoolean } from 'ahooks';
import Base from './base-editor';
import MarkdownEditor, {
  defaultCommands,
  ICommand,
} from '@uiw/react-markdown-editor';
import { cn } from '@/utils';
import { useDarkMode } from '@/context/dark-mode-context';

type MarkdownTextEditorProps = {
  value: string;
  onChange: (value: string) => void;
  leftAction?: (boolean) => JSX.Element;
  rightAction?: (boolean) => JSX.Element;
  onBlur?: () => void;
  enabledToolbar?: ICommand[];
  placeholder?: string;
  className?: string;
};

export const MarkdownTextEditor = React.forwardRef<
  HTMLDivElement,
  MarkdownTextEditorProps
>(
  (
    {
      value,
      onChange,
      leftAction,
      rightAction,
      onBlur,
      enabledToolbar = [
        defaultCommands.bold,
        defaultCommands.italic,
        defaultCommands.strike,
        defaultCommands.underline,
        defaultCommands.quote,
        defaultCommands.olist,
        defaultCommands.ulist,
        defaultCommands.code,
        defaultCommands.codeBlock,
      ],
      placeholder,
      className,
    }: MarkdownTextEditorProps,
    ref,
  ) => {
    const [isFocus, { setTrue: setIsFocus, setFalse: setIsNotFocus }] =
      useBoolean(false);

    const handleBlur = useCallback(() => {
      setIsNotFocus();
      onBlur?.();
    }, [setIsNotFocus, onBlur]);

    const ctx = useDarkMode();
    return (
      <Base
        value={value}
        isFocus={isFocus}
        leftAction={leftAction}
        rightAction={rightAction}
        className={cn(' max-w-full normal-text', className)}
      >
        <MarkdownEditor
          toolbars={enabledToolbar}
          toolbarsMode={[]}
          value={value}
          height="200px"
          className={cn(
            'text-gray-700! dark:text-gray-300! bg-gray-50! dark:bg-gray-900!',
            'markdown-editor max-w-full ! focus:outline-hidden border-transparent! break-all! cursor-text overflow-auto',
          )}
          enablePreview={false}
          theme={ctx.isDarkMode ? 'dark' : 'light'}
          basicSetup={{
            highlightActiveLine: false,
            highlightActiveLineGutter: false,
            lineNumbers: false,
            foldGutter: false,
          }}
          onChange={e => onChange(e)}
          onFocus={setIsFocus}
          onBlur={handleBlur}
          placeholder={placeholder}
        />
      </Base>
    );
  },
);
export default React.memo(MarkdownTextEditor);
