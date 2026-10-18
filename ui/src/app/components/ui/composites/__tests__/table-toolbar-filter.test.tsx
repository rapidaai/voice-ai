import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { TableToolbarFilter } from '../table-toolbar-filter';

jest.mock('@carbon/icons-react', () => ({
  Filter: () => <svg data-testid="filter-icon" />,
}));

jest.mock('@carbon/react', () => ({
  Button: ({
    children,
    hasIconOnly,
    iconDescription,
    kind,
    onClick,
    renderIcon: Icon,
    size,
  }: any) => (
    <button
      aria-label={hasIconOnly ? iconDescription : undefined}
      data-design-system-button-kind={kind}
      data-design-system-button-size={size}
      onClick={onClick}
      type="button"
    >
      {Icon ? <Icon /> : null}
      {children}
    </button>
  ),
  Checkbox: ({ checked, id, labelText, onChange }: any) => (
    <label htmlFor={id}>
      <input checked={checked} id={id} onChange={onChange} type="checkbox" />
      {labelText}
    </label>
  ),
  Popover: ({
    align,
    children,
    onRequestClose,
    open,
  }: React.PropsWithChildren<{
    align: string;
    onRequestClose: () => void;
    open: boolean;
  }>) => (
    <div data-align={align} data-open={open ? 'true' : 'false'}>
      {children}
      {open ? (
        <button type="button" onClick={onRequestClose}>
          Close popover
        </button>
      ) : null}
    </div>
  ),
  PopoverContent: ({ children, className }: React.PropsWithChildren<any>) => (
    <section className={className} data-design-system-popover-content>
      {children}
    </section>
  ),
}));

describe('TableToolbarFilter', () => {
  it('renders the Carbon filter trigger and opens a Carbon popover', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <TableToolbarFilter
        filters={[{ id: 'llm', label: 'LLM' }]}
        activeFilters={new Set(['llm'])}
        onApplyFilter={jest.fn()}
        onResetFilter={jest.fn()}
      />,
    );

    const trigger = screen.getByRole('button', { name: 'Filter' });

    expect(trigger).toHaveAttribute('data-design-system-button-kind', 'ghost');
    expect(screen.getByTestId('filter-icon')).toBeInTheDocument();
    expect(
      screen.queryByRole('checkbox', { name: 'LLM' }),
    ).not.toBeInTheDocument();

    await user.click(trigger);

    expect(screen.getByRole('checkbox', { name: 'LLM' })).toBeChecked();
    expect(
      screen
        .getByText('Filter by type')
        .closest('[data-design-system-popover-content]'),
    ).toBeInTheDocument();
  });

  it('applies local filter changes from the popover', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onApplyFilter = jest.fn();

    render(
      <TableToolbarFilter
        filters={[
          { id: 'llm', label: 'LLM' },
          { id: 'tool', label: 'Tool' },
        ]}
        activeFilters={new Set(['llm'])}
        onApplyFilter={onApplyFilter}
        onResetFilter={jest.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Filter' }));
    await user.click(screen.getByRole('checkbox', { name: 'Tool' }));
    await user.click(screen.getByRole('button', { name: 'Apply' }));

    expect(onApplyFilter).toHaveBeenCalledWith(new Set(['llm', 'tool']));
    expect(
      screen.queryByRole('checkbox', { name: 'Tool' }),
    ).not.toBeInTheDocument();
  });

  it('resets filters and honors the Carbon popover close request', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onResetFilter = jest.fn();
    const onReset = jest.fn();

    render(
      <TableToolbarFilter
        filters={[{ id: 'metric', label: 'Metric' }]}
        activeFilters={new Set(['metric'])}
        onApplyFilter={jest.fn()}
        onResetFilter={onResetFilter}
        onReset={onReset}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Filter' }));
    await user.click(screen.getByRole('button', { name: 'Reset' }));

    expect(onResetFilter).toHaveBeenCalledTimes(1);
    expect(onReset).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole('button', { name: 'Filter' }));
    await user.click(screen.getByRole('button', { name: 'Close popover' }));

    expect(
      screen.queryByRole('checkbox', { name: 'Metric' }),
    ).not.toBeInTheDocument();
  });
});
