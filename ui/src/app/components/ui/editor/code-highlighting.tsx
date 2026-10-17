import { useTheme } from '@/theme/theme-provider';
import { cn } from '@/utils';
import type { HTMLAttributes } from 'react';
import { memo } from 'react';
import Editor, { OnMount } from '@monaco-editor/react';
import '@/monaco';
import { CopyButton } from '@/app/components/ui/primitives/buttons/copy-button';
import {
  MONACO_EDITOR_CONTENT_PADDING,
  MONACO_EDITOR_VERTICAL_PADDING,
} from './monaco-editor-spacing';

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
        'code-highlighting relative bg-[var(--cds-field)] border border-[var(--cds-border-subtle-01)]',
        'p-0 m-0 flex flex-1',
        className,
      )}
    >
      <Editor
        className={cn('code-highlighting-editor flex flex-1', className)}
        language={language}
        value={code}
        theme={resolvedMode === 'dark' ? 'vs-dark' : 'vs'}
        onMount={handleEditorDidMount}
        options={{
          glyphMargin: false,
          readOnly: true,
          lineNumbers: 'off',
          folding: false,
          lineDecorationsWidth: MONACO_EDITOR_CONTENT_PADDING,
          padding: MONACO_EDITOR_VERTICAL_PADDING,
          lineNumbersMinChars: 0,
          wordWrap: 'on',
          fontSize: 15,
          fontFamily: 'IBM Plex Mono',
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
