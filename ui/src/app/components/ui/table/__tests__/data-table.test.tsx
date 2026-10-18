import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { ScrollableResizableTable } from '../data-table';

jest.mock('@/app/components/ui/primitives/input-checkbox', () => ({
  InputCheckbox: ({ onChange, ...props }: any) => (
    <input
      {...props}
      data-design-system-checkbox
      type="checkbox"
      onChange={onChange}
    />
  ),
}));

describe('ScrollableResizableTable', () => {
  it('renders the select-all action through the design-system checkbox', () => {
    render(
      <ScrollableResizableTable
        clms={[{ name: 'Name', key: 'name' }]}
        ontoggle={jest.fn()}
      >
        <tr>
          <td>Assistant</td>
        </tr>
      </ScrollableResizableTable>,
    );

    const checkbox = screen.getByRole('checkbox', { name: 'Select all rows' });
    expect(checkbox).toHaveAttribute('data-design-system-checkbox');
  });

  it('keeps the select-all checked callback contract', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const ontoggle = jest.fn();

    render(
      <ScrollableResizableTable
        clms={[{ name: 'Name', key: 'name' }]}
        ontoggle={ontoggle}
      >
        <tr>
          <td>Assistant</td>
        </tr>
      </ScrollableResizableTable>,
    );

    await user.click(screen.getByRole('checkbox', { name: 'Select all rows' }));

    expect(ontoggle).toHaveBeenCalledWith(true);
  });

  it('uses configured column widths instead of the default width', () => {
    render(
      <ScrollableResizableTable
        clms={[
          { name: 'Name', key: 'name', width: 320 },
          { name: 'Status', key: 'status' },
        ]}
        ontoggle={jest.fn()}
      >
        <tr>
          <td>Assistant</td>
          <td>Ready</td>
        </tr>
      </ScrollableResizableTable>,
    );

    expect(screen.getByText('Name')).toHaveStyle({ width: '320px' });
    expect(screen.getByText('Status')).toHaveStyle({ width: '200px' });
  });
});
