import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { DateFilter } from '../date-filter';

const mockSelectedDates = [new Date(2026, 8, 18), new Date(2026, 8, 19)];

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
  DatePicker: ({ children, onChange }: React.PropsWithChildren<any>) => (
    <div>
      {children}
      <button type="button" onClick={() => onChange(mockSelectedDates)}>
        Select dates
      </button>
    </div>
  ),
  DatePickerInput: ({ labelText }: { labelText: string }) => (
    <label>{labelText}</label>
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

describe('DateFilter', () => {
  it('renders the Carbon trigger and opens a Carbon popover', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<DateFilter onApply={jest.fn()} />);

    const trigger = screen.getByRole('button', { name: 'Filter by date' });

    expect(trigger).toHaveAttribute('data-design-system-button-kind', 'ghost');
    expect(screen.getByTestId('filter-icon')).toBeInTheDocument();
    expect(screen.queryByText('From')).not.toBeInTheDocument();

    await user.click(trigger);

    expect(screen.getByText('From')).toBeInTheDocument();
    expect(
      screen
        .getByText('Filter by date')
        .closest('[data-design-system-popover-content]'),
    ).toBeInTheDocument();
  });

  it('applies the selected date range', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onApply = jest.fn();

    render(<DateFilter onApply={onApply} />);

    await user.click(screen.getByRole('button', { name: 'Filter by date' }));
    await user.click(screen.getByRole('button', { name: 'Select dates' }));
    await user.click(screen.getByRole('button', { name: 'Apply' }));

    expect(onApply).toHaveBeenCalledWith(
      mockSelectedDates[0],
      mockSelectedDates[1],
    );
    expect(screen.queryByText('From')).not.toBeInTheDocument();
  });

  it('resets local dates and honors Carbon popover close requests', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onReset = jest.fn();

    render(<DateFilter onApply={jest.fn()} onReset={onReset} />);

    await user.click(screen.getByRole('button', { name: 'Filter by date' }));
    await user.click(screen.getByRole('button', { name: 'Select dates' }));
    await user.click(screen.getByRole('button', { name: 'Reset' }));

    expect(onReset).toHaveBeenCalledTimes(1);
    expect(screen.queryByText('From')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Filter by date' }));
    await user.click(screen.getByRole('button', { name: 'Close popover' }));

    expect(screen.queryByText('From')).not.toBeInTheDocument();
  });
});
