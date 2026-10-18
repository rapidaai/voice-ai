import { useState } from 'react';
import type { HTMLAttributes, ReactElement, ReactNode } from 'react';
import {
  Tabs as CarbonTabs,
  TabList as CarbonTabList,
  Tab as CarbonTab,
  TabPanels as CarbonTabPanels,
  TabPanel as CarbonTabPanel,
  TabsSkeleton,
} from '@carbon/react';
import { cn } from '@/utils';

export interface CarbonTabsProps {
  /** Tab labels. */
  tabs?: string[];
  /** Content for each tab. */
  children?: ReactNode | ReactNode[];
  /** Currently selected tab index. */
  selectedIndex?: number;
  /** Called when the selected tab changes. */
  onChange?: (index: number) => void;
  /** Use contained variant (solid background tabs). */
  contained?: boolean;
  /** Stretch tabs and panels to fill the parent flex layout. */
  fill?: boolean;
  /** Accessible label for the tab list. */
  'aria-label'?: string;
  /** Class applied to the Carbon Tabs root container. */
  className?: string;
  /** Class applied to each Carbon TabPanel. */
  panelClassName?: string;
  /** Class applied to the Carbon TabPanels wrapper. */
  panelsClassName?: string;
  /** Show TabsSkeleton instead of the real tabs + content. */
  isLoading?: boolean;
}

/** Carbon Tabs renders tab bar and panels, or a skeleton when loading. */
export function Tabs({
  tabs = [],
  children,
  selectedIndex = 0,
  onChange,
  contained = false,
  fill = false,
  'aria-label': ariaLabel = 'Tabs',
  className,
  panelClassName,
  panelsClassName,
  isLoading = false,
}: CarbonTabsProps) {
  if (isLoading) {
    return <TabsSkeleton className={cn(className)} />;
  }

  const panels = Array.isArray(children)
    ? children
    : children
      ? [children]
      : tabs.map(() => null);

  return (
    <div className={cn(fill && 'flex flex-1 min-h-0 flex-col', className)}>
      <CarbonTabs
        selectedIndex={selectedIndex}
        onChange={({ selectedIndex: idx }: { selectedIndex: number }) =>
          onChange?.(idx)
        }
      >
        <CarbonTabList contained={contained} aria-label={ariaLabel}>
          {tabs.map(label => (
            <CarbonTab key={label}>{label}</CarbonTab>
          ))}
        </CarbonTabList>
        <div
          className={cn(
            fill && 'flex flex-1 min-h-0 overflow-hidden',
            panelsClassName,
          )}
        >
          <CarbonTabPanels>
            {panels.map((panel, idx) => (
              <CarbonTabPanel
                key={idx}
                className={cn(
                  fill && 'flex flex-1 min-h-0 overflow-hidden',
                  panelClassName,
                )}
              >
                {panel}
              </CarbonTabPanel>
            ))}
          </CarbonTabPanels>
        </div>
      </CarbonTabs>
    </div>
  );
}

export interface TabProps extends HTMLAttributes<HTMLDivElement> {
  active: string;
  tabs: {
    label: string;
    labelIcon?: ReactElement;
    element: ReactElement;
  }[];
  strict?: boolean;
  linkClass?: string;
}

const getSelectedIndex = (tabs: TabProps['tabs'], active: string) => {
  const index = tabs.findIndex(tab => tab.label === active);
  return index >= 0 ? index : 0;
};

export function Tab({
  active,
  tabs,
  className,
  strict = true,
  linkClass,
}: TabProps) {
  const [selectedIndex, setSelectedIndex] = useState(() =>
    getSelectedIndex(tabs, active),
  );

  return (
    <div className={cn('flex min-h-0 flex-1 flex-col', className)}>
      <CarbonTabs
        selectedIndex={selectedIndex}
        onChange={({ selectedIndex: idx }: { selectedIndex: number }) =>
          setSelectedIndex(idx)
        }
      >
        <CarbonTabList aria-label="Tabs">
          {tabs.map(tab => (
            <CarbonTab key={tab.label}>
              <span className={cn('inline-flex items-center gap-2', linkClass)}>
                {tab.labelIcon}
                {tab.label}
              </span>
            </CarbonTab>
          ))}
        </CarbonTabList>
        <div className="flex min-h-0 flex-1 overflow-hidden">
          <CarbonTabPanels>
            {tabs.map((tab, idx) => (
              <CarbonTabPanel
                key={tab.label}
                className="flex min-h-0 flex-1 overflow-auto !p-0"
              >
                {strict || idx === selectedIndex ? tab.element : null}
              </CarbonTabPanel>
            ))}
          </CarbonTabPanels>
        </div>
      </CarbonTabs>
    </div>
  );
}

// Re-export raw Carbon components for direct use when needed.
export {
  CarbonTabs as RawTabs,
  CarbonTabList as RawTabList,
  CarbonTab as RawTab,
  CarbonTabPanels as RawTabPanels,
  CarbonTabPanel as RawTabPanel,
  TabsSkeleton,
};
