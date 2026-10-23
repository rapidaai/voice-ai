import type { ReactNode } from 'react';
import { Fragment } from 'react';
import { Tabs } from '@/app/components/ui/primitives';

interface DeploymentEditTab<T extends string> {
  code: T;
  name: string;
}

interface DeploymentEditTabsProps<T extends string> {
  tabs: readonly DeploymentEditTab<T>[];
  activeTab: T;
  onChange: (tab: T) => void;
  panels: Record<T, ReactNode>;
  'aria-label': string;
}

/** Keeps deployment editor content inside its matching Carbon TabPanel. */
export function DeploymentEditTabs<T extends string>({
  tabs,
  activeTab,
  onChange,
  panels,
  'aria-label': ariaLabel,
}: DeploymentEditTabsProps<T>) {
  const selectedIndex = Math.max(
    tabs.findIndex(tab => tab.code === activeTab),
    0,
  );

  return (
    <Tabs
      contained
      fill
      aria-label={ariaLabel}
      tabs={tabs.map(tab => tab.name)}
      selectedIndex={selectedIndex}
      onChange={index => {
        const nextTab = tabs[index];
        if (nextTab) onChange(nextTab.code);
      }}
      className="w-full"
      tabListClassName="w-full border-y border-border-subtle"
      panelsClassName="w-full"
      panelClassName="!w-full !overflow-auto !p-0"
    >
      {tabs.map(({ code }) => (
        <Fragment key={code}>
          {code === activeTab ? (
            <div data-deployment-form className="w-full max-w-3xl">
              {panels[code]}
            </div>
          ) : null}
        </Fragment>
      ))}
    </Tabs>
  );
}
