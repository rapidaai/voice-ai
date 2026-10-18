import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { CardOptionMenu, OptionMenu, OptionMenuItem } from '../menu';

jest.mock('@carbon/react', () => ({
  OverflowMenu: ({
    children,
    className,
    direction,
    flipped,
    iconDescription,
    renderIcon: Icon,
    size,
  }: any) => (
    <div
      data-design-system-overflow-menu
      data-direction={direction}
      data-flipped={String(flipped)}
      data-size={size}
    >
      <button type="button" aria-label={iconDescription} className={className}>
        {Icon ? <Icon /> : null}
      </button>
      <div>{children}</div>
    </div>
  ),
  OverflowMenuItem: ({ itemText, isDelete, hasDivider, onClick }: any) => (
    <button
      type="button"
      data-design-system-overflow-menu-item
      data-has-divider={String(hasDivider)}
      data-is-delete={String(isDelete)}
      onClick={onClick}
    >
      {itemText}
    </button>
  ),
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
            option: <span>Open</span>,
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
        classNames="w-9 h-9"
        options={[
          {
            option: <span>Re-index</span>,
            onActionClick: jest.fn(),
          },
        ]}
      />,
    );

    expect(screen.getByTestId('chevron-down-icon')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Menu' })).toHaveClass('w-9');
    expect(screen.getByRole('button', { name: 'Menu' })).toHaveClass('h-9');
  });

  it('maps danger options to Carbon delete menu items with a divider', () => {
    render(
      <OptionMenu
        options={[
          {
            option: <OptionMenuItem type="danger">Delete</OptionMenuItem>,
            onActionClick: jest.fn(),
          },
        ]}
      />,
    );

    const action = screen.getByRole('button', { name: 'Delete' });
    expect(action).toHaveAttribute('data-is-delete', 'true');
    expect(action).toHaveAttribute('data-has-divider', 'true');
  });
});
