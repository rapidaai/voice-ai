import { useTheme } from '@/theme/theme-provider';
import { cn } from '@/utils';
import type { HTMLAttributes } from 'react';
import { memo } from 'react';
import Editor, { OnMount } from '@monaco-editor/react';
import '@/monaco';
import { CopyButton } from '@/app/components/ui/primitives/buttons/copy-button';

export interface CodeHighlightingProps extends HTMLAttributes<HTMLDivElement> {
  code: string;
  language?: string;
  lineNumbers?: boolean;
  foldGutter?: boolean;
  editable?: boolean;
}

function CodeHighlightingComponent({
  code,
  language = 'javascript',
  className,
}: CodeHighlightingProps) {
  const { resolvedMode } = useTheme();
  const handleEditorDidMount: OnMount = editor => {
    editor.updateOptions({
      minimap: { enabled: false },
      scrollBeyondLastLine: false,
      renderLineHighlight: 'none',
      hideCursorInOverviewRuler: true,
      overviewRulerBorder: false,
    });
  };

  return (
    <div
      className={cn(
        'prose-base! relative bg-light-background dark:bg-gray-950 border',
        'p-0 m-0 flex flex-1',
        className,
      )}
    >
      <Editor
        className={cn('flex flex-1 ', className)}
        language={language}
        value={code}
        theme={resolvedMode === 'dark' ? 'vs-dark' : 'vs'}
        onMount={handleEditorDidMount}
        options={{
          glyphMargin: false,
          readOnly: true,
          lineNumbers: 'off',
          folding: false,
          lineDecorationsWidth: 0,
          lineNumbersMinChars: 0,
          wordWrap: 'on',
          fontSize: 15,
          fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
          automaticLayout: true,
          tabSize: language === 'html' ? 2 : 4,
          insertSpaces: true,
          formatOnPaste: true,
          formatOnType: true,
          scrollbar: {
            vertical: 'auto',
            horizontal: 'auto',
            verticalScrollbarSize: 8,
            horizontalScrollbarSize: 8,
          },
        }}
      />
      <div className="absolute top-0 right-0 p-2">
        <CopyButton>{code}</CopyButton>
      </div>
    </div>
  );
}

export const CodeHighlighting = memo(CodeHighlightingComponent);
