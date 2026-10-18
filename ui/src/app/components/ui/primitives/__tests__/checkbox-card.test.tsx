import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import CheckboxCard from '../checkbox-card';

jest.mock('@carbon/react', () => ({
  RadioTile: ({
    children,
    checked,
    className,
    disabled,
    id,
    name,
    onChange,
    required,
    tabIndex,
    value,
  }: any) => (
    <label className={className}>
      <input
        checked={checked}
        disabled={disabled}
        id={id}
        name={name}
        onChange={event => onChange?.(value, name, event)}
        required={required}
        tabIndex={tabIndex}
        type="radio"
        value={value}
        readOnly={!onChange}
      />
      {children}
    </label>
  ),
}));

describe('CheckboxCard', () => {
  it('renders children through the design-system radio tile', () => {
    render(
      <CheckboxCard
        id="hybrid-search-type"
        name="search-type"
        value="hybrid"
        checked
        onChange={jest.fn()}
      >
        Hybrid Search
      </CheckboxCard>,
    );

    const radio = screen.getByRole('radio', { name: 'Hybrid Search' });
    expect(radio).toBeChecked();
    expect(radio).toHaveAttribute('name', 'search-type');
    expect(radio).toHaveAttribute('value', 'hybrid');
  });

  it('keeps the existing input change handler contract', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const selectedValues: string[] = [];

    render(
      <CheckboxCard
        id="semantic-search-type"
        name="search-type"
        value="semantic"
        onChange={event => selectedValues.push(event.currentTarget.value)}
      >
        Semantic Search
      </CheckboxCard>,
    );

    await user.click(screen.getByRole('radio', { name: 'Semantic Search' }));

    expect(selectedValues).toEqual(['semantic']);
  });
});
