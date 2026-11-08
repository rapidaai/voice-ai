import { useState } from 'react';
import MarkdownPreview from '@uiw/react-markdown-preview';
import { NodeResizer } from 'reactflow';
import type { AgentflowNode } from '@/app/pages/assistant/actions/agentflow';

type StickyNoteNodeCardProps = {
  node: AgentflowNode;
  selected: boolean;
  note: string;
  noteColor: string;
  onChangeNote: (note: string) => void;
  onResizeEnd: (size: { width: number; height: number }) => void;
};

const stickyNoteColors: Record<
  string,
  { surface: string; fold: string; edge: string }
> = {
  yellow: {
    surface: 'var(--color-note-warning-background)',
    fold: 'var(--color-note-warning-fold)',
    edge: 'var(--color-note-warning-edge)',
  },
  blue: {
    surface: 'var(--color-note-info-background)',
    fold: 'var(--color-note-info-fold)',
    edge: 'var(--color-note-info-edge)',
  },
  green: {
    surface: 'var(--color-note-success-background)',
    fold: 'var(--color-note-success-fold)',
    edge: 'var(--color-note-success-edge)',
  },
  purple: {
    surface: 'var(--color-note-brand-background)',
    fold: 'var(--color-note-brand-fold)',
    edge: 'var(--color-note-brand-edge)',
  },
};
const stickyNoteHighlight =
  'linear-gradient(180deg, color-mix(in srgb, var(--color-inverse) 42%, transparent), transparent)';

export function StickyNoteNodeCard({
  node,
  selected,
  note,
  noteColor,
  onChangeNote,
  onResizeEnd,
}: StickyNoteNodeCardProps) {
  const [editing, setEditing] = useState(false);
  const noteValue = note || '# Quick start';
  const colors = stickyNoteColors[noteColor] ?? stickyNoteColors.yellow;
  const foldBackground = `linear-gradient(135deg, ${colors.fold} 0 50%, color-mix(in srgb, var(--color-inverse) 36%, transparent) 50% 100%)`;

  return (
    <>
      <NodeResizer
        isVisible={selected}
        minWidth={280}
        minHeight={128}
        color="var(--color-data-1)"
        lineClassName="!border !border-border-subtle"
        lineStyle={{ borderWidth: 1, borderColor: 'var(--color-muted)' }}
        handleClassName="!rounded-none"
        handleStyle={{
          width: 8,
          height: 8,
          borderRadius: 0,
          border: '1px solid var(--color-inverse)',
          backgroundColor: 'var(--color-data-1)',
        }}
        onResizeEnd={(_, size) =>
          onResizeEnd({ width: size.width, height: size.height })
        }
      />
      <div
        data-testid="note_node"
        className="relative h-full w-full overflow-hidden rounded-xl text-left shadow-lg transition-shadow hover:shadow-xl"
        style={{
          backgroundColor: colors.surface,
          borderTop: `1px solid ${colors.edge}`,
        }}
        onDoubleClick={() => setEditing(true)}
      >
        <div
          className="pointer-events-none absolute right-0 top-0 h-14 w-14"
          style={{
            background: foldBackground,
            boxShadow:
              '-2px 2px 5px color-mix(in srgb, var(--color-foreground) 12%, transparent)',
          }}
        />
        <div
          className="pointer-events-none absolute inset-x-0 top-0 h-3"
          style={{
            background: stickyNoteHighlight,
          }}
        />
        <div className="flex h-full min-h-0 p-2 pr-14">
          {editing ? (
            <textarea
              id={`${node.id}-sticky-note`}
              value={noteValue}
              placeholder="# Quick start"
              onChange={event => onChangeNote(event.target.value)}
              onBlur={() => setEditing(false)}
              autoFocus
              rows={4}
              aria-label="Sticky note"
              className="nodrag nopan block h-full min-h-0 w-full flex-1 resize-none self-stretch border-0 bg-transparent p-0 text-3xl font-bold leading-10 !text-foreground outline-none placeholder:text-foreground/60 focus:outline-none"
            />
          ) : (
            <div className="h-full w-full overflow-auto">
              <MarkdownPreview
                source={noteValue}
                className="!bg-transparent !text-foreground [&_*]:!text-foreground [&_.wmde-markdown]:!bg-transparent [&_.wmde-markdown]:!text-foreground [&_a]:!underline [&_code]:!bg-surface-inverse/10 [&_h1]:!mb-0 [&_h1]:!text-5xl [&_h1]:!font-bold [&_h1]:!leading-tight [&_h2]:!text-4xl [&_p]:!text-2xl [&_p]:!leading-9"
              />
            </div>
          )}
        </div>
      </div>
    </>
  );
}
