import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { Select } from '../select';
import userEvent from '@testing-library/user-event';

jest.mock('@carbon/react', () => {
  const React = require('react');

  return {
    Select: React.forwardRef(
      (
        {
          children,
          hideLabel,
          labelText,
          ...props
        }: React.SelectHTMLAttributes<HTMLSelectElement> & {
          hideLabel?: boolean;
          labelText: string;
        },
        ref: React.ForwardedRef<HTMLSelectElement>,
      ) => (
        <label>
          {!hideLabel && labelText}
          <select
            ref={ref}
            aria-label={hideLabel ? labelText : undefined}
            data-design-system-select
            {...props}
          >
            {children}
          </select>
        </label>
      ),
    ),
    SelectItem: ({ text, ...props }: any) => <option {...props}>{text}</option>,
  };
});

describe('Select', () => {
  it('renders placeholder and options through the design-system select', () => {
    render(
      <Select
        name="action"
        placeholder="Choose action"
        options={[
          { name: 'Block', value: 'block' },
          { name: 'Allow', value: 'allow' },
        ]}
      />,
    );

    const select = screen.getByRole('combobox', { name: 'action' });
    expect(select).toHaveAttribute('data-design-system-select');
    expect(
      screen.getByRole('option', { name: 'Choose action' }),
    ).toBeDisabled();
    expect(screen.getByRole('option', { name: 'Block' })).toHaveValue('block');
    expect(screen.getByRole('option', { name: 'Allow' })).toHaveValue('allow');
  });

  it('keeps the existing change handler contract', async () => {
    const user = userEvent.setup();
    const selectedValues: string[] = [];

    render(
      <Select
        aria-label="Action"
        onChange={event => selectedValues.push(event.currentTarget.value)}
        options={[
          { name: 'Block', value: 'block' },
          { name: 'Allow', value: 'allow' },
        ]}
      />,
    );

    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Action' }),
      String('allow'),
    );

    expect(selectedValues).toEqual(['allow']);
  });
});
