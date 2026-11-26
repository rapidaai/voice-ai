import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { ComboBox } from '../dropdown';

jest.mock('@carbon/react', () => ({
  ComboBox: ({
    disabled,
    id,
    items,
    itemToString,
    onChange,
    placeholder,
    selectedItem,
    titleText,
  }: any) => (
    <label htmlFor={id}>
      {titleText}
      <select
        id={id}
        aria-label={placeholder}
        disabled={disabled}
        value={selectedItem?.value || ''}
        onChange={event =>
          onChange({
            selectedItem:
              items.find((item: any) => item.value === event.target.value) ||
              null,
          })
        }
      >
        <option value="">{placeholder}</option>
        {items.map((item: any) => (
          <option key={item.value} value={item.value}>
            {itemToString(item)}
          </option>
        ))}
      </select>
    </label>
  ),
  Dropdown: () => null,
  DropdownSkeleton: () => null,
  MultiSelect: () => null,
}));

describe('ComboBox', () => {
  const roles = [
    { name: 'Project admin', value: 'admin' },
    { name: 'Project reader', value: 'reader' },
  ];

  it('delegates searchable selection to the Carbon combobox', async () => {
    const user = userEvent.setup();
    const onChange = jest.fn();

    render(
      <ComboBox
        id="role"
        titleText="Project role"
        placeholder="Search or select a role"
        items={roles}
        selectedItem={roles[0]}
        itemToString={item => item?.name || ''}
        onChange={onChange}
      />,
    );

    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Search or select a role' }),
      'reader',
    );

    expect(onChange).toHaveBeenCalledWith({ selectedItem: roles[1] });
  });

  it('preserves the disabled state', () => {
    render(
      <ComboBox
        id="role"
        titleText="Project role"
        placeholder="Search or select a role"
        items={roles}
        disabled
      />,
    );

    expect(screen.getByRole('combobox')).toBeDisabled();
  });
});
