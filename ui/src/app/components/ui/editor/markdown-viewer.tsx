import type { HTMLAttributes } from 'react';
import MarkdownPreview from '@uiw/react-markdown-preview';
import { cn } from '@/utils';

interface MarkdownViewerProps extends HTMLAttributes<HTMLDivElement> {
  text?: string;
  editorClassName?: string;
}

export function MarkdownViewer({
  text,
  className,
  editorClassName,
}: MarkdownViewerProps) {
  return (
    <div
      className={cn(
        'flex items-start justify-between group overflow-auto max-h-full w-full p-4 bg-surface ',
        className,
      )}
    >
      <MarkdownPreview
        source={text?.replaceAll('\n', '\n\n')}
        className={cn(
          'markdown-editor',
          'prose prose-lg max-w-none! break-words prose-headings:text-foreground prose-headings:underline prose-p:text-foreground prose-li:text-foreground prose-strong:text-foreground prose-code:text-foreground prose-blockquote:text-muted prose-a:text-link prose-hr:border-border-subtle prose-img:rounded-xl',
          'prose-h1:text-2xl prose-h2:text-xl prose-h3:text-lg prose-p:leading-normal prose-pre:p-0 prose-ol:leading-normal prose-ul:leading-normal prose-li:my-1 max-w-full break-words text-base! md:text-lg! bg-transparent! w-full',
          editorClassName,
        )}
        style={{
          overflowWrap: 'break-word',
          wordBreak: 'break-word',
          maxWidth: '100%',
        }}
      />
    </div>
  );
}
