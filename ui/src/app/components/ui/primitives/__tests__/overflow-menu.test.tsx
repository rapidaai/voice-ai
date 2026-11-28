import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { OverflowMenu, OverflowMenuItem } from '../overflow-menu';

jest.mock('@carbon/react', () => ({
  OverflowMenu: ({ children, label, menuAlignment, ...props }: any) => (
    <div
      className={props.className}
      data-testid="overflow-menu"
      data-menu-alignment={menuAlignment}
    >
      <button type="button" aria-label={label}>
        Open
      </button>
      {children}
    </div>
  ),
  MenuItem: ({ label, kind, ...props }: any) => (
    <button type="button" data-kind={kind} {...props}>
      {label}
    </button>
  ),
  MenuItemDivider: () => <hr data-testid="menu-item-divider" />,
}));

describe('OverflowMenu', () => {
  it('maps the app menu API to the v12 Carbon menu and handles actions', async () => {
    const user = userEvent.setup();
    const onClick = jest.fn();

    render(
      <OverflowMenu
        className="deployment-menu"
        size="sm"
        flipped
        aria-label="Deployment actions"
      >
        <OverflowMenuItem itemText="Edit deployment" onClick={onClick} />
      </OverflowMenu>,
    );

    expect(
      screen.getByRole('button', { name: 'Deployment actions' }),
    ).toBeInTheDocument();
    expect(screen.getByTestId('overflow-menu')).toHaveAttribute(
      'data-menu-alignment',
      'bottom-end',
    );
    expect(screen.getByTestId('overflow-menu')).toHaveClass('deployment-menu');
    expect(
      screen.getByRole('button', { name: 'Deployment actions' }),
    ).not.toHaveClass('deployment-menu');

    await user.click(screen.getByRole('button', { name: 'Edit deployment' }));

    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('preserves disabled, danger, divider, and top alignment behavior', () => {
    render(
      <OverflowMenu size="md" direction="top" iconDescription="Node options">
        <OverflowMenuItem itemText="Delete node" disabled isDelete hasDivider />
      </OverflowMenu>,
    );

    expect(screen.getByTestId('overflow-menu')).toHaveAttribute(
      'data-menu-alignment',
      'top-start',
    );
    expect(screen.getByTestId('menu-item-divider')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Delete node' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Delete node' })).toHaveAttribute(
      'data-kind',
      'danger',
    );
  });
});
