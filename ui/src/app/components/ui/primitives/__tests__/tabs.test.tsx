import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { Tab, Tabs } from '../tabs';

jest.mock('@carbon/react', () => {
  const React = require('react');
  const TabsContext = React.createContext({
    selectedIndex: 0,
    onChange: (_state: { selectedIndex: number }) => {},
  });

  return {
    Tabs: ({ children, onChange, selectedIndex }: any) => (
      <TabsContext.Provider value={{ onChange, selectedIndex }}>
        <div data-design-system-tabs>{children}</div>
      </TabsContext.Provider>
    ),
    TabList: ({ children }: any) => (
      <div role="tablist">
        {React.Children.map(
          children,
          (child: React.ReactElement, index: number) =>
            React.cloneElement(child, { index }),
        )}
      </div>
    ),
    Tab: ({ children, index }: any) => {
      const { onChange, selectedIndex } = React.useContext(TabsContext);
      return (
        <button
          type="button"
          role="tab"
          aria-selected={selectedIndex === index}
          onClick={() => onChange({ selectedIndex: index })}
        >
          {children}
        </button>
      );
    },
    TabPanels: ({ children }: any) => {
      const { selectedIndex } = React.useContext(TabsContext);
      return (
        <div>
          {React.Children.map(
            children,
            (child: React.ReactElement, index: number) =>
              React.cloneElement(child, { hidden: selectedIndex !== index }),
          )}
        </div>
      );
    },
    TabPanel: ({ children, hidden }: any) => (
      <div role="tabpanel" hidden={hidden}>
        {children}
      </div>
    ),
    TabsSkeleton: ({ className }: any) => (
      <div className={className} data-testid="tabs-skeleton" />
    ),
  };
});

describe('Tab', () => {
  it('keeps an associated panel for every tab when content renders elsewhere', () => {
    render(<Tabs tabs={['Messages', 'Details']} />);

    expect(screen.getAllByRole('tabpanel', { hidden: true })).toHaveLength(2);
  });

  it('selects the tab matching the active label', () => {
    render(
      <Tab
        active="Metadata"
        tabs={[
          { label: 'Output', element: <div>Output panel</div> },
          { label: 'Metadata', element: <div>Metadata panel</div> },
        ]}
      />,
    );

    expect(screen.getByRole('tab', { name: 'Metadata' })).toHaveAttribute(
      'aria-selected',
      'true',
    );
    expect(
      screen.getByText('Metadata panel').closest('[role="tabpanel"]'),
    ).not.toHaveAttribute('hidden');
  });

  it('changes selected panels through Carbon tabs', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <Tab
        active="Output"
        tabs={[
          { label: 'Output', element: <div>Output panel</div> },
          { label: 'Metrics', element: <div>Metrics panel</div> },
        ]}
      />,
    );

    await user.click(screen.getByRole('tab', { name: 'Metrics' }));

    expect(screen.getByRole('tab', { name: 'Metrics' })).toHaveAttribute(
      'aria-selected',
      'true',
    );
    expect(
      screen.getByText('Metrics panel').closest('[role="tabpanel"]'),
    ).not.toHaveAttribute('hidden');
  });

  it('only mounts the active panel when strict is false', () => {
    render(
      <Tab
        strict={false}
        active="Output"
        tabs={[
          { label: 'Output', element: <div>Output panel</div> },
          { label: 'Metrics', element: <div>Metrics panel</div> },
        ]}
      />,
    );

    expect(screen.getByText('Output panel')).toBeInTheDocument();
    expect(screen.queryByText('Metrics panel')).not.toBeInTheDocument();
  });
});
