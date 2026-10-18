import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { Dropdown } from '../listbox-dropdown';

jest.mock('@carbon/react', () => ({
  Dropdown: ({
    disabled,
    direction,
    itemToElement,
    itemToString,
    items,
    label,
    hideLabel,
    onChange,
    renderSelectedItem,
    selectedItem,
    titleText,
    ...props
  }: any) => (
    <div
      data-design-system-dropdown
      data-direction={direction || ''}
      data-disabled={disabled ? 'true' : 'false'}
      data-label={label}
      data-title-text={titleText}
      data-hide-label={hideLabel ? 'true' : 'false'}
      {...props}
    >
      <button type="button">
        {selectedItem
          ? renderSelectedItem?.(selectedItem)
          : itemToString?.(selectedItem) || label}
      </button>
      {items.map((item: any) => (
        <button
          key={itemToString(item)}
          type="button"
          onClick={() => onChange?.({ selectedItem: item })}
        >
          {itemToElement(item)}
        </button>
      ))}
    </div>
  ),
}));

describe('Dropdown', () => {
  it('renders placeholder and options through the design-system dropdown', () => {
    render(
      <Dropdown
        currentValue={null}
        setValue={jest.fn()}
        allValue={['one']}
        placeholder="Choose value"
        option={value => <span>{value}</span>}
      />,
    );

    expect(
      screen.getByRole('button', { name: 'Choose value' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'one' })).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Choose value' }).parentElement,
    ).toHaveAttribute('data-design-system-dropdown');
  });

  it('calls setValue from dropdown changes and marks the selected option', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setValue = jest.fn();

    render(
      <Dropdown
        currentValue="one"
        setValue={setValue}
        allValue={['one', 'two']}
        placeholder="Choose value"
        option={(value, selected) => (
          <span>
            {value}
            {selected ? ' selected' : ''}
          </span>
        )}
      />,
    );

    expect(screen.getByRole('button', { name: 'one' })).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'one selected' }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'two' }));

    expect(setValue).toHaveBeenCalledWith('two');
  });

  it('renders custom selected labels and maps placement to direction', () => {
    const selected = { code: 'openai', name: 'OpenAI' };

    render(
      <Dropdown
        currentValue={selected}
        setValue={jest.fn()}
        allValue={[selected]}
        placeholder="Select provider"
        placement="top"
        label={item => <span>{item.name}</span>}
      />,
    );

    const selectedButton = screen.getAllByRole('button', {
      name: 'OpenAI',
    })[0];
    expect(selectedButton).toBeInTheDocument();
    expect(selectedButton.parentElement).toHaveAttribute(
      'data-direction',
      'top',
    );
  });
});
