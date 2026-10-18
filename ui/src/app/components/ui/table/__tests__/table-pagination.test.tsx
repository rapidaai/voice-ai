import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { TablePagination } from '../table-pagination';

jest.mock('@/app/components/ui/primitives/button', () => ({
  IconOnlyButton: ({
    iconDescription,
    kind,
    renderIcon: Icon,
    size,
    ...props
  }: any) => (
    <button
      {...props}
      aria-label={iconDescription}
      data-design-system-icon-button-kind={kind}
      data-design-system-icon-button-size={size}
    >
      {Icon ? <Icon /> : null}
    </button>
  ),
}));

jest.mock('@/app/components/ui/primitives/pagination', () => ({
  Pagination: ({
    backwardText,
    className,
    forwardText,
    onChange,
    page,
    pageSize,
    pageSizes,
    totalItems,
  }: any) => {
    const totalPages = Math.max(Math.ceil(totalItems / pageSize), 1);

    return (
      <nav
        className={className}
        data-design-system-pagination
        data-page={page}
        data-page-size={pageSize}
        data-total-items={totalItems}
      >
        <button
          type="button"
          disabled={page <= 1}
          onClick={() => onChange({ page: page - 1, pageSize })}
        >
          {backwardText}
        </button>
        <button
          type="button"
          disabled={page >= totalPages}
          onClick={() => onChange({ page: page + 1, pageSize })}
        >
          {forwardText}
        </button>
        <button
          type="button"
          onClick={() => onChange({ page, pageSize: pageSizes[1] })}
        >
          Change page size
        </button>
      </nav>
    );
  },
}));

jest.mock('@/app/components/dialogs/shared/column-preference-modal', () => ({
  ColumnPreferencesDialog: ({ open }: any) =>
    open ? <div data-testid="column-preferences-dialog" /> : null,
}));

jest.mock('@carbon/icons-react', () => ({
  SettingsAdjust: () => <svg data-testid="settings-icon" />,
}));

describe('TablePagination', () => {
  it('renders Carbon pagination with the settings action', () => {
    render(
      <TablePagination
        currentPage={2}
        pageSize={10}
        totalItem={30}
        onChangeCurrentPage={jest.fn()}
      />,
    );

    const pagination = screen.getByRole('navigation');
    expect(pagination).toHaveAttribute('data-design-system-pagination');
    expect(pagination).toHaveAttribute('data-page', '2');
    expect(pagination).toHaveAttribute('data-page-size', '10');
    expect(pagination).toHaveAttribute('data-total-items', '30');
    expect(screen.getByTestId('settings-icon')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Configure column preference' }),
    ).toHaveAttribute('data-design-system-icon-button-kind', 'ghost');
  });

  it('changes pages, updates page size, and opens column preferences', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeCurrentPage = jest.fn();
    const onChangePageSize = jest.fn();

    render(
      <TablePagination
        currentPage={2}
        pageSize={10}
        totalItem={30}
        onChangeCurrentPage={onChangeCurrentPage}
        onChangePageSize={onChangePageSize}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Previous' }));
    expect(onChangeCurrentPage).toHaveBeenCalledWith(1);

    await user.click(screen.getByRole('button', { name: 'Next' }));
    expect(onChangeCurrentPage).toHaveBeenCalledWith(3);

    await user.click(screen.getByRole('button', { name: 'Change page size' }));
    expect(onChangePageSize).toHaveBeenCalledWith(20);

    await user.click(
      screen.getByRole('button', { name: 'Configure column preference' }),
    );
    expect(screen.getByTestId('column-preferences-dialog')).toBeInTheDocument();
  });

  it('disables unavailable page directions', () => {
    render(
      <TablePagination
        currentPage={1}
        pageSize={10}
        totalItem={10}
        onChangeCurrentPage={jest.fn()}
      />,
    );

    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();
  });
});
