import { useState, type FC } from 'react';
import { Tag } from '@carbon/react';
import { ChevronDown, ChevronRight } from '@carbon/icons-react';
import type { TimelineDocument, TimelineGroup, TimelineItem } from '../types';
import {
  formatDurationMs,
  formatTime,
  getDocumentColor,
  getDocumentComponent,
} from '../utils';

type ConversationWaterfallProps = {
  groups: TimelineGroup[];
  onSelectDocument: (doc: TimelineDocument) => void;
  selectedDocumentId?: string;
};

const getOutcomeTagType = (outcome: string): 'green' | 'red' | 'gray' => {
  if (outcome === 'success') return 'green';
  if (outcome === 'failure') return 'red';
  return 'gray';
};

const RowBar: FC<{ item: TimelineItem }> = ({ item }) => (
  <div className="relative h-8 min-w-[420px]">
    <div className="absolute left-0 right-0 top-1/2 h-px bg-gray-200 dark:bg-gray-800" />
    <div
      className="absolute top-[7px] h-[18px] border border-white/80 shadow-sm"
      style={{
        left: `${item.offsetPct}%`,
        width: `${Math.min(item.widthPct, 100 - item.offsetPct)}%`,
        backgroundColor: getDocumentColor(item),
      }}
      title={`${item.name} - ${formatDurationMs(item.durationMs)}`}
    />
  </div>
);

const ComponentRow: FC<{
  item: TimelineItem;
  onSelectDocument: (doc: TimelineDocument) => void;
  selected: boolean;
}> = ({ item, onSelectDocument, selected }) => (
  <button
    type="button"
    className={[
      'grid min-w-[880px] grid-cols-[360px_120px_120px_minmax(420px,1fr)] border-t border-gray-100 bg-white text-left hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-950 dark:hover:bg-gray-900',
      selected
        ? 'outline outline-2 -outline-offset-2 outline-[var(--cds-border-interactive)]'
        : '',
    ].join(' ')}
    onClick={() => onSelectDocument(item)}
  >
    <div className="flex min-w-0 items-center gap-2 px-4 py-2 pl-10">
      <span
        className="h-2.5 w-2.5 shrink-0"
        style={{ backgroundColor: getDocumentColor(item) }}
      />
      <div className="min-w-0">
        <p className="truncate text-sm text-gray-900 dark:text-gray-100">
          {item.title || item.name}
        </p>
      </div>
    </div>
    <div className="flex items-center px-3 py-2">
      <Tag type="cool-gray">{getDocumentComponent(item)}</Tag>
    </div>
    <div className="flex items-center px-3 py-2">
      <Tag type={getOutcomeTagType(item.outcome)}>
        {item.outcome || 'unknown'}
      </Tag>
    </div>
    <div className="px-3 py-2">
      <RowBar item={item} />
    </div>
  </button>
);

const GroupRow: FC<{
  group: TimelineGroup;
  expanded: boolean;
  onToggle: () => void;
}> = ({ group, expanded, onToggle }) => (
  <button
    type="button"
    className="grid min-w-[880px] grid-cols-[360px_120px_120px_minmax(420px,1fr)] border-t border-gray-200 bg-gray-50 text-left dark:border-gray-800 dark:bg-gray-900"
    onClick={onToggle}
  >
    <div className="flex min-w-0 items-center gap-2 px-4 py-2">
      {expanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
      <div className="min-w-0">
        <p className="truncate text-sm font-medium text-gray-900 dark:text-gray-100">
          {group.title}
        </p>
      </div>
    </div>
    <div className="flex items-center px-3 py-2 text-xs text-gray-500">
      {group.items.length} records
    </div>
    <div className="flex items-center px-3 py-2 font-mono text-xs text-gray-500">
      {formatDurationMs(group.durationMs)}
    </div>
    <div className="flex items-center px-3 py-2 text-xs text-gray-500">
      {formatTime(new Date(group.startMs).toISOString())}
    </div>
  </button>
);

export const ConversationWaterfall: FC<ConversationWaterfallProps> = ({
  groups,
  selectedDocumentId,
  onSelectDocument,
}) => {
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(
    new Set(),
  );
  const toggleGroup = (contextId: string) => {
    setCollapsedGroups(prev => {
      const next = new Set(prev);
      if (next.has(contextId)) next.delete(contextId);
      else next.add(contextId);
      return next;
    });
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="grid min-w-[880px] grid-cols-[360px_120px_120px_minmax(420px,1fr)] bg-white text-xs font-medium uppercase text-gray-500 dark:bg-gray-950">
        <div className="px-4 py-2">Message / Component</div>
        <div className="px-3 py-2">Component</div>
        <div className="px-3 py-2">Outcome</div>
        <div className="px-3 py-2">Timeline</div>
      </div>
      <div className="min-h-0 flex-1 overflow-auto">
        {groups.map(group => {
          const expanded = !collapsedGroups.has(group.contextId);
          return (
            <div key={group.contextId}>
              <GroupRow
                group={group}
                expanded={expanded}
                onToggle={() => toggleGroup(group.contextId)}
              />
              {expanded &&
                group.items.map(item => (
                  <ComponentRow
                    key={item.id}
                    item={item}
                    selected={item.id === selectedDocumentId}
                    onSelectDocument={onSelectDocument}
                  />
                ))}
            </div>
          );
        })}
      </div>
    </div>
  );
};
