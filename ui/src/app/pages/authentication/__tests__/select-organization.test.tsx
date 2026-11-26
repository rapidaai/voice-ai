import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { SelectOrganizationPage } from '@/app/pages/authentication/select-organization';

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: ({ title }: { title: string }) => <div data-page-title={title} />,
}));

jest.mock('@carbon/react', () => ({
  Link: ({ as: Component = 'a', children, inline: _inline, ...props }: any) => (
    <Component {...props}>{children}</Component>
  ),
}));

jest.mock('@/app/components/ui/primitives', () => ({
  ComboBox: ({
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
        value={selectedItem?.id || ''}
        onChange={event =>
          onChange({
            selectedItem:
              items.find(
                (item: { id: string }) => item.id === event.target.value,
              ) || null,
          })
        }
      >
        <option value="">{placeholder}</option>
        {items.map((item: { id: string }) => (
          <option key={item.id} value={item.id}>
            {itemToString(item)}
          </option>
        ))}
      </select>
    </label>
  ),
  PrimaryButton: ({
    children,
    isLoading: _isLoading,
    renderIcon: _renderIcon,
    ...props
  }: any) => <button {...props}>{children}</button>,
  SecondaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  Text: ({
    as: Component = 'span',
    children,
    variant: _variant,
    ...props
  }: any) => <Component {...props}>{children}</Component>,
}));

const organizations = [
  { id: 'organization-1', name: 'Acme Corporation' },
  { id: 'organization-2', name: 'Northwind Group' },
];

describe('SelectOrganizationPage', () => {
  it('uses the login panel and continues with the selected organization', async () => {
    const user = userEvent.setup();
    const onContinue = jest.fn();

    render(
      <SelectOrganizationPage
        organizations={organizations}
        onContinue={onContinue}
        onSignOut={jest.fn()}
      />,
    );

    expect(screen.getByTestId('organization-selection-panel')).toHaveClass(
      'max-w-[37rem]',
      'bg-surface',
    );
    expect(
      screen.getByRole('heading', { name: 'Select an organization' }),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId('organization-selection-actions'),
    ).not.toHaveClass('grid');
    expect(screen.getByRole('button', { name: 'Continue' })).toHaveClass(
      'mt-6!',
      'w-full!',
      'max-w-none!',
    );
    expect(screen.getByRole('button', { name: 'Continue' })).toBeDisabled();

    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Organization' }),
      'organization-2',
    );
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(onContinue).toHaveBeenCalledWith(organizations[1]);
  });

  it('does not continue without a selection and preserves the sign-out action', async () => {
    const user = userEvent.setup();
    const onContinue = jest.fn();
    const onSignOut = jest.fn();

    render(
      <SelectOrganizationPage
        organizations={[]}
        onContinue={onContinue}
        onSignOut={onSignOut}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Sign out' }));

    expect(screen.getByRole('button', { name: 'Continue' })).toBeDisabled();
    expect(onContinue).not.toHaveBeenCalled();
    expect(onSignOut).toHaveBeenCalledTimes(1);
  });
});
