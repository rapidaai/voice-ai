import type { ReactNode } from 'react';
import { Tabs } from '@/app/components/ui/primitives';
import { CodeHighlighting } from '@/app/components/ui/editor/code-highlighting';

const logTabsClassName =
  '[&_.cds--tabs__nav]:border-b [&_.cds--tabs__nav]:border-border-subtle [&_.cds--tab-content]:!h-full [&_.cds--tab-content]:!min-h-0 [&_.cds--tab-content]:!p-0';

interface LogTabsProps {
  tabs: string[];
  selectedIndex: number;
  onChange: (index: number) => void;
  label: string;
  children: ReactNode;
}

export function LogTabs({
  tabs,
  selectedIndex,
  onChange,
  label,
  children,
}: LogTabsProps) {
  return (
    <Tabs
      tabs={tabs}
      selectedIndex={selectedIndex}
      onChange={onChange}
      contained
      aria-label={label}
      fill
      className={logTabsClassName}
      panelClassName="!h-full !min-h-0 !overflow-auto !p-0"
    >
      {children}
    </Tabs>
  );
}

interface LogCodePanelProps {
  value: unknown;
}

export function LogCodePanel({ value }: LogCodePanelProps) {
  return (
    <div className="h-full min-h-0 w-full">
      <CodeHighlighting
        className="!h-full !min-h-0"
        language="json"
        lineNumbers={false}
        foldGutter={false}
        code={JSON.stringify(value ?? null, null, 2) ?? 'null'}
      />
    </div>
  );
}
