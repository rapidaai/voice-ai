import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { CardOptionMenu, OptionMenu } from '../menu';

jest.mock('@carbon/react', () => ({
  OverflowMenu: ({
    children,
    className,
    label,
    menuAlignment,
    renderIcon: Icon,
    size,
  }: any) => (
    <div
      className={className}
      data-design-system-overflow-menu
      data-menu-alignment={menuAlignment}
      data-size={size}
    >
      <button type="button" aria-label={label}>
        {Icon ? <Icon /> : null}
      </button>
      <div>{children}</div>
    </div>
  ),
  MenuItem: ({ label, kind, onClick }: any) => (
    <button
      type="button"
      data-design-system-overflow-menu-item
      data-kind={kind}
      onClick={onClick}
    >
      {label}
    </button>
  ),
  MenuItemDivider: () => <hr data-menu-item-divider />,
}));

jest.mock('@carbon/icons-react', () => ({
  ChevronDown: () => <svg data-testid="chevron-down-icon" />,
  OverflowMenuHorizontal: () => <svg data-testid="overflow-menu-icon" />,
}));

describe('Menu', () => {
  it('renders OptionMenu actions with the Carbon overflow menu', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onActionClick = jest.fn();

    render(
      <OptionMenu
        options={[
          {
            label: 'Open',
            onActionClick,
          },
        ]}
      />,
    );

    expect(screen.getByTestId('overflow-menu-icon')).toBeInTheDocument();
    const action = screen.getByRole('button', { name: 'Open' });
    expect(action).toHaveAttribute('data-design-system-overflow-menu-item');

    await user.click(action);

    expect(onActionClick).toHaveBeenCalledTimes(1);
  });

  it('keeps CardOptionMenu trigger classes and icon', () => {
    render(
      <CardOptionMenu
        classNames="menu-container [&_.cds--overflow-menu]:w-9 [&_.cds--overflow-menu]:h-9"
        options={[
          {
            label: 'Re-index',
            onActionClick: jest.fn(),
          },
        ]}
      />,
    );

    expect(screen.getByTestId('chevron-down-icon')).toBeInTheDocument();
    expect(
      document.querySelector('[data-design-system-overflow-menu]'),
    ).toHaveClass(
      'menu-container',
      '[&_.cds--overflow-menu]:w-9',
      '[&_.cds--overflow-menu]:h-9',
    );
  });

  it('maps danger options to Carbon delete menu items with a divider', () => {
    render(
      <OptionMenu
        options={[
          {
            label: 'Delete',
            kind: 'danger',
            onActionClick: jest.fn(),
          },
        ]}
      />,
    );

    const action = screen.getByRole('button', { name: 'Delete' });
    expect(action).toHaveAttribute('data-kind', 'danger');
    expect(document.querySelector('[data-menu-item-divider]')).not.toBeNull();
  });
});
