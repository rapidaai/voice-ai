import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { TableHead, TableHederWithCheckbox } from '../table-head';

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

describe('TableHead', () => {
  it('renders column headings and the action column label', () => {
    render(
      <table>
        <TableHead
          isActionable
          columns={[
            { name: 'Name', key: 'name' },
            { name: 'Status', key: 'status' },
          ]}
        />
      </table>,
    );

    expect(
      screen.getByRole('columnheader', { name: 'Name' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('columnheader', { name: 'Status' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Menu')).toBeInTheDocument();
  });

  it('uses the design-system checkbox for selectable headers', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const ontoggle = jest.fn();

    render(
      <table>
        <TableHederWithCheckbox
          ontoggle={ontoggle}
          columns={[{ name: 'Name', key: 'name' }]}
        />
      </table>,
    );

    const checkbox = screen.getByRole('checkbox', { name: 'Select all rows' });
    expect(checkbox).toHaveAttribute('data-design-system-checkbox');

    await user.click(checkbox);

    expect(ontoggle).toHaveBeenCalledWith(true);
  });
});
